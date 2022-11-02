package azure

import "go.mondoo.com/cnquery/motor/vault"

func (p *Provider) Identifier() (string, error) {
	return "//platformid.api.mondoo.app/runtime/azure/subscriptions/" + p.subscriptionID, nil
}

func (p *Provider) SubscriptionID() string {
	return p.subscriptionID
}

func (p *Provider) TenantID() string {
	return p.tenantID
}

func (p *Provider) ClientID() string {
	return p.clientID
}

func (p *Provider) Credential() *vault.Credential {
	return p.credential
}
