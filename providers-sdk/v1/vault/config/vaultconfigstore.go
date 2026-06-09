// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"encoding/json"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"

	// The in-memory vault has no heavy dependencies and stays in the SDK, so we
	// register it here to guarantee that the memory backend is always available
	// on the factory path. Heavy backends (AWS, GCP, HashiCorp, keyring) live in
	// the separate go.mondoo.com/mql/v13/vault module and are opted into by the
	// binary (see vault/register).
	_ "go.mondoo.com/mql/v13/providers-sdk/v1/vault/inmemory"
)

// New instantiates a vault from its configuration. The concrete implementation
// is resolved through the vault registry; the corresponding implementation
// package must be linked into the binary for its VaultType to be available.
func New(vCfg *vault.VaultConfiguration) (vault.Vault, error) {
	return vault.New(vCfg)
}

// ClientVaultConfig is the structured type where we store the client configuration for
// all user configured vaults. We use it to ensure the configuration is stored in structured
// format
type ClientVaultConfig map[string]vault.VaultConfiguration

func NewClientVaultConfig(secret *vault.Secret) (ClientVaultConfig, error) {
	var vCfg ClientVaultConfig
	err := json.Unmarshal(secret.Data, &vCfg)
	if err != nil {
		return nil, errors.Wrap(err, "corrupt vault configuration")
	}
	return vCfg, nil
}

func (avc ClientVaultConfig) Delete(key string) {
	delete(avc, key)
}

func (avc ClientVaultConfig) Set(key string, cfg vault.VaultConfiguration) {
	avc[key] = cfg
}

func (avc ClientVaultConfig) Get(key string) (vault.VaultConfiguration, error) {
	vCfg, ok := avc[key]
	if !ok {
		return vault.VaultConfiguration{}, errors.New("vault not found")
	}
	return vCfg, nil
}

// SecretData returns the marshaled data, it is compatible with New()
// In case the data structure cannot be marshalled, the function will panic
func (avc ClientVaultConfig) SecretData() []byte {
	data, err := json.Marshal(avc)
	if err != nil {
		panic(err)
	}
	return data
}
