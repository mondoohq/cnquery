// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	logic "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/logic/armlogic"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAzureSubscriptionLogicService) id() (string, error) {
	return "azure.subscription.logicService/" + a.SubscriptionId.Data, nil
}

func (a *mqlAzureSubscriptionLogicServiceWorkflow) id() (string, error) {
	return a.Id.Data, nil
}

func initAzureSubscriptionLogicService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}
	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	args["subscriptionId"] = llx.StringData(conn.SubId())
	return args, nil, nil
}

func (a *mqlAzureSubscriptionLogicService) workflows() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	subId := a.SubscriptionId.Data

	client, err := logic.NewWorkflowsClient(subId, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	pager := client.NewListBySubscriptionPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlWf, err := logicWorkflowToMQL(a.MqlRuntime, entry)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlWf)
		}
	}
	return res, nil
}

func logicWorkflowToMQL(runtime *plugin.Runtime, entry *logic.Workflow) (plugin.Resource, error) {
	var state, provisioningState, skuName, version, accessEndpoint string
	var integrationAccountId, integrationServiceEnvironmentId string
	identity := map[string]any{}
	endpointsConfig := map[string]any{}
	acTriggers := map[string]any{}
	acContents := map[string]any{}
	acActions := map[string]any{}
	acManagement := map[string]any{}
	hasIpRestrictions := false

	parameters := []any{}
	secureNames := []any{}
	triggers := []any{}
	actions := []any{}
	connectionNames := []any{}

	if entry.Identity != nil {
		d, err := convert.JsonToDict(entry.Identity)
		if err != nil {
			return nil, err
		}
		identity = d
	}

	props := entry.Properties
	if props != nil {
		if props.State != nil {
			state = string(*props.State)
		}
		if props.ProvisioningState != nil {
			provisioningState = string(*props.ProvisioningState)
		}
		if props.SKU != nil && props.SKU.Name != nil {
			skuName = string(*props.SKU.Name)
		}
		if props.Version != nil {
			version = *props.Version
		}
		if props.AccessEndpoint != nil {
			accessEndpoint = *props.AccessEndpoint
		}
		if props.IntegrationAccount != nil && props.IntegrationAccount.ID != nil {
			integrationAccountId = *props.IntegrationAccount.ID
		}
		if props.IntegrationServiceEnvironment != nil && props.IntegrationServiceEnvironment.ID != nil {
			integrationServiceEnvironmentId = *props.IntegrationServiceEnvironment.ID
		}
		if props.EndpointsConfiguration != nil {
			d, err := convert.JsonToDict(props.EndpointsConfiguration)
			if err != nil {
				return nil, err
			}
			endpointsConfig = d
		}
		if props.AccessControl != nil {
			ac := props.AccessControl
			var err error
			acTriggers, err = accessPolicyToDict(ac.Triggers)
			if err != nil {
				return nil, err
			}
			acContents, err = accessPolicyToDict(ac.Contents)
			if err != nil {
				return nil, err
			}
			acActions, err = accessPolicyToDict(ac.Actions)
			if err != nil {
				return nil, err
			}
			acManagement, err = accessPolicyToDict(ac.WorkflowManagement)
			if err != nil {
				return nil, err
			}
			hasIpRestrictions = policyHasIpAllowList(ac.Triggers) ||
				policyHasIpAllowList(ac.Contents) ||
				policyHasIpAllowList(ac.Actions) ||
				policyHasIpAllowList(ac.WorkflowManagement)
		}

		parameters, secureNames = workflowParametersToMQL(props.Parameters)
		triggers, actions, connectionNames = workflowDefinitionToMQL(props.Definition)
	}

	mqlWf, err := CreateResource(runtime, "azure.subscription.logicService.workflow",
		map[string]*llx.RawData{
			"id":                              llx.StringDataPtr(entry.ID),
			"name":                            llx.StringDataPtr(entry.Name),
			"location":                        llx.StringDataPtr(entry.Location),
			"tags":                            llx.MapData(convert.PtrMapStrToInterface(entry.Tags), types.String),
			"state":                           llx.StringData(state),
			"provisioningState":               llx.StringData(provisioningState),
			"skuName":                         llx.StringData(skuName),
			"version":                         llx.StringData(version),
			"accessEndpoint":                  llx.StringData(accessEndpoint),
			"identity":                        llx.DictData(identity),
			"integrationAccountId":            llx.StringData(integrationAccountId),
			"integrationServiceEnvironmentId": llx.StringData(integrationServiceEnvironmentId),
			"createdTime":                     llx.TimeDataPtr(workflowCreatedTime(props)),
			"changedTime":                     llx.TimeDataPtr(workflowChangedTime(props)),
			"endpointsConfiguration":          llx.DictData(endpointsConfig),
			"accessControlTriggers":           llx.DictData(acTriggers),
			"accessControlContents":           llx.DictData(acContents),
			"accessControlActions":            llx.DictData(acActions),
			"accessControlWorkflowManagement": llx.DictData(acManagement),
			"hasIpRestrictions":               llx.BoolData(hasIpRestrictions),
			"parameters":                      llx.ArrayData(parameters, types.Dict),
			"secureParameterNames":            llx.ArrayData(secureNames, types.String),
			"triggers":                        llx.ArrayData(triggers, types.Dict),
			"actions":                         llx.ArrayData(actions, types.Dict),
			"connectionNames":                 llx.ArrayData(connectionNames, types.String),
		})
	if err != nil {
		return nil, err
	}
	return mqlWf, nil
}

func workflowCreatedTime(p *logic.WorkflowProperties) *time.Time {
	if p == nil {
		return nil
	}
	return p.CreatedTime
}

func workflowChangedTime(p *logic.WorkflowProperties) *time.Time {
	if p == nil {
		return nil
	}
	return p.ChangedTime
}

// accessPolicyToDict converts an SDK access-control policy into a dict with
// shape {allowedCallerIpAddresses: []string, openAuthenticationPolicies: {...}}.
// Returns an empty map when the policy is nil so the field always has a stable shape.
func accessPolicyToDict(p *logic.FlowAccessControlConfigurationPolicy) (map[string]any, error) {
	out := map[string]any{
		"allowedCallerIpAddresses":   []any{},
		"openAuthenticationPolicies": map[string]any{},
	}
	if p == nil {
		return out, nil
	}
	ips := []any{}
	for _, r := range p.AllowedCallerIPAddresses {
		if r != nil && r.AddressRange != nil {
			ips = append(ips, *r.AddressRange)
		}
	}
	out["allowedCallerIpAddresses"] = ips
	if p.OpenAuthenticationPolicies != nil {
		d, err := convert.JsonToDict(p.OpenAuthenticationPolicies)
		if err != nil {
			return nil, err
		}
		out["openAuthenticationPolicies"] = d
	}
	return out, nil
}

func policyHasIpAllowList(p *logic.FlowAccessControlConfigurationPolicy) bool {
	if p == nil {
		return false
	}
	for _, r := range p.AllowedCallerIPAddresses {
		if r != nil && r.AddressRange != nil && *r.AddressRange != "" {
			return true
		}
	}
	return false
}

// secureParameterTypes are the parameter types Logic Apps treats as
// secret-bearing — `SecureString` and `SecureObject`. ARM never returns the
// resolved value of a secure parameter on read.
var secureParameterTypes = map[string]bool{
	string(logic.ParameterTypeSecureString): true,
	string(logic.ParameterTypeSecureObject): true,
}

// workflowParametersToMQL flattens the SDK's WorkflowParameter map into a
// stable-ordered slice of {name, type, hasDefaultValue, isSecure} entries
// plus the names of secure-typed parameters. Stable order keeps query
// output deterministic across runs.
func workflowParametersToMQL(params map[string]*logic.WorkflowParameter) ([]any, []any) {
	out := []any{}
	secure := []any{}
	if len(params) == 0 {
		return out, secure
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		p := params[k]
		entry := map[string]any{
			"name":            k,
			"type":            "",
			"hasDefaultValue": false,
			"isSecure":        false,
		}
		if p == nil {
			out = append(out, entry)
			continue
		}
		if p.Type != nil {
			entry["type"] = string(*p.Type)
			if secureParameterTypes[string(*p.Type)] {
				entry["isSecure"] = true
				secure = append(secure, k)
			}
		}
		if p.Value != nil {
			entry["hasDefaultValue"] = true
		}
		out = append(out, entry)
	}
	return out, secure
}

// workflowDefinitionToMQL parses the workflow definition JSON object for
// triggers and actions. Returns ([]trigger-dict, []action-dict, []connectionName).
// Each trigger/action dict has shape {name, type, kind?}.
//
// The Logic Apps definition is `any` in the SDK because it follows the
// Workflow Definition Language schema (deeply nested JSON). For audit
// purposes we only extract the top-level triggers and actions plus the
// `parameters.$connections.value` keys (the API connection names referenced
// by the workflow).
func workflowDefinitionToMQL(definition any) ([]any, []any, []any) {
	triggers := []any{}
	actions := []any{}
	connections := []any{}

	def, ok := definition.(map[string]any)
	if !ok {
		return triggers, actions, connections
	}

	triggers = parseDefinitionMap(def["triggers"], true)
	actions = parseDefinitionMap(def["actions"], false)

	if dp, ok := def["parameters"].(map[string]any); ok {
		if cn, ok := dp["$connections"].(map[string]any); ok {
			if val, ok := cn["value"].(map[string]any); ok {
				keys := make([]string, 0, len(val))
				for k := range val {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					connections = append(connections, k)
				}
			}
		}
	}

	return triggers, actions, connections
}

// parseDefinitionMap turns the `triggers` or `actions` map of the workflow
// definition into a stable-ordered slice of {name, type, kind?} entries.
// Returns an empty slice when the input is missing or not a map.
func parseDefinitionMap(raw any, includeKind bool) []any {
	out := []any{}
	m, ok := raw.(map[string]any)
	if !ok {
		return out
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, name := range keys {
		entry := map[string]any{"name": name, "type": ""}
		body, _ := m[name].(map[string]any)
		if body != nil {
			if t, ok := body["type"].(string); ok {
				entry["type"] = t
			}
			if includeKind {
				if k, ok := body["kind"].(string); ok && k != "" {
					entry["kind"] = k
				}
			}
		}
		out = append(out, entry)
	}
	return out
}
