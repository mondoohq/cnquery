// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/pkg/errors"
)

var exchangeReport = `
$appId = '%s'
$organization = '%s'
$tenantId = '%s'
$outlookToken= '%s'

Install-Module -Name ExchangeOnlineManagement -Scope CurrentUser -Force
Import-Module ExchangeOnlineManagement
Connect-ExchangeOnline -AccessToken $outlookToken -AppID $appId -Organization $organization -ShowBanner:$false -ShowProgress:$false 

$MalwareFilterPolicy = (Get-MalwareFilterPolicy)
$HostedOutboundSpamFilterPolicy = (Get-HostedOutboundSpamFilterPolicy)
$TransportRule = (Get-TransportRule)
$RemoteDomain = (Get-RemoteDomain Default)
$SafeLinksPolicy = (Get-SafeLinksPolicy)
$SafeAttachmentPolicy = (Get-SafeAttachmentPolicy)
$OrganizationConfig = (Get-OrganizationConfig)
$AuthenticationPolicy = (Get-AuthenticationPolicy)
$AntiPhishPolicy = (Get-AntiPhishPolicy)
$DkimSigningConfig = (Get-DkimSigningConfig)
$OwaMailboxPolicy = (Get-OwaMailboxPolicy)
$AdminAuditLogConfig = (Get-AdminAuditLogConfig)
$PhishFilterPolicy = (Get-PhishFilterPolicy)
$Mailbox = (Get-Mailbox -ResultSize Unlimited)
$AtpPolicyForO365 = (Get-AtpPolicyForO365)
$SharingPolicy = (Get-SharingPolicy)
$RoleAssignmentPolicy = (Get-RoleAssignmentPolicy)

$exchangeOnline = New-Object PSObject
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name MalwareFilterPolicy -Value @($MalwareFilterPolicy)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name HostedOutboundSpamFilterPolicy -Value @($HostedOutboundSpamFilterPolicy)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name TransportRule -Value @($TransportRule)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name RemoteDomain -Value  @($RemoteDomain)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name SafeLinksPolicy -Value @($SafeLinksPolicy)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name SafeAttachmentPolicy -Value @($SafeAttachmentPolicy)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name OrganizationConfig -Value $OrganizationConfig
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name AuthenticationPolicy -Value @($AuthenticationPolicy)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name AntiPhishPolicy -Value @($AntiPhishPolicy)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name DkimSigningConfig -Value @($DkimSigningConfig)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name OwaMailboxPolicy -Value @($OwaMailboxPolicy)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name AdminAuditLogConfig -Value $AdminAuditLogConfig
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name PhishFilterPolicy -Value @($PhishFilterPolicy)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name Mailbox -Value @($Mailbox)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name AtpPolicyForO365 -Value @($AtpPolicyForO365)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name SharingPolicy -Value @($SharingPolicy)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name RoleAssignmentPolicy -Value @($RoleAssignmentPolicy)

Disconnect-ExchangeOnline -Confirm:$false

ConvertTo-Json -Depth 4 $exchangeOnline
`

func (c *Ms365Connection) GetExchangeReport(ctx context.Context) (*ExchangeOnlineReport, error) {
	// if no organization is provided, we cannot fetch the exchange online report
	if c.organization == "" {
		return nil, errors.New("no organization provided, unable to fetch exchange online report")
	}
	c.exchangeReportLock.Lock()
	defer c.exchangeReportLock.Unlock()
	if c.exchangeReport != nil {
		return c.exchangeReport, nil
	}

	token := c.Token()
	outlookToken, err := token.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{OutlookScope},
	})
	if err != nil {
		return nil, err
	}
	report, err := c.getReport(outlookToken.Token)
	if err != nil {
		return nil, err
	}
	c.exchangeReport = report
	return report, nil
}

func (c *Ms365Connection) getReport(outlookToken string) (*ExchangeOnlineReport, error) {
	fmtScript := fmt.Sprintf(exchangeReport, c.organization, c.clientId, c.tenantId, outlookToken)
	res, err := c.checkAndRunPowershellScript(fmtScript)
	if err != nil {
		return nil, err
	}
	report := &ExchangeOnlineReport{}
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

		return nil, fmt.Errorf("failed to generate exchange online report (exit code %d): %s", res.ExitStatus, string(data))
	}
	return report, nil
}

type ExchangeOnlineReport struct {
	MalwareFilterPolicy            []interface{} `json:"MalwareFilterPolicy"`
	HostedOutboundSpamFilterPolicy []interface{} `json:"HostedOutboundSpamFilterPolicy"`
	TransportRule                  []interface{} `json:"TransportRule"`
	RemoteDomain                   []interface{} `json:"RemoteDomain"`
	SafeLinksPolicy                []interface{} `json:"SafeLinksPolicy"`
	SafeAttachmentPolicy           []interface{} `json:"SafeAttachmentPolicy"`
	OrganizationConfig             interface{}   `json:"OrganizationConfig"`
	AuthenticationPolicy           interface{}   `json:"AuthenticationPolicy"`
	AntiPhishPolicy                []interface{} `json:"AntiPhishPolicy"`
	DkimSigningConfig              interface{}   `json:"DkimSigningConfig"`
	OwaMailboxPolicy               interface{}   `json:"OwaMailboxPolicy"`
	AdminAuditLogConfig            interface{}   `json:"AdminAuditLogConfig"`
	PhishFilterPolicy              []interface{} `json:"PhishFilterPolicy"`
	Mailbox                        []interface{} `json:"Mailbox"`
	AtpPolicyForO365               []interface{} `json:"AtpPolicyForO365"`
	SharingPolicy                  []interface{} `json:"SharingPolicy"`
	RoleAssignmentPolicy           []interface{} `json:"RoleAssignmentPolicy"`
}
