package launchdarkly

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go"
)

func policyStatementsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MinItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				RESOURCES: {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Optional: true,
					MinItems: 1,
				},
				NOT_RESOURCES: {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Optional: true,
					MinItems: 1,
				},
				ACTIONS: {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Optional: true,
					MinItems: 1,
				},
				NOT_ACTIONS: {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Optional: true,
					MinItems: 1,
				},
				EFFECT: {
					Type:         schema.TypeString,
					Required:     true,
					ValidateFunc: validation.StringInSlice([]string{"allow", "deny"}, false),
				},
			},
		},
	}
}

func validatePolicyStatement(statement map[string]interface{}) error {
	resources := statement[RESOURCES].([]interface{})
	notResources := statement[NOT_RESOURCES].([]interface{})
	actions := statement[ACTIONS].([]interface{})
	notActions := statement[NOT_ACTIONS].([]interface{})
	if len(resources) > 0 && len(notResources) > 0 {
		return errors.New("policy_statements cannot contain both 'resources' and 'not_resources'")
	}
	if len(resources) == 0 && len(notResources) == 0 {
		return errors.New("policy_statements must contain either 'resources' or 'not_resources'")
	}
	if len(actions) > 0 && len(notActions) > 0 {
		return errors.New("policy_statements cannot contain both 'actions' and 'not_actions'")
	}
	if len(actions) == 0 && len(notActions) == 0 {
		return errors.New("policy_statements must contain either 'actions' or 'not_actions'")
	}
	return nil
}

func policyStatementsFromResourceData(d *schema.ResourceData) ([]ldapi.Statement, error) {
	schemaStatements := d.Get(POLICY_STATEMENTS).([]interface{})

	statements := make([]ldapi.Statement, 0, len(schemaStatements))
	for _, stmt := range schemaStatements {
		statement := stmt.(map[string]interface{})
		err := validatePolicyStatement(statement)
		if err != nil {
			return statements, err
		}
		s := policyStatementFromResourceData(statement)
		statements = append(statements, s)
	}
	return statements, nil
}

func policyStatementFromResourceData(statement map[string]interface{}) ldapi.Statement {
	ret := ldapi.Statement{
		Effect: statement[EFFECT].(string),
	}
	for _, r := range statement[RESOURCES].([]interface{}) {
		ret.Resources = append(ret.Resources, r.(string))
	}
	for _, n := range statement[NOT_RESOURCES].([]interface{}) {
		ret.NotResources = append(ret.NotResources, n.(string))
	}
	for _, a := range statement[ACTIONS].([]interface{}) {
		ret.Actions = append(ret.Actions, a.(string))
	}
	for _, n := range statement[NOT_ACTIONS].([]interface{}) {
		ret.NotActions = append(ret.NotActions, n.(string))
	}
	return ret
}

func policyStatementsToResourceData(statements []ldapi.Statement) []interface{} {
	transformed := make([]interface{}, 0, len(statements))
	for _, s := range statements {
		t := map[string]interface{}{
			EFFECT: s.Effect,
		}
		if len(s.Resources) > 0 {
			t[RESOURCES] = stringSliceToInterfaceSlice(s.Resources)
		}
		if len(s.NotResources) > 0 {
			t[NOT_RESOURCES] = stringSliceToInterfaceSlice(s.NotResources)
		}
		if len(s.Actions) > 0 {
			t[ACTIONS] = stringSliceToInterfaceSlice(s.Actions)
		}
		if len(s.NotActions) > 0 {
			t[NOT_ACTIONS] = stringSliceToInterfaceSlice(s.NotActions)
		}
		transformed = append(transformed, t)
	}
	return transformed
}

func statementsToPolicies(statements []ldapi.Statement) []ldapi.Policy {
	policies := make([]ldapi.Policy, 0, len(statements))
	for _, s := range statements {
		policies = append(policies, ldapi.Policy(s))
	}
	return policies
}

func policiesToStatements(policies []ldapi.Policy) []ldapi.Statement {
	statements := make([]ldapi.Statement, 0, len(policies))
	for _, p := range policies {
		statements = append(statements, ldapi.Statement(p))
	}
	return statements
}
