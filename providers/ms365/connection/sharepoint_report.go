// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"go.mondoo.com/cnquery/v10/logger"
)

var sharepointReport = `
$ErrorActionPreference = "Stop"
$token = '%s'
$url = "%s"
Install-Module PnP.PowerShell -Force -Scope CurrentUser
Import-Module PnP.PowerShell
Connect-PnPOnline -AccessToken $token -Url $url

$SPOTenant = (Get-PnPTenant)
$SPOTenantSyncClientRestriction = (Get-PnPTenantSyncClientRestriction)
$SPOSite = (Get-PnPTenantSite)

$sharepoint = New-Object PSObject
Add-Member -InputObject $sharepoint -MemberType NoteProperty -Name SPOTenant -Value $SPOTenant
Add-Member -InputObject $sharepoint -MemberType NoteProperty -Name SPOTenantSyncClientRestriction -Value $SPOTenantSyncClientRestriction
Add-Member -InputObject $sharepoint -MemberType NoteProperty -Name SPOSite -Value $SPOSite

Disconnect-PnPOnline 

ConvertTo-Json -Depth 4 $sharepoint -EnumsAsStrings
`

func (c *Ms365Connection) GetSharepointOnlineReport(ctx context.Context, tenant string) (*SharepointOnlineReport, error) {
	if tenant == "" {
		return nil, fmt.Errorf("tenant cannot be empty, cannot fetch sharepoint online report")
	}
	// for some reasons, tokens issued by a client secret do not work. only certificates do
	// TODO: ^ we should try and investigate why, its unclear to me why it happens.
	if !c.IsCertProvided() {
		return nil, fmt.Errorf("only certificate authentication is supported for fetching sharepoint onine report")
	}
	c.sharepointLock.Lock()
	defer c.sharepointLock.Unlock()
	if c.sharepointReport != nil {
		return c.sharepointReport, nil
	}

	token := c.Token()
	tokenScope := fmt.Sprintf("https://%s-admin.sharepoint.com/.default", tenant)
	sharepointUrl := fmt.Sprintf("https://%s.sharepoint.com", tenant)
	spToken, err := token.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{tokenScope},
	})
	if err != nil {
		return nil, err
	}
	report, err := c.getSharepointReport(spToken.Token, sharepointUrl)
	if err != nil {
		return nil, err
	}
	c.sharepointReport = report
	return report, nil
}

func (c *Ms365Connection) getSharepointReport(spToken, url string) (*SharepointOnlineReport, error) {
	fmtScript := fmt.Sprintf(sharepointReport, spToken, url)
	res, err := c.checkAndRunPowershellScript(fmtScript)
	if err != nil {
		return nil, err
	}
	report := &SharepointOnlineReport{}
	if res.ExitStatus == 0 {
		data, err := io.ReadAll(res.Stdout)
		if err != nil {
			return nil, err
		}

		logger.DebugDumpJSON("sharepoint-online-report", string(data))

		err = json.Unmarshal(data, report)
		if err != nil {
			return nil, err
		}
	} else {
		data, err := io.ReadAll(res.Stderr)
		if err != nil {
			return nil, err
		}

		logger.DebugDumpJSON("sharepoint-online-report", string(data))
		return nil, fmt.Errorf("failed to generate sharepoint online report (exit code %d): %s", res.ExitStatus, string(data))
	}
	return report, nil
}

type SharepointOnlineReport struct {
	SpoTenant                      interface{} `json:"SPOTenant"`
	SpoTenantSyncClientRestriction interface{} `json:"SPOTenantSyncClientRestriction"`
	SpoSite                        []*SpoSite  `json:"SPOSite"`
}

type SpoSite struct {
	DenyAddAndCustomizePages string `json:"DenyAddAndCustomizePages"`
	Url                      string `json:"Url"`
}
