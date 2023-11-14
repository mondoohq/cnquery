// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

var sharepointReport = `
$token = '%s'
$url = "%s"
Install-Module PnP.PowerShell -Force
Connect-PnPOnline -AccessToken $token -Url $url

$SPOTenant = (Get-PnPTenant)
$SPOTenantSyncClientRestriction = (Get-PnPTenantSyncClientRestriction)

$sharepoint = New-Object PSObject
Add-Member -InputObject $sharepoint -MemberType NoteProperty -Name SPOTenant -Value $SPOTenant
Add-Member -InputObject $sharepoint -MemberType NoteProperty -Name SPOTenantSyncClientRestriction -Value $SPOTenantSyncClientRestriction

Disconnect-PnPOnline 

ConvertTo-Json -Depth 4 $sharepoint
`

func (c *Ms365Connection) GetSharepointOnlineReport(ctx context.Context, tenantDomain string) (*SharepointOnlineReport, error) {
	if tenantDomain == "" {
		return nil, fmt.Errorf("tenant domain cannot be empty, cannot fetch sharepoint online report")
	}
	c.sharepointLock.Lock()
	defer c.sharepointLock.Unlock()
	if c.sharepointReport != nil {
		return c.sharepointReport, nil
	}

	token := c.Token()
	domainParts := strings.Split(tenantDomain, ".")
	if len(domainParts) < 2 {
		return nil, fmt.Errorf("invalid tenant domain url: %s", tenantDomain)
	}
	// we only care about the tenant name, so we take the first part in the split domain
	tenantName := domainParts[0]
	tokenScope := fmt.Sprintf("https://%s-admin.sharepoint.com/.default", tenantName)
	sharepointUrl := fmt.Sprintf("https://%s.sharepoint.com", tenantName)
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

		err = json.Unmarshal(data, report)
		if err != nil {
			return nil, err
		}
	} else {
		data, err := io.ReadAll(res.Stderr)
		if err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("failed to generate sharepoint online report (exit code %d): %s", res.ExitStatus, string(data))
	}
	return report, nil
}

type SharepointOnlineReport struct {
	SpoTenant                      interface{} `json:"SPOTenant"`
	SpoTenantSyncClientRestriction interface{} `json:"SPOTenantSyncClientRestriction"`
}
