package ms365

func (p *Provider) Identifier() (string, error) {
	return "//platformid.api.mondoo.app/runtime/ms365/tenant/" + p.tenantID, nil
}

func (p *Provider) TenantID() string {
	return p.tenantID
}
