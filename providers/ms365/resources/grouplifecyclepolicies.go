// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

# Try to fetch LCM settings from Azure AD internal API
# This requires O365Essentials module or proper authentication headers
$Uri = 'https://main.iam.ad.ext.azure.com/api/Directories/LcmSettings'
$headers = @{
    "Authorization" = "Bearer %s"
    "Content-Type" = "application/json"
}

try {
    $response = Invoke-RestMethod -Uri $Uri -Method Get -Headers $headers -UseBasicParsing
    ConvertTo-Json -Depth 10 -InputObject $response
} catch {
    # If we get 401, the API might not be accessible with these credentials
    # Return an empty object to indicate no groups are monitored
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

	groupIds, err := fetchGroupIdsToMonitorExpirations(a.MqlRuntime)
	if err != nil {
		return nil, err
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

	// Get access token for Microsoft Graph API
	token := conn.Token()
	ctx := context.Background()

	accessToken, err := token.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://graph.microsoft.com/.default"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// Execute PowerShell script
	fmtScript := fmt.Sprintf(lcmSettingsScript, accessToken.Token)
	res, err := conn.CheckAndRunPowershellScript(fmtScript)
	if err != nil {
		return nil, fmt.Errorf("failed to run PowerShell script: %w", err)
	}

	if res.ExitStatus != 0 {
		data, _ := io.ReadAll(res.Stderr)
		return nil, fmt.Errorf("PowerShell script failed (exit code %d): %s", res.ExitStatus, string(data))
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
