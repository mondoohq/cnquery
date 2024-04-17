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
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/logger"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/ms365/connection"
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
	sharepointLock sync.Mutex
	fetched        bool
	fetchErr       error
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

func (r *mqlMs365Sharepointonline) getSharepointOnlineReport() error {
	conn := r.MqlRuntime.Connection.(*connection.Ms365Connection)

	r.sharepointLock.Lock()
	defer r.sharepointLock.Unlock()

	// only fetch once
	if r.fetched {
		return r.fetchErr
	}

	errHandler := func(err error) error {
		r.fetchErr = err
		return err
	}

	// for some reasons, tokens issued by a client secret do not work. only certificates do
	// TODO: ^ we should try and investigate why, its unclear to me why it happens.
	if !conn.IsCertProvided() {
		return errHandler(fmt.Errorf("only certificate authentication is supported for fetching sharepoint onine report"))
	}

	tenant, err := r.getTenant()
	if tenant == "" || err != nil {
		return errHandler(fmt.Errorf("tenant cannot be empty, cannot fetch sharepoint online report"))
	}

	ctx := context.Background()
	token := conn.Token()
	tokenScope := fmt.Sprintf("https://%s-admin.sharepoint.com/.default", tenant)
	sharepointUrl := fmt.Sprintf("https://%s.sharepoint.com", tenant)
	spToken, err := token.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{tokenScope},
	})
	if err != nil {
		return errHandler(err)
	}

	fmtScript := fmt.Sprintf(sharepointReport, spToken.Token, sharepointUrl)
	res, err := conn.CheckAndRunPowershellScript(fmtScript)
	if err != nil {
		return errHandler(err)
	}
	report := &SharepointOnlineReport{}
	if res.ExitStatus == 0 {
		data, err := io.ReadAll(res.Stdout)
		if err != nil {
			return errHandler(err)
		}

		logger.DebugDumpJSON("sharepoint-online-report", string(data))

		err = json.Unmarshal(data, report)
		if err != nil {
			return errHandler(err)
		}
	} else {
		data, err := io.ReadAll(res.Stderr)
		if err != nil {
			return errHandler(err)
		}

		str := string(data)
		if strings.Contains(strings.ToLower(str), "unauthorized") {
			return errHandler(errors.New("access denied, please ensure the credentials have the right permissions in Azure AD"))
		}

		logger.DebugDumpJSON("sharepoint-online-report", string(data))
		return fmt.Errorf("failed to generate sharepoint online report (exit code %d): %s", res.ExitStatus, string(data))
	}

	spoTenant, spoTenantErr := convert.JsonToDict(report.SpoTenant)
	r.SpoTenant = plugin.TValue[interface{}]{Data: spoTenant, State: plugin.StateIsSet, Error: spoTenantErr}

	spoTenantSyncClientRestriction, spoTenantSyncClientRestrictionErr := convert.JsonToDict(report.SpoTenantSyncClientRestriction)
	r.SpoTenantSyncClientRestriction = plugin.TValue[interface{}]{Data: spoTenantSyncClientRestriction, State: plugin.StateIsSet, Error: spoTenantSyncClientRestrictionErr}

	sites := []interface{}{}
	var sitesErr error
	for _, s := range report.SpoSite {
		mqlSpoSite, err := CreateResource(r.MqlRuntime, "ms365.sharepointonline.site",
			map[string]*llx.RawData{
				"denyAddAndCustomizePages": llx.BoolData(s.DenyAddAndCustomizePages == "Enabled"),
				"url":                      llx.StringData(s.Url),
			})
		if err != nil {
			sitesErr = err
			break
		}
		sites = append(sites, mqlSpoSite)
	}
	r.SpoSites = plugin.TValue[[]interface{}]{Data: sites, State: plugin.StateIsSet, Error: sitesErr}
	return nil
}

func (r *mqlMs365Sharepointonline) spoTenant() (interface{}, error) {
	return nil, r.getSharepointOnlineReport()
}

func (r *mqlMs365Sharepointonline) spoTenantSyncClientRestriction() (interface{}, error) {
	return nil, r.getSharepointOnlineReport()
}

func (r *mqlMs365Sharepointonline) spoSites() ([]interface{}, error) {
	return nil, r.getSharepointOnlineReport()
}
