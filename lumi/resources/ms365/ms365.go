package ms365

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

type SharepointOnlineReport struct {
	SPOTenant                      interface{} `json:"SPOTenant"`
	SPOTenantSyncClientRestriction interface{} `json:"SPOTenantSyncClientRestriction"`
}

type MsTeamsReport struct {
	CsTeamsClientConfiguration interface{}   `json:"CsTeamsClientConfiguration"`
	CsOAuthConfiguration       []interface{} `json:"CsOAuthConfiguration"`
}

type Microsoft365Report struct {
	ExchangeOnline   ExchangeOnlineReport   `json:"ExchangeOnline"`
	SharepointOnline SharepointOnlineReport `json:"SharepointOnline"`
	MsTeams          MsTeamsReport          `json:"MsTeams"`
}
