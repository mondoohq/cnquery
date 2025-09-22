// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers/ms365/connection"
	"go.mondoo.com/cnquery/v12/types"
)

func (m *mqlMicrosoftConditionalAccessIpNamedLocation) id() (string, error) {
	return m.Name.Data, nil
}

func (a *mqlMicrosoftConditionalAccessNamedLocations) ipLocations() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	namedLocations, err := graphClient.Identity().ConditionalAccess().NamedLocations().Get(ctx, nil)
	if err != nil {
		return nil, transformError(err)
	}

	var locationDetails []any
	for _, location := range namedLocations.GetValue() {
		if ipLocation, ok := location.(*models.IpNamedLocation); ok {
			displayName := ipLocation.GetDisplayName()
			isTrusted := ipLocation.GetIsTrusted()

			if displayName != nil {
				trusted := false
				if isTrusted != nil {
					trusted = *isTrusted
				}

				locationInfo, err := CreateResource(a.MqlRuntime, "microsoft.conditionalAccess.ipNamedLocation",
					map[string]*llx.RawData{
						"name":    llx.StringDataPtr(displayName),
						"trusted": llx.BoolData(trusted),
					})
				if err != nil {
					return nil, err
				}
				locationDetails = append(locationDetails, locationInfo)
			}
		}
	}

	if len(locationDetails) == 0 {
		return nil, nil
	}

	return locationDetails, nil
}

func (m *mqlMicrosoftConditionalAccessCountryNamedLocation) id() (string, error) {
	return m.Name.Data, nil
}

func (a *mqlMicrosoftConditionalAccessNamedLocations) countryLocations() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	namedLocations, err := graphClient.Identity().ConditionalAccess().NamedLocations().Get(ctx, nil)
	if err != nil {
		return nil, transformError(err)
	}

	var locationDetails []any
	for _, location := range namedLocations.GetValue() {
		if countryLocation, ok := location.(*models.CountryNamedLocation); ok {
			displayName := countryLocation.GetDisplayName()
			countryLookupMethod := countryLocation.GetCountryLookupMethod()

			var lookupMethodStr *string
			if countryLookupMethod != nil {
				method := countryLookupMethod.String()
				lookupMethodStr = &method
			}

			if displayName != nil && lookupMethodStr != nil {
				locationInfo, err := CreateResource(a.MqlRuntime, "microsoft.conditionalAccess.countryNamedLocation",
					map[string]*llx.RawData{
						"name":         llx.StringDataPtr(displayName),
						"lookupMethod": llx.StringDataPtr(lookupMethodStr),
					})
				if err != nil {
					return nil, err
				}
				locationDetails = append(locationDetails, locationInfo)
			}
		}
	}

	if len(locationDetails) == 0 {
		return nil, nil
	}

	return locationDetails, nil
}

func (m *mqlMicrosoftConditionalAccessPolicy) id() (string, error) {
	return m.Id.Data, nil
}

func (m *mqlMicrosoftConditionalAccessPolicyConditions) id() (string, error) {
	return m.Id.Data, nil
}

func (m *mqlMicrosoftConditionalAccessPolicyGrantControls) id() (string, error) {
	return m.Id.Data, nil
}

func (m *mqlMicrosoftConditionalAccessPolicySessionControls) id() (string, error) {
	return m.Id.Data, nil
}

// Conditional Access Policies
func (a *mqlMicrosoftConditionalAccess) policies() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	policies, err := graphClient.Identity().ConditionalAccess().Policies().Get(ctx, nil)
	if err != nil {
		return nil, transformError(err)
	}

	var policyDetails []any
	for _, policy := range policies.GetValue() {
		id := policy.GetId()
		displayName := policy.GetDisplayName()

		if id != nil && displayName != nil {
			policyResource, err := a.createPolicyResource(policy)
			if err != nil {
				return nil, err
			}
			policyDetails = append(policyDetails, policyResource)
		}
	}

	return policyDetails, nil
}

// Helper methods for creating policy sub-resources
func (a *mqlMicrosoftConditionalAccess) createPolicyResource(policy models.ConditionalAccessPolicyable) (any, error) {
	id := policy.GetId()
	if id == nil {
		return nil, fmt.Errorf("policy ID cannot be nil")
	}

	var conditions plugin.Resource
	if mConditions := policy.GetConditions(); mConditions != nil {
		var err error
		// Create conditions resource
		conditions, err = a.createConditionsResource(*id, mConditions)
		if err != nil {
			return nil, fmt.Errorf("failed to create conditions resource: %w", err)
		}
	}

	// Create grant controls resource
	var grantControls plugin.Resource
	if mGrantControls := policy.GetGrantControls(); mGrantControls != nil {
		var err error
		grantControls, err = a.createGrantControlsResource(*id, mGrantControls)
		if err != nil {
			return nil, fmt.Errorf("failed to create grant controls resource: %w", err)
		}
	}

	var sessionControls plugin.Resource
	if mSessionControls := policy.GetSessionControls(); mSessionControls != nil {
		var err error
		// Create session controls resource
		sessionControls, err = a.createSessionControlsResource(*id, mSessionControls)
		if err != nil {
			return nil, fmt.Errorf("failed to create session controls resource: %w", err)
		}
	}

	policyInfo, err := CreateResource(a.MqlRuntime, "microsoft.conditionalAccess.policy",
		map[string]*llx.RawData{
			"__id":             llx.StringDataPtr(id),
			"id":               llx.StringDataPtr(id),
			"templateId":       llx.StringDataPtr(policy.GetTemplateId()),
			"displayName":      llx.StringDataPtr(policy.GetDisplayName()),
			"createdDateTime":  llx.TimeDataPtr(policy.GetCreatedDateTime()),
			"modifiedDateTime": llx.TimeDataPtr(policy.GetModifiedDateTime()),
			"state":            llx.StringData(policy.GetState().String()),
			"sessionControls":  llx.ResourceData(sessionControls, "microsoft.conditionalAccess.policy.sessionControls"),
			"conditions":       llx.ResourceData(conditions, "microsoft.conditionalAccess.policy.conditions"),
			"grantControls":    llx.ResourceData(grantControls, "microsoft.conditionalAccess.policy.grantControls"),
		})
	if err != nil {
		return nil, fmt.Errorf("failed to create policy resource: %w", err)
	}

	return policyInfo, nil
}

func (a *mqlMicrosoftConditionalAccess) createConditionsResource(
	policyId string,
	conditions models.ConditionalAccessConditionSetable,
) (plugin.Resource, error) {
	conditionsId := policyId + "_conditions"

	var err error
	var mqlApplications plugin.Resource
	var mqlUsers plugin.Resource
	var mqlLocations plugin.Resource
	var mqlPlatforms plugin.Resource
	var mqlClientApplications plugin.Resource
	var mqlAuthenticationFlows plugin.Resource

	if conditions != nil && conditions.GetAuthenticationFlows() != nil {
		authenticationFlows := conditions.GetAuthenticationFlows()
		usersData := map[string]*llx.RawData{
			"__id":            llx.StringData(policyId + "_conditions_authflows"),
			"transferMethods": llx.StringData(authenticationFlows.GetTransferMethods().String()),
		}

		mqlAuthenticationFlows, err = CreateResource(a.MqlRuntime, "microsoft.conditionalAccess.policy.conditions.authenticationFlows", usersData)
		if err != nil {
			return nil, err
		}
	}

	// Extract users
	if conditions != nil && conditions.GetUsers() != nil {
		users := conditions.GetUsers()
		usersData := map[string]*llx.RawData{
			"__id":          llx.StringData(policyId + "_conditions_users"),
			"includeUsers":  llx.ArrayData(convert.SliceAnyToInterface(users.GetIncludeUsers()), types.String),
			"excludeUsers":  llx.ArrayData(convert.SliceAnyToInterface(users.GetExcludeUsers()), types.String),
			"includeGroups": llx.ArrayData(convert.SliceAnyToInterface(users.GetIncludeGroups()), types.String),
			"excludeGroups": llx.ArrayData(convert.SliceAnyToInterface(users.GetExcludeGroups()), types.String),
			"includeRoles":  llx.ArrayData(convert.SliceAnyToInterface(users.GetIncludeRoles()), types.String),
			"excludeRoles":  llx.ArrayData(convert.SliceAnyToInterface(users.GetExcludeRoles()), types.String),
		}

		mqlUsers, err = CreateResource(a.MqlRuntime, "microsoft.conditionalAccess.policy.conditions.users", usersData)
		if err != nil {
			return nil, err
		}
	}

	if conditions != nil && conditions.GetApplications() != nil {
		apps := conditions.GetApplications()
		mqlApplications, err = CreateResource(a.MqlRuntime, "microsoft.conditionalAccess.policy.conditions.applications", map[string]*llx.RawData{
			"__id":                llx.StringData(policyId + "_conditions_applications"),
			"includeApplications": llx.ArrayData(convert.SliceAnyToInterface(apps.GetIncludeApplications()), types.String),
			"excludeApplications": llx.ArrayData(convert.SliceAnyToInterface(apps.GetExcludeApplications()), types.String),
			"includeUserActions":  llx.ArrayData(convert.SliceAnyToInterface(apps.GetIncludeUserActions()), types.String),
		})
		if err != nil {
			return nil, err
		}
	}

	// Extract locations and create mqlLocations resource
	if conditions != nil && conditions.GetLocations() != nil {
		locations := conditions.GetLocations()
		locationsData := map[string]*llx.RawData{
			"__id":             llx.StringData(policyId + "_conditions_locations"),
			"includeLocations": llx.ArrayData(convert.SliceAnyToInterface(locations.GetIncludeLocations()), types.String),
			"excludeLocations": llx.ArrayData(convert.SliceAnyToInterface(locations.GetExcludeLocations()), types.String),
		}
		mqlLocations, err = CreateResource(a.MqlRuntime, "microsoft.conditionalAccess.policy.conditions.locations", locationsData)
		if err != nil {
			return nil, err
		}
	}

	// Extract platforms and create mqlPlatforms resource
	if conditions != nil && conditions.GetPlatforms() != nil {
		platforms := conditions.GetPlatforms()
		platformsData := map[string]*llx.RawData{"__id": llx.StringData(policyId + "_conditions_platforms")}

		var includePlatformStrings []string
		for _, platform := range platforms.GetIncludePlatforms() {
			if platform.String() != "" {
				includePlatformStrings = append(includePlatformStrings, platform.String())
			}
		}
		platformsData["includePlatforms"] = llx.ArrayData(convert.SliceAnyToInterface(includePlatformStrings), types.String)

		var excludePlatformStrings []string
		for _, platform := range platforms.GetExcludePlatforms() {
			if platform.String() != "" {
				excludePlatformStrings = append(excludePlatformStrings, platform.String())
			}
		}
		platformsData["excludePlatforms"] = llx.ArrayData(convert.SliceAnyToInterface(excludePlatformStrings), types.String)

		mqlPlatforms, err = CreateResource(a.MqlRuntime, "microsoft.conditionalAccess.policy.conditions.platforms", platformsData)
		if err != nil {
			return nil, err
		}
	}

	// Extract client applications and create mqlClientApplications resource
	if conditions != nil && conditions.GetClientApplications() != nil {
		clientApps := conditions.GetClientApplications()
		clientApplicationsData := map[string]*llx.RawData{
			"__id":                     llx.StringData(policyId + "_conditions_client_applications"),
			"includeServicePrincipals": llx.ArrayData(convert.SliceAnyToInterface(clientApps.GetIncludeServicePrincipals()), types.String),
			"excludeServicePrincipals": llx.ArrayData(convert.SliceAnyToInterface(clientApps.GetExcludeServicePrincipals()), types.String),
		}
		mqlClientApplications, err = CreateResource(a.MqlRuntime, "microsoft.conditionalAccess.policy.conditions.clientApplications", clientApplicationsData)
		if err != nil {
			return nil, err
		}
	}

	var insiderRiskLevelsStr string
	if conditions != nil && conditions.GetInsiderRiskLevels() != nil {
		insiderRiskLevelsStr = conditions.GetInsiderRiskLevels().String()
	}

	conditionsResource, err := CreateResource(a.MqlRuntime, "microsoft.conditionalAccess.policy.conditions",
		map[string]*llx.RawData{
			"__id":                       llx.StringData(conditionsId),
			"id":                         llx.StringData(conditionsId),
			"applications":               llx.ResourceData(mqlApplications, "microsoft.conditionalAccess.policy.conditions.applications"),
			"authenticationFlows":        llx.ResourceData(mqlAuthenticationFlows, "microsoft.conditionalAccess.policy.conditions.authenticationFlows"),
			"users":                      llx.ResourceData(mqlUsers, "microsoft.conditionalAccess.policy.conditions.users"),
			"locations":                  llx.ResourceData(mqlLocations, "microsoft.conditionalAccess.policy.conditions.locations"),
			"platforms":                  llx.ResourceData(mqlPlatforms, "microsoft.conditionalAccess.policy.conditions.platforms"),
			"clientApplications":         llx.ResourceData(mqlClientApplications, "microsoft.conditionalAccess.policy.conditions.clientApplications"),
			"insiderRiskLevels":          llx.StringData(insiderRiskLevelsStr),
			"clientAppTypes":             llx.ArrayData(convert.SliceAnyToInterface(convertEnumCollectionToStrings(conditions.GetClientAppTypes())), types.String),
			"userRiskLevels":             llx.ArrayData(convert.SliceAnyToInterface(convertEnumCollectionToStrings(conditions.GetUserRiskLevels())), types.String),
			"signInRiskLevels":           llx.ArrayData(convert.SliceAnyToInterface(convertEnumCollectionToStrings(conditions.GetSignInRiskLevels())), types.String),
			"servicePrincipalRiskLevels": llx.ArrayData(convert.SliceAnyToInterface(convertEnumCollectionToStrings(conditions.GetServicePrincipalRiskLevels())), types.String),
		})

	return conditionsResource, err
}

func (a *mqlMicrosoftConditionalAccess) createGrantControlsResource(
	policyId string,
	grantControls models.ConditionalAccessGrantControlsable,
) (plugin.Resource, error) {
	grantControlsId := policyId + "_grantControls"

	var mqlAuthStrength plugin.Resource

	// Create authenticationStrength resource if it exists
	if grantControls.GetAuthenticationStrength() != nil {
		authStrength := grantControls.GetAuthenticationStrength()
		authStrengthData := map[string]*llx.RawData{
			"__id":                  llx.StringDataPtr(authStrength.GetId()),
			"id":                    llx.StringDataPtr(authStrength.GetId()),
			"displayName":           llx.StringDataPtr(authStrength.GetDisplayName()),
			"description":           llx.StringDataPtr(authStrength.GetDescription()),
			"policyType":            llx.StringData(authStrength.GetPolicyType().String()),
			"requirementsSatisfied": llx.StringData(authStrength.GetRequirementsSatisfied().String()),
			"allowedCombinations":   llx.ArrayData(convert.SliceAnyToInterface(convertEnumCollectionToStrings(authStrength.GetAllowedCombinations())), types.String),
			"createdDateTime":       llx.TimeDataPtr(authStrength.GetCreatedDateTime()),
			"modifiedDateTime":      llx.TimeDataPtr(authStrength.GetModifiedDateTime()),
		}

		var err error
		mqlAuthStrength, err = CreateResource(a.MqlRuntime, "microsoft.conditionalAccess.policy.grantControls.authenticationStrength", authStrengthData)
		if err != nil {
			return nil, err
		}
	}

	customAuthFactors := grantControls.GetCustomAuthenticationFactors()
	termsOfUse := grantControls.GetTermsOfUse()
	builtInControls := convertEnumCollectionToStrings(grantControls.GetBuiltInControls())

	return CreateResource(a.MqlRuntime, "microsoft.conditionalAccess.policy.grantControls",
		map[string]*llx.RawData{
			"__id":                        llx.StringData(grantControlsId),
			"id":                          llx.StringData(grantControlsId),
			"operator":                    llx.StringDataPtr(grantControls.GetOperator()),
			"builtInControls":             llx.ArrayData(convert.SliceAnyToInterface(builtInControls), types.String),
			"customAuthenticationFactors": llx.ArrayData(convert.SliceAnyToInterface(customAuthFactors), types.String),
			"termsOfUse":                  llx.ArrayData(convert.SliceAnyToInterface(termsOfUse), types.String),
			"authenticationStrength":      llx.ResourceData(mqlAuthStrength, "microsoft.conditionalAccess.policy.grantControls.authenticationStrength"),
		})
}

func (a *mqlMicrosoftConditionalAccess) createSessionControlsResource(
	policyId string,
	sessionControls models.ConditionalAccessSessionControlsable,
) (plugin.Resource, error) {
	sessionControlsId := policyId + "_sessionControls"

	var mqlSignInFreq plugin.Resource
	var mqlCloudAppSecurity plugin.Resource
	var mqlPersistentBrowser plugin.Resource
	var mqlAppEnforcedRestrictions plugin.Resource
	var mqlSecureSignInSession plugin.Resource

	// Create signInFrequency resource
	if sessionControls != nil && sessionControls.GetSignInFrequency() != nil {
		signInFreq := sessionControls.GetSignInFrequency()
		signInFreqData := map[string]*llx.RawData{
			"__id":               llx.StringData(policyId + "_session_signInFrequency"),
			"authenticationType": llx.StringData(signInFreq.GetAuthenticationType().String()),
			"frequencyInterval":  llx.StringData(signInFreq.GetFrequencyInterval().String()),
			"isEnabled":          llx.BoolDataPtr(signInFreq.GetIsEnabled()),
		}
		var err error
		mqlSignInFreq, err = CreateResource(a.MqlRuntime, "microsoft.conditionalAccess.policy.sessionControls.signInFrequency", signInFreqData)
		if err != nil {
			return nil, err
		}
	}

	// Create cloudAppSecurity resource
	if sessionControls != nil && sessionControls.GetCloudAppSecurity() != nil {
		cloudAppSecurity := sessionControls.GetCloudAppSecurity()
		cloudAppSecurityData := map[string]*llx.RawData{
			"__id":                 llx.StringData(policyId + "_session_cloudAppSecurity"),
			"cloudAppSecurityType": llx.StringData(cloudAppSecurity.GetCloudAppSecurityType().String()),
			"isEnabled":            llx.BoolDataPtr(cloudAppSecurity.GetIsEnabled()),
		}
		var err error
		mqlCloudAppSecurity, err = CreateResource(a.MqlRuntime, "microsoft.conditionalAccess.policy.sessionControls.cloudAppSecurity", cloudAppSecurityData)
		if err != nil {
			return nil, err
		}
	}

	// Create persistentBrowser resource
	if sessionControls != nil && sessionControls.GetPersistentBrowser() != nil {
		persistentBrowser := sessionControls.GetPersistentBrowser()
		persistentBrowserData := map[string]*llx.RawData{
			"__id":      llx.StringData(policyId + "_session_persistentBrowser"),
			"mode":      llx.StringData(persistentBrowser.GetMode().String()),
			"isEnabled": llx.BoolDataPtr(persistentBrowser.GetIsEnabled()),
		}
		var err error
		mqlPersistentBrowser, err = CreateResource(a.MqlRuntime, "microsoft.conditionalAccess.policy.sessionControls.persistentBrowser", persistentBrowserData)
		if err != nil {
			return nil, err
		}
	}

	// Create applicationEnforcedRestrictions resource
	if sessionControls != nil && sessionControls.GetApplicationEnforcedRestrictions() != nil {
		appEnforcedRestrictions := sessionControls.GetApplicationEnforcedRestrictions()
		appEnforcedRestrictionsData := map[string]*llx.RawData{
			"__id":      llx.StringData(policyId + "_session_appEnforcedRestrictions"),
			"isEnabled": llx.BoolDataPtr(appEnforcedRestrictions.GetIsEnabled()),
		}
		var err error
		mqlAppEnforcedRestrictions, err = CreateResource(a.MqlRuntime, "microsoft.conditionalAccess.policy.sessionControls.applicationEnforcedRestrictions", appEnforcedRestrictionsData)
		if err != nil {
			return nil, err
		}
	}

	// Create secureSignInSession resource
	if sessionControls != nil && sessionControls.GetDisableResilienceDefaults() != nil {
		secureSignInSessionData := map[string]*llx.RawData{
			"__id":                      llx.StringData(policyId + "_session_secureSignInSession"),
			"disableResilienceDefaults": llx.BoolDataPtr(sessionControls.GetDisableResilienceDefaults()),
		}
		var err error
		mqlSecureSignInSession, err = CreateResource(a.MqlRuntime, "microsoft.conditionalAccess.policy.sessionControls.secureSignInSession", secureSignInSessionData)
		if err != nil {
			return nil, err
		}
	}

	return CreateResource(a.MqlRuntime, "microsoft.conditionalAccess.policy.sessionControls",
		map[string]*llx.RawData{
			"__id":                            llx.StringData(sessionControlsId),
			"id":                              llx.StringData(sessionControlsId),
			"signInFrequency":                 llx.ResourceData(mqlSignInFreq, "microsoft.conditionalAccess.policy.sessionControls.signInFrequency"),
			"cloudAppSecurity":                llx.ResourceData(mqlCloudAppSecurity, "microsoft.conditionalAccess.policy.sessionControls.cloudAppSecurity"),
			"persistentBrowser":               llx.ResourceData(mqlPersistentBrowser, "microsoft.conditionalAccess.policy.sessionControls.persistentBrowser"),
			"applicationEnforcedRestrictions": llx.ResourceData(mqlAppEnforcedRestrictions, "microsoft.conditionalAccess.policy.sessionControls.applicationEnforcedRestrictions"),
			"secureSignInSession":             llx.ResourceData(mqlSecureSignInSession, "microsoft.conditionalAccess.policy.sessionControls.secureSignInSession"),
		})
}

func convertEnumCollectionToStrings[T fmt.Stringer](enums []T) []string {
	var result []string
	if enums == nil {
		return nil
	}
	for _, enum := range enums {
		if enum.String() != "" {
			result = append(result, enum.String())
		}
	}
	return result
}

func (a *mqlMicrosoftConditionalAccess) authenticationMethodsPolicy() (*mqlMicrosoftConditionalAccessAuthenticationMethodsPolicy, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	policy, err := graphClient.Policies().AuthenticationMethodsPolicy().Get(context.Background(), nil)
	if err != nil {
		return nil, transformError(err)
	}

	if policy == nil {
		return nil, nil
	}

	return newMqlAuthenticationMethodsPolicy(a.MqlRuntime, policy)
}

func newMqlAuthenticationMethodsPolicy(runtime *plugin.Runtime, policy models.AuthenticationMethodsPolicyable) (*mqlMicrosoftConditionalAccessAuthenticationMethodsPolicy, error) {
	if policy.GetId() == nil {
		return nil, fmt.Errorf("authentication methods policy has a nil ID")
	}

	var authMethodConfigs []any
	for _, config := range policy.GetAuthenticationMethodConfigurations() {
		mqlConfig, err := newMqlAuthenticationMethodConfiguration(runtime, config)
		if err != nil {
			return nil, err
		}
		authMethodConfigs = append(authMethodConfigs, mqlConfig)
	}

	resource, err := CreateResource(runtime, "microsoft.conditionalAccess.authenticationMethodsPolicy", map[string]*llx.RawData{
		"__id":                               llx.StringDataPtr(policy.GetId()),
		"id":                                 llx.StringDataPtr(policy.GetId()),
		"displayName":                        llx.StringDataPtr(policy.GetDisplayName()),
		"description":                        llx.StringDataPtr(policy.GetDescription()),
		"lastModifiedDateTime":               llx.TimeDataPtr(policy.GetLastModifiedDateTime()),
		"policyVersion":                      llx.StringDataPtr(policy.GetPolicyVersion()),
		"authenticationMethodConfigurations": llx.ArrayData(llx.TArr2Raw(authMethodConfigs), types.Resource("microsoft.conditionalAccess.authenticationMethodConfiguration")),
	})
	if err != nil {
		return nil, err
	}

	return resource.(*mqlMicrosoftConditionalAccessAuthenticationMethodsPolicy), nil
}

func newMqlAuthenticationMethodConfiguration(runtime *plugin.Runtime, config models.AuthenticationMethodConfigurationable) (*mqlMicrosoftConditionalAccessAuthenticationMethodConfiguration, error) {
	resource, err := CreateResource(runtime, "microsoft.conditionalAccess.authenticationMethodConfiguration", map[string]*llx.RawData{
		"__id":  llx.StringDataPtr(config.GetId()),
		"id":    llx.StringDataPtr(config.GetId()),
		"state": llx.StringData(config.GetState().String()),
	})
	if err != nil {
		return nil, err
	}

	return resource.(*mqlMicrosoftConditionalAccessAuthenticationMethodConfiguration), nil
}
