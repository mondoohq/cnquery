// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cache

import (
	"context"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/vault"
)

type cachedVault struct {
	secrets map[string]*vault.Secret
	vault   vault.Vault
}

func New(v vault.Vault) vault.Vault {
	return &cachedVault{
		secrets: map[string]*vault.Secret{},
		vault:   v,
	}
}

func (c *cachedVault) About(ctx context.Context, e *vault.Empty) (*vault.VaultInfo, error) {
	// return the info about the underlying vault. The cached vault is only an abstraction
	return c.vault.About(ctx, e)
}

func (c *cachedVault) Get(ctx context.Context, id *vault.SecretID) (*vault.Secret, error) {
	if s, ok := c.secrets[id.Key]; ok {
		return s, nil
	}
	s, err := c.vault.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	c.secrets[id.Key] = s
	return s, nil
}

func (c *cachedVault) Set(ctx context.Context, s *vault.Secret) (*vault.SecretID, error) {
	return c.vault.Set(ctx, s)
}
