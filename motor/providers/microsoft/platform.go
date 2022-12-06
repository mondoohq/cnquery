package microsoft

import (
	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/vault"
)

func (p *Provider) Identifier() (string, error) {
	if p.assetType == azure {
		return "//platformid.api.mondoo.app/runtime/azure/subscriptions/" + p.subscriptionID, nil
	}
	return "//platformid.api.mondoo.app/runtime/ms365/tenant/" + p.tenantID, nil
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
	return p.cred
}

func (p *Provider) Roles() map[string]struct{} {
	return p.rolesMap
}

func (p *Provider) PlatformInfo() (*platform.Platform, error) {
	switch p.assetType {
	case azure:
		return &platform.Platform{
			Name:    "azure",
			Title:   "Microsoft Azure",
			Kind:    providers.Kind_KIND_API,
			Runtime: providers.RUNTIME_AZ,
		}, nil
	case ms365:
		return &platform.Platform{
			Name:    "microsoft365",
			Title:   "Microsoft 365",
			Kind:    providers.Kind_KIND_API,
			Runtime: providers.RUNTIME_MICROSOFT_GRAPH,
		}, nil
	default:
		return nil, errors.New("unknown microsoft asset type")
	}
}
