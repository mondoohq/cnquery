package multivault

import (
	"context"
	"errors"

	"go.mondoo.io/mondoo/motor/vault"
)

// New creates a new vault that can query multiple vaults. It is a
// read-only vault that allows the user to have a unified view for secrets
// located in e.g. and inmemory vault and in e.g. a keyring vault. The order
// matters. When a secret is requested, this implementation iterates over
// each vault and returns the first secret with the requested id
func New(vaults ...vault.Vault) *multiVault {
	return &multiVault{
		vaults: vaults,
	}
}

type multiVault struct {
	vaults []vault.Vault
}

func (m *multiVault) Set(ctx context.Context, secret *vault.Secret) (*vault.SecretID, error) {
	return nil, errors.New("this vault is read only")
}

func (m *multiVault) Get(ctx context.Context, id *vault.SecretID) (*vault.Secret, error) {
	if id == nil {
		return nil, errors.New("secret id is empty")
	}

	// iterate over each vault and return the first finding
	for i := range m.vaults {
		v := m.vaults[i]
		secret, err := v.Get(ctx, id)
		if err == nil {
			return secret, nil
		}
	}

	return nil, errors.New("secret not found")
}
