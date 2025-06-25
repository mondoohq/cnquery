// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"strings"

	graphidentitygovernance "github.com/microsoftgraph/msgraph-sdk-go/identitygovernance"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	graphpolicies "github.com/microsoftgraph/msgraph-sdk-go/policies"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/ms365/connection"
	"go.mondoo.com/cnquery/v11/types"
)

const (
	defaultRequestFilterDirectoryRole = "scopeId eq '/' and scopeType eq 'DirectoryRole'"
)

func (a *mqlMicrosoft) identityAndAccess() (*mqlMicrosoftIdentityAndAccess, error) {
	resource, err := CreateResource(a.MqlRuntime, "microsoft.identityAndAccess", nil)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlMicrosoftIdentityAndAccess), nil
}

func initMicrosoftIdentityAndAccess(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if filter, ok := args["filter"]; ok {
		args["filter"] = filter
	}

	return args, nil, nil
}

// The data-fetching logic is now in the list() method of the new resource.
func (a *mqlMicrosoftIdentityAndAccess) list() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	// Use the Filter field from our struct, which was populated during init.
	requestFilter := a.Filter.Data
	if requestFilter == "" {
		requestFilter = defaultRequestFilterDirectoryRole
	}

	requestParameters := &graphpolicies.RoleManagementPoliciesRequestBuilderGetQueryParameters{}

	switch {
	case strings.Contains(requestFilter, "scopeType eq 'DirectoryRole'"):
		requestParameters = &graphpolicies.RoleManagementPoliciesRequestBuilderGetQueryParameters{
			Filter: &requestFilter,
		}
	// we can only get rules if scopeType set to 'Directory'
	case strings.Contains(requestFilter, "scopeType eq 'Directory'"):
		requestParameters = &graphpolicies.RoleManagementPoliciesRequestBuilderGetQueryParameters{
			Filter: &requestFilter,
			Expand: []string{"rules"},
		}

	default:
		return nil, fmt.Errorf("scopeType in the filter needs to equal to 'Directory' or 'DirectoryRole', got %q", requestFilter)
	}

	configuration := &graphpolicies.RoleManagementPoliciesRequestBuilderGetRequestConfiguration{
		QueryParameters: requestParameters,
	}

	policies, err := graphClient.Policies().RoleManagementPolicies().Get(context.Background(), configuration)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve role management policies with filter '%s': %w", requestFilter, err)
	}

	var policyResources []interface{}
	if policies == nil {
		return nil, nil
	}

	for _, policy := range policies.GetValue() {
		if policy.GetId() != nil && policy.GetDisplayName() != nil {
			policyResource, err := newMqlRoleManagementPolicy(a.MqlRuntime, policy)
			if err != nil {
				return nil, fmt.Errorf("failed to create MQL resource for policy ID %s: %w", *policy.GetId(), err)
			}
			policyResources = append(policyResources, policyResource)
		}
	}

	return policyResources, nil
}

func newMqlRoleManagementPolicy(runtime *plugin.Runtime, u models.UnifiedRoleManagementPolicyable) (*mqlMicrosoftIdentityAndAccessPolicy, error) {
	lastModifiedByDict := map[string]interface{}{}
	var err error

	if u.GetLastModifiedBy() != nil {
		lastModifiedByDict, err = convert.JsonToDict(newLastModifiedBy(u.GetLastModifiedBy()))
		if err != nil {
			return nil, err
		}
	}

	resource, err := CreateResource(runtime, "microsoft.identityAndAccess.policy",
		map[string]*llx.RawData{
			"__id":                  llx.StringDataPtr(u.GetId()),
			"id":                    llx.StringDataPtr(u.GetId()),
			"displayName":           llx.StringDataPtr(u.GetDisplayName()),
			"description":           llx.StringDataPtr(u.GetDescription()),
			"isOrganizationDefault": llx.BoolDataPtr(u.GetIsOrganizationDefault()),
			"scopeId":               llx.StringDataPtr(u.GetScopeId()),
			"scopeType":             llx.StringDataPtr(u.GetScopeType()),
			"lastModifiedDateTime":  llx.TimeDataPtr(u.GetLastModifiedDateTime()),
			"lastModifiedBy":        llx.DictData(lastModifiedByDict),
		})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlMicrosoftIdentityAndAccessPolicy), nil
}

func (m *mqlMicrosoftIdentityAndAccessPolicy) rules() ([]interface{}, error) {
	conn := m.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	policyId := m.Id.Data
	if policyId == "" {
		return nil, fmt.Errorf("policy resource has an empty ID, cannot fetch rules")
	}

	ctx := context.Background()

	rulesResult, err := graphClient.Policies().RoleManagementPolicies().ByUnifiedRoleManagementPolicyId(policyId).Rules().Get(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get rules for policy %s: %w", policyId, err)
	}

	var ruleResources []interface{}
	if rulesResult == nil {
		return nil, nil
	}

	for _, rule := range rulesResult.GetValue() {
		if rule.GetId() == nil {
			continue
		}
		ruleResource, err := newMqlRoleManagementPolicyRule(m.MqlRuntime, rule)
		if err != nil {
			return nil, fmt.Errorf("failed to create MQL resource for rule ID %s: %w", *rule.GetId(), err)
		}
		ruleResources = append(ruleResources, ruleResource)
	}

	return ruleResources, nil
}

func newMqlRoleManagementPolicyRule(runtime *plugin.Runtime, rule models.UnifiedRoleManagementPolicyRuleable) (*mqlMicrosoftIdentityAndAccessPolicyRule, error) {
	var mqlPolicyRuleTarget plugin.Resource
	var err error

	if rule.GetTarget() != nil {
		targetData := map[string]*llx.RawData{
			"caller":              llx.StringDataPtr(rule.GetTarget().GetCaller()),
			"enforcedSettings":    llx.ArrayData(convert.SliceAnyToInterface(rule.GetTarget().GetEnforcedSettings()), types.String),
			"inheritableSettings": llx.ArrayData(convert.SliceAnyToInterface(rule.GetTarget().GetInheritableSettings()), types.String),
			"level":               llx.StringDataPtr(rule.GetTarget().GetLevel()),
			"operations":          llx.ArrayData(convert.SliceAnyToInterface(convertEnumCollectionToStrings(rule.GetTarget().GetOperations())), types.String),
		}

		mqlPolicyRuleTarget, err = CreateResource(runtime, "microsoft.identityAndAccess.policy.rule.target", targetData)
		if err != nil {
			return nil, err
		}
	}

	resource, err := CreateResource(runtime, "microsoft.identityAndAccess.policy.rule",
		map[string]*llx.RawData{
			"__id":   llx.StringDataPtr(rule.GetId()),
			"id":     llx.StringDataPtr(rule.GetId()),
			"target": llx.ResourceData(mqlPolicyRuleTarget, "microsoft.identityAndAccess.policy.rule.target"),
		})
	if err != nil {
		return nil, err
	}

	return resource.(*mqlMicrosoftIdentityAndAccessPolicyRule), nil
}

// Least privileged permissions: RoleEligibilitySchedule.Read.Directory
func (a *mqlMicrosoftIdentityAndAccess) roleEligibilityScheduleInstances() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	roleEligibilityScheduleInstances, err := graphClient.RoleManagement().Directory().RoleEligibilityScheduleInstances().Get(context.Background(), nil)
	if err != nil {
		return nil, transformError(err)
	}

	if roleEligibilityScheduleInstances == nil {
		return nil, nil
	}

	var instances []interface{}
	for _, inst := range roleEligibilityScheduleInstances.GetValue() {
		if inst.GetId() == nil {
			continue
		}
		instanceResource, err := newMqlRoleEligibilityScheduleInstance(a.MqlRuntime, inst)
		if err != nil {
			return nil, fmt.Errorf("failed to create MQL resource for rule ID %s: %w", *inst.GetId(), err)
		}
		instances = append(instances, instanceResource)
	}

	return instances, nil
}

func newMqlRoleEligibilityScheduleInstance(runtime *plugin.Runtime, inst models.UnifiedRoleEligibilityScheduleInstanceable) (*mqlMicrosoftIdentityAndAccessRoleEligibilityScheduleInstance, error) {
	resource, err := CreateResource(runtime, "microsoft.identityAndAccess.roleEligibilityScheduleInstance", map[string]*llx.RawData{
		"id":                        llx.StringDataPtr(inst.GetId()),
		"__id":                      llx.StringDataPtr(inst.GetId()),
		"principalId":               llx.StringDataPtr(inst.GetPrincipalId()),
		"roleDefinitionId":          llx.StringDataPtr(inst.GetRoleDefinitionId()),
		"directoryScopeId":          llx.StringDataPtr(inst.GetDirectoryScopeId()),
		"appScopeId":                llx.StringDataPtr(inst.GetAppScopeId()),
		"startDateTime":             llx.TimeDataPtr(inst.GetStartDateTime()),
		"endDateTime":               llx.TimeDataPtr(inst.GetEndDateTime()),
		"memberType":                llx.StringDataPtr(inst.GetMemberType()),
		"roleEligibilityScheduleId": llx.StringDataPtr(inst.GetRoleEligibilityScheduleId()),
	})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlMicrosoftIdentityAndAccessRoleEligibilityScheduleInstance), nil
}

// Needs the permission AccessReview.Read.All
func (a *mqlMicrosoft) accessReviews() (*mqlMicrosoftIdentityAndAccessAccessReviews, error) {
	mqlResource, err := CreateResource(a.MqlRuntime, "microsoft.identityAndAccess.accessReviews", map[string]*llx.RawData{})
	return mqlResource.(*mqlMicrosoftIdentityAndAccessAccessReviews), err
}

// The $filter query parameter with the contains operator is supported on
// the scope property of accessReviewScheduleDefinition. Use the following format for the request:
// filter=contains(scope/microsoft.graph.accessReviewQueryScope/query, '{object}')
// The {object} can have one of the following values:
// /groups: List every accessReviewScheduleDefinition on individual groups (excludes definitions scoped to all Microsoft 365 groups with guests).
// /groups/{group_id}:	List every accessReviewScheduleDefinition on a specific group (excludes definitions scoped to all Microsoft 365 groups with guests).
// ./members: List every accessReviewScheduleDefinition scoped to all Microsoft 365 groups with guests.
// accessPackageAssignments:	List every accessReviewScheduleDefinition on an access package.
// roleAssignmentScheduleInstances:	List every accessReviewScheduleDefinition for principals that are assigned to a privileged role.
func (a *mqlMicrosoftIdentityAndAccessAccessReviews) list() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	configuration := &graphidentitygovernance.AccessReviewsDefinitionsRequestBuilderGetRequestConfiguration{}

	requestFilter := a.Filter.Data
	if requestFilter != "" {
		requestParameters := &graphidentitygovernance.AccessReviewsDefinitionsRequestBuilderGetQueryParameters{
			Filter: &requestFilter,
		}
		configuration = &graphidentitygovernance.AccessReviewsDefinitionsRequestBuilderGetRequestConfiguration{
			QueryParameters: requestParameters,
		}
	}

	definitions, err := graphClient.
		IdentityGovernance().
		AccessReviews().
		Definitions().
		Get(context.Background(), configuration)
	if err != nil {
		return nil, transformError(err)
	}

	if definitions == nil {
		return nil, nil
	}

	var accessReviewResources []interface{}
	for _, accessReviewSchedule := range definitions.GetValue() {
		if accessReviewSchedule.GetId() != nil {
			reviewResource, err := newMqlAccessReviewDefinition(a.MqlRuntime, accessReviewSchedule)
			if err != nil {
				return nil, fmt.Errorf("failed to create MQL resource for access review ID %s: %w", *accessReviewSchedule.GetId(), err)
			}
			accessReviewResources = append(accessReviewResources, reviewResource)
		}
	}

	return accessReviewResources, nil
}

func newMqlAccessReviewDefinition(runtime *plugin.Runtime, d models.AccessReviewScheduleDefinitionable) (*mqlMicrosoftIdentityAndAccessAccessReviewDefinition, error) {
	reviewersDict := []interface{}{}
	if d.GetReviewers() != nil {
		for _, reviewer := range d.GetReviewers() {
			reviewerDict := map[string]*llx.RawData{
				"reviewer":  llx.StringDataPtr(reviewer.GetQuery()),
				"queryType": llx.StringDataPtr(reviewer.GetQueryType()),
				"queryRoot": llx.StringDataPtr(reviewer.GetQueryRoot()),
			}

			reviewersDict = append(reviewersDict, reviewerDict)
		}
	}

	var mqlAccessReviewScheduleSettings plugin.Resource
	var err error

	if d.GetSettings() != nil {
		settingsId := *d.GetId() + "_settings"

		var patternDict map[string]interface{}
		var rangeDict map[string]interface{}

		if recurrence := d.GetSettings().GetRecurrence(); recurrence != nil {
			if pattern := recurrence.GetPattern(); pattern != nil {
				patternDict, err = convert.JsonToDict(pattern)
				if err != nil {
					return nil, err
				}
			}

			if recurrenceRange := recurrence.GetRangeEscaped(); recurrenceRange != nil {
				rangeDict, err = convert.JsonToDict(recurrenceRange)
				if err != nil {
					return nil, err
				}
			}
		}

		recurrenceDict := map[string]*llx.RawData{
			"pattern": llx.DictData(patternDict),
			"range":   llx.DictData(rangeDict),
		}

		targetData := map[string]*llx.RawData{
			"__id":                                 llx.StringData(settingsId),
			"autoApplyDecisionsEnabled":            llx.BoolDataPtr(d.GetSettings().GetAutoApplyDecisionsEnabled()),
			"decisionHistoriesForReviewersEnabled": llx.BoolDataPtr(d.GetSettings().GetDecisionHistoriesForReviewersEnabled()),
			"defaultDecision":                      llx.StringDataPtr(d.GetSettings().GetDefaultDecision()),
			"defaultDecisionEnabled":               llx.BoolDataPtr(d.GetSettings().GetDefaultDecisionEnabled()),
			"instanceDurationInDays":               llx.IntDataPtr(d.GetSettings().GetInstanceDurationInDays()),
			"justificationRequiredOnApproval":      llx.BoolDataPtr(d.GetSettings().GetJustificationRequiredOnApproval()),
			"mailNotificationsEnabled":             llx.BoolDataPtr(d.GetSettings().GetMailNotificationsEnabled()),
			"recommendationsEnabled":               llx.BoolDataPtr(d.GetSettings().GetRecommendationsEnabled()),
			"recurrence":                           llx.DictData(recurrenceDict),
		}

		mqlAccessReviewScheduleSettings, err = CreateResource(runtime, "microsoft.identityAndAccess.accessReviewDefinition.accessReviewScheduleSettings", targetData)
		if err != nil {
			return nil, err
		}
	}

	resource, err := CreateResource(runtime, "microsoft.identityAndAccess.accessReviewDefinition",
		map[string]*llx.RawData{
			"__id":        llx.StringDataPtr(d.GetId()),
			"id":          llx.StringDataPtr(d.GetId()),
			"displayName": llx.StringDataPtr(d.GetDisplayName()),
			"status":      llx.StringDataPtr(d.GetStatus()),
			"reviewers":   llx.DictData(reviewersDict),
			"settings":    llx.ResourceData(mqlAccessReviewScheduleSettings, "microsoft.identityAndAccess.accessReviewDefinition.accessReviewScheduleSettings"),
		})
	if err != nil {
		return nil, err
	}

	return resource.(*mqlMicrosoftIdentityAndAccessAccessReviewDefinition), nil
}
