package launchdarkly

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	ldapi "github.com/launchdarkly/api-client-go"
	"github.com/stretchr/testify/require"
)

const (
	testAccDataSourceFeatureFlag = `
data "launchdarkly_feature_flag" "test" {
	key = "%s"
	project_key = "%s"
}
`
)

func TestAccDataSourceFeatureFlag_noMatchReturnsError(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false)
	require.NoError(t, err)
	projectKey := "tf-flag-test-proj"
	projectBody := ldapi.ProjectBody{
		Name: "Terraform Flag Test Project",
		Key:  projectKey,
	}
	project, err := testAccDataSourceProjectCreate(client, projectBody)
	require.NoError(t, err)

	flagKey := "nonexistent-flag"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccDataSourceFeatureFlag, flagKey, project.Key),
				ExpectError: regexp.MustCompile(`errors during refresh: failed to get flag "nonexistent-flag" of project "tf-flag-test-proj": 404 Not Found:`),
			},
		},
	})

	err = testAccDataSourceProjectDelete(client, projectKey)
	require.NoError(t, err)
}

func TestAccDataSourceFeatureFlag_exists(t *testing.T) {
	accTest := os.Getenv("TF_ACC")
	if accTest == "" {
		t.SkipNow()
	}

	projectKey := "flag-test-project"
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), os.Getenv(LAUNCHDARKLY_API_HOST), false)
	require.NoError(t, err)

	flagName := "Flag Data Source Test"
	flagKey := "flag-ds-test"
	flagBody := ldapi.FeatureFlagBody{
		Name: flagName,
		Key:  flagKey,
		Variations: []ldapi.Variation{
			{Value: intfPtr(true)},
			{Value: intfPtr(false)},
		},
		Description: "a flag to test the terraform flag data source",
		Temporary:   true,
		ClientSideAvailability: &ldapi.ClientSideAvailability{
			UsingEnvironmentId: true,
			UsingMobileKey:     false,
		},
	}
	flag, err := testAccDataSourceFeatureFlagScaffold(client, projectKey, flagBody)
	require.NoError(t, err)

	defer func() {
		err := testAccDataSourceProjectDelete(client, projectKey)
		require.NoError(t, err)
	}()

	resourceName := "data.launchdarkly_feature_flag.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceFeatureFlag, flagKey, projectKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "key"),
					resource.TestCheckResourceAttrSet(resourceName, "name"),
					resource.TestCheckResourceAttrSet(resourceName, "project_key"),
					resource.TestCheckResourceAttr(resourceName, "key", flag.Key),
					resource.TestCheckResourceAttr(resourceName, "name", flag.Name),
					resource.TestCheckResourceAttr(resourceName, "description", flag.Description),
					resource.TestCheckResourceAttr(resourceName, "temporary", "true"),
					resource.TestCheckResourceAttr(resourceName, "variations.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "variations.0.value", "true"),
					resource.TestCheckResourceAttr(resourceName, "variations.1.value", "false"),
					resource.TestCheckResourceAttr(resourceName, "id", projectKey+"/"+flag.Key),
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.using_environment_id", "true"),
					resource.TestCheckResourceAttr(resourceName, "client_side_availability.using_mobile_key", "false"),
				),
			},
		},
	})
}
