package ms365

import (
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (m *mqlMs365Exchangeonline) id() (string, error) {
	return "ms365.exchangeonline", nil
}

func (m *mqlMs365Exchangeonline) init(args *resources.Args) (*resources.Args, Ms365Exchangeonline, error) {
	mt, err := ms365Provider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	report, err := mt.GetMs365DataReport()
	if err != nil {
		return nil, nil, err
	}

	malwareFilterPolicy, _ := core.JsonToDictSlice(report.ExchangeOnline.MalwareFilterPolicy)
	hostedOutboundSpamFilterPolicy, _ := core.JsonToDictSlice(report.ExchangeOnline.HostedOutboundSpamFilterPolicy)
	transportRule, _ := core.JsonToDictSlice(report.ExchangeOnline.TransportRule)
	remoteDomain, _ := core.JsonToDictSlice(report.ExchangeOnline.RemoteDomain)
	safeLinksPolicy, _ := core.JsonToDictSlice(report.ExchangeOnline.SafeLinksPolicy)
	safeAttachmentPolicy, _ := core.JsonToDictSlice(report.ExchangeOnline.SafeAttachmentPolicy)
	organizationConfig, _ := core.JsonToDict(report.ExchangeOnline.OrganizationConfig)
	authenticationPolicy, _ := core.JsonToDictSlice(report.ExchangeOnline.AuthenticationPolicy)
	antiPhishPolicy, _ := core.JsonToDictSlice(report.ExchangeOnline.AntiPhishPolicy)
	dkimSigningConfig, _ := core.JsonToDictSlice(report.ExchangeOnline.DkimSigningConfig)
	owaMailboxPolicy, _ := core.JsonToDictSlice(report.ExchangeOnline.OwaMailboxPolicy)
	adminAuditLogConfig, _ := core.JsonToDict(report.ExchangeOnline.AdminAuditLogConfig)
	phishFilterPolicy, _ := core.JsonToDictSlice(report.ExchangeOnline.PhishFilterPolicy)
	mailbox, _ := core.JsonToDictSlice(report.ExchangeOnline.Mailbox)
	atpPolicyForO365, _ := core.JsonToDictSlice(report.ExchangeOnline.AtpPolicyForO365)
	sharingPolicy, _ := core.JsonToDictSlice(report.ExchangeOnline.SharingPolicy)
	roleAssignmentPolicy, _ := core.JsonToDictSlice(report.ExchangeOnline.RoleAssignmentPolicy)

	(*args)["malwareFilterPolicy"] = malwareFilterPolicy
	(*args)["hostedOutboundSpamFilterPolicy"] = hostedOutboundSpamFilterPolicy
	(*args)["transportRule"] = transportRule
	(*args)["remoteDomain"] = remoteDomain
	(*args)["safeLinksPolicy"] = safeLinksPolicy
	(*args)["safeAttachmentPolicy"] = safeAttachmentPolicy
	(*args)["organizationConfig"] = organizationConfig
	(*args)["authenticationPolicy"] = authenticationPolicy
	(*args)["antiPhishPolicy"] = antiPhishPolicy
	(*args)["dkimSigningConfig"] = dkimSigningConfig
	(*args)["owaMailboxPolicy"] = owaMailboxPolicy
	(*args)["adminAuditLogConfig"] = adminAuditLogConfig
	(*args)["phishFilterPolicy"] = phishFilterPolicy
	(*args)["mailbox"] = mailbox
	(*args)["atpPolicyForO365"] = atpPolicyForO365
	(*args)["sharingPolicy"] = sharingPolicy
	(*args)["roleAssignmentPolicy"] = roleAssignmentPolicy

	return args, nil, nil
}

func (m *mqlMs365Sharepointonline) id() (string, error) {
	return "ms365.sharepointonline", nil
}

func (m *mqlMs365Sharepointonline) init(args *resources.Args) (*resources.Args, Ms365Sharepointonline, error) {
	mt, err := ms365Provider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	report, err := mt.GetMs365DataReport()
	if err != nil {
		return nil, nil, err
	}

	spoTenant, _ := core.JsonToDict(report.SharepointOnline.SPOTenant)
	spoTenantSyncClientRestriction, _ := core.JsonToDict(report.SharepointOnline.SPOTenantSyncClientRestriction)

	(*args)["spoTenant"] = spoTenant
	(*args)["spoTenantSyncClientRestriction"] = spoTenantSyncClientRestriction

	return args, nil, nil
}

func (m *mqlMs365Teams) id() (string, error) {
	return "ms365.teams", nil
}

func (m *mqlMs365Teams) init(args *resources.Args) (*resources.Args, Ms365Teams, error) {
	mt, err := ms365Provider(m.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	report, err := mt.GetMs365DataReport()
	if err != nil {
		return nil, nil, err
	}

	csTeamsClientConfiguration, _ := core.JsonToDict(report.MsTeams.CsTeamsClientConfiguration)
	csOAuthConfiguration, _ := core.JsonToDictSlice(report.MsTeams.CsOAuthConfiguration)

	(*args)["csTeamsClientConfiguration"] = csTeamsClientConfiguration
	(*args)["csOAuthConfiguration"] = csOAuthConfiguration

	return args, nil, nil
}
