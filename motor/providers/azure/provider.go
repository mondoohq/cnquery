package azure

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
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

func (p *Provider) GetTokenCredential() (azcore.TokenCredential, error) {
	var credential azcore.TokenCredential
	var err error

	// fallback to CLI authorizer if no credentials are specified
	if p.credential == nil {
		log.Debug().Msg("using azure cli to get authorizer")
		credential, err = azidentity.NewAzureCLICredential(&azidentity.AzureCLICredentialOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "error creating cli credentials")
		}
	} else {
		// we only support private key authentication for ms 365
		switch p.credential.Type {
		case vault.CredentialType_pkcs12:
			certs, privateKey, err := azidentity.ParseCertificates(p.credential.Secret, []byte(p.credential.Password))
			if err != nil {
				return nil, errors.Wrap(err, "could not parse pfx file")
			}

			credential, err = azidentity.NewClientCertificateCredential(p.tenantID, p.clientID, certs, privateKey, &azidentity.ClientCertificateCredentialOptions{})
			if err != nil {
				return nil, errors.Wrap(err, "error creating credentials")
			}
		case vault.CredentialType_password:
			credential, err = azidentity.NewClientSecretCredential(p.tenantID, p.clientID, string(p.credential.Secret), &azidentity.ClientSecretCredentialOptions{})
			if err != nil {
				return nil, errors.Wrap(err, "error creating credentials")
			}
		default:
			return nil, errors.New("invalid secret configuration for ms365 transport: " + p.credential.Type.String())
		}
	}
	return credential, nil
}

func (p *Provider) ParseResourceID(id string) (*ResourceID, error) {
	return ParseResourceID(id)
}
