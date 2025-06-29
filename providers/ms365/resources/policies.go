// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/google/uuid"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/policies"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/ms365/connection"
	"go.mondoo.com/cnquery/v11/types"
)

// PowerShell script to get activity-based timeout policies using Microsoft Graph PowerShell SDK
var activityBasedTimeoutPoliciesScript = `
$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"
$graphToken = '%s'

# Install and import Microsoft Graph Identity SignIns module
Install-Module -Name Microsoft.Graph.Identity.SignIns -Scope CurrentUser -Force -AllowClobber
Import-Module Microsoft.Graph.Identity.SignIns

# Convert the access token string to SecureString (required by Microsoft Graph PowerShell v2.0+)
$secureToken = ConvertTo-SecureString -String $graphToken -AsPlainText -Force

# Connect to Microsoft Graph using the secure access token (suppress welcome message)
Connect-MgGraph -AccessToken $secureToken -NoWelcome

# Get activity-based timeout policies
$policies = @(Get-MgPolicyActivityBasedTimeoutPolicy)

# Disconnect from Microsoft Graph
Disconnect-MgGraph

# Convert to JSON output
$result = @{
    ActivityBasedTimeoutPolicies = $policies
}

ConvertTo-Json -Depth 10 $result
`

func (a *mqlMicrosoftPolicies) authorizationPolicy() (interface{}, error) {
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

func (a *mqlMicrosoftPolicies) identitySecurityDefaultsEnforcementPolicy() (interface{}, error) {
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
func (a *mqlMicrosoftPolicies) permissionGrantPolicies() ([]interface{}, error) {
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

func (a *mqlMicrosoftPolicies) consentPolicySettings() (interface{}, error) {
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

	actualSettingsMap := make(map[string]map[string]interface{})
	for _, setting := range groupSettings.GetValue() {
		displayName := setting.GetDisplayName()
		if displayName != nil {
			if _, exists := actualSettingsMap[*displayName]; !exists {
				actualSettingsMap[*displayName] = make(map[string]interface{})
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

func newAuthenticationMethodConfigurations(runtime *plugin.Runtime, configs []models.AuthenticationMethodConfigurationable) ([]interface{}, error) {
	var configResources []interface{}
	for _, config := range configs {
		excludeTargets := []interface{}{}
		for _, target := range config.GetExcludeTargets() {
			targetDict := map[string]interface{}{}
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

	var reviewers []interface{}
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

func (a *mqlMicrosoftPolicies) activityBasedTimeoutPolicies() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)

	// Get Microsoft Graph token for PowerShell authentication
	ctx := context.Background()
	token := conn.Token()
	graphToken, err := token.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{connection.DefaultMSGraphScope},
	})
	if err != nil {
		log.Error().Err(err).Msg("activityBasedTimeoutPolicies: Failed to get Graph token")
		return nil, err
	}

	// Format the PowerShell script with the access token
	fmtScript := fmt.Sprintf(activityBasedTimeoutPoliciesScript, graphToken.Token)

	// Execute the PowerShell script
	res, err := conn.CheckAndRunPowershellScript(fmtScript)
	if err != nil {
		log.Error().Err(err).Msg("activityBasedTimeoutPolicies: Failed to execute PowerShell script")
		return nil, err
	}

	// Process the results
	if res.ExitStatus == 0 {
		data, err := io.ReadAll(res.Stdout)
		if err != nil {
			log.Error().Err(err).Msg("activityBasedTimeoutPolicies: Failed to read PowerShell output")
			return nil, err
		}

		log.Debug().Msgf("activityBasedTimeoutPolicies: PowerShell script executed successfully, output length: %d", len(data))

		// Extract JSON from the output (in case there are other messages)
		outputStr := string(data)

		// Look for our specific JSON structure containing "ActivityBasedTimeoutPolicies"
		jsonStart := strings.Index(outputStr, `"ActivityBasedTimeoutPolicies"`)
		if jsonStart == -1 {
			log.Error().Str("output", outputStr).Msg("activityBasedTimeoutPolicies: No ActivityBasedTimeoutPolicies JSON found in PowerShell output")
			return nil, fmt.Errorf("no ActivityBasedTimeoutPolicies JSON found in PowerShell output")
		}

		// Find the opening brace before "ActivityBasedTimeoutPolicies"
		openBrace := strings.LastIndex(outputStr[:jsonStart], "{")
		if openBrace == -1 {
			log.Error().Str("output", outputStr).Msg("activityBasedTimeoutPolicies: No opening brace found before ActivityBasedTimeoutPolicies")
			return nil, fmt.Errorf("no opening brace found before ActivityBasedTimeoutPolicies")
		}

		// Find the closing brace after "ActivityBasedTimeoutPolicies"
		closeBrace := strings.Index(outputStr[jsonStart:], "}")
		if closeBrace == -1 {
			log.Error().Str("output", outputStr).Msg("activityBasedTimeoutPolicies: No closing brace found after ActivityBasedTimeoutPolicies")
			return nil, fmt.Errorf("no closing brace found after ActivityBasedTimeoutPolicies")
		}

		// Extract the JSON object
		jsonData := outputStr[openBrace : jsonStart+closeBrace+1]
		log.Debug().Str("json", jsonData).Msg("activityBasedTimeoutPolicies: Extracted JSON from PowerShell output")

		// Parse the JSON response
		var result struct {
			ActivityBasedTimeoutPolicies []map[string]interface{} `json:"ActivityBasedTimeoutPolicies"`
		}

		err = json.Unmarshal([]byte(jsonData), &result)
		if err != nil {
			log.Error().Err(err).Str("json", jsonData).Msg("activityBasedTimeoutPolicies: Failed to parse JSON response")
			return nil, fmt.Errorf("failed to parse PowerShell JSON response: %w", err)
		}

		log.Debug().Msgf("activityBasedTimeoutPolicies: Successfully parsed %d policies from PowerShell output", len(result.ActivityBasedTimeoutPolicies))

		// Convert to []interface{} for MQL compatibility
		policies := make([]interface{}, len(result.ActivityBasedTimeoutPolicies))
		for i, policy := range result.ActivityBasedTimeoutPolicies {
			policies[i] = policy
			log.Debug().Msgf("activityBasedTimeoutPolicies: Policy %d - ID: %v, DisplayName: %v, IsOrganizationDefault: %v",
				i, policy["Id"], policy["DisplayName"], policy["IsOrganizationDefault"])
		}

		return policies, nil
	} else {
		// Handle PowerShell execution errors
		data, err := io.ReadAll(res.Stderr)
		if err != nil {
			log.Error().Err(err).Msg("activityBasedTimeoutPolicies: Failed to read PowerShell error output")
			return nil, fmt.Errorf("PowerShell script failed with exit code %d", res.ExitStatus)
		}

		errorOutput := string(data)
		log.Error().Str("stderr", errorOutput).Int("exitCode", res.ExitStatus).Msg("activityBasedTimeoutPolicies: PowerShell script failed")
		return nil, fmt.Errorf("PowerShell script failed (exit code %d): %s", res.ExitStatus, errorOutput)
	}
}
