// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/logger"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers/ms365/connection"
	"go.mondoo.com/cnquery/v12/types"
)

var lcmSettingsLock sync.Mutex

const lcmSettingsScript = `
$ErrorActionPreference = "Stop"

$Uri = 'https://main.iam.ad.ext.azure.com/api/Directories/LcmSettings'
$headers = @{
    "Authorization"          = "Bearer %s"
    "X-Ms-Client-Request-Id" = (New-Guid).Guid
}

try {
    $response = Invoke-RestMethod -Uri $Uri -Method Get -Headers $headers -UseBasicParsing
    ConvertTo-Json -Depth 10 -InputObject $response
} catch {
    # Return empty object if API is not accessible
    [PSCustomObject]@{
        groupIdsToMonitorExpirations = @()
        expiresAfterInDays = 0
        groupLifetimeCustomValueInDays = 180
        managedGroupTypes = 0
        adminNotificationEmails = ""
        policyIdentifier = ""
    } | ConvertTo-Json -Depth 10
}
`

func (a *mqlMicrosoft) groupLifecyclePolicies() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	resp, err := graphClient.GroupLifecyclePolicies().Get(context.Background(), nil)
	if err != nil {
		return nil, transformError(err)
	}

	// Fetch groupIdsToMonitorExpirations from internal Azure AD LCM API
	// Note: This API requires delegated authentication (user sign-in) or special Azure management permissions.
	// With certificate-based app-only authentication, this will return an empty array.
	groupIds, err := fetchGroupIdsToMonitorExpirations(a.MqlRuntime)
	if err != nil {
		// Log the error but continue with empty array - this is expected with app-only auth
		groupIds = []string{}
	}

	res := []any{}
	for _, p := range resp.GetValue() {
		policy, err := newMqlMicrosoftGroupLifecyclePolicy(a.MqlRuntime, p, groupIds)
		if err != nil {
			return nil, err
		}
		res = append(res, policy)
	}

	return res, nil
}

func newMqlMicrosoftGroupLifecyclePolicy(runtime *plugin.Runtime, p models.GroupLifecyclePolicyable, groupIds []string) (*mqlMicrosoftGroupLifecyclePolicy, error) {
	if p.GetId() == nil {
		return nil, errors.New("group lifecycle policy response is missing an ID")
	}

	data := map[string]*llx.RawData{
		"__id":                         llx.StringDataPtr(p.GetId()),
		"id":                           llx.StringDataPtr(p.GetId()),
		"groupLifetimeInDays":          llx.IntDataPtr(p.GetGroupLifetimeInDays()),
		"managedGroupTypes":            llx.StringDataPtr(p.GetManagedGroupTypes()),
		"alternateNotificationEmails":  llx.StringDataPtr(p.GetAlternateNotificationEmails()),
		"groupIdsToMonitorExpirations": llx.ArrayData(llx.TArr2Raw(convert.SliceAnyToInterface(groupIds)), types.String),
	}

	resource, err := CreateResource(runtime, "microsoft.groupLifecyclePolicy", data)
	if err != nil {
		return nil, err
	}

	return resource.(*mqlMicrosoftGroupLifecyclePolicy), nil
}

// Azure AD LCM settings response structure
type lcmSettingsResponse struct {
	GroupIdsToMonitorExpirations   []string `json:"groupIdsToMonitorExpirations"`
	ExpiresAfterInDays             int      `json:"expiresAfterInDays"`
	GroupLifetimeCustomValueInDays int      `json:"groupLifetimeCustomValueInDays"`
	ManagedGroupTypes              int      `json:"managedGroupTypes"`
	AdminNotificationEmails        string   `json:"adminNotificationEmails"`
	PolicyIdentifier               string   `json:"policyIdentifier"`
}

// fetchGroupIdsToMonitorExpirations fetches the group IDs from Azure AD internal API using PowerShell
func fetchGroupIdsToMonitorExpirations(runtime *plugin.Runtime) ([]string, error) {
	lcmSettingsLock.Lock()
	defer lcmSettingsLock.Unlock()

	conn := runtime.Connection.(*connection.Ms365Connection)

	var tokenString string

	// Check if LCM_BEARER_TOKEN environment variable is set (for testing with delegated token)
	if envToken := os.Getenv("LCM_BEARER_TOKEN"); envToken != "" {
		tokenString = envToken
	} else {
		// Get access token for Azure Portal API (required for main.iam.ad.ext.azure.com)
		// The Azure Portal API audience is 74658136-14ec-4630-ad9b-26e160ff0fc6
		token := conn.Token()
		ctx := context.Background()

		// Try with Azure Portal API audience
		accessToken, err := token.GetToken(ctx, policy.TokenRequestOptions{
			Scopes: []string{"74658136-14ec-4630-ad9b-26e160ff0fc6/.default"},
		})
		if err != nil {
			// Fallback to management scope if portal scope doesn't work
			accessToken, err = token.GetToken(ctx, policy.TokenRequestOptions{
				Scopes: []string{"https://management.azure.com/.default"},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to get access token: %w", err)
			}
		}
		tokenString = accessToken.Token
	}

	// Execute PowerShell script
	fmtScript := fmt.Sprintf(lcmSettingsScript, tokenString)
	res, err := conn.CheckAndRunPowershellScript(fmtScript)
	if err != nil {
		return nil, fmt.Errorf("failed to run PowerShell script: %w", err)
	}

	if res.ExitStatus != 0 {
		data, _ := io.ReadAll(res.Stderr)
		stderrStr := string(data)
		logger.DebugDumpJSON("lcm-settings-stderr", []byte(stderrStr))
		return nil, fmt.Errorf("PowerShell script failed (exit code %d): %s", res.ExitStatus, stderrStr)
	}

	data, err := io.ReadAll(res.Stdout)
	if err != nil {
		return nil, fmt.Errorf("failed to read PowerShell output: %w", err)
	}

	logger.DebugDumpJSON("lcm-settings-response", data)

	var lcmSettings lcmSettingsResponse
	if err := json.Unmarshal(data, &lcmSettings); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return lcmSettings.GroupIdsToMonitorExpirations, nil
}
