package resources

import (
	"encoding/json"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/exchangeonline"
	"io/ioutil"
)

func (p *lumiMsexchangeOnline) init(args *lumi.Args) (*lumi.Args, MsexchangeOnline, error) {

	// TODO: get path from transport option
	data, err := ioutil.ReadFile("/Users/chris-rock/go/src/go.mondoo.io/mondoo/lumi/resources/exchangeonline/testdata/exchangeonlinereport.json")
	if err != nil {
		return nil, nil, err
	}
	report := exchangeonline.ExchangeOnlineExportReport{}
	json.Unmarshal(data, &report)

	malwareFilterPolicy, _ := jsonToDictSlice(report.MalwareFilterPolicy)
	hostedOutboundSpamFilterPolicy, _ := jsonToDictSlice(report.HostedOutboundSpamFilterPolicy)
	transportRule, _ := jsonToDictSlice(report.TransportRule)
	remoteDomain, _ := jsonToDictSlice(report.RemoteDomain)
	safeLinksPolicy, _ := jsonToDictSlice(report.SafeLinksPolicy)
	safeAttachmentPolicy, _ := jsonToDictSlice(report.SafeAttachmentPolicy)
	organizationConfig, _ := jsonToDict(report.OrganizationConfig)
	authenticationPolicy, _ := jsonToDictSlice(report.AuthenticationPolicy)
	antiPhishPolicy, _ := jsonToDictSlice(report.AntiPhishPolicy)
	dkimSigningConfig, _ := jsonToDictSlice(report.DkimSigningConfig)
	owaMailboxPolicy, _ := jsonToDictSlice(report.OwaMailboxPolicy)
	adminAuditLogConfig, _ := jsonToDict(report.AdminAuditLogConfig)
	phishFilterPolicy, _ := jsonToDictSlice(report.PhishFilterPolicy)
	mailbox, _ := jsonToDictSlice(report.Mailbox)

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

	return args, nil, nil
}

func (m *lumiMsexchangeOnline) id() (string, error) {
	return "msexchange.online", nil
}
