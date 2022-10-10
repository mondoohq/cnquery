package azure

import (
	"errors"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/providers"
)

var (
	_ providers.Instance           = (*Provider)(nil)
	_ providers.PlatformIdentifier = (*Provider)(nil)
)

func New(cfg *providers.Config) (*Provider, error) {
	if cfg.Backend != providers.ProviderType_AZURE {
		return nil, providers.ErrProviderTypeDoesNotMatch
	}
	if cfg.Options == nil {
		return nil, errors.New("azure provider requires options")
	}

	tenantId := cfg.Options["tenant-id"]

	// deprecated options for backward compatibility with older inventory files
	if tenantId == "" {
		tid, ok := cfg.Options["tenantId"]
		if ok {
			log.Warn().Str("tenantId", tid).Msg("tenantId is deprecated, use tenant-id instead")
		}
		tenantId = tid
	}

	subscriptionId := cfg.Options["subscription-id"]
	if subscriptionId == "" {
		sid, ok := cfg.Options["subscriptionId"]
		if ok {
			log.Warn().Str("subscriptionId", sid).Msg("subscriptionId is deprecated, use subscription-id instead")
		}
		subscriptionId = sid
	}

	if subscriptionId == "" {
		return nil, errors.New("azure provider requires a subscriptionID")
	}

	if tenantId == "" {
		return nil, errors.New("azure provider requires a tenantID")
	}

	return &Provider{
		subscriptionID: subscriptionId,
		tenantID:       tenantId,
		opts:           cfg.Options,
	}, nil
}

type Provider struct {
	subscriptionID string
	tenantID       string
	opts           map[string]string
}

func (p *Provider) Close() {}

func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		providers.Capability_Azure,
	}
}

func (p *Provider) Options() map[string]string {
	return p.opts
}

func (p *Provider) Kind() providers.Kind {
	return providers.Kind_KIND_API
}

func (p *Provider) Runtime() string {
	return providers.RUNTIME_AZ
}

func (p *Provider) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}

func GetAuthorizer() (autorest.Authorizer, error) {
	return auth.NewAuthorizerFromCLI()
}

func (p *Provider) Authorizer() (autorest.Authorizer, error) {
	return GetAuthorizer()
}

func (p *Provider) AuthorizerWithAudience(audience string) (autorest.Authorizer, error) {
	return auth.NewAuthorizerFromCLIWithResource(audience)
}

func (p *Provider) ParseResourceID(id string) (*ResourceID, error) {
	return ParseResourceID(id)
}
