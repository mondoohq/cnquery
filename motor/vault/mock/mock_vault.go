package mockvault

import (
	"context"

	"errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/vault"
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
	MockPassword, _ = vault.NewSecret(&vault.Credential{
		Type:   vault.CredentialType_password,
		Secret: []byte("password"),
	}, vault.SecretEncoding_encoding_proto)
	MockPKey, _ = vault.NewSecret(&vault.Credential{
		Type:   vault.CredentialType_private_key,
		Secret: []byte("BEGIN_PRIVATE_KEY...."),
	}, vault.SecretEncoding_encoding_proto)
	MockPrivateKeyPassword, _ = vault.NewSecret(&vault.Credential{
		Type:     vault.CredentialType_private_key,
		User:     "that-user",
		Secret:   []byte("blabla"),
		Password: "supersecure",
	}, vault.SecretEncoding_encoding_proto)
}

func (v *Vault) About(context.Context, *vault.Empty) (*vault.VaultInfo, error) {
	return &vault.VaultInfo{Name: "Mock Vault"}, nil
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
