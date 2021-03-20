package exchangeonline

type ExchangeOnlineExportReport struct {
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
}
