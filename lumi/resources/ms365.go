package resources

import (
	"go.mondoo.io/mondoo/lumi"
)

func (m *lumiMs365Exchangeonline) id() (string, error) {
	return "ms365.exchangeonline", nil
}

func (m *lumiMs365Exchangeonline) init(args *lumi.Args) (*lumi.Args, Ms365Exchangeonline, error) {
	mt, err := ms365transport(m.Runtime.Motor.Transport)
	if err != nil {
		return nil, nil, err
	}

	report, err := mt.GetMs365DataReport()
	if err != nil {
		return nil, nil, err
	}

	malwareFilterPolicy, _ := jsonToDictSlice(report.ExchangeOnline.MalwareFilterPolicy)
	hostedOutboundSpamFilterPolicy, _ := jsonToDictSlice(report.ExchangeOnline.HostedOutboundSpamFilterPolicy)
	transportRule, _ := jsonToDictSlice(report.ExchangeOnline.TransportRule)
	remoteDomain, _ := jsonToDictSlice(report.ExchangeOnline.RemoteDomain)
	safeLinksPolicy, _ := jsonToDictSlice(report.ExchangeOnline.SafeLinksPolicy)
	safeAttachmentPolicy, _ := jsonToDictSlice(report.ExchangeOnline.SafeAttachmentPolicy)
	organizationConfig, _ := jsonToDict(report.ExchangeOnline.OrganizationConfig)
	authenticationPolicy, _ := jsonToDictSlice(report.ExchangeOnline.AuthenticationPolicy)
	antiPhishPolicy, _ := jsonToDictSlice(report.ExchangeOnline.AntiPhishPolicy)
	dkimSigningConfig, _ := jsonToDictSlice(report.ExchangeOnline.DkimSigningConfig)
	owaMailboxPolicy, _ := jsonToDictSlice(report.ExchangeOnline.OwaMailboxPolicy)
	adminAuditLogConfig, _ := jsonToDict(report.ExchangeOnline.AdminAuditLogConfig)
	phishFilterPolicy, _ := jsonToDictSlice(report.ExchangeOnline.PhishFilterPolicy)
	mailbox, _ := jsonToDictSlice(report.ExchangeOnline.Mailbox)
	atpPolicyForO365, _ := jsonToDictSlice(report.ExchangeOnline.AtpPolicyForO365)
	sharingPolicy, _ := jsonToDictSlice(report.ExchangeOnline.SharingPolicy)
	roleAssignmentPolicy, _ := jsonToDictSlice(report.ExchangeOnline.RoleAssignmentPolicy)

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

func (m *lumiMs365Sharepointonline) id() (string, error) {
	return "ms365.sharepointonline", nil
}

func (m *lumiMs365Sharepointonline) init(args *lumi.Args) (*lumi.Args, Ms365Sharepointonline, error) {
	mt, err := ms365transport(m.Runtime.Motor.Transport)
	if err != nil {
		return nil, nil, err
	}

	report, err := mt.GetMs365DataReport()
	if err != nil {
		return nil, nil, err
	}

	spoTenant, _ := jsonToDict(report.SharepointOnline.SPOTenant)
	spoTenantSyncClientRestriction, _ := jsonToDict(report.SharepointOnline.SPOTenantSyncClientRestriction)

	(*args)["spoTenant"] = spoTenant
	(*args)["spoTenantSyncClientRestriction"] = spoTenantSyncClientRestriction

	return args, nil, nil
}

func (m *lumiMs365Teams) id() (string, error) {
	return "ms365.teams", nil
}

func (m *lumiMs365Teams) init(args *lumi.Args) (*lumi.Args, Ms365Teams, error) {
	mt, err := ms365transport(m.Runtime.Motor.Transport)
	if err != nil {
		return nil, nil, err
	}

	report, err := mt.GetMs365DataReport()
	if err != nil {
		return nil, nil, err
	}

	csTeamsClientConfiguration, _ := jsonToDict(report.MsTeams.CsTeamsClientConfiguration)
	csOAuthConfiguration, _ := jsonToDictSlice(report.MsTeams.CsOAuthConfiguration)

	(*args)["csTeamsClientConfiguration"] = csTeamsClientConfiguration
	(*args)["csOAuthConfiguration"] = csOAuthConfiguration

	return args, nil, nil
}
