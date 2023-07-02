package microsoft

import (
	"fmt"

	"errors"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
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
			return nil, errors.Join(err, errors.New("error creating cli credentials"))
		}
	} else {
		// we only support private key authentication for ms 365
		switch p.cred.Type {
		case vault.CredentialType_pkcs12:
			certs, privateKey, err := azidentity.ParseCertificates(p.cred.Secret, []byte(p.cred.Password))
			if err != nil {
				return nil, errors.Join(err, errors.New(fmt.Sprintf("could not parse provided certificate at %s", p.cred.PrivateKeyPath)))
			}
			credential, err = azidentity.NewClientCertificateCredential(p.tenantID, p.clientID, certs, privateKey, &azidentity.ClientCertificateCredentialOptions{})
			if err != nil {
				return nil, errors.Join(err, errors.New("error creating credentials from a certificate"))
			}
		case vault.CredentialType_password:
			credential, err = azidentity.NewClientSecretCredential(p.tenantID, p.clientID, string(p.cred.Secret), &azidentity.ClientSecretCredentialOptions{})
			if err != nil {
				return nil, errors.Join(err, errors.New("error creating credentials from a secret"))
			}
		default:
			return nil, errors.New("invalid secret configuration for microsoft transport: " + p.cred.Type.String())
		}
	}
	return credential, nil
}
