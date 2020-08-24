package ms365

func (t *Transport) Identifier() (string, error) {
	return "//platformid.api.mondoo.app/runtime/ms365/tenant/" + t.tenantID, nil
}

func (t *Transport) TenantID() string {
	return t.tenantID
}
