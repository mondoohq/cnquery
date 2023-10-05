// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v9/providers/ms365/connection"
	"go.mondoo.com/cnquery/v9/types"
)

func initMs365Exchangeonline(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn := runtime.Connection.(*connection.Ms365Connection)
	report, err := conn.GetMs365DataReport()
	if err != nil {
		return args, nil, err
	}

	malwareFilterPolicy, _ := convert.JsonToDictSlice(report.ExchangeOnline.MalwareFilterPolicy)
	hostedOutboundSpamFilterPolicy, _ := convert.JsonToDictSlice(report.ExchangeOnline.HostedOutboundSpamFilterPolicy)
	transportRule, _ := convert.JsonToDictSlice(report.ExchangeOnline.TransportRule)
	remoteDomain, _ := convert.JsonToDictSlice(report.ExchangeOnline.RemoteDomain)
	safeLinksPolicy, _ := convert.JsonToDictSlice(report.ExchangeOnline.SafeLinksPolicy)
	safeAttachmentPolicy, _ := convert.JsonToDictSlice(report.ExchangeOnline.SafeAttachmentPolicy)
	organizationConfig, _ := convert.JsonToDict(report.ExchangeOnline.OrganizationConfig)
	authenticationPolicy, _ := convert.JsonToDictSlice(report.ExchangeOnline.AuthenticationPolicy)
	antiPhishPolicy, _ := convert.JsonToDictSlice(report.ExchangeOnline.AntiPhishPolicy)
	dkimSigningConfig, _ := convert.JsonToDictSlice(report.ExchangeOnline.DkimSigningConfig)
	owaMailboxPolicy, _ := convert.JsonToDictSlice(report.ExchangeOnline.OwaMailboxPolicy)
	adminAuditLogConfig, _ := convert.JsonToDict(report.ExchangeOnline.AdminAuditLogConfig)
	phishFilterPolicy, _ := convert.JsonToDictSlice(report.ExchangeOnline.PhishFilterPolicy)
	mailbox, _ := convert.JsonToDictSlice(report.ExchangeOnline.Mailbox)
	atpPolicyForO365, _ := convert.JsonToDictSlice(report.ExchangeOnline.AtpPolicyForO365)
	sharingPolicy, _ := convert.JsonToDictSlice(report.ExchangeOnline.SharingPolicy)
	roleAssignmentPolicy, _ := convert.JsonToDictSlice(report.ExchangeOnline.RoleAssignmentPolicy)

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
	report, err := conn.GetMs365DataReport()
	if err != nil {
		return args, nil, err
	}
	spoTenant, _ := convert.JsonToDict(report.SharepointOnline.SPOTenant)
	spoTenantSyncClientRestriction, _ := convert.JsonToDict(report.SharepointOnline.SPOTenantSyncClientRestriction)

	args["spoTenant"] = llx.DictData(spoTenant)
	args["spoTenantSyncClientRestriction"] = llx.DictData(spoTenantSyncClientRestriction)

	return args, nil, nil
}

func initMs365Teams(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn := runtime.Connection.(*connection.Ms365Connection)
	report, err := conn.GetMs365DataReport()
	if err != nil {
		return args, nil, err
	}
	csTeamsClientConfiguration, _ := convert.JsonToDict(report.MsTeams.CsTeamsClientConfiguration)
	csOAuthConfiguration, _ := convert.JsonToDictSlice(report.MsTeams.CsOAuthConfiguration)

	args["csTeamsClientConfiguration"] = llx.DictData(csTeamsClientConfiguration)
	args["csOAuthConfiguration"] = llx.ArrayData(csOAuthConfiguration, types.Any)

	return args, nil, nil
}
