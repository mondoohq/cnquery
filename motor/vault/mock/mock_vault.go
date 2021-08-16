package mockvault

import (
	"context"

	"go.mondoo.io/mondoo/motor/transports"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/vault"
)

var notImplemented = errors.New("not implemented")

func New() *Vault {
	return &Vault{}
}

type Vault struct{}

var (
	MockPassword           *vault.Secret
	MockPKey               *vault.Secret
	MockPrivateKeyPassword *vault.Secret
)

func init() {
	MockPassword, _ = vault.NewSecret(&transports.Credential{
		Type:   transports.CredentialType_password,
		Secret: []byte("password"),
	}, vault.SecretEncoding_PROTO)
	MockPKey, _ = vault.NewSecret(&transports.Credential{
		Type:   transports.CredentialType_private_key,
		Secret: []byte("BEGIN_PRIVATE_KEY...."),
	}, vault.SecretEncoding_PROTO)
	MockPrivateKeyPassword, _ = vault.NewSecret(&transports.Credential{
		Type:     transports.CredentialType_private_key,
		User:     "that-user",
		Secret:   []byte("blabla"),
		Password: "supersecure",
	}, vault.SecretEncoding_PROTO)
}

func (v *Vault) Get(ctx context.Context, id *vault.SecretID) (*vault.Secret, error) {
	log.Debug().Msgf("getting cred from mock vault %s", id.Key)
	switch id.Key {
	case "mockPassword":
		return MockPassword, nil
	case "mockPKey":
		return MockPKey, nil
	case "mockPKeyPassword":
		return MockPrivateKeyPassword, nil
	}
	return nil, vault.NotFoundError
}

func (v *Vault) Set(ctx context.Context, cred *vault.Secret) (*vault.SecretID, error) {
	return nil, errors.New("not implemented")
}
