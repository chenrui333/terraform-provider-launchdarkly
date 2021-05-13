package launchdarkly

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	ldapi "github.com/launchdarkly/api-client-go"
	"github.com/stretchr/testify/require"
)

const (
	testAccDataSourceWebhook = `
data "launchdarkly_webhook" "test" {
	id = "%s"
}
`
)

func testAccDataSourceWebhookCreate(client *Client, webhookName string) (*ldapi.Webhook, error) {
	webhookBody := ldapi.WebhookBody{
		Url:  "https://www.example.com",
		Sign: false,
		On:   true,
		Name: webhookName,
		Tags: []string{"terraform"},
		Statements: []ldapi.Statement{
			{
				Resources: []string{"proj/*"},
				Actions:   []string{"turnFlagOn"},
				Effect:    "allow",
			},
		},
	}
	webhookRaw, _, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.WebhooksApi.PostWebhook(client.ctx, webhookBody)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook with name %q: %s", webhookName, handleLdapiErr(err))
	}

	if webhook, ok := webhookRaw.(ldapi.Webhook); ok {
		return &webhook, nil
	}
	return nil, fmt.Errorf("failed to create webhook")
}

func testAccDataSourceWebhookDelete(client *Client, webhookId string) error {
	_, _, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		res, err := client.ld.WebhooksApi.DeleteWebhook(client.ctx, webhookId)
		return nil, res, err
	})
	if err != nil {
		return fmt.Errorf("failed to delete webhook with id %q: %s", webhookId, handleLdapiErr(err))
	}
	return nil
}

func TestAccDataSourceWebhook_noMatchReturnsError(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	webhookId := acctest.RandStringFromCharSet(24, acctest.CharSetAlphaNum)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccDataSourceWebhook, webhookId),
				ExpectError: regexp.MustCompile(fmt.Sprintf(`errors during refresh: failed to get webhook with id "%s": 404 Not Found:`, webhookId)),
			},
		},
	})
}

func TestAccDataSourceWebhook_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	webhookName := "Data Source Test"
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false)
	require.NoError(t, err)
	webhook, err := testAccDataSourceWebhookCreate(client, webhookName)
	require.NoError(t, err)
	defer func() {
		err := testAccDataSourceWebhookDelete(client, webhook.Id)
		require.NoError(t, err)
	}()

	resourceName := "data.launchdarkly_webhook.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceWebhook, webhook.Id),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "id", webhook.Id),
					resource.TestCheckResourceAttr(resourceName, "name", webhookName),
					resource.TestCheckResourceAttr(resourceName, "url", webhook.Url),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "secret", webhook.Secret),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.resources.0", "proj/*"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.actions.0", "turnFlagOn"),
					resource.TestCheckResourceAttr(resourceName, "policy_statements.0.effect", "allow"),
				),
			},
		},
	})
}
