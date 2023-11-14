// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v9/providers/ms365/connection"
	"go.mondoo.com/cnquery/v9/types"
)

func initMs365Exchangeonline(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn := runtime.Connection.(*connection.Ms365Connection)
	ctx := context.Background()
	microsoft, err := runtime.CreateResource(runtime, "microsoft", map[string]*llx.RawData{})
	if err != nil {
		return args, nil, err
	}
	mqlMicrosoft := microsoft.(*mqlMicrosoft)
	tenantDomainName := mqlMicrosoft.GetTenantDomainName()
	if tenantDomainName.Error != nil {
		return args, nil, tenantDomainName.Error
	}
	report, err := conn.GetExchangeReport(ctx, tenantDomainName.Data)
	if err != nil {
		return args, nil, err
	}

	malwareFilterPolicy, _ := convert.JsonToDictSlice(report.MalwareFilterPolicy)
	hostedOutboundSpamFilterPolicy, _ := convert.JsonToDictSlice(report.HostedOutboundSpamFilterPolicy)
	transportRule, _ := convert.JsonToDictSlice(report.TransportRule)
	remoteDomain, _ := convert.JsonToDictSlice(report.RemoteDomain)
	safeLinksPolicy, _ := convert.JsonToDictSlice(report.SafeLinksPolicy)
	safeAttachmentPolicy, _ := convert.JsonToDictSlice(report.SafeAttachmentPolicy)
	organizationConfig, _ := convert.JsonToDict(report.OrganizationConfig)
	authenticationPolicy, _ := convert.JsonToDictSlice(report.AuthenticationPolicy)
	antiPhishPolicy, _ := convert.JsonToDictSlice(report.AntiPhishPolicy)
	dkimSigningConfig, _ := convert.JsonToDictSlice(report.DkimSigningConfig)
	owaMailboxPolicy, _ := convert.JsonToDictSlice(report.OwaMailboxPolicy)
	adminAuditLogConfig, _ := convert.JsonToDict(report.AdminAuditLogConfig)
	phishFilterPolicy, _ := convert.JsonToDictSlice(report.PhishFilterPolicy)
	mailbox, _ := convert.JsonToDictSlice(report.Mailbox)
	atpPolicyForO365, _ := convert.JsonToDictSlice(report.AtpPolicyForO365)
	sharingPolicy, _ := convert.JsonToDictSlice(report.SharingPolicy)
	roleAssignmentPolicy, _ := convert.JsonToDictSlice(report.RoleAssignmentPolicy)

	args["malwareFilterPolicy"] = llx.ArrayData(malwareFilterPolicy, types.Any)
	args["hostedOutboundSpamFilterPolicy"] = llx.ArrayData(hostedOutboundSpamFilterPolicy, types.Any)
	args["transportRule"] = llx.ArrayData(transportRule, types.Any)
	args["remoteDomain"] = llx.ArrayData(remoteDomain, types.Any)
	args["safeLinksPolicy"] = llx.ArrayData(safeLinksPolicy, types.Any)
	args["safeAttachmentPolicy"] = llx.ArrayData(safeAttachmentPolicy, types.Any)
	args["organizationConfig"] = llx.DictData(organizationConfig)
	args["authenticationPolicy"] = llx.ArrayData(authenticationPolicy, types.Any)
	args["antiPhishPolicy"] = llx.ArrayData(antiPhishPolicy, types.Any)
	args["dkimSigningConfig"] = llx.ArrayData(dkimSigningConfig, types.Any)
	args["owaMailboxPolicy"] = llx.ArrayData(owaMailboxPolicy, types.Any)
	args["adminAuditLogConfig"] = llx.DictData(adminAuditLogConfig)
	args["phishFilterPolicy"] = llx.ArrayData(phishFilterPolicy, types.Any)
	args["mailbox"] = llx.ArrayData(mailbox, types.Any)
	args["atpPolicyForO365"] = llx.ArrayData(atpPolicyForO365, types.Any)
	args["sharingPolicy"] = llx.ArrayData(sharingPolicy, types.Any)
	args["roleAssignmentPolicy"] = llx.ArrayData(roleAssignmentPolicy, types.Any)

	return args, nil, nil
}

func initMs365Sharepointonline(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn := runtime.Connection.(*connection.Ms365Connection)
	ctx := context.Background()

	microsoft, err := runtime.CreateResource(runtime, "microsoft", map[string]*llx.RawData{})
	if err != nil {
		return args, nil, err
	}
	mqlMicrosoft := microsoft.(*mqlMicrosoft)
	tenantDomainName := mqlMicrosoft.GetTenantDomainName()
	if tenantDomainName.Error != nil {
		return args, nil, tenantDomainName.Error
	}

	report, err := conn.GetSharepointOnlineReport(ctx, tenantDomainName.Data)
	if err != nil {
		return args, nil, err
	}
	spoTenant, _ := convert.JsonToDict(report.SpoTenant)
	spoTenantSyncClientRestriction, _ := convert.JsonToDict(report.SpoTenantSyncClientRestriction)

	args["spoTenant"] = llx.DictData(spoTenant)
	args["spoTenantSyncClientRestriction"] = llx.DictData(spoTenantSyncClientRestriction)
	return args, nil, nil
}

func initMs365Teams(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn := runtime.Connection.(*connection.Ms365Connection)
	ctx := context.Background()
	report, err := conn.GetTeamsReport(ctx)
	if err != nil {
		return args, nil, err
	}
	csTeamsClientConfiguration, _ := convert.JsonToDict(report.CsTeamsClientConfiguration)
	args["csTeamsClientConfiguration"] = llx.DictData(csTeamsClientConfiguration)

	return args, nil, nil
}
