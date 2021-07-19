package inmemory

import (
	"context"
	"errors"

	"go.mondoo.io/mondoo/motor/vault"
)

type Option func(*inmemoryVault)

func WithSecretMap(secrets map[string]*vault.Secret) Option {
	return func(iv *inmemoryVault) {
		iv.secrets = secrets
	}
}

func New(opts ...Option) *inmemoryVault {
	iv := &inmemoryVault{
		secrets: map[string]*vault.Secret{},
	}

	for _, option := range opts {
		option(iv)
	}

	return iv
}

type inmemoryVault struct {
	secrets map[string]*vault.Secret
}

func (v *inmemoryVault) Set(ctx context.Context, secret *vault.Secret) (*vault.SecretID, error) {
	if secret == nil {
		return nil, errors.New("secret is empty")
	}
	v.secrets[secret.Key] = secret
	return &vault.SecretID{Key: secret.Key}, nil
}

func (v *inmemoryVault) Get(ctx context.Context, id *vault.SecretID) (*vault.Secret, error) {
	if id == nil {
		return nil, errors.New("secret id is empty")
	}

	s, ok := v.secrets[id.Key]
	if !ok {
		return nil, errors.New("secret not found: " + id.Key)
	}
	return s, nil
}
