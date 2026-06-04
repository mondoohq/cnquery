// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
)

// decodeExchangeList re-decodes the raw cmdlet payload (already unmarshalled
// into []any by the report parser) into a struct slice. The full object is
// serialized in the report, so the decoded view is a faithful subset of the
// same data with no additional API call.
func decodeExchangeList[T any](raw any) ([]*T, error) {
	if raw == nil {
		return nil, nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var out []*T
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// --- Transport rules ---

type ExchangeTransportRule struct {
	Identity string `json:"Identity"`
	Name     string `json:"Name"`
	Priority int64  `json:"Priority"`
	State    string `json:"State"`
	Mode     string `json:"Mode"`
	Comments string `json:"Comments"`
}

func convertTransportRules(r *mqlMs365Exchangeonline, raw any) ([]any, error) {
	rules, err := decodeExchangeList[ExchangeTransportRule](raw)
	if err != nil {
		return nil, err
	}
	result := []any{}
	for _, t := range rules {
		if t == nil {
			continue
		}
		mql, err := CreateResource(r.MqlRuntime, "ms365.exchangeonline.transportRuleEntry",
			map[string]*llx.RawData{
				"__id":     llx.StringData("transportRule-" + t.Identity),
				"identity": llx.StringData(t.Identity),
				"name":     llx.StringData(t.Name),
				"priority": llx.IntData(t.Priority),
				"state":    llx.StringData(t.State),
				"mode":     llx.StringData(t.Mode),
				"comments": llx.StringData(t.Comments),
			})
		if err != nil {
			return nil, err
		}
		result = append(result, mql)
	}
	return result, nil
}

// --- Anti-phishing policies ---

type ExchangeAntiPhishPolicy struct {
	Identity                            string `json:"Identity"`
	Name                                string `json:"Name"`
	Enabled                             bool   `json:"Enabled"`
	PhishThresholdLevel                 int64  `json:"PhishThresholdLevel"`
	EnableMailboxIntelligence           bool   `json:"EnableMailboxIntelligence"`
	EnableMailboxIntelligenceProtection bool   `json:"EnableMailboxIntelligenceProtection"`
	EnableSpoofIntelligence             bool   `json:"EnableSpoofIntelligence"`
	EnableFirstContactSafetyTips        bool   `json:"EnableFirstContactSafetyTips"`
	EnableTargetedUserProtection        bool   `json:"EnableTargetedUserProtection"`
	EnableTargetedDomainsProtection     bool   `json:"EnableTargetedDomainsProtection"`
	EnableOrganizationDomainsProtection bool   `json:"EnableOrganizationDomainsProtection"`
	TargetedUserProtectionAction        string `json:"TargetedUserProtectionAction"`
	TargetedDomainProtectionAction      string `json:"TargetedDomainProtectionAction"`
	AuthenticationFailAction            string `json:"AuthenticationFailAction"`
}

func convertAntiPhishPolicies(r *mqlMs365Exchangeonline, raw any) ([]any, error) {
	policies, err := decodeExchangeList[ExchangeAntiPhishPolicy](raw)
	if err != nil {
		return nil, err
	}
	result := []any{}
	for _, p := range policies {
		if p == nil {
			continue
		}
		mql, err := CreateResource(r.MqlRuntime, "ms365.exchangeonline.antiPhishPolicyEntry",
			map[string]*llx.RawData{
				"__id":                                llx.StringData("antiPhishPolicy-" + p.Identity),
				"identity":                            llx.StringData(p.Identity),
				"name":                                llx.StringData(p.Name),
				"enabled":                             llx.BoolData(p.Enabled),
				"phishThresholdLevel":                 llx.IntData(p.PhishThresholdLevel),
				"enableMailboxIntelligence":           llx.BoolData(p.EnableMailboxIntelligence),
				"enableMailboxIntelligenceProtection": llx.BoolData(p.EnableMailboxIntelligenceProtection),
				"enableSpoofIntelligence":             llx.BoolData(p.EnableSpoofIntelligence),
				"enableFirstContactSafetyTips":        llx.BoolData(p.EnableFirstContactSafetyTips),
				"enableTargetedUserProtection":        llx.BoolData(p.EnableTargetedUserProtection),
				"enableTargetedDomainsProtection":     llx.BoolData(p.EnableTargetedDomainsProtection),
				"enableOrganizationDomainsProtection": llx.BoolData(p.EnableOrganizationDomainsProtection),
				"targetedUserProtectionAction":        llx.StringData(p.TargetedUserProtectionAction),
				"targetedDomainProtectionAction":      llx.StringData(p.TargetedDomainProtectionAction),
				"authenticationFailAction":            llx.StringData(p.AuthenticationFailAction),
			})
		if err != nil {
			return nil, err
		}
		result = append(result, mql)
	}
	return result, nil
}

// --- Safe Links policies ---

type ExchangeSafeLinksPolicy struct {
	Identity                 string `json:"Identity"`
	Name                     string `json:"Name"`
	EnableSafeLinksForEmail  bool   `json:"EnableSafeLinksForEmail"`
	EnableSafeLinksForTeams  bool   `json:"EnableSafeLinksForTeams"`
	EnableSafeLinksForOffice bool   `json:"EnableSafeLinksForOffice"`
	TrackClicks              bool   `json:"TrackClicks"`
	AllowClickThrough        bool   `json:"AllowClickThrough"`
	ScanUrls                 bool   `json:"ScanUrls"`
	EnableForInternalSenders bool   `json:"EnableForInternalSenders"`
	DeliverMessageAfterScan  bool   `json:"DeliverMessageAfterScan"`
	DisableUrlRewrite        bool   `json:"DisableUrlRewrite"`
}

func convertSafeLinksPolicies(r *mqlMs365Exchangeonline, raw any) ([]any, error) {
	policies, err := decodeExchangeList[ExchangeSafeLinksPolicy](raw)
	if err != nil {
		return nil, err
	}
	result := []any{}
	for _, p := range policies {
		if p == nil {
			continue
		}
		mql, err := CreateResource(r.MqlRuntime, "ms365.exchangeonline.safeLinksPolicyEntry",
			map[string]*llx.RawData{
				"__id":                     llx.StringData("safeLinksPolicy-" + p.Identity),
				"identity":                 llx.StringData(p.Identity),
				"name":                     llx.StringData(p.Name),
				"enableSafeLinksForEmail":  llx.BoolData(p.EnableSafeLinksForEmail),
				"enableSafeLinksForTeams":  llx.BoolData(p.EnableSafeLinksForTeams),
				"enableSafeLinksForOffice": llx.BoolData(p.EnableSafeLinksForOffice),
				"trackClicks":              llx.BoolData(p.TrackClicks),
				"allowClickThrough":        llx.BoolData(p.AllowClickThrough),
				"scanUrls":                 llx.BoolData(p.ScanUrls),
				"enableForInternalSenders": llx.BoolData(p.EnableForInternalSenders),
				"deliverMessageAfterScan":  llx.BoolData(p.DeliverMessageAfterScan),
				"disableUrlRewrite":        llx.BoolData(p.DisableUrlRewrite),
			})
		if err != nil {
			return nil, err
		}
		result = append(result, mql)
	}
	return result, nil
}

// --- Safe Attachment policies ---

type ExchangeSafeAttachmentPolicy struct {
	Identity        string `json:"Identity"`
	Name            string `json:"Name"`
	Enable          bool   `json:"Enable"`
	Action          string `json:"Action"`
	Redirect        bool   `json:"Redirect"`
	RedirectAddress string `json:"RedirectAddress"`
}

func convertSafeAttachmentPolicies(r *mqlMs365Exchangeonline, raw any) ([]any, error) {
	policies, err := decodeExchangeList[ExchangeSafeAttachmentPolicy](raw)
	if err != nil {
		return nil, err
	}
	result := []any{}
	for _, p := range policies {
		if p == nil {
			continue
		}
		mql, err := CreateResource(r.MqlRuntime, "ms365.exchangeonline.safeAttachmentPolicyEntry",
			map[string]*llx.RawData{
				"__id":            llx.StringData("safeAttachmentPolicy-" + p.Identity),
				"identity":        llx.StringData(p.Identity),
				"name":            llx.StringData(p.Name),
				"enable":          llx.BoolData(p.Enable),
				"action":          llx.StringData(p.Action),
				"redirect":        llx.BoolData(p.Redirect),
				"redirectAddress": llx.StringData(p.RedirectAddress),
			})
		if err != nil {
			return nil, err
		}
		result = append(result, mql)
	}
	return result, nil
}

// --- Malware filter policies ---

type ExchangeMalwareFilterPolicy struct {
	Identity                               string   `json:"Identity"`
	Name                                   string   `json:"Name"`
	EnableFileFilter                       bool     `json:"EnableFileFilter"`
	ZapEnabled                             bool     `json:"ZapEnabled"`
	EnableInternalSenderAdminNotifications bool     `json:"EnableInternalSenderAdminNotifications"`
	EnableExternalSenderAdminNotifications bool     `json:"EnableExternalSenderAdminNotifications"`
	FileTypes                              []string `json:"FileTypes"`
}

func convertMalwareFilterPolicies(r *mqlMs365Exchangeonline, raw any) ([]any, error) {
	policies, err := decodeExchangeList[ExchangeMalwareFilterPolicy](raw)
	if err != nil {
		return nil, err
	}
	result := []any{}
	for _, p := range policies {
		if p == nil {
			continue
		}
		mql, err := CreateResource(r.MqlRuntime, "ms365.exchangeonline.malwareFilterPolicyEntry",
			map[string]*llx.RawData{
				"__id":                                   llx.StringData("malwareFilterPolicy-" + p.Identity),
				"identity":                               llx.StringData(p.Identity),
				"name":                                   llx.StringData(p.Name),
				"enableFileFilter":                       llx.BoolData(p.EnableFileFilter),
				"zapEnabled":                             llx.BoolData(p.ZapEnabled),
				"enableInternalSenderAdminNotifications": llx.BoolData(p.EnableInternalSenderAdminNotifications),
				"enableExternalSenderAdminNotifications": llx.BoolData(p.EnableExternalSenderAdminNotifications),
				"fileTypes":                              llx.ArrayData(llx.TArr2Raw(p.FileTypes), types.String),
			})
		if err != nil {
			return nil, err
		}
		result = append(result, mql)
	}
	return result, nil
}

// --- Hosted content (spam) filter policies ---

type ExchangeHostedContentFilterPolicy struct {
	Identity                       string   `json:"Identity"`
	Name                           string   `json:"Name"`
	SpamAction                     string   `json:"SpamAction"`
	HighConfidenceSpamAction       string   `json:"HighConfidenceSpamAction"`
	PhishSpamAction                string   `json:"PhishSpamAction"`
	HighConfidencePhishAction      string   `json:"HighConfidencePhishAction"`
	BulkSpamAction                 string   `json:"BulkSpamAction"`
	BulkThreshold                  int64    `json:"BulkThreshold"`
	EnableEndUserSpamNotifications bool     `json:"EnableEndUserSpamNotifications"`
	AllowedSenderDomains           []string `json:"AllowedSenderDomains"`
	BlockedSenderDomains           []string `json:"BlockedSenderDomains"`
}

func convertHostedContentFilterPolicies(r *mqlMs365Exchangeonline, raw any) ([]any, error) {
	policies, err := decodeExchangeList[ExchangeHostedContentFilterPolicy](raw)
	if err != nil {
		return nil, err
	}
	result := []any{}
	for _, p := range policies {
		if p == nil {
			continue
		}
		mql, err := CreateResource(r.MqlRuntime, "ms365.exchangeonline.hostedContentFilterPolicyEntry",
			map[string]*llx.RawData{
				"__id":                           llx.StringData("hostedContentFilterPolicy-" + p.Identity),
				"identity":                       llx.StringData(p.Identity),
				"name":                           llx.StringData(p.Name),
				"spamAction":                     llx.StringData(p.SpamAction),
				"highConfidenceSpamAction":       llx.StringData(p.HighConfidenceSpamAction),
				"phishSpamAction":                llx.StringData(p.PhishSpamAction),
				"highConfidencePhishAction":      llx.StringData(p.HighConfidencePhishAction),
				"bulkSpamAction":                 llx.StringData(p.BulkSpamAction),
				"bulkThreshold":                  llx.IntData(p.BulkThreshold),
				"enableEndUserSpamNotifications": llx.BoolData(p.EnableEndUserSpamNotifications),
				"allowedSenderDomains":           llx.ArrayData(llx.TArr2Raw(p.AllowedSenderDomains), types.String),
				"blockedSenderDomains":           llx.ArrayData(llx.TArr2Raw(p.BlockedSenderDomains), types.String),
			})
		if err != nil {
			return nil, err
		}
		result = append(result, mql)
	}
	return result, nil
}

// --- Hosted outbound spam filter policies ---

type ExchangeHostedOutboundSpamFilterPolicy struct {
	Identity                      string `json:"Identity"`
	Name                          string `json:"Name"`
	RecipientLimitExternalPerHour int64  `json:"RecipientLimitExternalPerHour"`
	RecipientLimitInternalPerHour int64  `json:"RecipientLimitInternalPerHour"`
	RecipientLimitPerDay          int64  `json:"RecipientLimitPerDay"`
	ActionWhenThresholdReached    string `json:"ActionWhenThresholdReached"`
	AutoForwardingMode            string `json:"AutoForwardingMode"`
	BccSuspiciousOutboundMail     bool   `json:"BccSuspiciousOutboundMail"`
	NotifyOutboundSpam            bool   `json:"NotifyOutboundSpam"`
}

func convertHostedOutboundSpamFilterPolicies(r *mqlMs365Exchangeonline, raw any) ([]any, error) {
	policies, err := decodeExchangeList[ExchangeHostedOutboundSpamFilterPolicy](raw)
	if err != nil {
		return nil, err
	}
	result := []any{}
	for _, p := range policies {
		if p == nil {
			continue
		}
		mql, err := CreateResource(r.MqlRuntime, "ms365.exchangeonline.hostedOutboundSpamFilterPolicyEntry",
			map[string]*llx.RawData{
				"__id":                          llx.StringData("hostedOutboundSpamFilterPolicy-" + p.Identity),
				"identity":                      llx.StringData(p.Identity),
				"name":                          llx.StringData(p.Name),
				"recipientLimitExternalPerHour": llx.IntData(p.RecipientLimitExternalPerHour),
				"recipientLimitInternalPerHour": llx.IntData(p.RecipientLimitInternalPerHour),
				"recipientLimitPerDay":          llx.IntData(p.RecipientLimitPerDay),
				"actionWhenThresholdReached":    llx.StringData(p.ActionWhenThresholdReached),
				"autoForwardingMode":            llx.StringData(p.AutoForwardingMode),
				"bccSuspiciousOutboundMail":     llx.BoolData(p.BccSuspiciousOutboundMail),
				"notifyOutboundSpam":            llx.BoolData(p.NotifyOutboundSpam),
			})
		if err != nil {
			return nil, err
		}
		result = append(result, mql)
	}
	return result, nil
}

// --- DKIM signing configurations ---

type ExchangeDkimSigningConfig struct {
	Identity string `json:"Identity"`
	Domain   string `json:"Domain"`
	Enabled  bool   `json:"Enabled"`
	Status   string `json:"Status"`
}

func convertDkimSigningConfigs(r *mqlMs365Exchangeonline, raw any) ([]any, error) {
	configs, err := decodeExchangeList[ExchangeDkimSigningConfig](raw)
	if err != nil {
		return nil, err
	}
	result := []any{}
	for _, c := range configs {
		if c == nil {
			continue
		}
		mql, err := CreateResource(r.MqlRuntime, "ms365.exchangeonline.dkimSigningConfigEntry",
			map[string]*llx.RawData{
				"__id":     llx.StringData("dkimSigningConfig-" + c.Identity),
				"identity": llx.StringData(c.Identity),
				"domain":   llx.StringData(c.Domain),
				"enabled":  llx.BoolData(c.Enabled),
				"status":   llx.StringData(c.Status),
			})
		if err != nil {
			return nil, err
		}
		result = append(result, mql)
	}
	return result, nil
}

// --- Authentication policies ---

type ExchangeAuthenticationPolicy struct {
	Identity                           string `json:"Identity"`
	Name                               string `json:"Name"`
	AllowBasicAuthActiveSync           bool   `json:"AllowBasicAuthActiveSync"`
	AllowBasicAuthAutodiscover         bool   `json:"AllowBasicAuthAutodiscover"`
	AllowBasicAuthImap                 bool   `json:"AllowBasicAuthImap"`
	AllowBasicAuthMapi                 bool   `json:"AllowBasicAuthMapi"`
	AllowBasicAuthOfflineAddressBook   bool   `json:"AllowBasicAuthOfflineAddressBook"`
	AllowBasicAuthOutlookService       bool   `json:"AllowBasicAuthOutlookService"`
	AllowBasicAuthPop                  bool   `json:"AllowBasicAuthPop"`
	AllowBasicAuthPowershell           bool   `json:"AllowBasicAuthPowershell"`
	AllowBasicAuthReportingWebServices bool   `json:"AllowBasicAuthReportingWebServices"`
	AllowBasicAuthRpc                  bool   `json:"AllowBasicAuthRpc"`
	AllowBasicAuthSmtp                 bool   `json:"AllowBasicAuthSmtp"`
	AllowBasicAuthWebServices          bool   `json:"AllowBasicAuthWebServices"`
}

func convertAuthenticationPolicies(r *mqlMs365Exchangeonline, raw any) ([]any, error) {
	policies, err := decodeExchangeList[ExchangeAuthenticationPolicy](raw)
	if err != nil {
		return nil, err
	}
	result := []any{}
	for _, p := range policies {
		if p == nil {
			continue
		}
		mql, err := CreateResource(r.MqlRuntime, "ms365.exchangeonline.authenticationPolicyEntry",
			map[string]*llx.RawData{
				"__id":                               llx.StringData("authenticationPolicy-" + p.Identity),
				"identity":                           llx.StringData(p.Identity),
				"name":                               llx.StringData(p.Name),
				"allowBasicAuthActiveSync":           llx.BoolData(p.AllowBasicAuthActiveSync),
				"allowBasicAuthAutodiscover":         llx.BoolData(p.AllowBasicAuthAutodiscover),
				"allowBasicAuthImap":                 llx.BoolData(p.AllowBasicAuthImap),
				"allowBasicAuthMapi":                 llx.BoolData(p.AllowBasicAuthMapi),
				"allowBasicAuthOfflineAddressBook":   llx.BoolData(p.AllowBasicAuthOfflineAddressBook),
				"allowBasicAuthOutlookService":       llx.BoolData(p.AllowBasicAuthOutlookService),
				"allowBasicAuthPop":                  llx.BoolData(p.AllowBasicAuthPop),
				"allowBasicAuthPowershell":           llx.BoolData(p.AllowBasicAuthPowershell),
				"allowBasicAuthReportingWebServices": llx.BoolData(p.AllowBasicAuthReportingWebServices),
				"allowBasicAuthRpc":                  llx.BoolData(p.AllowBasicAuthRpc),
				"allowBasicAuthSmtp":                 llx.BoolData(p.AllowBasicAuthSmtp),
				"allowBasicAuthWebServices":          llx.BoolData(p.AllowBasicAuthWebServices),
			})
		if err != nil {
			return nil, err
		}
		result = append(result, mql)
	}
	return result, nil
}

// --- OWA mailbox policies ---

type ExchangeOwaMailboxPolicy struct {
	Identity                                  string   `json:"Identity"`
	Name                                      string   `json:"Name"`
	AdditionalStorageProvidersAvailable       bool     `json:"AdditionalStorageProvidersAvailable"`
	DirectFileAccessOnPublicComputersEnabled  bool     `json:"DirectFileAccessOnPublicComputersEnabled"`
	DirectFileAccessOnPrivateComputersEnabled bool     `json:"DirectFileAccessOnPrivateComputersEnabled"`
	ForceSaveAttachmentFilteringEnabled       bool     `json:"ForceSaveAttachmentFilteringEnabled"`
	AllowedFileTypes                          []string `json:"AllowedFileTypes"`
	BlockedFileTypes                          []string `json:"BlockedFileTypes"`
}

func convertOwaMailboxPolicies(r *mqlMs365Exchangeonline, raw any) ([]any, error) {
	policies, err := decodeExchangeList[ExchangeOwaMailboxPolicy](raw)
	if err != nil {
		return nil, err
	}
	result := []any{}
	for _, p := range policies {
		if p == nil {
			continue
		}
		mql, err := CreateResource(r.MqlRuntime, "ms365.exchangeonline.owaMailboxPolicyEntry",
			map[string]*llx.RawData{
				"__id":                                llx.StringData("owaMailboxPolicy-" + p.Identity),
				"identity":                            llx.StringData(p.Identity),
				"name":                                llx.StringData(p.Name),
				"additionalStorageProvidersAvailable": llx.BoolData(p.AdditionalStorageProvidersAvailable),
				"directFileAccessOnPublicComputersEnabled":  llx.BoolData(p.DirectFileAccessOnPublicComputersEnabled),
				"directFileAccessOnPrivateComputersEnabled": llx.BoolData(p.DirectFileAccessOnPrivateComputersEnabled),
				"forceSaveAttachmentFilteringEnabled":       llx.BoolData(p.ForceSaveAttachmentFilteringEnabled),
				"allowedFileTypes":                          llx.ArrayData(llx.TArr2Raw(p.AllowedFileTypes), types.String),
				"blockedFileTypes":                          llx.ArrayData(llx.TArr2Raw(p.BlockedFileTypes), types.String),
			})
		if err != nil {
			return nil, err
		}
		result = append(result, mql)
	}
	return result, nil
}

// --- Remote domains ---

type ExchangeRemoteDomain struct {
	Identity              string `json:"Identity"`
	Name                  string `json:"Name"`
	DomainName            string `json:"DomainName"`
	AllowedOOFType        string `json:"AllowedOOFType"`
	AutoReplyEnabled      bool   `json:"AutoReplyEnabled"`
	AutoForwardEnabled    bool   `json:"AutoForwardEnabled"`
	DeliveryReportEnabled bool   `json:"DeliveryReportEnabled"`
	NDREnabled            bool   `json:"NDREnabled"`
	IsInternal            bool   `json:"IsInternal"`
}

func convertRemoteDomains(r *mqlMs365Exchangeonline, raw any) ([]any, error) {
	domains, err := decodeExchangeList[ExchangeRemoteDomain](raw)
	if err != nil {
		return nil, err
	}
	result := []any{}
	for _, d := range domains {
		if d == nil {
			continue
		}
		mql, err := CreateResource(r.MqlRuntime, "ms365.exchangeonline.remoteDomainEntry",
			map[string]*llx.RawData{
				"__id":                  llx.StringData("remoteDomain-" + d.Identity),
				"identity":              llx.StringData(d.Identity),
				"name":                  llx.StringData(d.Name),
				"domainName":            llx.StringData(d.DomainName),
				"allowedOOFType":        llx.StringData(d.AllowedOOFType),
				"autoReplyEnabled":      llx.BoolData(d.AutoReplyEnabled),
				"autoForwardEnabled":    llx.BoolData(d.AutoForwardEnabled),
				"deliveryReportEnabled": llx.BoolData(d.DeliveryReportEnabled),
				"ndrEnabled":            llx.BoolData(d.NDREnabled),
				"isInternal":            llx.BoolData(d.IsInternal),
			})
		if err != nil {
			return nil, err
		}
		result = append(result, mql)
	}
	return result, nil
}

// --- Quarantine policies ---

type ExchangeQuarantinePolicy struct {
	Identity                          string `json:"Identity"`
	Name                              string `json:"Name"`
	EndUserQuarantinePermissionsValue int64  `json:"EndUserQuarantinePermissionsValue"`
	ESNEnabled                        bool   `json:"ESNEnabled"`
}

func convertQuarantinePolicies(r *mqlMs365Exchangeonline, raw any) ([]any, error) {
	policies, err := decodeExchangeList[ExchangeQuarantinePolicy](raw)
	if err != nil {
		return nil, err
	}
	result := []any{}
	for _, p := range policies {
		if p == nil {
			continue
		}
		mql, err := CreateResource(r.MqlRuntime, "ms365.exchangeonline.quarantinePolicyEntry",
			map[string]*llx.RawData{
				"__id":                              llx.StringData("quarantinePolicy-" + p.Identity),
				"identity":                          llx.StringData(p.Identity),
				"name":                              llx.StringData(p.Name),
				"endUserQuarantinePermissionsValue": llx.IntData(p.EndUserQuarantinePermissionsValue),
				"esnEnabled":                        llx.BoolData(p.ESNEnabled),
			})
		if err != nil {
			return nil, err
		}
		result = append(result, mql)
	}
	return result, nil
}

// --- ATP policy for Office 365 ---

type ExchangeAtpPolicyForO365 struct {
	Identity                string `json:"Identity"`
	Name                    string `json:"Name"`
	EnableSafeDocs          bool   `json:"EnableSafeDocs"`
	AllowSafeDocsOpen       bool   `json:"AllowSafeDocsOpen"`
	EnableATPForSPOTeamsODB bool   `json:"EnableATPForSPOTeamsODB"`
}

func convertAtpPoliciesForO365(r *mqlMs365Exchangeonline, raw any) ([]any, error) {
	policies, err := decodeExchangeList[ExchangeAtpPolicyForO365](raw)
	if err != nil {
		return nil, err
	}
	result := []any{}
	for _, p := range policies {
		if p == nil {
			continue
		}
		mql, err := CreateResource(r.MqlRuntime, "ms365.exchangeonline.atpPolicyForO365Entry",
			map[string]*llx.RawData{
				"__id":                    llx.StringData("atpPolicyForO365-" + p.Identity),
				"identity":                llx.StringData(p.Identity),
				"name":                    llx.StringData(p.Name),
				"enableSafeDocs":          llx.BoolData(p.EnableSafeDocs),
				"allowSafeDocsOpen":       llx.BoolData(p.AllowSafeDocsOpen),
				"enableATPForSPOTeamsODB": llx.BoolData(p.EnableATPForSPOTeamsODB),
			})
		if err != nil {
			return nil, err
		}
		result = append(result, mql)
	}
	return result, nil
}

// --- Sharing policies ---

type ExchangeSharingPolicy struct {
	Identity string   `json:"Identity"`
	Name     string   `json:"Name"`
	Enabled  bool     `json:"Enabled"`
	Default  bool     `json:"Default"`
	Domains  []string `json:"Domains"`
}

func convertSharingPolicies(r *mqlMs365Exchangeonline, raw any) ([]any, error) {
	policies, err := decodeExchangeList[ExchangeSharingPolicy](raw)
	if err != nil {
		return nil, err
	}
	result := []any{}
	for _, p := range policies {
		if p == nil {
			continue
		}
		mql, err := CreateResource(r.MqlRuntime, "ms365.exchangeonline.sharingPolicyEntry",
			map[string]*llx.RawData{
				"__id":      llx.StringData("sharingPolicy-" + p.Identity),
				"identity":  llx.StringData(p.Identity),
				"name":      llx.StringData(p.Name),
				"enabled":   llx.BoolData(p.Enabled),
				"isDefault": llx.BoolData(p.Default),
				"domains":   llx.ArrayData(llx.TArr2Raw(p.Domains), types.String),
			})
		if err != nil {
			return nil, err
		}
		result = append(result, mql)
	}
	return result, nil
}

// --- Role assignment policies ---

type ExchangeRoleAssignmentPolicy struct {
	Identity      string   `json:"Identity"`
	Name          string   `json:"Name"`
	IsDefault     bool     `json:"IsDefault"`
	Description   string   `json:"Description"`
	AssignedRoles []string `json:"AssignedRoles"`
}

func convertRoleAssignmentPolicies(r *mqlMs365Exchangeonline, raw any) ([]any, error) {
	policies, err := decodeExchangeList[ExchangeRoleAssignmentPolicy](raw)
	if err != nil {
		return nil, err
	}
	result := []any{}
	for _, p := range policies {
		if p == nil {
			continue
		}
		mql, err := CreateResource(r.MqlRuntime, "ms365.exchangeonline.roleAssignmentPolicyEntry",
			map[string]*llx.RawData{
				"__id":          llx.StringData("roleAssignmentPolicy-" + p.Identity),
				"identity":      llx.StringData(p.Identity),
				"name":          llx.StringData(p.Name),
				"isDefault":     llx.BoolData(p.IsDefault),
				"description":   llx.StringData(p.Description),
				"assignedRoles": llx.ArrayData(llx.TArr2Raw(p.AssignedRoles), types.String),
			})
		if err != nil {
			return nil, err
		}
		result = append(result, mql)
	}
	return result, nil
}

// --- Resource accessors (plural field name, singular element resource) ---

func (r *mqlMs365Exchangeonline) transportRules() ([]any, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) antiPhishPolicies() ([]any, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) safeLinksPolicies() ([]any, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) safeAttachmentPolicies() ([]any, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) malwareFilterPolicies() ([]any, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) hostedContentFilterPolicies() ([]any, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) hostedOutboundSpamFilterPolicies() ([]any, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) dkimSigningConfigs() ([]any, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) authenticationPolicies() ([]any, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) owaMailboxPolicies() ([]any, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) remoteDomains() ([]any, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) quarantinePolicies() ([]any, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) atpPoliciesForO365() ([]any, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) sharingPolicies() ([]any, error) {
	return nil, r.getExchangeReport()
}

func (r *mqlMs365Exchangeonline) roleAssignmentPolicies() ([]any, error) {
	return nil, r.getExchangeReport()
}
