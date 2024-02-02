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
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/logger"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/ms365/connection"
	"go.mondoo.com/cnquery/v10/types"
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
	ExoMailbox []*ExoMailbox `json:"ExoMailbox"`
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

type mqlMs365ExchangeonlineInternal struct {
	exchangeReport     *ExchangeOnlineReport
	exchangeReportLock sync.Mutex
	once               sync.Once
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

func (r *mqlMs365Exchangeonline) getExchangeReport(ctx context.Context) (*ExchangeOnlineReport, error) {
	conn := r.MqlRuntime.Connection.(*connection.Ms365Connection)

	r.exchangeReportLock.Lock()
	defer r.exchangeReportLock.Unlock()
	if r.exchangeReport != nil {
		return r.exchangeReport, nil
	}

	organization, err := r.getOrg()
	if organization == "" || err != nil {
		return nil, errors.New("no organization provided, unable to fetch exchange online report")
	}

	token := conn.Token()
	outlookToken, err := token.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{outlookScope},
	})
	if err != nil {
		return nil, err
	}

	fmtScript := fmt.Sprintf(exchangeReport, organization, conn.ClientId(), conn.TenantId(), outlookToken)
	res, err := conn.CheckAndRunPowershellScript(fmtScript)
	if err != nil {
		return nil, err
	}
	report := &ExchangeOnlineReport{}
	if res.ExitStatus == 0 {
		data, err := io.ReadAll(res.Stdout)
		if err != nil {
			return nil, err
		}

		logger.DebugDumpJSON("exchange-online-report", data)

		err = json.Unmarshal(data, report)
		if err != nil {
			return nil, err
		}
	} else {
		data, err := io.ReadAll(res.Stderr)
		if err != nil {
			return nil, err
		}

		logger.DebugDumpJSON("exchange-online-report", data)
		return nil, fmt.Errorf("failed to generate exchange online report (exit code %d): %s", res.ExitStatus, string(data))
	}

	r.exchangeReport = report
	return report, nil
}

func (r *mqlMs365Exchangeonline) malwareFilterPolicy() ([]interface{}, error) {
	ctx := context.Background()
	report, err := r.getExchangeReport(ctx)
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(report.MalwareFilterPolicy)
}

func (r *mqlMs365Exchangeonline) hostedOutboundSpamFilterPolicy() ([]interface{}, error) {
	ctx := context.Background()
	report, err := r.getExchangeReport(ctx)
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(report.HostedOutboundSpamFilterPolicy)
}

func (r *mqlMs365Exchangeonline) transportRule() ([]interface{}, error) {
	ctx := context.Background()
	report, err := r.getExchangeReport(ctx)
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(report.TransportRule)
}

func (r *mqlMs365Exchangeonline) remoteDomain() ([]interface{}, error) {
	ctx := context.Background()
	report, err := r.getExchangeReport(ctx)
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(report.RemoteDomain)
}

func (r *mqlMs365Exchangeonline) safeLinksPolicy() ([]interface{}, error) {
	ctx := context.Background()
	report, err := r.getExchangeReport(ctx)
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(report.SafeLinksPolicy)
}

func (r *mqlMs365Exchangeonline) safeAttachmentPolicy() ([]interface{}, error) {
	ctx := context.Background()
	report, err := r.getExchangeReport(ctx)
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(report.SafeAttachmentPolicy)
}

func (r *mqlMs365Exchangeonline) organizationConfig() (interface{}, error) {
	ctx := context.Background()
	report, err := r.getExchangeReport(ctx)
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(report.OrganizationConfig)
}

func (r *mqlMs365Exchangeonline) authenticationPolicy() ([]interface{}, error) {
	ctx := context.Background()
	report, err := r.getExchangeReport(ctx)
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(report.AuthenticationPolicy)
}

func (r *mqlMs365Exchangeonline) antiPhishPolicy() ([]interface{}, error) {
	ctx := context.Background()
	report, err := r.getExchangeReport(ctx)
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(report.AntiPhishPolicy)
}

func (r *mqlMs365Exchangeonline) dkimSigningConfig() ([]interface{}, error) {
	ctx := context.Background()
	report, err := r.getExchangeReport(ctx)
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(report.DkimSigningConfig)
}

func (r *mqlMs365Exchangeonline) owaMailboxPolicy() ([]interface{}, error) {
	ctx := context.Background()
	report, err := r.getExchangeReport(ctx)
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(report.OwaMailboxPolicy)
}

func (r *mqlMs365Exchangeonline) adminAuditLogConfig() (interface{}, error) {
	ctx := context.Background()
	report, err := r.getExchangeReport(ctx)
	if err != nil {
		return nil, err
	}
	return convert.JsonToDict(report.AdminAuditLogConfig)
}

func (r *mqlMs365Exchangeonline) phishFilterPolicy() ([]interface{}, error) {
	ctx := context.Background()
	report, err := r.getExchangeReport(ctx)
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(report.PhishFilterPolicy)
}

func (r *mqlMs365Exchangeonline) mailbox() ([]interface{}, error) {
	ctx := context.Background()
	report, err := r.getExchangeReport(ctx)
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(report.Mailbox)
}

func (r *mqlMs365Exchangeonline) atpPolicyForO365() ([]interface{}, error) {
	ctx := context.Background()
	report, err := r.getExchangeReport(ctx)
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(report.AtpPolicyForO365)
}

func (r *mqlMs365Exchangeonline) sharingPolicy() ([]interface{}, error) {
	ctx := context.Background()
	report, err := r.getExchangeReport(ctx)
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(report.SharingPolicy)
}

func (r *mqlMs365Exchangeonline) roleAssignmentPolicy() ([]interface{}, error) {
	ctx := context.Background()
	report, err := r.getExchangeReport(ctx)
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(report.RoleAssignmentPolicy)
}

func (r *mqlMs365Exchangeonline) externalInOutlook() ([]interface{}, error) {
	ctx := context.Background()
	report, err := r.getExchangeReport(ctx)
	if err != nil {
		return nil, err
	}
	externalInOutlook := []interface{}{}
	for _, e := range report.ExternalInOutlook {
		mql, err := CreateResource(r.MqlRuntime, "ms365.exchangeonline.externalSender",
			map[string]*llx.RawData{
				"identity":  llx.StringData(e.Identity),
				"enabled":   llx.BoolData(e.Enabled),
				"allowList": llx.ArrayData(llx.TArr2Raw(e.AllowList), types.Any),
			})
		if err != nil {
			return nil, err
		}

		externalInOutlook = append(externalInOutlook, mql)
	}
	return externalInOutlook, nil
}

func (r *mqlMs365ExchangeonlineExternalSender) id() (string, error) {
	return r.Identity.Data, nil
}

func (r *mqlMs365Exchangeonline) sharedMailboxes() ([]interface{}, error) {
	ctx := context.Background()
	report, err := r.getExchangeReport(ctx)
	if err != nil {
		return nil, err
	}
	sharedMailboxes := []interface{}{}
	for _, m := range report.ExoMailbox {
		mql, err := CreateResource(r.MqlRuntime, "ms365.exchangeonline.exoMailbox",
			map[string]*llx.RawData{
				"identity":                  llx.StringData(m.Identity),
				"externalDirectoryObjectId": llx.StringData(m.ExternalDirectoryObjectId),
			})
		if err != nil {
			return nil, err
		}

		sharedMailboxes = append(sharedMailboxes, mql)
	}
	return sharedMailboxes, nil
}

func (m *mqlMs365ExchangeonlineExoMailbox) id() (string, error) {
	return m.Identity.Data, nil
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
