// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/logger"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/ms365/connection"
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

type SharepointOnlineReport struct {
	SpoTenant                      interface{} `json:"SPOTenant"`
	SpoTenantSyncClientRestriction interface{} `json:"SPOTenantSyncClientRestriction"`
	SpoSite                        []*SpoSite  `json:"SPOSite"`
}

type SpoSite struct {
	DenyAddAndCustomizePages string `json:"DenyAddAndCustomizePages"`
	Url                      string `json:"Url"`
}

func (m *mqlMs365SharepointonlineSite) id() (string, error) {
	return m.Url.Data, nil
}

type mqlMs365SharepointonlineInternal struct {
	sharepointReport *SharepointOnlineReport
	sharepointLock   sync.Mutex
}

func (r *mqlMs365Sharepointonline) getTenant() (string, error) {
	conn := r.MqlRuntime.Connection.(*connection.Ms365Connection)

	microsoft, err := CreateResource(r.MqlRuntime, "microsoft", map[string]*llx.RawData{})
	if err != nil {
		return "", err
	}
	mqlMicrosoft := microsoft.(*mqlMicrosoft)

	// we prefer the explicitly passed in sharepoint url, if there is one
	spUrl := conn.SharepointUrl()
	if spUrl == "" {
		tenantDomainName := mqlMicrosoft.GetTenantDomainName()
		if tenantDomainName.Error != nil {
			// note: we dont want to err here. maybe the app registration has no perms to get the organization
			// in that case we try and get the report by using the explicitly passed in sharepoint url
			log.Debug().Err(tenantDomainName.Error).Msg("unable to get tenant domain name")
		} else {
			spUrl = tenantDomainName.Data
		}
	}

	if spUrl == "" {
		return "", errors.New("no sharepoint url provided, unable to fetch sharepoint online report")
	}

	domainParts := strings.Split(spUrl, ".")
	if len(domainParts) < 2 {
		return "", fmt.Errorf("invalid sharepoint url: %s", spUrl)
	}

	// we only care about the tenant name, so we take the first part in the split domain
	tenant := domainParts[0]
	return tenant, nil
}

func (r *mqlMs365Sharepointonline) GetSharepointOnlineReport(ctx context.Context) (*SharepointOnlineReport, error) {
	conn := r.MqlRuntime.Connection.(*connection.Ms365Connection)

	// for some reasons, tokens issued by a client secret do not work. only certificates do
	// TODO: ^ we should try and investigate why, its unclear to me why it happens.
	if !conn.IsCertProvided() {
		return nil, fmt.Errorf("only certificate authentication is supported for fetching sharepoint onine report")
	}

	r.sharepointLock.Lock()
	defer r.sharepointLock.Unlock()
	if r.sharepointReport != nil {
		return r.sharepointReport, nil
	}

	tenant, err := r.getTenant()
	if tenant == "" || err != nil {
		return nil, fmt.Errorf("tenant cannot be empty, cannot fetch sharepoint online report")
	}

	token := conn.Token()
	tokenScope := fmt.Sprintf("https://%s-admin.sharepoint.com/.default", tenant)
	sharepointUrl := fmt.Sprintf("https://%s.sharepoint.com", tenant)
	spToken, err := token.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{tokenScope},
	})
	if err != nil {
		return nil, err
	}

	fmtScript := fmt.Sprintf(sharepointReport, spToken.Token, sharepointUrl)
	res, err := conn.CheckAndRunPowershellScript(fmtScript)
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

	r.sharepointReport = report
	return report, nil
}

func (r *mqlMs365Sharepointonline) spoTenant() (interface{}, error) {
	ctx := context.Background()
	report, err := r.GetSharepointOnlineReport(ctx)
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(report.SpoTenant)
}

func (r *mqlMs365Sharepointonline) spoTenantSyncClientRestriction() (interface{}, error) {
	ctx := context.Background()
	report, err := r.GetSharepointOnlineReport(ctx)
	if err != nil {
		return nil, err
	}
	return convert.JsonToDict(report.SpoTenantSyncClientRestriction)
}

func (r *mqlMs365Sharepointonline) spoSites() ([]interface{}, error) {
	ctx := context.Background()
	report, err := r.GetSharepointOnlineReport(ctx)
	if err != nil {
		return nil, err
	}

	sites := []interface{}{}
	for _, s := range report.SpoSite {
		mqlSpoSite, err := CreateResource(r.MqlRuntime, "ms365.sharepointonline.site",
			map[string]*llx.RawData{
				"denyAddAndCustomizePages": llx.BoolData(s.DenyAddAndCustomizePages == "Enabled"),
				"url":                      llx.StringData(s.Url),
			})
		if err != nil {
			return nil, err
		}
		sites = append(sites, mqlSpoSite)
	}
	return sites, nil
}
