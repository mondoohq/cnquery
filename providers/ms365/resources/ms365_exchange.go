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
	"go.mondoo.com/cnquery/v11/types"
)

const outlookScope = "https://outlook.office.com/.default"

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
$ExternalInOutlook = (Get-ExternalInOutlook)
$ExoMailbox = (Get-EXOMailbox -RecipientTypeDetails SharedMailbox)
$TeamsProtectionPolicy = (Get-TeamsProtectionPolicy)

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
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name ExternalInOutlook -Value @($ExternalInOutlook)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name ExoMailbox -Value @($ExoMailbox)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name TeamsProtectionPolicy -Value @($TeamsProtectionPolicy)

Disconnect-ExchangeOnline -Confirm:$false

ConvertTo-Json -Depth 4 $exchangeOnline
`

type ExchangeOnlineReport struct {
	MalwareFilterPolicy            []interface{}     `json:"MalwareFilterPolicy"`
	HostedOutboundSpamFilterPolicy []interface{}     `json:"HostedOutboundSpamFilterPolicy"`
	TransportRule                  []interface{}     `json:"TransportRule"`
	RemoteDomain                   []interface{}     `json:"RemoteDomain"`
	SafeLinksPolicy                []interface{}     `json:"SafeLinksPolicy"`
	SafeAttachmentPolicy           []interface{}     `json:"SafeAttachmentPolicy"`
	OrganizationConfig             interface{}       `json:"OrganizationConfig"`
	AuthenticationPolicy           interface{}       `json:"AuthenticationPolicy"`
	AntiPhishPolicy                []interface{}     `json:"AntiPhishPolicy"`
	DkimSigningConfig              interface{}       `json:"DkimSigningConfig"`
	OwaMailboxPolicy               interface{}       `json:"OwaMailboxPolicy"`
	AdminAuditLogConfig            interface{}       `json:"AdminAuditLogConfig"`
	PhishFilterPolicy              []interface{}     `json:"PhishFilterPolicy"`
	Mailbox                        []interface{}     `json:"Mailbox"`
	AtpPolicyForO365               []interface{}     `json:"AtpPolicyForO365"`
	SharingPolicy                  []interface{}     `json:"SharingPolicy"`
	RoleAssignmentPolicy           []interface{}     `json:"RoleAssignmentPolicy"`
	ExternalInOutlook              []*ExternalSender `json:"ExternalInOutlook"`
	// note: this only contains shared mailboxes
	ExoMailbox            []*ExoMailbox            `json:"ExoMailbox"`
	TeamsProtectionPolicy []*TeamsProtectionPolicy `json:"TeamsProtectionPolicy"`
}

type ExternalSender struct {
	Identity  string   `json:"Identity"`
	Enabled   bool     `json:"Enabled"`
	AllowList []string `json:"AllowList"`
}

type ExoMailbox struct {
	ExternalDirectoryObjectId string   `json:"ExternalDirectoryObjectId"`
	UserPrincipalName         string   `json:"UserPrincipalName"`
	Alias                     string   `json:"Alias"`
	DisplayName               string   `json:"DisplayName"`
	EmailAddresses            []string `json:"EmailAddresses"`
	PrimarySmtpAddress        string   `json:"PrimarySmtpAddress"`
	RecipientType             string   `json:"RecipientType"`
	RecipientTypeDetails      string   `json:"RecipientTypeDetails"`
	Identity                  string   `json:"Identity"`
	Id                        string   `json:"Id"`
	ExchangeVersion           string   `json:"ExchangeVersion"`
	Name                      string   `json:"Name"`
	DistinguishedName         string   `json:"DistinguishedName"`
	OrganizationId            string   `json:"OrganizationId"`
	Guid                      string   `json:"Guid"`
}

type TeamsProtectionPolicy struct {
	ZapEnabled bool `json:"ZapEnabled"`
	IsValid    bool `json:"IsValid"`
}

type mqlMs365ExchangeonlineInternal struct {
	exchangeReportLock sync.Mutex
	fetched            bool
	fetchErr           error
	org                string
}

func (r *mqlMs365Exchangeonline) getOrg() (string, error) {
	conn := r.MqlRuntime.Connection.(*connection.Ms365Connection)
	microsoft, err := CreateResource(r.MqlRuntime, "microsoft", map[string]*llx.RawData{})
	if err != nil {
		return "", err
	}
	mqlMicrosoft := microsoft.(*mqlMicrosoft)
	// we prefer the explicitly passed in org, if there is one
	org := conn.Organization()
	if org == "" {
		tenantDomainName := mqlMicrosoft.GetTenantDomainName()
		if tenantDomainName.Error != nil {
			// note: we dont want to err here. maybe the app registration has no perms to get the organization
			// in that case we try and get the report by using the explicitly passed in exchange organization
			log.Debug().Err(tenantDomainName.Error).Msg("unable to get tenant domain name")
		} else {
			org = tenantDomainName.Data
		}
	}
	return org, nil
}

func (r *mqlMs365Exchangeonline) getExchangeReport() error {
	conn := r.MqlRuntime.Connection.(*connection.Ms365Connection)

	r.exchangeReportLock.Lock()
	defer r.exchangeReportLock.Unlock()

	// only fetch once
	if r.fetched {
		return r.fetchErr
	}

	errHandler := func(err error) error {
		r.fetchErr = err
		return err
	}

	organization, err := r.getOrg()
	if organization == "" || err != nil {
		return errHandler(errors.New("no organization provided, unable to fetch exchange online report"))
	}

	ctx := context.Background()
	token := conn.Token()
	outlookToken, err := token.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{outlookScope},
	})
	if err != nil {
		return errHandler(err)
	}

	fmtScript := fmt.Sprintf(exchangeReport, organization, conn.ClientId(), conn.TenantId(), outlookToken.Token)
	res, err := conn.CheckAndRunPowershellScript(fmtScript)
	if err != nil {
		return err
	}
	report := &ExchangeOnlineReport{}
	if res.ExitStatus == 0 {
		data, err := io.ReadAll(res.Stdout)
		if err != nil {
			return errHandler(err)
		}
		logger.DebugDumpJSON("exchange-online-report", data)

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

		logger.DebugDumpJSON("exchange-online-report", data)
		err = fmt.Errorf("failed to generate exchange online report (exit code %d): %s", res.ExitStatus, string(data))
		return errHandler(err)
	}

	malwareFilterPolicy, malwareFilterPolicyErr := convert.JsonToDictSlice(report.MalwareFilterPolicy)
	r.MalwareFilterPolicy = plugin.TValue[[]interface{}]{Data: malwareFilterPolicy, State: plugin.StateIsSet, Error: malwareFilterPolicyErr}

	hostedOutboundSpamFilterPolicy, hostedOutboundSpamFilterPolicyErr := convert.JsonToDictSlice(report.HostedOutboundSpamFilterPolicy)
	r.HostedOutboundSpamFilterPolicy = plugin.TValue[[]interface{}]{Data: hostedOutboundSpamFilterPolicy, State: plugin.StateIsSet, Error: hostedOutboundSpamFilterPolicyErr}

	transportRule, transportRuleErr := convert.JsonToDictSlice(report.TransportRule)
	r.TransportRule = plugin.TValue[[]interface{}]{Data: transportRule, State: plugin.StateIsSet, Error: transportRuleErr}

	remoteDomain, remoteDomainErr := convert.JsonToDictSlice(report.RemoteDomain)
	r.RemoteDomain = plugin.TValue[[]interface{}]{Data: remoteDomain, State: plugin.StateIsSet, Error: remoteDomainErr}

	safeLinksPolicy, safeLinksPolicyErr := convert.JsonToDictSlice(report.SafeLinksPolicy)
	r.SafeLinksPolicy = plugin.TValue[[]interface{}]{Data: safeLinksPolicy, State: plugin.StateIsSet, Error: safeLinksPolicyErr}

	safeAttachmentPolicy, safeAttachmentPolicyErr := convert.JsonToDictSlice(report.SafeAttachmentPolicy)
	r.SafeAttachmentPolicy = plugin.TValue[[]interface{}]{Data: safeAttachmentPolicy, State: plugin.StateIsSet, Error: safeAttachmentPolicyErr}

	organizationConfig, organizationConfigErr := convert.JsonToDict(report.OrganizationConfig)
	r.OrganizationConfig = plugin.TValue[interface{}]{Data: organizationConfig, State: plugin.StateIsSet, Error: organizationConfigErr}

	authenticationPolicy, authenticationPolicyErr := convert.JsonToDictSlice(report.AuthenticationPolicy)
	r.AuthenticationPolicy = plugin.TValue[[]interface{}]{Data: authenticationPolicy, State: plugin.StateIsSet, Error: authenticationPolicyErr}

	antiPhishPolicy, antiPhishPolicyErr := convert.JsonToDictSlice(report.AntiPhishPolicy)
	r.AntiPhishPolicy = plugin.TValue[[]interface{}]{Data: antiPhishPolicy, State: plugin.StateIsSet, Error: antiPhishPolicyErr}

	dkimSigningConfig, dkimSigningConfigErr := convert.JsonToDictSlice(report.DkimSigningConfig)
	r.DkimSigningConfig = plugin.TValue[[]interface{}]{Data: dkimSigningConfig, State: plugin.StateIsSet, Error: dkimSigningConfigErr}

	owaMailboxPolicy, owaMailboxPolicyErr := convert.JsonToDictSlice(report.OwaMailboxPolicy)
	r.OwaMailboxPolicy = plugin.TValue[[]interface{}]{Data: owaMailboxPolicy, State: plugin.StateIsSet, Error: owaMailboxPolicyErr}

	adminAuditLogConfig, adminAuditLogConfigErr := convert.JsonToDict(report.AdminAuditLogConfig)
	r.AdminAuditLogConfig = plugin.TValue[interface{}]{Data: adminAuditLogConfig, State: plugin.StateIsSet, Error: adminAuditLogConfigErr}

	phishFilterPolicy, phishFilterPolicyErr := convert.JsonToDictSlice(report.PhishFilterPolicy)
	r.PhishFilterPolicy = plugin.TValue[[]interface{}]{Data: phishFilterPolicy, State: plugin.StateIsSet, Error: phishFilterPolicyErr}

	mailbox, mailboxErr := convert.JsonToDictSlice(report.Mailbox)
	r.Mailbox = plugin.TValue[[]interface{}]{Data: mailbox, State: plugin.StateIsSet, Error: mailboxErr}

	atpPolicyForO365, atpPolicyForO365Err := convert.JsonToDictSlice(report.AtpPolicyForO365)
	r.AtpPolicyForO365 = plugin.TValue[[]interface{}]{Data: atpPolicyForO365, State: plugin.StateIsSet, Error: atpPolicyForO365Err}

	sharingPolicy, sharingPolicyErr := convert.JsonToDictSlice(report.SharingPolicy)
	r.SharingPolicy = plugin.TValue[[]interface{}]{Data: sharingPolicy, State: plugin.StateIsSet, Error: sharingPolicyErr}

	roleAssignmentPolicy, roleAssignmentPolicyErr := convert.JsonToDictSlice(report.RoleAssignmentPolicy)
	r.RoleAssignmentPolicy = plugin.TValue[[]interface{}]{Data: roleAssignmentPolicy, State: plugin.StateIsSet, Error: roleAssignmentPolicyErr}

	externalInOutlook := []interface{}{}
	var externalInOutlookErr error
	for _, e := range report.ExternalInOutlook {
		mql, err := CreateResource(r.MqlRuntime, "ms365.exchangeonline.externalSender",
			map[string]*llx.RawData{
				"identity":  llx.StringData(e.Identity),
				"enabled":   llx.BoolData(e.Enabled),
				"allowList": llx.ArrayData(llx.TArr2Raw(e.AllowList), types.Any),
			})
		if err != nil {
			externalInOutlookErr = err
			break
		}

		externalInOutlook = append(externalInOutlook, mql)
	}
	r.ExternalInOutlook = plugin.TValue[[]interface{}]{Data: externalInOutlook, State: plugin.StateIsSet, Error: externalInOutlookErr}

	sharedMailboxes := []interface{}{}
	var sharedMailboxesErr error
	for _, m := range report.ExoMailbox {
		mql, err := CreateResource(r.MqlRuntime, "ms365.exchangeonline.exoMailbox",
			map[string]*llx.RawData{
				"identity":                  llx.StringData(m.Identity),
				"externalDirectoryObjectId": llx.StringData(m.ExternalDirectoryObjectId),
			})
		if err != nil {
			sharedMailboxesErr = err
			break
		}

		sharedMailboxes = append(sharedMailboxes, mql)
	}
	r.SharedMailboxes = plugin.TValue[[]interface{}]{Data: sharedMailboxes, State: plugin.StateIsSet, Error: sharedMailboxesErr}

	teamsProtectionPolicy := []interface{}{}
	var teamsProtectionPolicyErr error
	for _, t := range report.TeamsProtectionPolicy {
		policy, err := CreateResource(r.MqlRuntime, "ms365.exchangeonline.teamsProtectionPolicy",
			map[string]*llx.RawData{
				"zapEnabled": llx.BoolData(t.ZapEnabled),
				"isValid":    llx.BoolData(t.IsValid),
			})
		if err != nil {
			teamsProtectionPolicyErr = err
			break
		}

		teamsProtectionPolicy = append(teamsProtectionPolicy, policy)
	}
	r.TeamsProtectionPolicy = plugin.TValue[[]interface{}]{Data: teamsProtectionPolicy, State: plugin.StateIsSet, Error: teamsProtectionPolicyErr}

	return nil
}

func (r *mqlMs365Exchangeonline) malwareFilterPolicy() ([]interface{}, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) hostedOutboundSpamFilterPolicy() ([]interface{}, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) transportRule() ([]interface{}, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) remoteDomain() ([]interface{}, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) safeLinksPolicy() ([]interface{}, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) safeAttachmentPolicy() ([]interface{}, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) organizationConfig() (interface{}, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) authenticationPolicy() ([]interface{}, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) antiPhishPolicy() ([]interface{}, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) dkimSigningConfig() ([]interface{}, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) owaMailboxPolicy() ([]interface{}, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) adminAuditLogConfig() (interface{}, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) phishFilterPolicy() ([]interface{}, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) mailbox() ([]interface{}, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) atpPolicyForO365() ([]interface{}, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) sharingPolicy() ([]interface{}, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) roleAssignmentPolicy() ([]interface{}, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) externalInOutlook() ([]interface{}, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365ExchangeonlineExternalSender) id() (string, error) {
	return r.Identity.Data, nil
}

func (r *mqlMs365Exchangeonline) sharedMailboxes() ([]interface{}, error) {
	return nil, r.getExchangeReport()
}

func (m *mqlMs365ExchangeonlineExoMailbox) id() (string, error) {
	return m.Identity.Data, nil
}

func (r *mqlMs365Exchangeonline) teamsProtectionPolicy() ([]interface{}, error) {
	return nil, r.getExchangeReport()
}

func (m *mqlMs365ExchangeonlineExoMailbox) user() (*mqlMicrosoftUser, error) {
	externalId := m.ExternalDirectoryObjectId.Data
	if externalId == "" {
		return nil, errors.New("no externalDirectoryObjectId provided, cannot find user for mailbox")
	}
	microsoft, err := m.MqlRuntime.CreateResource(m.MqlRuntime, "microsoft", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	mqlMicrosoft := microsoft.(*mqlMicrosoft)
	users := mqlMicrosoft.GetUsers()
	if users.Error != nil {
		return nil, users.Error
	}
	for _, u := range users.Data {
		mqlUser := u.(*mqlMicrosoftUser)
		if mqlUser.Id.Data == externalId {
			return mqlUser, nil
		}
	}
	return nil, errors.New("cannot find user for exchange mailbox")
}
