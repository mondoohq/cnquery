// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package common

import (
	"github.com/rs/zerolog/log"
	inventory "go.mondoo.com/cnquery/motor/inventory/v1"
	"go.mondoo.com/cnquery/motor/vault"
)

type (
	// QuerySecretFn is used during discovery phase to identify a secret for an asset
	QuerySecretFn func(a *inventory.Asset) (*vault.Credential, error)
)

func EnrichAssetWithSecrets(a *inventory.Asset, sfn QuerySecretFn) {
	for j := range a.Connections {
		conn := a.Connections[j]

		// NOTE: for now we only add credentials for ssh, we may revisit that in the future
		if len(conn.Credentials) == 0 && conn.Type == "ssh" {
			creds, err := sfn(a)
			if err == nil && creds != nil {
				conn.Credentials = []*vault.Credential{creds}
			} else {
				log.Warn().Str("name", a.Name).Msg("could not determine credentials for asset")
			}
		}
	}
}
