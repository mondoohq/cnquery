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

# Fetch LCM settings from Azure AD internal API
# This mimics the O365Essentials Get-O365AzureGroupExpiration function
$Uri = 'https://main.iam.ad.ext.azure.com/api/Directories/LcmSettings'

# Try using O365Essentials module if available, otherwise fall back to direct call
try {
    # Check if O365Essentials module is available
    if (Get-Module -ListAvailable -Name O365Essentials) {
        Import-Module O365Essentials -Force
        
        # Build headers as dictionary (like the original function expects)
        $Headers = @{
            "Authorization" = "Bearer %s"
        }
        
        $Output = Invoke-O365Admin -Uri $Uri -Headers $Headers -Method Get
        
        if ($Output) {
            [PSCustomObject]@{
                groupIdsToMonitorExpirations = if ($Output.groupIdsToMonitorExpirations) { $Output.groupIdsToMonitorExpirations } else { @() }
                expiresAfterInDays = $Output.expiresAfterInDays
                groupLifetimeCustomValueInDays = $Output.groupLifetimeCustomValueInDays
                managedGroupTypes = $Output.managedGroupTypes
                adminNotificationEmails = $Output.adminNotificationEmails
                policyIdentifier = $Output.policyIdentifier
            } | ConvertTo-Json -Depth 10
            exit 0
        }
    }
} catch {
    Write-Host "O365Essentials not available or failed: $($_.Exception.Message)" -ForegroundColor Yellow
}

# Fallback to direct REST call if O365Essentials not available
try {
    $headers = @{
        "Authorization"          = "Bearer %s"
        "Content-Type"           = "application/json"
        "x-ms-client-request-id" = (New-Guid).Guid
        "x-ms-session-id"        = "12345678910111213141516"
        "Sec-Fetch-Dest"         = "empty"
        "Sec-Fetch-Mode"         = "cors"
        "Accept"                 = "*/*"
        "x-requested-with"       = "XMLHttpRequest"
        "user-agent"             = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
    }
    
    Write-Host "Calling: $Uri" -ForegroundColor Cyan
    Write-Host "Headers: $($headers.Keys -join ', ')" -ForegroundColor Cyan
    
    try {
        $response = Invoke-RestMethod -Uri $Uri -Method Get -Headers $headers -UseBasicParsing -ErrorAction Stop
        Write-Host "Success: Got response" -ForegroundColor Green
        ConvertTo-Json -Depth 10 -InputObject $response
    } catch {
        $statusCode = $_.Exception.Response.StatusCode.value__
        $statusDescription = $_.Exception.Response.StatusDescription
        Write-Host "HTTP Error: Status $statusCode - $statusDescription" -ForegroundColor Red
        Write-Host "Error details: $($_.Exception.Message)" -ForegroundColor Red
        
        # Try to read error response
        $reader = [System.IO.StreamReader]::new($_.Exception.Response.GetResponseStream())
        $errorBody = $reader.ReadToEnd()
        Write-Host "Response body: $errorBody" -ForegroundColor Red
        
        # Throw to be caught by outer catch
        throw $_
    }
} catch {
    Write-Host "All methods failed, returning empty object" -ForegroundColor Yellow
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

	// Get access token for Azure Portal API (required for main.iam.ad.ext.azure.com)
	token := conn.Token()
	ctx := context.Background()

	// Try with Azure Management scope (what the internal API needs)
	accessToken, err := token.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// Execute PowerShell script
	fmtScript := fmt.Sprintf(lcmSettingsScript, accessToken.Token, accessToken.Token)
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
