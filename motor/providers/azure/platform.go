package azure

func (t *Provider) Identifier() (string, error) {
	return "//platformid.api.mondoo.app/runtime/azure/subscriptions/" + t.subscriptionID, nil
}

func (t *Provider) SubscriptionID() string {
	return t.subscriptionID
}

func (t *Provider) TenantID() string {
	return t.tenantID
}
