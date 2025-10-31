// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/policies"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers/ms365/connection"
	"go.mondoo.com/cnquery/v12/types"
)

// Package-level cache for cross-tenant access policies
var (
	crossTenantAccessPolicyCacheMu sync.Mutex
	crossTenantAccessPolicyCache   = make(map[string]*mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefaultInternal)
)

func (a *mqlMicrosoftPolicies) authorizationPolicy() (any, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	resp, err := graphClient.Policies().AuthorizationPolicy().Get(ctx, &policies.AuthorizationPolicyRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, transformError(err)
	}

	return convert.JsonToDict(newAuthorizationPolicy(resp))
}

func (a *mqlMicrosoftPolicies) identitySecurityDefaultsEnforcementPolicy() (any, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	policy, err := graphClient.Policies().IdentitySecurityDefaultsEnforcementPolicy().Get(ctx, &policies.IdentitySecurityDefaultsEnforcementPolicyRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, transformError(err)
	}

	return convert.JsonToDict(newIdentitySecurityDefaultsEnforcementPolicy(policy))
}

// https://docs.microsoft.com/en-us/azure/active-directory/manage-apps/configure-user-consent?tabs=azure-powershell
// https://docs.microsoft.com/en-us/graph/api/permissiongrantpolicy-list?view=graph-rest-1.0&tabs=http
func (a *mqlMicrosoftPolicies) permissionGrantPolicies() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	resp, err := graphClient.Policies().PermissionGrantPolicies().Get(ctx, &policies.PermissionGrantPoliciesRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, transformError(err)
	}
	return convert.JsonToDictSlice(newPermissionGrantPolicies(resp.GetValue()))
}

// https://learn.microsoft.com/en-us/graph/api/groupsetting-get?view=graph-rest-1.0&tabs=http

func (a *mqlMicrosoftPolicies) consentPolicySettings() (any, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	groupSettings, err := graphClient.GroupSettings().Get(ctx, nil)
	if err != nil {
		return nil, transformError(err)
	}

	actualSettingsMap := make(map[string]map[string]any)
	for _, setting := range groupSettings.GetValue() {
		displayName := setting.GetDisplayName()
		if displayName != nil {
			if _, exists := actualSettingsMap[*displayName]; !exists {
				actualSettingsMap[*displayName] = make(map[string]any)
			}

			for _, settingValue := range setting.GetValues() {
				name := settingValue.GetName()
				value := settingValue.GetValue()
				if name != nil && value != nil {
					actualSettingsMap[*displayName][*name] = *value
				}
			}
		}
	}

	return convert.JsonToDict(actualSettingsMap)
}

func (a *mqlMicrosoftPolicies) authenticationMethodsPolicy() (*mqlMicrosoftPoliciesAuthenticationMethodsPolicy, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	// expand authenticationMethodConfigurations to get all the details in one call
	requestConfiguration := &policies.AuthenticationMethodsPolicyRequestBuilderGetRequestConfiguration{
		QueryParameters: &policies.AuthenticationMethodsPolicyRequestBuilderGetQueryParameters{
			Expand: []string{"authenticationMethodConfigurations"},
		},
	}

	resp, err := graphClient.Policies().AuthenticationMethodsPolicy().Get(ctx, requestConfiguration)
	if err != nil {
		return nil, transformError(err)
	}

	return newAuthenticationMethodsPolicy(a.MqlRuntime, resp)
}

func (a *mqlMicrosoftPolicies) activityBasedTimeoutPolicies() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	resp, err := graphClient.Policies().ActivityBasedTimeoutPolicies().Get(ctx, nil)
	if err != nil {
		return nil, transformError(err)
	}

	var activityBasedTimeoutPolicies []any
	for _, policy := range resp.GetValue() {
		mqlPolicy, err := CreateResource(a.MqlRuntime, "microsoft.policies.activityBasedTimeoutPolicy",
			map[string]*llx.RawData{
				"__id":                  llx.StringDataPtr(policy.GetId()),
				"id":                    llx.StringDataPtr(policy.GetId()),
				"definition":            llx.ArrayData(convert.SliceAnyToInterface(policy.GetDefinition()), types.String),
				"displayName":           llx.StringDataPtr(policy.GetDisplayName()),
				"isOrganizationDefault": llx.BoolDataPtr(policy.GetIsOrganizationDefault()),
			})
		if err != nil {
			return nil, err
		}
		activityBasedTimeoutPolicies = append(activityBasedTimeoutPolicies, mqlPolicy)
	}

	return activityBasedTimeoutPolicies, nil
}

func newAuthenticationMethodsPolicy(runtime *plugin.Runtime, policy models.AuthenticationMethodsPolicyable) (*mqlMicrosoftPoliciesAuthenticationMethodsPolicy, error) {
	authMethodConfigs, err := newAuthenticationMethodConfigurations(runtime, policy.GetAuthenticationMethodConfigurations())
	if err != nil {
		return nil, err
	}

	mqlAuthenticationMethodsPolicy, err := CreateResource(runtime, "microsoft.policies.authenticationMethodsPolicy",
		map[string]*llx.RawData{
			"__id":                               llx.StringDataPtr(policy.GetId()),
			"id":                                 llx.StringDataPtr(policy.GetId()),
			"description":                        llx.StringDataPtr(policy.GetDescription()),
			"displayName":                        llx.StringDataPtr(policy.GetDisplayName()),
			"lastModifiedDateTime":               llx.TimeDataPtr(policy.GetLastModifiedDateTime()),
			"policyVersion":                      llx.StringDataPtr(policy.GetPolicyVersion()),
			"authenticationMethodConfigurations": llx.ArrayData(authMethodConfigs, "microsoft.policies.authenticationMethodConfiguration"),
		})
	if err != nil {
		return nil, err
	}

	return mqlAuthenticationMethodsPolicy.(*mqlMicrosoftPoliciesAuthenticationMethodsPolicy), nil
}

func newAuthenticationMethodConfigurations(runtime *plugin.Runtime, configs []models.AuthenticationMethodConfigurationable) ([]any, error) {
	var configResources []any
	for _, config := range configs {
		excludeTargets := []any{}
		for _, target := range config.GetExcludeTargets() {
			targetDict := map[string]any{}
			if target.GetId() != nil {
				targetDict["id"] = *target.GetId()
			}
			if target.GetTargetType() != nil {
				targetDict["targetType"] = target.GetTargetType().String()
			}
			excludeTargets = append(excludeTargets, targetDict)
		}

		configData := map[string]*llx.RawData{
			"__id":           llx.StringDataPtr(config.GetId()),
			"id":             llx.StringDataPtr(config.GetId()),
			"state":          llx.StringData(config.GetState().String()),
			"excludeTargets": llx.ArrayData(excludeTargets, types.Dict),
		}

		mqlConfig, err := CreateResource(runtime, "microsoft.policies.authenticationMethodConfiguration", configData)
		if err != nil {
			return nil, err
		}

		configResources = append(configResources, mqlConfig)
	}

	return configResources, nil
}

// https://docs.microsoft.com/en-us/graph/api/adminconsentrequestpolicy-get?view=graph-rest-
func (a *mqlMicrosoftPolicies) adminConsentRequestPolicy() (*mqlMicrosoftAdminConsentRequestPolicy, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	adminConsentRequestPolicy, err := graphClient.Policies().AdminConsentRequestPolicy().Get(context.Background(), nil)
	if err != nil {
		return nil, transformError(err)
	}

	if adminConsentRequestPolicy == nil {
		return nil, nil
	}

	pId := uuid.NewString()

	var reviewers []any
	if adminConsentRequestPolicy.GetReviewers() != nil {
		for i, reviewer := range adminConsentRequestPolicy.GetReviewers() {
			revId := fmt.Sprintf("%s-reviewer-scope-%d", pId, i)
			resource, err := CreateResource(a.MqlRuntime, "microsoft.graph.accessReviewReviewerScope",
				map[string]*llx.RawData{
					"__id":      llx.StringData(revId),
					"query":     llx.StringDataPtr(reviewer.GetQuery()),
					"queryRoot": llx.StringDataPtr(reviewer.GetQueryRoot()),
					"queryType": llx.StringDataPtr(reviewer.GetQueryType()),
				})
			if err != nil {
				return nil, err
			}

			reviewers = append(reviewers, resource)
		}
	}

	data := map[string]*llx.RawData{
		"__id":                  llx.StringData(pId),
		"reviewers":             llx.ArrayData(reviewers, "microsoft.graph.accessReviewReviewerScope"),
		"isEnabled":             llx.BoolDataPtr(adminConsentRequestPolicy.GetIsEnabled()),
		"notifyReviewers":       llx.BoolDataPtr(adminConsentRequestPolicy.GetNotifyReviewers()),
		"remindersEnabled":      llx.BoolDataPtr(adminConsentRequestPolicy.GetRemindersEnabled()),
		"requestDurationInDays": llx.IntDataPtr(adminConsentRequestPolicy.GetRequestDurationInDays()),
		"version":               llx.IntDataPtr(adminConsentRequestPolicy.GetVersion()),
	}

	resource, err := CreateResource(a.MqlRuntime, "microsoft.adminConsentRequestPolicy", data)
	if err != nil {
		return nil, err
	}

	return resource.(*mqlMicrosoftAdminConsentRequestPolicy), nil
}

func (a *mqlMicrosoftPolicies) externalIdentitiesPolicy() (*mqlMicrosoftPoliciesExternalIdentitiesPolicy, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	betaGraphClient, err := conn.BetaGraphClient()
	if err != nil {
		return nil, err
	}
	policy, err := betaGraphClient.Policies().ExternalIdentitiesPolicy().Get(context.Background(), nil)
	if err != nil {
		return nil, transformError(err)
	}

	mqlPolicy, err := CreateResource(a.MqlRuntime, "microsoft.policies.externalIdentitiesPolicy",
		map[string]*llx.RawData{
			"__id":                           llx.StringDataPtr(policy.GetId()),
			"id":                             llx.StringDataPtr(policy.GetId()),
			"displayName":                    llx.StringDataPtr(policy.GetDisplayName()),
			"description":                    llx.StringDataPtr(policy.GetDescription()),
			"allowExternalIdentitiesToLeave": llx.BoolDataPtr(policy.GetAllowExternalIdentitiesToLeave()),
		})
	if err != nil {
		return nil, err
	}

	return mqlPolicy.(*mqlMicrosoftPoliciesExternalIdentitiesPolicy), nil
}

// Internal struct for caching cross-tenant access policy data
// This will be embedded in mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefault after code generation
type mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefaultInternal struct {
	policyLock sync.Mutex
	fetched    bool
	fetchErr   error
	policy     models.CrossTenantAccessPolicyConfigurationDefaultable
}

func (a *mqlMicrosoftPolicies) crossTenantAccessPolicies() (*mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefault, error) {
	resource, err := CreateResource(a.MqlRuntime, "microsoft.policies.crossTenantAccessPolicyConfigurationDefault",
		map[string]*llx.RawData{
			"__id": llx.StringData("crossTenantAccessPolicyConfigurationDefault"),
		})
	if err != nil {
		return nil, err
	}

	return resource.(*mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefault), nil
}

// getCrossTenantAccessPolicy fetches and caches the policy data
// Following the same pattern as getExchangeReport() in ms365_exchange.go
func (a *mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefault) getCrossTenantAccessPolicy() error {
	// After code generation, these fields will be embedded and accessible directly:
	// a.policyLock, a.fetched, a.fetchErr, a.policy

	cacheKey := "crossTenantAccessPolicy_" + a.__id

	crossTenantAccessPolicyCacheMu.Lock()
	internal, exists := crossTenantAccessPolicyCache[cacheKey]
	if !exists {
		internal = &mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefaultInternal{}
		crossTenantAccessPolicyCache[cacheKey] = internal
	}
	crossTenantAccessPolicyCacheMu.Unlock()

	internal.policyLock.Lock()
	defer internal.policyLock.Unlock()

	// only fetch once
	if internal.fetched {
		return internal.fetchErr
	}

	errHandler := func(err error) error {
		internal.fetchErr = err
		return err
	}

	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return errHandler(err)
	}

	policy, err := graphClient.Policies().CrossTenantAccessPolicy().DefaultEscaped().Get(context.Background(), nil)
	if err != nil {
		return errHandler(transformError(err))
	}

	// Cache the policy
	internal.policy = policy
	internal.fetched = true
	internal.fetchErr = nil

	return nil
}

func (a *mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefault) automaticUserConsentSettings() (*mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefaultAutomaticUserConsentSettings, error) {
	if err := a.getCrossTenantAccessPolicy(); err != nil {
		return nil, err
	}

	cacheKey := "crossTenantAccessPolicy_" + a.__id

	crossTenantAccessPolicyCacheMu.Lock()
	internal := crossTenantAccessPolicyCache[cacheKey]
	crossTenantAccessPolicyCacheMu.Unlock()

	if internal == nil || internal.policy == nil || internal.policy.GetAutomaticUserConsentSettings() == nil {
		return nil, nil
	}

	consentSettings := internal.policy.GetAutomaticUserConsentSettings()
	consentResource, err := CreateResource(a.MqlRuntime, "microsoft.policies.crossTenantAccessPolicyConfigurationDefault.automaticUserConsentSettings",
		map[string]*llx.RawData{
			"__id":            llx.StringData("automaticUserConsentSettings"),
			"inboundAllowed":  llx.BoolDataPtr(consentSettings.GetInboundAllowed()),
			"outboundAllowed": llx.BoolDataPtr(consentSettings.GetOutboundAllowed()),
		})
	if err != nil {
		return nil, err
	}

	return consentResource.(*mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefaultAutomaticUserConsentSettings), nil
}

func (a *mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefault) b2bCollaborationInbound() (*mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefaultB2bSetting, error) {
	if err := a.getCrossTenantAccessPolicy(); err != nil {
		return nil, err
	}

	cacheKey := "crossTenantAccessPolicy_" + a.__id

	crossTenantAccessPolicyCacheMu.Lock()
	internal := crossTenantAccessPolicyCache[cacheKey]
	crossTenantAccessPolicyCacheMu.Unlock()

	if internal == nil || internal.policy == nil {
		return nil, nil
	}

	return newB2BSetting(a.MqlRuntime, internal.policy.GetB2bCollaborationInbound(), "b2bCollaborationInbound")
}

func (a *mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefault) b2bCollaborationOutbound() (*mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefaultB2bSetting, error) {
	if err := a.getCrossTenantAccessPolicy(); err != nil {
		return nil, err
	}

	cacheKey := "crossTenantAccessPolicy_" + a.__id

	crossTenantAccessPolicyCacheMu.Lock()
	internal := crossTenantAccessPolicyCache[cacheKey]
	crossTenantAccessPolicyCacheMu.Unlock()

	if internal == nil || internal.policy == nil {
		return nil, nil
	}

	return newB2BSetting(a.MqlRuntime, internal.policy.GetB2bCollaborationOutbound(), "b2bCollaborationOutbound")
}

func (a *mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefault) b2bDirectConnectInbound() (*mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefaultB2bSetting, error) {
	if err := a.getCrossTenantAccessPolicy(); err != nil {
		return nil, err
	}

	cacheKey := "crossTenantAccessPolicy_" + a.__id

	crossTenantAccessPolicyCacheMu.Lock()
	internal := crossTenantAccessPolicyCache[cacheKey]
	crossTenantAccessPolicyCacheMu.Unlock()

	if internal == nil || internal.policy == nil {
		return nil, nil
	}

	return newB2BSetting(a.MqlRuntime, internal.policy.GetB2bDirectConnectInbound(), "b2bDirectConnectInbound")
}

func (a *mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefault) b2bDirectConnectOutbound() (*mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefaultB2bSetting, error) {
	if err := a.getCrossTenantAccessPolicy(); err != nil {
		return nil, err
	}

	cacheKey := "crossTenantAccessPolicy_" + a.__id

	crossTenantAccessPolicyCacheMu.Lock()
	internal := crossTenantAccessPolicyCache[cacheKey]
	crossTenantAccessPolicyCacheMu.Unlock()

	if internal == nil || internal.policy == nil {
		return nil, nil
	}

	return newB2BSetting(a.MqlRuntime, internal.policy.GetB2bDirectConnectOutbound(), "b2bDirectConnectOutbound")
}

func (a *mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefault) tenantRestrictions() (*mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefaultB2bSetting, error) {
	if err := a.getCrossTenantAccessPolicy(); err != nil {
		return nil, err
	}

	cacheKey := "crossTenantAccessPolicy_" + a.__id

	crossTenantAccessPolicyCacheMu.Lock()
	internal := crossTenantAccessPolicyCache[cacheKey]
	crossTenantAccessPolicyCacheMu.Unlock()

	if internal == nil || internal.policy == nil {
		return nil, nil
	}

	return newB2BSetting(a.MqlRuntime, internal.policy.GetTenantRestrictions(), "tenantRestrictions")
}

func (a *mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefault) inboundTrust() (*mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefaultInboundTrust, error) {
	if err := a.getCrossTenantAccessPolicy(); err != nil {
		return nil, err
	}

	cacheKey := "crossTenantAccessPolicy_" + a.__id

	crossTenantAccessPolicyCacheMu.Lock()
	internal := crossTenantAccessPolicyCache[cacheKey]
	crossTenantAccessPolicyCacheMu.Unlock()

	if internal == nil || internal.policy == nil || internal.policy.GetInboundTrust() == nil {
		return nil, nil
	}

	inboundTrustValue := internal.policy.GetInboundTrust()
	inboundTrustResource, err := CreateResource(a.MqlRuntime, "microsoft.policies.crossTenantAccessPolicyConfigurationDefault.inboundTrust",
		map[string]*llx.RawData{
			"__id":                                llx.StringData("inboundTrust"),
			"isMfaAccepted":                       llx.BoolDataPtr(inboundTrustValue.GetIsMfaAccepted()),
			"isCompliantDeviceAccepted":           llx.BoolDataPtr(inboundTrustValue.GetIsCompliantDeviceAccepted()),
			"isHybridAzureADJoinedDeviceAccepted": llx.BoolDataPtr(inboundTrustValue.GetIsHybridAzureADJoinedDeviceAccepted()),
		})
	if err != nil {
		return nil, err
	}

	return inboundTrustResource.(*mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefaultInboundTrust), nil
}

func (a *mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefault) invitationRedemptionIdentityProviderConfiguration() (*mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefaultInvitationRedemptionIdentityProviderConfiguration, error) {
	if err := a.getCrossTenantAccessPolicy(); err != nil {
		return nil, err
	}

	cacheKey := "crossTenantAccessPolicy_" + a.__id

	crossTenantAccessPolicyCacheMu.Lock()
	internal := crossTenantAccessPolicyCache[cacheKey]
	crossTenantAccessPolicyCacheMu.Unlock()

	if internal == nil || internal.policy == nil || internal.policy.GetInvitationRedemptionIdentityProviderConfiguration() == nil {
		return nil, nil
	}

	invConfig := internal.policy.GetInvitationRedemptionIdentityProviderConfiguration()
	var fallbackProvider string
	if invConfig.GetFallbackIdentityProvider() != nil {
		fallbackProvider = invConfig.GetFallbackIdentityProvider().String()
	}
	var precedenceOrder []any
	for _, provider := range invConfig.GetPrimaryIdentityProviderPrecedenceOrder() {
		// provider is of type B2bIdentityProvidersType (an enum), not a pointer
		precedenceOrder = append(precedenceOrder, provider.String())
	}

	invResource, err := CreateResource(a.MqlRuntime, "microsoft.policies.crossTenantAccessPolicyConfigurationDefault.invitationRedemptionIdentityProviderConfiguration",
		map[string]*llx.RawData{
			"__id":                                   llx.StringData("invitationRedemptionIdentityProviderConfiguration"),
			"fallbackIdentityProvider":               llx.StringData(fallbackProvider),
			"primaryIdentityProviderPrecedenceOrder": llx.ArrayData(precedenceOrder, types.String),
		})
	if err != nil {
		return nil, err
	}

	return invResource.(*mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefaultInvitationRedemptionIdentityProviderConfiguration), nil
}

func newB2BSetting(runtime *plugin.Runtime, setting models.CrossTenantAccessPolicyB2BSettingable, settingId string) (*mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefaultB2bSetting, error) {
	if setting == nil {
		// Return nil if setting is nil (caller should handle this)
		return nil, nil
	}

	usersAndGroups, err := newUsersAndGroups(runtime, setting.GetUsersAndGroups(), settingId+"-usersAndGroups")
	if err != nil {
		return nil, err
	}

	applications, err := newUsersAndGroups(runtime, setting.GetApplications(), settingId+"-applications")
	if err != nil {
		return nil, err
	}

	resource, err := CreateResource(runtime, "microsoft.policies.crossTenantAccessPolicyConfigurationDefault.b2bSetting",
		map[string]*llx.RawData{
			"__id":           llx.StringData(settingId),
			"usersAndGroups": llx.ResourceData(usersAndGroups, "microsoft.policies.crossTenantAccessPolicyConfigurationDefault.b2bSetting.usersAndGroups"),
			"applications":   llx.ResourceData(applications, "microsoft.policies.crossTenantAccessPolicyConfigurationDefault.b2bSetting.usersAndGroups"),
		})
	if err != nil {
		return nil, err
	}

	return resource.(*mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefaultB2bSetting), nil
}

// usersAndGroups returns the usersAndGroups field from the b2bSetting resource
func (a *mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefaultB2bSetting) usersAndGroups() (*mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefaultB2bSettingUsersAndGroups, error) {
	return a.UsersAndGroups.Data, a.UsersAndGroups.Error
}

// applications returns the applications field from the b2bSetting resource
func (a *mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefaultB2bSetting) applications() (*mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefaultB2bSettingUsersAndGroups, error) {
	return a.Applications.Data, a.Applications.Error
}

func newUsersAndGroups(runtime *plugin.Runtime, usersAndGroups models.CrossTenantAccessPolicyTargetConfigurationable, id string) (*mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefaultB2bSettingUsersAndGroups, error) {
	if usersAndGroups == nil {
		return nil, nil
	}

	var accessType string
	if usersAndGroups.GetAccessType() != nil {
		accessType = usersAndGroups.GetAccessType().String()
	}

	var targetResources []any
	for _, target := range usersAndGroups.GetTargets() {
		var targetType string
		if target.GetTargetType() != nil {
			targetType = target.GetTargetType().String()
		}
		var targetValue string
		if target.GetTarget() != nil {
			targetValue = *target.GetTarget()
		}

		targetResource, err := CreateResource(runtime, "microsoft.policies.crossTenantAccessPolicyConfigurationDefault.b2bSetting.target",
			map[string]*llx.RawData{
				"__id":       llx.StringData(fmt.Sprintf("%s-%s", id, targetValue)),
				"target":     llx.StringData(targetValue),
				"targetType": llx.StringData(targetType),
			})
		if err != nil {
			return nil, err
		}
		targetResources = append(targetResources, targetResource)
	}

	resource, err := CreateResource(runtime, "microsoft.policies.crossTenantAccessPolicyConfigurationDefault.b2bSetting.usersAndGroups",
		map[string]*llx.RawData{
			"__id":       llx.StringData(id),
			"accessType": llx.StringData(accessType),
			"targets":    llx.ArrayData(targetResources, "microsoft.policies.crossTenantAccessPolicyConfigurationDefault.b2bSetting.target"),
		})
	if err != nil {
		return nil, err
	}

	return resource.(*mqlMicrosoftPoliciesCrossTenantAccessPolicyConfigurationDefaultB2bSettingUsersAndGroups), nil
}
