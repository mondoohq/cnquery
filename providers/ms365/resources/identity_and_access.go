// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"strings"

	graphidentitygovernance "github.com/microsoftgraph/msgraph-sdk-go/identitygovernance"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/organization"
	graphpolicies "github.com/microsoftgraph/msgraph-sdk-go/policies"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/ms365/connection"
	"go.mondoo.com/mql/v13/types"
)

const (
	defaultRequestFilterDirectoryRole = "scopeId eq '/' and scopeType eq 'DirectoryRole'"

	identityAndAccessPrivilegedIdentityManagementID         = "microsoft.identityAndAccess/privilegedIdentityManagement"
	identityAndAccessPrivilegedIdentityManagementPoliciesID = "microsoft.identityAndAccess/privilegedIdentityManagement/policies"
)

func (a *mqlMicrosoft) identityAndAccess() (*mqlMicrosoftIdentityAndAccess, error) {
	resource, err := CreateResource(a.MqlRuntime, "microsoft.identityAndAccess", nil)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlMicrosoftIdentityAndAccess), nil
}

func (a *mqlMicrosoftIdentityAndAccess) privilegedIdentityManagement() (*mqlMicrosoftIdentityAndAccessPrivilegedIdentityManagement, error) {
	resource, err := CreateResource(a.MqlRuntime, ResourceMicrosoftIdentityAndAccessPrivilegedIdentityManagement, map[string]*llx.RawData{
		"__id": llx.StringData(identityAndAccessPrivilegedIdentityManagementID),
	})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlMicrosoftIdentityAndAccessPrivilegedIdentityManagement), nil
}

func (a *mqlMicrosoftIdentityAndAccess) organization() (*mqlMicrosoftTenant, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	tenant, err := graphClient.Organization().ByOrganizationId(conn.TenantId()).Get(context.Background(), &organization.OrganizationItemRequestBuilderGetRequestConfiguration{
		QueryParameters: &organization.OrganizationItemRequestBuilderGetQueryParameters{
			Select: tenantFields,
		},
	})
	if err != nil {
		return nil, transformError(err)
	}
	if tenant == nil {
		return nil, fmt.Errorf("organization not found for tenant %q", conn.TenantId())
	}

	return newMicrosoftTenant(a.MqlRuntime, tenant)
}

func (pim *mqlMicrosoftIdentityAndAccessPrivilegedIdentityManagement) policies() (*mqlMicrosoftIdentityAndAccessPrivilegedIdentityManagementPolicies, error) {
	resource, err := CreateResource(pim.MqlRuntime, ResourceMicrosoftIdentityAndAccessPrivilegedIdentityManagementPolicies, map[string]*llx.RawData{
		"__id": llx.StringData(identityAndAccessPrivilegedIdentityManagementPoliciesID),
	})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlMicrosoftIdentityAndAccessPrivilegedIdentityManagementPolicies), nil
}

func initMicrosoftIdentityAndAccess(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return args, nil, nil
}

func initMicrosoftIdentityAndAccessPrivilegedIdentityManagementPolicies(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return args, nil, nil
}

func (a *mqlMicrosoftIdentityAndAccess) list() ([]any, error) {
	return listRoleManagementPolicies(a.MqlRuntime, a.Filter.Data, newDeprecatedMqlRoleManagementPolicy)
}

func (a *mqlMicrosoftIdentityAndAccessPrivilegedIdentityManagementPolicies) list() ([]any, error) {
	return listRoleManagementPolicies(a.MqlRuntime, a.Filter.Data, newMqlRoleManagementPolicy)
}

type roleManagementPolicyFactory func(*plugin.Runtime, models.UnifiedRoleManagementPolicyable) (any, error)

func listRoleManagementPolicies(runtime *plugin.Runtime, requestFilter string, createPolicy roleManagementPolicyFactory) ([]any, error) {
	conn := runtime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

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

	ctx := context.Background()
	resp, err := graphClient.Policies().RoleManagementPolicies().Get(ctx, configuration)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve role management policies with filter '%s': %w", requestFilter, err)
	}
	if resp == nil {
		return nil, nil
	}
	policies, err := iterate[models.UnifiedRoleManagementPolicyable](ctx, resp, graphClient.GetAdapter(), models.CreateUnifiedRoleManagementPolicyCollectionResponseFromDiscriminatorValue)
	if err != nil {
		return nil, err
	}

	var policyResources []any
	for _, policy := range policies {
		if policy.GetId() != nil && policy.GetDisplayName() != nil {
			policyResource, err := createPolicy(runtime, policy)
			if err != nil {
				return nil, fmt.Errorf("failed to create MQL resource for policy ID %s: %w", *policy.GetId(), err)
			}
			policyResources = append(policyResources, policyResource)
		}
	}

	return policyResources, nil
}

func newDeprecatedMqlRoleManagementPolicy(runtime *plugin.Runtime, u models.UnifiedRoleManagementPolicyable) (any, error) {
	resource, err := newRoleManagementPolicyResource(runtime, u, ResourceMicrosoftIdentityAndAccessPolicy)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlMicrosoftIdentityAndAccessPolicy), nil
}

func newMqlRoleManagementPolicy(runtime *plugin.Runtime, u models.UnifiedRoleManagementPolicyable) (any, error) {
	resource, err := newRoleManagementPolicyResource(runtime, u, ResourceMicrosoftIdentityAndAccessPrivilegedIdentityManagementPolicy)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlMicrosoftIdentityAndAccessPrivilegedIdentityManagementPolicy), nil
}

func newRoleManagementPolicyResource(runtime *plugin.Runtime, u models.UnifiedRoleManagementPolicyable, resourceName string) (plugin.Resource, error) {
	lastModifiedByDict := map[string]any{}
	var err error

	if u.GetLastModifiedBy() != nil {
		lastModifiedByDict, err = convert.JsonToDict(newLastModifiedBy(u.GetLastModifiedBy()))
		if err != nil {
			return nil, err
		}
	}

	return CreateResource(runtime, resourceName,
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
}

func (m *mqlMicrosoftIdentityAndAccessPolicy) rules() ([]any, error) {
	return listRoleManagementPolicyRules(m.MqlRuntime, m.Id.Data, newDeprecatedMqlRoleManagementPolicyRule)
}

func (m *mqlMicrosoftIdentityAndAccessPrivilegedIdentityManagementPolicy) rules() ([]any, error) {
	return listRoleManagementPolicyRules(m.MqlRuntime, m.Id.Data, newMqlRoleManagementPolicyRule)
}

type roleManagementPolicyRuleFactory func(*plugin.Runtime, models.UnifiedRoleManagementPolicyRuleable) (any, error)

func listRoleManagementPolicyRules(runtime *plugin.Runtime, policyID string, createRule roleManagementPolicyRuleFactory) ([]any, error) {
	conn := runtime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	if policyID == "" {
		return nil, fmt.Errorf("policy resource has an empty ID, cannot fetch rules")
	}

	ctx := context.Background()

	rulesResult, err := graphClient.Policies().RoleManagementPolicies().ByUnifiedRoleManagementPolicyId(policyID).Rules().Get(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get rules for policy %s: %w", policyID, err)
	}

	var ruleResources []any
	if rulesResult == nil {
		return nil, nil
	}

	for _, rule := range rulesResult.GetValue() {
		if rule.GetId() == nil {
			continue
		}
		ruleResource, err := createRule(runtime, rule)
		if err != nil {
			return nil, fmt.Errorf("failed to create MQL resource for rule ID %s: %w", *rule.GetId(), err)
		}
		ruleResources = append(ruleResources, ruleResource)
	}

	return ruleResources, nil
}

func newDeprecatedMqlRoleManagementPolicyRule(runtime *plugin.Runtime, rule models.UnifiedRoleManagementPolicyRuleable) (any, error) {
	resource, err := newRoleManagementPolicyRuleResource(
		runtime,
		rule,
		ResourceMicrosoftIdentityAndAccessPolicyRule,
		ResourceMicrosoftIdentityAndAccessPolicyRuleTarget,
	)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlMicrosoftIdentityAndAccessPolicyRule), nil
}

func newMqlRoleManagementPolicyRule(runtime *plugin.Runtime, rule models.UnifiedRoleManagementPolicyRuleable) (any, error) {
	resource, err := newRoleManagementPolicyRuleResource(
		runtime,
		rule,
		ResourceMicrosoftIdentityAndAccessPrivilegedIdentityManagementPolicyRule,
		ResourceMicrosoftIdentityAndAccessPrivilegedIdentityManagementPolicyRuleTarget,
	)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlMicrosoftIdentityAndAccessPrivilegedIdentityManagementPolicyRule), nil
}

func newRoleManagementPolicyRuleResource(runtime *plugin.Runtime, rule models.UnifiedRoleManagementPolicyRuleable, ruleResourceName string, targetResourceName string) (plugin.Resource, error) {
	var mqlPolicyRuleTarget plugin.Resource
	var err error

	if rule.GetTarget() != nil {
		ruleTargetID := fmt.Sprintf("%s-ruleTarget", *rule.GetId())
		targetData := map[string]*llx.RawData{
			"__id":                llx.StringData(ruleTargetID),
			"caller":              llx.StringDataPtr(rule.GetTarget().GetCaller()),
			"enforcedSettings":    llx.ArrayData(convert.SliceAnyToInterface(rule.GetTarget().GetEnforcedSettings()), types.String),
			"inheritableSettings": llx.ArrayData(convert.SliceAnyToInterface(rule.GetTarget().GetInheritableSettings()), types.String),
			"level":               llx.StringDataPtr(rule.GetTarget().GetLevel()),
			"operations":          llx.ArrayData(convert.SliceAnyToInterface(convertEnumCollectionToStrings(rule.GetTarget().GetOperations())), types.String),
		}

		mqlPolicyRuleTarget, err = CreateResource(runtime, targetResourceName, targetData)
		if err != nil {
			return nil, err
		}
	}

	return CreateResource(runtime, ruleResourceName,
		map[string]*llx.RawData{
			"__id":   llx.StringDataPtr(rule.GetId()),
			"id":     llx.StringDataPtr(rule.GetId()),
			"target": llx.ResourceData(mqlPolicyRuleTarget, targetResourceName),
		})
}

// Least privileged permissions: RoleEligibilitySchedule.Read.Directory
func (a *mqlMicrosoftIdentityAndAccess) roleEligibilityScheduleInstances() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	resp, err := graphClient.RoleManagement().Directory().RoleEligibilityScheduleInstances().Get(ctx, nil)
	if err != nil {
		return nil, transformError(err)
	}
	if resp == nil {
		return nil, nil
	}
	roleEligibilityScheduleInstances, err := iterate[models.UnifiedRoleEligibilityScheduleInstanceable](ctx, resp, graphClient.GetAdapter(), models.CreateUnifiedRoleEligibilityScheduleInstanceCollectionResponseFromDiscriminatorValue)
	if err != nil {
		return nil, err
	}

	var instances []any
	for _, inst := range roleEligibilityScheduleInstances {
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

// roleDefinition resolves the role definition this eligibility grants.
func (a *mqlMicrosoftIdentityAndAccessRoleEligibilityScheduleInstance) roleDefinition() (*mqlMicrosoftRolemanagementRoledefinition, error) {
	id := a.RoleDefinitionId.Data
	if id == "" {
		a.RoleDefinition.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "microsoft.rolemanagement.roledefinition", map[string]*llx.RawData{
		"id": llx.StringData(id),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlMicrosoftRolemanagementRoledefinition), nil
}

// Implementation for the new identityAndSignIn resource
func (a *mqlMicrosoftIdentityAndAccess) identityAndSignIn() (*mqlMicrosoftIdentityAndAccessIdentityAndSignIn, error) {
	resource, err := CreateResource(a.MqlRuntime, "microsoft.identityAndAccess.identityAndSignIn", map[string]*llx.RawData{
		"__id": llx.StringData("identityAndSignIn"),
	})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlMicrosoftIdentityAndAccessIdentityAndSignIn), nil
}

// Implementation for the policies resource
func (a *mqlMicrosoftIdentityAndAccessIdentityAndSignIn) policies() (*mqlMicrosoftIdentityAndAccessIdentityAndSignInPolicies, error) {
	resource, err := CreateResource(a.MqlRuntime, "microsoft.identityAndAccess.identityAndSignIn.policies", map[string]*llx.RawData{
		"__id": llx.StringData("policies"),
	})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlMicrosoftIdentityAndAccessIdentityAndSignInPolicies), nil
}

// initMicrosoftIdentityAndAccessIdentityAndSignInPoliciesIdentitySecurityDefaultsEnforcementPolicy
// populates the policy when it is queried directly via its full dotted path
// rather than through the parent accessor chain. Without this, the terminal
// resource is built as a bare husk with all fields null. We rebuild the chain
// (identityAndSignIn -> policies -> identitySecurityDefaultsEnforcementPolicy),
// which fetches the policy.
func initMicrosoftIdentityAndAccessIdentityAndSignInPoliciesIdentitySecurityDefaultsEnforcementPolicy(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	signIn, err := CreateResource(runtime, "microsoft.identityAndAccess.identityAndSignIn", map[string]*llx.RawData{
		"__id": llx.StringData("identityAndSignIn"),
	})
	if err != nil {
		return nil, nil, err
	}

	policies, err := signIn.(*mqlMicrosoftIdentityAndAccessIdentityAndSignIn).policies()
	if err != nil {
		return nil, nil, err
	}

	policy, err := policies.identitySecurityDefaultsEnforcementPolicy()
	if err != nil {
		return nil, nil, err
	}

	return nil, policy, nil
}

// Implementation for the identitySecurityDefaultsEnforcementPolicy resource
func (a *mqlMicrosoftIdentityAndAccessIdentityAndSignInPolicies) identitySecurityDefaultsEnforcementPolicy() (*mqlMicrosoftIdentityAndAccessIdentityAndSignInPoliciesIdentitySecurityDefaultsEnforcementPolicy, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	policy, err := graphClient.Policies().IdentitySecurityDefaultsEnforcementPolicy().Get(ctx, &graphpolicies.IdentitySecurityDefaultsEnforcementPolicyRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, transformError(err)
	}

	if policy == nil {
		return nil, fmt.Errorf("identity security defaults enforcement policy not found")
	}

	// Extract the policy data
	policyId := ""
	if policy.GetId() != nil {
		policyId = *policy.GetId()
	}

	displayName := ""
	if policy.GetDisplayName() != nil {
		displayName = *policy.GetDisplayName()
	}

	description := ""
	if policy.GetDescription() != nil {
		description = *policy.GetDescription()
	}

	isEnabled := false
	if policy.GetIsEnabled() != nil {
		isEnabled = *policy.GetIsEnabled()
	}

	resource, err := CreateResource(a.MqlRuntime, "microsoft.identityAndAccess.identityAndSignIn.policies.identitySecurityDefaultsEnforcementPolicy", map[string]*llx.RawData{
		"__id":        llx.StringData(policyId),
		"id":          llx.StringData(policyId),
		"displayName": llx.StringData(displayName),
		"description": llx.StringData(description),
		"isEnabled":   llx.BoolData(isEnabled),
	})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlMicrosoftIdentityAndAccessIdentityAndSignInPoliciesIdentitySecurityDefaultsEnforcementPolicy), nil
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
func (a *mqlMicrosoftIdentityAndAccessAccessReviews) list() ([]any, error) {
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

	ctx := context.Background()
	resp, err := graphClient.
		IdentityGovernance().
		AccessReviews().
		Definitions().
		Get(ctx, configuration)
	if err != nil {
		return nil, transformError(err)
	}
	if resp == nil {
		return nil, nil
	}
	definitions, err := iterate[models.AccessReviewScheduleDefinitionable](ctx, resp, graphClient.GetAdapter(), models.CreateAccessReviewScheduleDefinitionCollectionResponseFromDiscriminatorValue)
	if err != nil {
		return nil, err
	}

	var accessReviewResources []any
	for _, accessReviewSchedule := range definitions {
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
	reviewersDict := []any{}
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

	var mqlScope plugin.Resource
	if scope := d.GetScope(); scope != nil {
		if queryScope, ok := scope.(models.AccessReviewQueryScopeable); ok {
			var err error
			mqlScope, err = CreateResource(runtime, ResourceMicrosoftIdentityAndAccessAccessReviewDefinitionScope, map[string]*llx.RawData{
				"__id":      llx.StringData(*d.GetId() + "_scope"),
				"query":     llx.StringDataPtr(queryScope.GetQuery()),
				"queryType": llx.StringDataPtr(queryScope.GetQueryType()),
				"queryRoot": llx.StringDataPtr(queryScope.GetQueryRoot()),
			})
			if err != nil {
				return nil, err
			}
		}
	}

	var mqlAccessReviewScheduleSettings plugin.Resource
	var err error

	if d.GetSettings() != nil {
		settingsId := *d.GetId() + "_settings"

		var patternDict map[string]any
		var rangeDict map[string]any

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

		recurrenceDict := map[string]any{
			"pattern": patternDict,
			"range":   rangeDict,
		}

		targetData := map[string]*llx.RawData{
			"__id":                                 llx.StringData(settingsId),
			"autoApplyDecisionsEnabled":            llx.BoolDataPtr(d.GetSettings().GetAutoApplyDecisionsEnabled()),
			"decisionHistoriesForReviewersEnabled": llx.BoolDataPtr(d.GetSettings().GetDecisionHistoriesForReviewersEnabled()),
			"defaultDecision":                      llx.StringDataPtr(d.GetSettings().GetDefaultDecision()),
			"defaultDecisionEnabled":               llx.BoolDataPtr(d.GetSettings().GetDefaultDecisionEnabled()),
			"instanceDurationInDays":               llx.IntDataPtr(d.GetSettings().GetInstanceDurationInDays()),
			"reminderNotificationsEnabled":         llx.BoolDataPtr(d.GetSettings().GetReminderNotificationsEnabled()),
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
			"scope":       llx.ResourceData(mqlScope, ResourceMicrosoftIdentityAndAccessAccessReviewDefinitionScope),
			"reviewers":   llx.DictData(reviewersDict),
			"settings":    llx.ResourceData(mqlAccessReviewScheduleSettings, "microsoft.identityAndAccess.accessReviewDefinition.accessReviewScheduleSettings"),
		})
	if err != nil {
		return nil, err
	}

	return resource.(*mqlMicrosoftIdentityAndAccessAccessReviewDefinition), nil
}
