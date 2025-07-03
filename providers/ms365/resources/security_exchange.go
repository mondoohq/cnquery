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
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/logger"
	"go.mondoo.com/cnquery/v11/providers/ms365/connection"
	"go.mondoo.com/cnquery/v11/types"
)

// Note: outlookScope is already defined in ms365_exchange.go

var hostedConnectionFilterPolicyScript = `
$appId = '%s'
$organization = '%s'
$tenantId = '%s'
$outlookToken = '%s'

Install-Module -Name ExchangeOnlineManagement -Scope CurrentUser -Force
Import-Module ExchangeOnlineManagement
Connect-ExchangeOnline -AccessToken $outlookToken -AppID $appId -Organization $organization -ShowBanner:$false -ShowProgress:$false

$HostedConnectionFilterPolicy = (Get-HostedConnectionFilterPolicy -Identity Default)

$result = New-Object PSObject
Add-Member -InputObject $result -MemberType NoteProperty -Name HostedConnectionFilterPolicy -Value $HostedConnectionFilterPolicy

Disconnect-ExchangeOnline -Confirm:$false

ConvertTo-Json -Depth 4 $result
`

type HostedConnectionFilterPolicyReport struct {
	HostedConnectionFilterPolicy *HostedConnectionFilterPolicyData `json:"HostedConnectionFilterPolicy"`
}

type HostedConnectionFilterPolicyData struct {
	Identity           string   `json:"Identity"`
	AdminDisplayName   string   `json:"AdminDisplayName"`
	IPAllowList        []string `json:"IPAllowList"`
	IPBlockList        []string `json:"IPBlockList"`
	EnableSafeList     bool     `json:"EnableSafeList"`
}

type mqlMicrosoftSecurityExchangeInternal struct {
	reportLock sync.Mutex
	fetched    bool
	fetchErr   error
	report     *HostedConnectionFilterPolicyReport
}

type mqlMicrosoftSecurityExchangeAntispamInternal struct {
	// Inherits from parent
}

type mqlMicrosoftSecurityExchangeAntispamHostedConnectionFilterPolicyInternal struct {
	// Inherits from parent
}

func (r *mqlMicrosoftSecurity) exchange() (*mqlMicrosoftSecurityExchange, error) {
	resource, err := CreateResource(r.MqlRuntime, "microsoft.security.exchange", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlMicrosoftSecurityExchange), nil
}

func (r *mqlMicrosoftSecurityExchange) antispam() (*mqlMicrosoftSecurityExchangeAntispam, error) {
	resource, err := CreateResource(r.MqlRuntime, "microsoft.security.exchange.antispam", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlMicrosoftSecurityExchangeAntispam), nil
}

func (r *mqlMicrosoftSecurityExchangeAntispam) hostedConnectionFilterPolicy() (*mqlMicrosoftSecurityExchangeAntispamHostedConnectionFilterPolicy, error) {
	// Create a new exchange resource to get the report
	exchangeResource, err := CreateResource(r.MqlRuntime, "microsoft.security.exchange", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}

	report, err := exchangeResource.(*mqlMicrosoftSecurityExchange).getHostedConnectionFilterPolicyReport()
	if err != nil {
		return nil, err
	}

	if report.HostedConnectionFilterPolicy == nil {
		return nil, errors.New("no hosted connection filter policy found")
	}

	policy := report.HostedConnectionFilterPolicy

	resource, err := CreateResource(r.MqlRuntime, "microsoft.security.exchange.antispam.hostedConnectionFilterPolicy",
		map[string]*llx.RawData{
			"identity":           llx.StringData(policy.Identity),
			"adminDisplayName":   llx.StringData(policy.AdminDisplayName),
			"ipAllowList":      	llx.ArrayData(convert.SliceAnyToInterface(policy.IPAllowList), types.String),
			"ipBlockList":      	llx.ArrayData(convert.SliceAnyToInterface(policy.IPBlockList), types.String),
			"enableSafeList":     llx.BoolData(policy.EnableSafeList),
		})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlMicrosoftSecurityExchangeAntispamHostedConnectionFilterPolicy), nil
}

func (r *mqlMicrosoftSecurityExchange) getHostedConnectionFilterPolicyReport() (*HostedConnectionFilterPolicyReport, error) {
	r.reportLock.Lock()
	defer r.reportLock.Unlock()

	if r.fetched {
		return r.report, r.fetchErr
	}

	errHandler := func(err error) (*HostedConnectionFilterPolicyReport, error) {
		r.fetchErr = err
		r.fetched = true
		return nil, err
	}

	conn := r.MqlRuntime.Connection.(*connection.Ms365Connection)
	
	// Get organization info
	microsoft, err := CreateResource(r.MqlRuntime, "microsoft", map[string]*llx.RawData{})
	if err != nil {
		return errHandler(err)
	}
	
	tenantDomainName := microsoft.(*mqlMicrosoft).GetTenantDomainName()
	if tenantDomainName.Error != nil {
		return errHandler(tenantDomainName.Error)
	}
	
	organization := tenantDomainName.Data
	if organization == "" {
		return errHandler(errors.New("no organization provided, unable to fetch hosted connection filter policy"))
	}

	ctx := context.Background()
	token := conn.Token()
	outlookToken, err := token.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{outlookScope},
	})
	if err != nil {
		return errHandler(err)
	}

	fmtScript := fmt.Sprintf(hostedConnectionFilterPolicyScript, conn.ClientId(), organization, conn.TenantId(), outlookToken.Token)
	res, err := conn.CheckAndRunPowershellScript(fmtScript)
	if err != nil {
		return errHandler(err)
	}

	report := &HostedConnectionFilterPolicyReport{}
	if res.ExitStatus == 0 {
		data, err := io.ReadAll(res.Stdout)
		if err != nil {
			return errHandler(err)
		}
		logger.DebugDumpJSON("hosted-connection-filter-policy-report", data)

		err = json.Unmarshal(data, report)
		if err != nil {
			return errHandler(err)
		}
	} else {
		data, err := io.ReadAll(res.Stderr)
		if err != nil {
			return errHandler(err)
		}
		
		err = fmt.Errorf("failed to generate hosted connection filter policy report (exit code %d): %s", res.ExitStatus, string(data))
		return errHandler(err)
	}

	r.report = report
	r.fetched = true
	return report, nil
}
