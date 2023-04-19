package microsoft

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/vault"
)

func (p *Provider) GetTokenCredential() (azcore.TokenCredential, error) {
	var credential azcore.TokenCredential
	var err error

	// fallback to CLI authorizer if no credentials are specified
	if p.cred == nil {
		log.Debug().Msg("using azure cli to get authorizer")
		credential, err = azidentity.NewAzureCLICredential(&azidentity.AzureCLICredentialOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "error creating cli credentials")
		}
	} else {
		// we only support private key authentication for ms 365
		switch p.cred.Type {
		case vault.CredentialType_pkcs12:
			certs, privateKey, err := azidentity.ParseCertificates(p.cred.Secret, []byte(p.cred.Password))
			if err != nil {
				return nil, errors.Wrap(err, "could not parse provided certificate")
			}

			credential, err = azidentity.NewClientCertificateCredential(p.tenantID, p.clientID, certs, privateKey, &azidentity.ClientCertificateCredentialOptions{})
			if err != nil {
				return nil, errors.Wrap(err, "error creating credentials")
			}
		case vault.CredentialType_password:
			credential, err = azidentity.NewClientSecretCredential(p.tenantID, p.clientID, string(p.cred.Secret), &azidentity.ClientSecretCredentialOptions{})
			if err != nil {
				return nil, errors.Wrap(err, "error creating credentials")
			}
		default:
			return nil, errors.New("invalid secret configuration for microsoft transport: " + p.cred.Type.String())
		}
	}
	return credential, nil
}
