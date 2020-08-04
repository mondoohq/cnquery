package azure

func (t *Transport) Identifier() (string, error) {
	return "//platformid.api.mondoo.app/runtime/azure/subscription/" + t.subscriptionID, nil
}

func (t *Transport) SubscriptionID() string {
	return t.subscriptionID
}

func (t *Transport) TenantID() string {
	return t.tenantID
}
