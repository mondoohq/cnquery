// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v9/providers/ms365/connection"
	"go.mondoo.com/cnquery/v9/types"
)

func (m *mqlMs365ExchangeonlineExternalSender) id() (string, error) {
	return m.Identity.Data, nil
}

func (m *mqlMs365SharepointonlineSite) id() (string, error) {
	return m.Url.Data, nil
}

func initMs365Exchangeonline(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn := runtime.Connection.(*connection.Ms365Connection)
	ctx := context.Background()
	microsoft, err := runtime.CreateResource(runtime, "microsoft", map[string]*llx.RawData{})
	if err != nil {
		return args, nil, err
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

	report, err := conn.GetExchangeReport(ctx, org)
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

	externalInOutlook := []interface{}{}
	for _, e := range report.ExternalInOutlook {
		mql, err := CreateResource(runtime, "ms365.exchangeonline.externalSender",
			map[string]*llx.RawData{
				"identity":  llx.StringData(e.Identity),
				"enabled":   llx.BoolData(e.Enabled),
				"allowList": llx.ArrayData(llx.TArr2Raw(e.AllowList), types.Any),
			})
		if err != nil {
			return args, nil, err
		}

		externalInOutlook = append(externalInOutlook, mql)
	}
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
	args["externalInOutlook"] = llx.ArrayData(externalInOutlook, types.ResourceLike)

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
		return args, nil, errors.New("no sharepoint url provided, unable to fetch sharepoint online report")
	}

	domainParts := strings.Split(spUrl, ".")
	if len(domainParts) < 2 {
		return args, nil, fmt.Errorf("invalid sharepoint url: %s", spUrl)
	}

	// we only care about the tenant name, so we take the first part in the split domain
	tenant := domainParts[0]

	report, err := conn.GetSharepointOnlineReport(ctx, tenant)
	if err != nil {
		return args, nil, err
	}
	spoTenant, _ := convert.JsonToDict(report.SpoTenant)
	spoTenantSyncClientRestriction, _ := convert.JsonToDict(report.SpoTenantSyncClientRestriction)

	sites := []interface{}{}
	for _, s := range report.SpoSite {
		mqlSpoSite, err := CreateResource(runtime, "ms365.sharepointonline.site",
			map[string]*llx.RawData{
				"denyAddAndCustomizePages": llx.BoolData(s.DenyAddAndCustomizePages == "Enabled"),
				"url":                      llx.StringData(s.Url),
			})
		if err != nil {
			return args, nil, err
		}
		sites = append(sites, mqlSpoSite)

	}
	args["spoTenant"] = llx.DictData(spoTenant)
	args["spoTenantSyncClientRestriction"] = llx.DictData(spoTenantSyncClientRestriction)
	args["spoSites"] = llx.ArrayData(sites, types.ResourceLike)
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

	tenantConfig := report.CsTenantFederationConfiguration
	teamsPolicy := report.CsTeamsMeetingPolicy
	tenantConfigBlockedDomains, _ := convert.JsonToDict(tenantConfig.BlockedDomains)

	mqlTenantConfig, err := CreateResource(runtime, "ms365.teams.tenantFederationConfig",
		map[string]*llx.RawData{
			"identity":                                    llx.StringData(tenantConfig.Identity),
			"blockedDomains":                              llx.DictData(tenantConfigBlockedDomains),
			"allowFederatedUsers":                         llx.BoolData(tenantConfig.AllowFederatedUsers),
			"allowPublicUsers":                            llx.BoolData(tenantConfig.AllowPublicUsers),
			"allowTeamsConsumer":                          llx.BoolData(tenantConfig.AllowTeamsConsumer),
			"allowTeamsConsumerInbound":                   llx.BoolData(tenantConfig.AllowTeamsConsumerInbound),
			"treatDiscoveredPartnersAsUnverified":         llx.BoolData(tenantConfig.TreatDiscoveredPartnersAsUnverified),
			"sharedSipAddressSpace":                       llx.BoolData(tenantConfig.SharedSipAddressSpace),
			"restrictTeamsConsumerToExternalUserProfiles": llx.BoolData(tenantConfig.RestrictTeamsConsumerToExternalUserProfiles),
		})
	if err != nil {
		return args, nil, err
	}
	mqlTeamsPolicy, err := CreateResource(runtime, "ms365.teams.teamsMeetingPolicyConfig",
		map[string]*llx.RawData{
			"allowAnonymousUsersToJoinMeeting":           llx.BoolData(teamsPolicy.AllowAnonymousUsersToJoinMeeting),
			"allowAnonymousUsersToStartMeeting":          llx.BoolData(teamsPolicy.AllowAnonymousUsersToStartMeeting),
			"autoAdmittedUsers":                          llx.StringData(teamsPolicy.AutoAdmittedUsers),
			"allowPSTNUsersToBypassLobby":                llx.BoolData(teamsPolicy.AllowPSTNUsersToBypassLobby),
			"meetingChatEnabledType":                     llx.StringData(teamsPolicy.MeetingChatEnabledType),
			"designatedPresenterRoleMode":                llx.StringData(teamsPolicy.DesignatedPresenterRoleMode),
			"allowExternalParticipantGiveRequestControl": llx.BoolData(teamsPolicy.AllowExternalParticipantGiveRequestControl),
			"allowSecurityEndUserReporting": 							llx.BoolData(teamsPolicy.AllowSecurityEndUserReporting),
		})
	if err != nil {
		return args, nil, err
	}

	args["csTenantFederationConfiguration"] = llx.ResourceData(mqlTenantConfig, mqlTenantConfig.MqlName())
	args["csTeamsMeetingPolicy"] = llx.ResourceData(mqlTeamsPolicy, mqlTeamsPolicy.MqlName())

	return args, nil, nil
}
