package common

import (
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/transports"
)

type (
	// CredentialFn retrieves the credentials to connect to the platform
	CredentialFn func(secretId string) (*transports.Credential, error)
	// QuerySecretFn is used during discovery phase to identify a secret for an asset
	QuerySecretFn func(a *asset.Asset) (*transports.Credential, error)
)
