package ms365

func (t *Provider) Identifier() (string, error) {
	return "//platformid.api.mondoo.app/runtime/ms365/tenant/" + t.tenantID, nil
}

func (t *Provider) TenantID() string {
	return t.tenantID
}
