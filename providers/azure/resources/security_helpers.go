// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

// newKeyVaultKeyResource creates a typed azure.subscription.keyVaultService.key
// reference from a Key Vault key URI (e.g. https://myvault.vault.azure.net/keys/mykey/version).
// Returns nil resource if keyURI is empty.
func newKeyVaultKeyResource(runtime *plugin.Runtime, keyURI string) (*mqlAzureSubscriptionKeyVaultServiceKey, error) {
	if keyURI == "" {
		return nil, nil
	}

	// Use NewResource so that if the key is already cached it gets reused.
	// The KID field is the canonical identifier for key vault keys.
	mqlKey, err := NewResource(runtime, "azure.subscription.keyVaultService.key",
		map[string]*llx.RawData{
			"kid":     llx.StringData(keyURI),
			"managed": llx.BoolData(false),
			"tags":    llx.MapData(map[string]interface{}{}, types.String),
		})
	if err != nil {
		return nil, err
	}
	return mqlKey.(*mqlAzureSubscriptionKeyVaultServiceKey), nil
}

// newKeyVaultSecretResource creates a typed azure.subscription.keyVaultService.secret
// reference from a Key Vault secret URI (e.g. https://myvault.vault.azure.net/secrets/mysecret/version).
// Returns nil resource if secretURI is empty.
func newKeyVaultSecretResource(runtime *plugin.Runtime, secretURI string) (*mqlAzureSubscriptionKeyVaultServiceSecret, error) {
	if secretURI == "" {
		return nil, nil
	}

	// Use NewResource so that if the secret is already cached it gets reused.
	// The id field (the secret URI) is the canonical identifier for key vault secrets.
	mqlSecret, err := NewResource(runtime, "azure.subscription.keyVaultService.secret",
		map[string]*llx.RawData{
			"id":      llx.StringData(secretURI),
			"managed": llx.BoolData(false),
			"tags":    llx.MapData(map[string]interface{}{}, types.String),
		})
	if err != nil {
		return nil, err
	}
	return mqlSecret.(*mqlAzureSubscriptionKeyVaultServiceSecret), nil
}
