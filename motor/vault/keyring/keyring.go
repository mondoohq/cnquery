package keyring

import (
	"context"
	"encoding/json"

	"github.com/99designs/keyring"
	"go.mondoo.io/mondoo/motor/vault"
)

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

	// we json-encode the value, while proto would be more efficient, json allows humans to read the data more easily
	data, err := json.Marshal(cred.Fields)
	if err != nil {
		return nil, err
	}

	// TODO: store data as json encoding
	err = ring.Set(keyring.Item{
		Key:   cred.Key,
		Label: cred.Label,
		Data:  data,
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

	var fields map[string]string
	err = json.Unmarshal(i.Data, &fields)
	if err != nil {
		return nil, err
	}

	return &vault.Credential{
		Key:    i.Key,
		Label:  i.Label,
		Fields: fields,
	}, nil
}

func (v *Vault) Delete(ctx context.Context, id *vault.CredentialID) (*vault.CredentialDeletedResp, error) {
	ring, err := v.open()
	if err != nil {
		return nil, err
	}

	err = ring.Remove(id.Key)
	return &vault.CredentialDeletedResp{}, err
}
