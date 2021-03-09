package mockvault

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/vault"
)

var notImplemented = errors.New("not implemented")

func New() *Vault {
	return &Vault{}
}

type Vault struct{}

const (
	MockPassword = "password"
	MockPKey     = "BEGIN_PRIVATE_KEY...."
	MockJson     = `{"backend": "ssh","private_key":"blabla","user":"that-user","password": "supersecure"}`
)

func (v *Vault) Get(ctx context.Context, id *vault.CredentialID) (*vault.Credential, error) {
	log.Debug().Msgf("getting cred from mock vault %s", id.Key)
	switch id.Key {
	case "mockPassword":
		return &vault.Credential{
			Key:    id.Key,
			Secret: MockPassword,
		}, nil
	case "mockPKey":
		return &vault.Credential{
			Key:    id.Key,
			Secret: MockPKey,
		}, nil
	case "mockJson":
		return &vault.Credential{
			Key:    id.Key,
			Secret: MockJson,
		}, nil
	}
	return nil, nil
}

func (v *Vault) Set(ctx context.Context, cred *vault.Credential) (*vault.CredentialID, error) {
	return nil, errors.New("not implemented")
}
