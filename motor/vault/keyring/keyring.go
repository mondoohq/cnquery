package keyring

import (
	"context"
	"errors"

	"github.com/99designs/keyring"
	"go.mondoo.io/mondoo/motor/vault"
)

var notImplemented = errors.New("not implemented")

func New(serviceName string) *Vault {
	return &Vault{
		ServiceName: serviceName,
	}
}

type Vault struct {
	ServiceName string
}

func (v *Vault) open() (keyring.Keyring, error) {
	return keyring.Open(keyring.Config{
		ServiceName: v.ServiceName,
		FileDir:     "~/.mondoo/",
		FilePasswordFunc: func(s string) (string, error) {
			// TODO: this only applies to cases where we have no real keychain available
			// we need to find a better way to manage this, maybe this is going to land in
			// the mondoo configuration
			return "random", nil
		},
	})
}

func (v *Vault) Set(ctx context.Context, cred *vault.Credential) (*vault.CredentialID, error) {
	ring, err := v.open()
	if err != nil {
		return nil, err
	}

	// TODO: store data as json encoding
	err = ring.Set(keyring.Item{
		Key:   cred.Key,
		Label: cred.Label,
		Data:  []byte(cred.Secret),
	})

	return &vault.CredentialID{
		Key: cred.Key,
	}, err
}

func (v *Vault) Get(ctx context.Context, id *vault.CredentialID) (*vault.Credential, error) {
	ring, err := v.open()
	if err != nil {
		return nil, err
	}

	i, err := ring.Get(id.Key)
	if err != nil {
		return nil, err
	}

	return &vault.Credential{
		Key:    i.Key,
		Label:  i.Label,
		Secret: string(i.Data),
	}, nil
}
