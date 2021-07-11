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

func (v *Vault) Get(ctx context.Context, id *vault.SecretID) (*vault.Secret, error) {
	log.Debug().Msgf("getting cred from mock vault %s", id.Key)
	switch id.Key {
	case "mockPassword":
		return &vault.Secret{
			Key:    id.Key,
			Secret: []byte(MockPassword),
		}, nil
	case "mockPKey":
		return &vault.Secret{
			Key:    id.Key,
			Secret: []byte(MockPKey),
		}, nil
	case "mockJson":
		return &vault.Secret{
			Key:    id.Key,
			Secret: []byte(MockJson),
		}, nil
	}
	return nil, nil
}

func (v *Vault) Set(ctx context.Context, cred *vault.Secret) (*vault.SecretID, error) {
	return nil, errors.New("not implemented")
}
