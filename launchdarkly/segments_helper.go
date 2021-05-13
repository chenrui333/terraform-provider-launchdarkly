package launchdarkly

import (
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
)

func baseSegmentSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		DESCRIPTION: {
			Type:     schema.TypeString,
			Optional: true,
		},
		TAGS: tagsSchema(),
		INCLUDED: {
			Type:     schema.TypeList,
			Elem:     &schema.Schema{Type: schema.TypeString},
			Optional: true,
		},
		EXCLUDED: {
			Type:     schema.TypeList,
			Elem:     &schema.Schema{Type: schema.TypeString},
			Optional: true,
		},
		RULES: segmentRulesSchema(),
	}
}

func segmentRead(d *schema.ResourceData, raw interface{}, isDataSource bool) error {
	client := raw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	envKey := d.Get(ENV_KEY).(string)
	segmentKey := d.Get(KEY).(string)

	segmentRaw, res, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.UserSegmentsApi.GetUserSegment(client.ctx, projectKey, envKey, segmentKey)
	})
	segment := segmentRaw.(ldapi.UserSegment)
	if isStatusNotFound(res) && !isDataSource {
		log.Printf("[WARN] failed to find segment %q in project %q, environment %q, removing from state", segmentKey, projectKey, envKey)
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get segment %q of project %q: %s", segmentKey, projectKey, handleLdapiErr(err))
	}

	if isDataSource {
		d.SetId(projectKey + "/" + envKey + "/" + segmentKey)
	}
	_ = d.Set(NAME, segment.Name)
	_ = d.Set(DESCRIPTION, segment.Description)

	err = d.Set(TAGS, segment.Tags)
	if err != nil {
		return fmt.Errorf("failed to set tags on segment with key %q: %v", segmentKey, err)
	}

	err = d.Set(INCLUDED, segment.Included)
	if err != nil {
		return fmt.Errorf("failed to set included on segment with key %q: %v", segmentKey, err)
	}

	err = d.Set(EXCLUDED, segment.Excluded)
	if err != nil {
		return fmt.Errorf("failed to set excluded on segment with key %q: %v", segmentKey, err)
	}

	rules, err := segmentRulesToResourceData(segment.Rules)
	if err != nil {
		return fmt.Errorf("failed to read rules on segment with key %q: %v", segmentKey, err)
	}
	err = d.Set(RULES, rules)
	if err != nil {
		return fmt.Errorf("failed to set excluded on segment with key %q: %v", segmentKey, err)
	}
	return nil
}
