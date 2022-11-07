package azure

import (
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/vault"
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

	clientId := cfg.Options["client-id"]
	tenantId := cfg.Options["tenant-id"]
	subscriptionId := cfg.Options["subscription-id"]
	var cred *vault.Credential
	if len(cfg.Credentials) != 0 {
		cred = cfg.Credentials[0]
	}

	// deprecated options for backward compatibility with older inventory files
	if subscriptionId == "" {
		sid, ok := cfg.Options["subscriptionId"]
		if ok {
			log.Warn().Str("subscriptionId", sid).Msg("subscriptionId is deprecated, use subscription-id instead")
		}
		subscriptionId = sid
	}
	if tenantId == "" {
		tid, ok := cfg.Options["tenantId"]
		if ok {
			log.Warn().Str("tenantId", tid).Msg("tenantId is deprecated, use tenant-id instead")
		}
		tenantId = tid
	}

	if clientId == "" && cred != nil {
		return nil, errors.New("azure provider requires client id besides credentials")
	}

	return &Provider{
		clientID:       clientId,
		subscriptionID: subscriptionId,
		tenantID:       tenantId,
		opts:           cfg.Options,
		credential:     cred,
	}, nil
}

type Provider struct {
	clientID       string
	subscriptionID string
	tenantID       string
	opts           map[string]string
	credential     *vault.Credential
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

// GetAuthorizer determines what authorizer to use, based on the passed in configs. Possible options are:
// - Authorizer via the CLI that uses the `az` executable
// - Authorizer via password that uses the sdk
// - Authorizer via certificate that uses the sdk
func getAuthorizer(clientId, tenantId, resource string, credential *vault.Credential) (autorest.Authorizer, error) {
	var authorizer autorest.Authorizer
	var err error

	// fallback to CLI authorizer if no credentials are specified
	if credential == nil {
		log.Debug().Msg("using azure cli to get authorizer")
		if resource != "" {
			return auth.NewAuthorizerFromCLIWithResource(resource)
		}
		return auth.NewAuthorizerFromCLI()
	}

	switch credential.Type {
	case vault.CredentialType_password:
		config := auth.NewClientCredentialsConfig(clientId, string(credential.Secret), tenantId)
		if resource != "" {
			config.Resource = resource
		}
		authorizer, err = config.Authorizer()
		if err != nil {
			return nil, errors.Wrap(err, "error creating credentials from secret")
		}
	case vault.CredentialType_pkcs12:
		config := auth.NewClientCertificateConfig(credential.PrivateKeyPath, credential.Password, clientId, tenantId)
		if resource != "" {
			config.Resource = resource
		}
		authorizer, err = config.Authorizer()
		if err != nil {
			return nil, errors.Wrap(err, "error creating credentials from certificate")
		}
	default:
		return nil, errors.New("invalid secret configuration for azure transport: " + credential.Type.String())
	}

	return authorizer, nil
}

func (p *Provider) Authorizer() (autorest.Authorizer, error) {
	return getAuthorizer(p.clientID, p.tenantID, "", p.credential)
}

func (p *Provider) AuthorizerWithAudience(audience string) (autorest.Authorizer, error) {
	return getAuthorizer(p.clientID, p.tenantID, audience, p.credential)
}

func (p *Provider) ParseResourceID(id string) (*ResourceID, error) {
	return ParseResourceID(id)
}
