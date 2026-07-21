// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/policyinsights/armpolicyinsights"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

func initAzureSubscriptionPolicy(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

func (a *mqlAzureSubscriptionPolicyAssignment) id() (string, error) {
	return "azure.subscription.policy.assignment/" + a.AssignmentId.Data, nil
}

// initAzureSubscriptionPolicyAssignment resolves a single policy assignment by
// its resource ID. The assignments() list path passes every field, so this
// only fetches when the resource is created from just an assignmentId (e.g.
// via the typed reference on a policy exemption).
func initAzureSubscriptionPolicyAssignment(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}
	idRaw, ok := args["assignmentId"]
	if !ok {
		return args, nil, nil
	}
	assignmentID, ok := idRaw.Value.(string)
	if !ok || assignmentID == "" {
		return args, nil, nil
	}

	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}

	clientFactory, err := policyClientFactory(conn, conn.SubId())
	if err != nil {
		return nil, nil, err
	}

	resp, err := clientFactory.NewAssignmentsClient().GetByID(context.Background(), assignmentID, nil)
	if err != nil {
		return nil, nil, err
	}

	props := resp.Properties
	if props == nil {
		props = &armpolicy.AssignmentProperties{}
	}
	parameters, err := convert.JsonToDict(props.Parameters)
	if err != nil {
		return nil, nil, err
	}

	args["id"] = llx.StringDataPtr(props.PolicyDefinitionID)
	args["assignmentId"] = llx.StringDataPtr(resp.ID)
	args["name"] = llx.StringDataPtr(props.DisplayName)
	args["scope"] = llx.StringDataPtr(props.Scope)
	args["description"] = llx.StringDataPtr(props.Description)
	args["enforcementMode"] = llx.StringData(string(convert.ToValue(props.EnforcementMode)))
	args["parameters"] = llx.DictData(parameters)

	return args, nil, nil
}

func (a *mqlAzureSubscriptionPolicy) assignments() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	subId := a.SubscriptionId.Data

	armConn, err := getArmSecurityConnection(ctx, conn, subId)
	if err != nil {
		return nil, err
	}

	// The unfiltered subscription list also returns assignments inherited from the
	// management groups that contain the subscription; their scope is a
	// managementGroups path rather than the subscription. Every returned assignment
	// is mapped, inherited ones included.
	pas, err := getPolicyAssignments(ctx, armConn)
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, assignment := range pas.PolicyAssignments {
		assignmentData, err := policyAssignmentArgs(assignment)
		if err != nil {
			return nil, err
		}

		mqlAssignment, err := CreateResource(a.MqlRuntime, "azure.subscription.policy.assignment", assignmentData)
		if err != nil {
			return nil, err
		}

		sysData, err := convert.JsonToDict(assignment.SystemData)
		if err != nil {
			return nil, err
		}
		mqlAssignment.(*mqlAzureSubscriptionPolicyAssignment).cacheSystemData = sysData

		res = append(res, mqlAssignment)
	}
	return res, nil
}

type mqlAzureSubscriptionPolicyAssignmentInternal struct {
	cacheSystemData any
}

func (a *mqlAzureSubscriptionPolicyAssignment) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	return systemMetadataFromRaw(a.MqlRuntime, a.Id.Data, a.cacheSystemData, &a.SystemMetadata)
}

// policyAssignmentArgs maps a single policy assignment to the resource fields.
// The assignment's scope is preserved verbatim, so subscription-scoped and
// management-group-inherited assignments are surfaced identically.
func policyAssignmentArgs(assignment PolicyAssignment) (map[string]*llx.RawData, error) {
	parameters, err := convert.JsonToDict(assignment.Properties.Parameters)
	if err != nil {
		return nil, err
	}

	return map[string]*llx.RawData{
		"id":              llx.StringData(assignment.Properties.PolicyDefinitionID),
		"assignmentId":    llx.StringData(assignment.ID),
		"name":            llx.StringData(assignment.Properties.DisplayName),
		"scope":           llx.StringData(assignment.Properties.Scope),
		"description":     llx.StringData(assignment.Properties.Description),
		"enforcementMode": llx.StringData(assignment.Properties.EnforcementMode),
		"parameters":      llx.DictData(parameters),
	}, nil
}

func policyClientFactory(conn *connection.AzureConnection, subId string) (*armpolicy.ClientFactory, error) {
	return armpolicy.NewClientFactory(subId, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
}

func (a *mqlAzureSubscriptionPolicy) definitions() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()

	clientFactory, err := policyClientFactory(conn, a.SubscriptionId.Data)
	if err != nil {
		return nil, err
	}

	pager := clientFactory.NewDefinitionsClient().NewListPager(nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, def := range page.Value {
			mqlDef, err := newMqlPolicyDefinition(a.MqlRuntime, def)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlDef)
		}
	}
	return res, nil
}

func newMqlPolicyDefinition(runtime *plugin.Runtime, def *armpolicy.Definition) (plugin.Resource, error) {
	props := def.Properties
	if props == nil {
		props = &armpolicy.DefinitionProperties{}
	}

	metadata, err := convert.JsonToDict(props.Metadata)
	if err != nil {
		return nil, err
	}
	parameters, err := convert.JsonToDict(props.Parameters)
	if err != nil {
		return nil, err
	}
	policyRule, err := convert.JsonToDict(props.PolicyRule)
	if err != nil {
		return nil, err
	}

	res, err := CreateResource(runtime, "azure.subscription.policy.definition", map[string]*llx.RawData{
		"__id":        llx.StringDataPtr(def.ID),
		"id":          llx.StringDataPtr(def.ID),
		"name":        llx.StringDataPtr(def.Name),
		"displayName": llx.StringDataPtr(props.DisplayName),
		"description": llx.StringDataPtr(props.Description),
		"policyType":  llx.StringData(string(convert.ToValue(props.PolicyType))),
		"mode":        llx.StringDataPtr(props.Mode),
		"policyRule":  llx.DictData(policyRule),
		"parameters":  llx.DictData(parameters),
		"metadata":    llx.DictData(metadata),
		"version":     llx.StringDataPtr(props.Version),
	})
	if err != nil {
		return nil, err
	}

	sysData, err := convert.JsonToDict(def.SystemData)
	if err != nil {
		return nil, err
	}
	res.(*mqlAzureSubscriptionPolicyDefinition).cacheSystemData = sysData

	return res, nil
}

type mqlAzureSubscriptionPolicyDefinitionInternal struct {
	cacheSystemData any
}

func (a *mqlAzureSubscriptionPolicyDefinition) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	return systemMetadataFromRaw(a.MqlRuntime, a.Id.Data, a.cacheSystemData, &a.SystemMetadata)
}

func (a *mqlAzureSubscriptionPolicy) setDefinitions() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()

	clientFactory, err := policyClientFactory(conn, a.SubscriptionId.Data)
	if err != nil {
		return nil, err
	}

	pager := clientFactory.NewSetDefinitionsClient().NewListPager(nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, setDef := range page.Value {
			props := setDef.Properties
			if props == nil {
				props = &armpolicy.SetDefinitionProperties{}
			}

			metadata, err := convert.JsonToDict(props.Metadata)
			if err != nil {
				return nil, err
			}
			parameters, err := convert.JsonToDict(props.Parameters)
			if err != nil {
				return nil, err
			}
			policyDefinitions, err := convert.JsonToDictSlice(props.PolicyDefinitions)
			if err != nil {
				return nil, err
			}

			mqlSetDef, err := CreateResource(a.MqlRuntime, "azure.subscription.policy.setDefinition", map[string]*llx.RawData{
				"__id":              llx.StringDataPtr(setDef.ID),
				"id":                llx.StringDataPtr(setDef.ID),
				"name":              llx.StringDataPtr(setDef.Name),
				"displayName":       llx.StringDataPtr(props.DisplayName),
				"description":       llx.StringDataPtr(props.Description),
				"policyType":        llx.StringData(string(convert.ToValue(props.PolicyType))),
				"policyDefinitions": llx.ArrayData(policyDefinitions, types.Dict),
				"parameters":        llx.DictData(parameters),
				"metadata":          llx.DictData(metadata),
				"version":           llx.StringDataPtr(props.Version),
			})
			if err != nil {
				return nil, err
			}

			sysData, err := convert.JsonToDict(setDef.SystemData)
			if err != nil {
				return nil, err
			}
			mqlSetDef.(*mqlAzureSubscriptionPolicySetDefinition).cacheSystemData = sysData

			res = append(res, mqlSetDef)
		}
	}
	return res, nil
}

type mqlAzureSubscriptionPolicySetDefinitionInternal struct {
	cacheSystemData any
}

func (a *mqlAzureSubscriptionPolicySetDefinition) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	return systemMetadataFromRaw(a.MqlRuntime, a.Id.Data, a.cacheSystemData, &a.SystemMetadata)
}

// azurePolicyExemption mirrors the Microsoft.Authorization/policyExemptions REST
// resource. Exemptions are not exposed by the armpolicy SDK, so they are fetched
// directly, like policy assignments above.
type azurePolicyExemption struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	SystemData struct {
		CreatedAt      *time.Time `json:"createdAt"`
		LastModifiedAt *time.Time `json:"lastModifiedAt"`
	} `json:"systemData"`
	Properties struct {
		PolicyAssignmentID           string     `json:"policyAssignmentId"`
		PolicyDefinitionReferenceIDs []string   `json:"policyDefinitionReferenceIds"`
		ExemptionCategory            string     `json:"exemptionCategory"`
		DisplayName                  string     `json:"displayName"`
		Description                  string     `json:"description"`
		ExpiresOn                    *time.Time `json:"expiresOn"`
		Metadata                     any        `json:"metadata"`
	} `json:"properties"`
}

type azurePolicyExemptionList struct {
	Value    []azurePolicyExemption `json:"value"`
	NextLink string                 `json:"nextLink"`
}

func (a *mqlAzureSubscriptionPolicy) exemptions() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	subId := a.SubscriptionId.Data

	armConn, err := getArmSecurityConnection(ctx, conn, subId)
	if err != nil {
		return nil, err
	}
	token, err := armConn.GetToken()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/subscriptions/%s/providers/Microsoft.Authorization/policyExemptions?api-version=2022-07-01-preview",
		armConn.host, subId)
	client := http.Client{}
	res := []any{}
	for url != "" {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer "+token.Token)

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		raw, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return nil, errors.New("failed to fetch policy exemptions: " + resp.Status)
		}

		list := azurePolicyExemptionList{}
		if err := json.Unmarshal(raw, &list); err != nil {
			return nil, err
		}
		for i := range list.Value {
			mqlExemption, err := newMqlPolicyExemption(a.MqlRuntime, &list.Value[i])
			if err != nil {
				return nil, err
			}
			res = append(res, mqlExemption)
		}
		url = list.NextLink
	}
	return res, nil
}

func newMqlPolicyExemption(runtime *plugin.Runtime, exemption *azurePolicyExemption) (plugin.Resource, error) {
	metadata, err := convert.JsonToDict(exemption.Properties.Metadata)
	if err != nil {
		return nil, err
	}

	// the exemption scope is the resource ID without the policyExemptions suffix
	scope := exemption.ID
	const marker = "/providers/microsoft.authorization/policyexemptions/"
	if idx := strings.Index(strings.ToLower(scope), marker); idx >= 0 {
		scope = scope[:idx]
	}

	res, err := CreateResource(runtime, "azure.subscription.policy.exemption", map[string]*llx.RawData{
		"__id":                         llx.StringData(exemption.ID),
		"id":                           llx.StringData(exemption.ID),
		"name":                         llx.StringData(exemption.Name),
		"displayName":                  llx.StringData(exemption.Properties.DisplayName),
		"description":                  llx.StringData(exemption.Properties.Description),
		"exemptionCategory":            llx.StringData(exemption.Properties.ExemptionCategory),
		"scope":                        llx.StringData(scope),
		"policyDefinitionReferenceIds": llx.ArrayData(convert.SliceAnyToInterface(exemption.Properties.PolicyDefinitionReferenceIDs), types.String),
		"expiresOn":                    llx.TimeDataPtr(exemption.Properties.ExpiresOn),
		"metadata":                     llx.DictData(metadata),
		"createdAt":                    llx.TimeDataPtr(exemption.SystemData.CreatedAt),
		"updatedAt":                    llx.TimeDataPtr(exemption.SystemData.LastModifiedAt),
	})
	if err != nil {
		return nil, err
	}
	mqlExemption := res.(*mqlAzureSubscriptionPolicyExemption)
	mqlExemption.cachePolicyAssignmentID = exemption.Properties.PolicyAssignmentID
	return mqlExemption, nil
}

// mqlAzureSubscriptionPolicyExemptionInternal caches the raw policy assignment
// ID so the typed policyAssignment() reference can resolve it lazily.
type mqlAzureSubscriptionPolicyExemptionInternal struct {
	cachePolicyAssignmentID string
}

func (a *mqlAzureSubscriptionPolicyExemption) policyAssignment() (*mqlAzureSubscriptionPolicyAssignment, error) {
	if a.cachePolicyAssignmentID == "" {
		a.PolicyAssignment.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	res, err := NewResource(a.MqlRuntime, "azure.subscription.policy.assignment", map[string]*llx.RawData{
		"assignmentId": llx.StringData(a.cachePolicyAssignmentID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionPolicyAssignment), nil
}

func initAzureSubscriptionPolicyComplianceSummary(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

func (a *mqlAzureSubscriptionPolicyComplianceSummary) id() (string, error) {
	return "azure.subscription.policy.complianceSummary/" + a.SubscriptionId.Data, nil
}

func (a *mqlAzureSubscriptionPolicy) states() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	client, err := armpolicyinsights.NewPolicyStatesClient(conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListQueryResultsForSubscriptionPager(armpolicyinsights.PolicyStatesResourceLatest, a.SubscriptionId.Data, nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, ps := range page.Value {
			if ps == nil {
				continue
			}
			resourceId := convert.ToValue(ps.ResourceID)
			policyAssignmentId := convert.ToValue(ps.PolicyAssignmentID)
			policyDefinitionId := convert.ToValue(ps.PolicyDefinitionID)
			policySetDefinitionId := convert.ToValue(ps.PolicySetDefinitionID)
			policyDefinitionReferenceId := convert.ToValue(ps.PolicyDefinitionReferenceID)
			// A compliance state is unique per (resource, assignment, definition
			// reference); there is no single stable ID the API returns, so key on
			// the composite.
			stateId := resourceId + "/" + policyAssignmentId + "/" + policyDefinitionReferenceId
			mqlState, err := CreateResource(a.MqlRuntime, "azure.subscription.policy.state", map[string]*llx.RawData{
				"__id":                        llx.StringData(stateId),
				"complianceState":             llx.StringData(convert.ToValue(ps.ComplianceState)),
				"resourceId":                  llx.StringData(resourceId),
				"resourceType":                llx.StringData(convert.ToValue(ps.ResourceType)),
				"resourceGroup":               llx.StringData(convert.ToValue(ps.ResourceGroup)),
				"resourceLocation":            llx.StringData(convert.ToValue(ps.ResourceLocation)),
				"policyAssignmentName":        llx.StringData(convert.ToValue(ps.PolicyAssignmentName)),
				"policyAssignmentScope":       llx.StringData(convert.ToValue(ps.PolicyAssignmentScope)),
				"policyDefinitionName":        llx.StringData(convert.ToValue(ps.PolicyDefinitionName)),
				"policyDefinitionReferenceId": llx.StringData(policyDefinitionReferenceId),
				"policyDefinitionAction":      llx.StringData(convert.ToValue(ps.PolicyDefinitionAction)),
				"policyDefinitionCategory":    llx.StringData(convert.ToValue(ps.PolicyDefinitionCategory)),
				"policySetDefinitionName":     llx.StringData(convert.ToValue(ps.PolicySetDefinitionName)),
				"timestamp":                   llx.TimeDataPtr(ps.Timestamp),
			})
			if err != nil {
				return nil, err
			}
			st := mqlState.(*mqlAzureSubscriptionPolicyState)
			st.subscriptionId = a.SubscriptionId.Data
			st.policyAssignmentId = policyAssignmentId
			st.policyDefinitionId = policyDefinitionId
			st.policySetDefinitionId = policySetDefinitionId
			res = append(res, mqlState)
		}
	}
	return res, nil
}

// mqlAzureSubscriptionPolicyStateInternal caches the raw assignment, definition,
// and set-definition IDs so the typed references can resolve them lazily against
// the subscription's policy collections.
type mqlAzureSubscriptionPolicyStateInternal struct {
	subscriptionId        string
	policyAssignmentId    string
	policyDefinitionId    string
	policySetDefinitionId string
}

// policyForState returns the subscription's policy resource, whose assignment,
// definition, and set-definition lists are cached and shared across all states.
func (a *mqlAzureSubscriptionPolicyState) policyForState() (*mqlAzureSubscriptionPolicy, error) {
	res, err := CreateResource(a.MqlRuntime, "azure.subscription.policy", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(a.subscriptionId),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionPolicy), nil
}

func (a *mqlAzureSubscriptionPolicyState) policyAssignment() (*mqlAzureSubscriptionPolicyAssignment, error) {
	if a.policyAssignmentId == "" {
		a.PolicyAssignment.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	pol, err := a.policyForState()
	if err != nil {
		return nil, err
	}
	list := pol.GetAssignments()
	if list.Error != nil {
		return nil, list.Error
	}
	for _, x := range list.Data {
		asg, ok := x.(*mqlAzureSubscriptionPolicyAssignment)
		if ok && strings.EqualFold(asg.AssignmentId.Data, a.policyAssignmentId) {
			return asg, nil
		}
	}
	a.PolicyAssignment.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

func (a *mqlAzureSubscriptionPolicyState) policyDefinition() (*mqlAzureSubscriptionPolicyDefinition, error) {
	if a.policyDefinitionId == "" {
		a.PolicyDefinition.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	pol, err := a.policyForState()
	if err != nil {
		return nil, err
	}
	list := pol.GetDefinitions()
	if list.Error != nil {
		return nil, list.Error
	}
	for _, x := range list.Data {
		def, ok := x.(*mqlAzureSubscriptionPolicyDefinition)
		if ok && strings.EqualFold(def.Id.Data, a.policyDefinitionId) {
			return def, nil
		}
	}
	a.PolicyDefinition.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

func (a *mqlAzureSubscriptionPolicyState) policySetDefinition() (*mqlAzureSubscriptionPolicySetDefinition, error) {
	if a.policySetDefinitionId == "" {
		a.PolicySetDefinition.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	pol, err := a.policyForState()
	if err != nil {
		return nil, err
	}
	list := pol.GetSetDefinitions()
	if list.Error != nil {
		return nil, list.Error
	}
	for _, x := range list.Data {
		setDef, ok := x.(*mqlAzureSubscriptionPolicySetDefinition)
		if ok && strings.EqualFold(setDef.Id.Data, a.policySetDefinitionId) {
			return setDef, nil
		}
	}
	a.PolicySetDefinition.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

func (a *mqlAzureSubscriptionPolicy) complianceSummary() (*mqlAzureSubscriptionPolicyComplianceSummary, error) {
	res, err := NewResource(a.MqlRuntime, "azure.subscription.policy.complianceSummary", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(a.SubscriptionId.Data),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionPolicyComplianceSummary), nil
}

// mqlAzureSubscriptionPolicyComplianceSummaryInternal caches the single
// SummarizeForSubscription response so every compliance field shares one call.
type mqlAzureSubscriptionPolicyComplianceSummaryInternal struct {
	fetched                     bool
	lock                        sync.Mutex
	cachedNonCompliantResources int64
	cachedNonCompliantPolicies  int64
	cachedResourceCounts        map[string]any
	cachedPolicyCounts          map[string]any
	cachedAssignments           []any
}

func (a *mqlAzureSubscriptionPolicyComplianceSummary) fetchSummary() error {
	if a.fetched {
		return nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()

	client, err := armpolicyinsights.NewPolicyStatesClient(conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return err
	}

	resp, err := client.SummarizeForSubscription(ctx, armpolicyinsights.PolicyStatesSummaryResourceTypeLatest, a.SubscriptionId.Data, nil)
	if err != nil {
		return err
	}

	a.cachedResourceCounts = map[string]any{}
	a.cachedPolicyCounts = map[string]any{}
	a.cachedAssignments = []any{}

	if len(resp.Value) > 0 && resp.Value[0] != nil {
		summary := resp.Value[0]
		if results := summary.Results; results != nil {
			a.cachedNonCompliantResources = int64(convert.ToValue(results.NonCompliantResources))
			a.cachedNonCompliantPolicies = int64(convert.ToValue(results.NonCompliantPolicies))
			a.cachedResourceCounts = complianceCounts(results.ResourceDetails)
			a.cachedPolicyCounts = complianceCounts(results.PolicyDetails)
		}
		for _, pa := range summary.PolicyAssignments {
			if pa == nil {
				continue
			}
			entry := map[string]any{
				"policyAssignmentId":    convert.ToValue(pa.PolicyAssignmentID),
				"policySetDefinitionId": convert.ToValue(pa.PolicySetDefinitionID),
				"nonCompliantResources": int64(0),
				"nonCompliantPolicies":  int64(0),
			}
			if pa.Results != nil {
				entry["nonCompliantResources"] = int64(convert.ToValue(pa.Results.NonCompliantResources))
				entry["nonCompliantPolicies"] = int64(convert.ToValue(pa.Results.NonCompliantPolicies))
			}
			a.cachedAssignments = append(a.cachedAssignments, entry)
		}
	}

	a.fetched = true
	return nil
}

func (a *mqlAzureSubscriptionPolicyComplianceSummary) nonCompliantResources() (int64, error) {
	if err := a.fetchSummary(); err != nil {
		return 0, err
	}
	return a.cachedNonCompliantResources, nil
}

func (a *mqlAzureSubscriptionPolicyComplianceSummary) nonCompliantPolicies() (int64, error) {
	if err := a.fetchSummary(); err != nil {
		return 0, err
	}
	return a.cachedNonCompliantPolicies, nil
}

func (a *mqlAzureSubscriptionPolicyComplianceSummary) resourceComplianceCounts() (map[string]any, error) {
	if err := a.fetchSummary(); err != nil {
		return nil, err
	}
	return a.cachedResourceCounts, nil
}

func (a *mqlAzureSubscriptionPolicyComplianceSummary) policyComplianceCounts() (map[string]any, error) {
	if err := a.fetchSummary(); err != nil {
		return nil, err
	}
	return a.cachedPolicyCounts, nil
}

func (a *mqlAzureSubscriptionPolicyComplianceSummary) assignments() ([]any, error) {
	if err := a.fetchSummary(); err != nil {
		return nil, err
	}
	return a.cachedAssignments, nil
}

func complianceCounts(details []*armpolicyinsights.ComplianceDetail) map[string]any {
	counts := map[string]any{}
	for _, d := range details {
		if d == nil || d.ComplianceState == nil {
			continue
		}
		counts[*d.ComplianceState] = int64(convert.ToValue(d.Count))
	}
	return counts
}
