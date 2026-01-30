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

func (a *mqlMicrosoftPolicies) authenticationMethodsPolicy() (*mqlMicrosoftAuthenticationMethodsPolicy, error) {
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

func newAuthenticationMethodsPolicy(runtime *plugin.Runtime, policy models.AuthenticationMethodsPolicyable) (*mqlMicrosoftAuthenticationMethodsPolicy, error) {
	authMethodConfigs, err := newAuthenticationMethodConfigurations(runtime, policy.GetAuthenticationMethodConfigurations())
	if err != nil {
		return nil, err
	}

	mqlAuthenticationMethodsPolicy, err := CreateResource(runtime, "microsoft.authenticationMethodsPolicy",
		map[string]*llx.RawData{
			"__id":                               llx.StringDataPtr(policy.GetId()),
			"id":                                 llx.StringDataPtr(policy.GetId()),
			"description":                        llx.StringDataPtr(policy.GetDescription()),
			"displayName":                        llx.StringDataPtr(policy.GetDisplayName()),
			"lastModifiedDateTime":               llx.TimeDataPtr(policy.GetLastModifiedDateTime()),
			"policyVersion":                      llx.StringDataPtr(policy.GetPolicyVersion()),
			"authenticationMethodConfigurations": llx.ArrayData(authMethodConfigs, "microsoft.authenticationMethodConfiguration"),
		})
	if err != nil {
		return nil, err
	}

	return mqlAuthenticationMethodsPolicy.(*mqlMicrosoftAuthenticationMethodsPolicy), nil
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

		state := ""
		if config.GetState() != nil {
			state = config.GetState().String()
		}

		configData := map[string]*llx.RawData{
			"__id":           llx.StringDataPtr(config.GetId()),
			"id":             llx.StringDataPtr(config.GetId()),
			"state":          llx.StringData(state),
			"excludeTargets": llx.ArrayData(excludeTargets, types.Dict),
		}

		mqlConfig, err := CreateResource(runtime, "microsoft.authenticationMethodConfiguration", configData)
		if err != nil {
			return nil, err
		}

		configResources = append(configResources, mqlConfig)
	}

	return configResources, nil
}

func (a *mqlMicrosoftAuthenticationMethodsPolicy) systemCredentialPreferences() (*mqlMicrosoftSystemCredentialPreferences, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	betaGraphClient, err := conn.BetaGraphClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	policy, err := betaGraphClient.Policies().AuthenticationMethodsPolicy().Get(ctx, nil)
	if err != nil {
		return nil, transformError(err)
	}

	systemCredPrefs := policy.GetSystemCredentialPreferences()
	if systemCredPrefs == nil {
		return nil, nil
	}

	// Convert include targets to []dict
	var includeTargets []any
	for _, target := range systemCredPrefs.GetIncludeTargets() {
		targetDict := map[string]any{}
		if target.GetId() != nil {
			targetDict["id"] = *target.GetId()
		}
		if target.GetTargetType() != nil {
			targetDict["targetType"] = target.GetTargetType().String()
		}
		includeTargets = append(includeTargets, targetDict)
	}

	// Convert exclude targets to []dict
	var excludeTargets []any
	for _, target := range systemCredPrefs.GetExcludeTargets() {
		targetDict := map[string]any{}
		if target.GetId() != nil {
			targetDict["id"] = *target.GetId()
		}
		if target.GetTargetType() != nil {
			targetDict["targetType"] = target.GetTargetType().String()
		}
		excludeTargets = append(excludeTargets, targetDict)
	}

	state := ""
	if systemCredPrefs.GetState() != nil {
		state = systemCredPrefs.GetState().String()
	}

	policyId := a.Id.Data

	mqlSystemCredPrefs, err := CreateResource(a.MqlRuntime, ResourceMicrosoftSystemCredentialPreferences,
		map[string]*llx.RawData{
			"__id":           llx.StringData(policyId + "/systemCredentialPreferences"),
			"state":          llx.StringData(state),
			"includeTargets": llx.ArrayData(includeTargets, types.Dict),
			"excludeTargets": llx.ArrayData(excludeTargets, types.Dict),
		})
	if err != nil {
		return nil, err
	}

	return mqlSystemCredPrefs.(*mqlMicrosoftSystemCredentialPreferences), nil
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

func (a *mqlMicrosoftPolicies) externalIdentitiesPolicy() (*mqlMicrosoftExternalIdentitiesPolicy, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	betaGraphClient, err := conn.BetaGraphClient()
	if err != nil {
		return nil, err
	}
	policy, err := betaGraphClient.Policies().ExternalIdentitiesPolicy().Get(context.Background(), nil)
	if err != nil {
		return nil, transformError(err)
	}

	mqlPolicy, err := CreateResource(a.MqlRuntime, "microsoft.externalIdentitiesPolicy",
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

	return mqlPolicy.(*mqlMicrosoftExternalIdentitiesPolicy), nil
}

func initMicrosoftExternalIdentitiesPolicy(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// Create the parent policies resource and call its method
	policiesResource, err := CreateResource(runtime, ResourceMicrosoftPolicies, map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}

	policy, err := policiesResource.(*mqlMicrosoftPolicies).externalIdentitiesPolicy()
	if err != nil {
		return nil, nil, err
	}

	return nil, policy, nil
}

// Internal struct for caching cross-tenant access policy data
// This will be embedded in mqlMicrosoftCrossTenantAccessPolicyDefault after code generation
type mqlMicrosoftCrossTenantAccessPolicyDefaultInternal struct {
	policyLock                                              sync.Mutex
	fetched                                                 bool
	fetchErr                                                error
	policy                                                  models.CrossTenantAccessPolicyConfigurationDefaultable
	cachedAutomaticUserConsentSettings                      *mqlMicrosoftCrossTenantAccessPolicyDefaultAutomaticUserConsentSettings
	cachedB2bCollaborationInbound                           *mqlMicrosoftCrossTenantAccessPolicyDefaultB2bSetting
	cachedB2bCollaborationOutbound                          *mqlMicrosoftCrossTenantAccessPolicyDefaultB2bSetting
	cachedB2bDirectConnectInbound                           *mqlMicrosoftCrossTenantAccessPolicyDefaultB2bSetting
	cachedB2bDirectConnectOutbound                          *mqlMicrosoftCrossTenantAccessPolicyDefaultB2bSetting
	cachedInvitationRedemptionIdentityProviderConfiguration *mqlMicrosoftCrossTenantAccessPolicyDefaultInvitationRedemptionIdentityProviderConfiguration
	cachedInboundTrust                                      *mqlMicrosoftCrossTenantAccessPolicyDefaultInboundTrust
	cachedTenantRestrictions                                *mqlMicrosoftCrossTenantAccessPolicyDefaultB2bSetting
}

func (a *mqlMicrosoftPolicies) crossTenantAccessPolicy() (*mqlMicrosoftCrossTenantAccessPolicyDefault, error) {
	resource, err := CreateResource(a.MqlRuntime, ResourceMicrosoftCrossTenantAccessPolicyDefault,
		map[string]*llx.RawData{
			"__id": llx.StringData("crossTenantAccessPolicyDefault"),
		})
	if err != nil {
		return nil, err
	}

	return resource.(*mqlMicrosoftCrossTenantAccessPolicyDefault), nil
}

func (a *mqlMicrosoftCrossTenantAccessPolicyDefault) getCrossTenantAccessPolicy() error {
	a.policyLock.Lock()
	defer a.policyLock.Unlock()

	if a.fetched {
		return a.fetchErr
	}

	a.fetched = true

	errHandler := func(err error) error {
		a.fetchErr = err
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

	a.policy = policy

	if policy.GetIsServiceDefault() != nil {
		a.IsServiceDefault = plugin.TValue[bool]{Data: *policy.GetIsServiceDefault(), State: plugin.StateIsSet}
	} else {
		a.IsServiceDefault = plugin.TValue[bool]{State: plugin.StateIsNull}
	}

	if policy.GetAutomaticUserConsentSettings() != nil {
		consentSettings := policy.GetAutomaticUserConsentSettings()
		consentResource, err := CreateResource(a.MqlRuntime, ResourceMicrosoftCrossTenantAccessPolicyDefaultAutomaticUserConsentSettings,
			map[string]*llx.RawData{
				"__id":            llx.StringData(a.__id + "-automaticUserConsentSettings"),
				"inboundAllowed":  llx.BoolDataPtr(consentSettings.GetInboundAllowed()),
				"outboundAllowed": llx.BoolDataPtr(consentSettings.GetOutboundAllowed()),
			})
		if err == nil {
			a.cachedAutomaticUserConsentSettings = consentResource.(*mqlMicrosoftCrossTenantAccessPolicyDefaultAutomaticUserConsentSettings)
		}
	}

	if policy.GetB2bCollaborationInbound() != nil {
		b2bResource, err := newB2BSetting(a.MqlRuntime, policy.GetB2bCollaborationInbound(), a.__id+"-b2bCollaborationInbound")
		if err == nil {
			a.cachedB2bCollaborationInbound = b2bResource
		}
	}

	if policy.GetB2bCollaborationOutbound() != nil {
		b2bResource, err := newB2BSetting(a.MqlRuntime, policy.GetB2bCollaborationOutbound(), a.__id+"-b2bCollaborationOutbound")
		if err == nil {
			a.cachedB2bCollaborationOutbound = b2bResource
		}
	}

	if policy.GetB2bDirectConnectInbound() != nil {
		b2bResource, err := newB2BSetting(a.MqlRuntime, policy.GetB2bDirectConnectInbound(), a.__id+"-b2bDirectConnectInbound")
		if err == nil {
			a.cachedB2bDirectConnectInbound = b2bResource
		}
	}

	if policy.GetB2bDirectConnectOutbound() != nil {
		b2bResource, err := newB2BSetting(a.MqlRuntime, policy.GetB2bDirectConnectOutbound(), a.__id+"-b2bDirectConnectOutbound")
		if err == nil {
			a.cachedB2bDirectConnectOutbound = b2bResource
		}
	}

	if policy.GetInvitationRedemptionIdentityProviderConfiguration() != nil {
		invConfig := policy.GetInvitationRedemptionIdentityProviderConfiguration()
		var fallbackProvider string
		if invConfig.GetFallbackIdentityProvider() != nil {
			fallbackProvider = invConfig.GetFallbackIdentityProvider().String()
		}
		var precedenceOrder []any
		for _, provider := range invConfig.GetPrimaryIdentityProviderPrecedenceOrder() {
			precedenceOrder = append(precedenceOrder, provider.String())
		}

		invResource, err := CreateResource(a.MqlRuntime, ResourceMicrosoftCrossTenantAccessPolicyDefaultInvitationRedemptionIdentityProviderConfiguration,
			map[string]*llx.RawData{
				"__id":                                   llx.StringData(a.__id + "-invitationRedemptionIdentityProviderConfiguration"),
				"fallbackIdentityProvider":               llx.StringData(fallbackProvider),
				"primaryIdentityProviderPrecedenceOrder": llx.ArrayData(precedenceOrder, types.String),
			})
		if err == nil {
			a.cachedInvitationRedemptionIdentityProviderConfiguration = invResource.(*mqlMicrosoftCrossTenantAccessPolicyDefaultInvitationRedemptionIdentityProviderConfiguration)
		}
	}

	if policy.GetInboundTrust() != nil {
		inboundTrustValue := policy.GetInboundTrust()
		inboundTrustResource, err := CreateResource(a.MqlRuntime, ResourceMicrosoftCrossTenantAccessPolicyDefaultInboundTrust,
			map[string]*llx.RawData{
				"__id":                                llx.StringData(a.__id + "-inboundTrust"),
				"isMfaAccepted":                       llx.BoolDataPtr(inboundTrustValue.GetIsMfaAccepted()),
				"isCompliantDeviceAccepted":           llx.BoolDataPtr(inboundTrustValue.GetIsCompliantDeviceAccepted()),
				"isHybridAzureADJoinedDeviceAccepted": llx.BoolDataPtr(inboundTrustValue.GetIsHybridAzureADJoinedDeviceAccepted()),
			})
		if err == nil {
			a.cachedInboundTrust = inboundTrustResource.(*mqlMicrosoftCrossTenantAccessPolicyDefaultInboundTrust)
		}
	}

	if policy.GetTenantRestrictions() != nil {
		b2bResource, err := newB2BSetting(a.MqlRuntime, policy.GetTenantRestrictions(), a.__id+"-tenantRestrictions")
		if err == nil {
			a.cachedTenantRestrictions = b2bResource
		}
	}

	return nil
}

func (a *mqlMicrosoftCrossTenantAccessPolicyDefault) automaticUserConsentSettings() (*mqlMicrosoftCrossTenantAccessPolicyDefaultAutomaticUserConsentSettings, error) {
	if err := a.getCrossTenantAccessPolicy(); err != nil {
		return nil, err
	}

	return a.cachedAutomaticUserConsentSettings, nil
}

func (a *mqlMicrosoftCrossTenantAccessPolicyDefault) b2bCollaborationInbound() (*mqlMicrosoftCrossTenantAccessPolicyDefaultB2bSetting, error) {
	if err := a.getCrossTenantAccessPolicy(); err != nil {
		return nil, err
	}

	return a.cachedB2bCollaborationInbound, nil
}

func (a *mqlMicrosoftCrossTenantAccessPolicyDefault) b2bCollaborationOutbound() (*mqlMicrosoftCrossTenantAccessPolicyDefaultB2bSetting, error) {
	if err := a.getCrossTenantAccessPolicy(); err != nil {
		return nil, err
	}

	return a.cachedB2bCollaborationOutbound, nil
}

func (a *mqlMicrosoftCrossTenantAccessPolicyDefault) b2bDirectConnectInbound() (*mqlMicrosoftCrossTenantAccessPolicyDefaultB2bSetting, error) {
	if err := a.getCrossTenantAccessPolicy(); err != nil {
		return nil, err
	}

	return a.cachedB2bDirectConnectInbound, nil
}

func (a *mqlMicrosoftCrossTenantAccessPolicyDefault) b2bDirectConnectOutbound() (*mqlMicrosoftCrossTenantAccessPolicyDefaultB2bSetting, error) {
	if err := a.getCrossTenantAccessPolicy(); err != nil {
		return nil, err
	}

	return a.cachedB2bDirectConnectOutbound, nil
}

func (a *mqlMicrosoftCrossTenantAccessPolicyDefault) tenantRestrictions() (*mqlMicrosoftCrossTenantAccessPolicyDefaultB2bSetting, error) {
	if err := a.getCrossTenantAccessPolicy(); err != nil {
		return nil, err
	}

	return a.cachedTenantRestrictions, nil
}

func (a *mqlMicrosoftCrossTenantAccessPolicyDefault) inboundTrust() (*mqlMicrosoftCrossTenantAccessPolicyDefaultInboundTrust, error) {
	if err := a.getCrossTenantAccessPolicy(); err != nil {
		return nil, err
	}

	return a.cachedInboundTrust, nil
}

func (a *mqlMicrosoftCrossTenantAccessPolicyDefault) invitationRedemptionIdentityProviderConfiguration() (*mqlMicrosoftCrossTenantAccessPolicyDefaultInvitationRedemptionIdentityProviderConfiguration, error) {
	if err := a.getCrossTenantAccessPolicy(); err != nil {
		return nil, err
	}

	return a.cachedInvitationRedemptionIdentityProviderConfiguration, nil
}

func newB2BSetting(runtime *plugin.Runtime, setting models.CrossTenantAccessPolicyB2BSettingable, settingId string) (*mqlMicrosoftCrossTenantAccessPolicyDefaultB2bSetting, error) {
	usersAndGroups, err := newCrossTenantAccessPolicyTarget(runtime, setting.GetUsersAndGroups(), settingId+"-usersAndGroups")
	if err != nil {
		return nil, err
	}

	applications, err := newCrossTenantAccessPolicyTarget(runtime, setting.GetApplications(), settingId+"-applications")
	if err != nil {
		return nil, err
	}

	resource, err := CreateResource(runtime, ResourceMicrosoftCrossTenantAccessPolicyDefaultB2bSetting,
		map[string]*llx.RawData{
			"__id":           llx.StringData(settingId),
			"usersAndGroups": llx.ResourceData(usersAndGroups, string(ResourceMicrosoftCrossTenantAccessPolicyDefaultB2bSettingTargetConfig)),
			"applications":   llx.ResourceData(applications, string(ResourceMicrosoftCrossTenantAccessPolicyDefaultB2bSettingTargetConfig)),
		})
	if err != nil {
		return nil, err
	}

	return resource.(*mqlMicrosoftCrossTenantAccessPolicyDefaultB2bSetting), nil
}

func (a *mqlMicrosoftCrossTenantAccessPolicyDefaultB2bSetting) usersAndGroups() (*mqlMicrosoftCrossTenantAccessPolicyDefaultB2bSettingTargetConfig, error) {
	return a.UsersAndGroups.Data, a.UsersAndGroups.Error
}

func (a *mqlMicrosoftCrossTenantAccessPolicyDefaultB2bSetting) applications() (*mqlMicrosoftCrossTenantAccessPolicyDefaultB2bSettingTargetConfig, error) {
	return a.Applications.Data, a.Applications.Error
}

func newCrossTenantAccessPolicyTarget(runtime *plugin.Runtime, accessPolicyTargetConfiguration models.CrossTenantAccessPolicyTargetConfigurationable, id string) (*mqlMicrosoftCrossTenantAccessPolicyDefaultB2bSettingTargetConfig, error) {
	var accessType string
	if accessPolicyTargetConfiguration.GetAccessType() != nil {
		accessType = accessPolicyTargetConfiguration.GetAccessType().String()
	}

	var targetResources []any
	for _, target := range accessPolicyTargetConfiguration.GetTargets() {
		var targetType string
		if target.GetTargetType() != nil {
			targetType = target.GetTargetType().String()
		}
		var targetValue string
		if target.GetTarget() != nil {
			targetValue = *target.GetTarget()
		}

		targetResource, err := CreateResource(runtime, ResourceMicrosoftCrossTenantAccessPolicyDefaultB2bSettingTarget,
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

	resource, err := CreateResource(runtime, ResourceMicrosoftCrossTenantAccessPolicyDefaultB2bSettingTargetConfig,
		map[string]*llx.RawData{
			"__id":       llx.StringData(id),
			"accessType": llx.StringData(accessType),
			"targets":    llx.ArrayData(targetResources, types.Resource(string(ResourceMicrosoftCrossTenantAccessPolicyDefaultB2bSettingTarget))),
		})
	if err != nil {
		return nil, err
	}

	return resource.(*mqlMicrosoftCrossTenantAccessPolicyDefaultB2bSettingTargetConfig), nil
}
