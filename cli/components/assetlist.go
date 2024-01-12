// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package components

import (
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/cli/theme"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
)

func AssetList(theme *theme.Theme, assetList []*inventory.Asset) string {
	b := strings.Builder{}

	log.Info().Msgf("discovered %d asset(s)", len(assetList))

	for i := range assetList {
		assetObj := assetList[i]

		b.WriteString(theme.Primary("name:\t\t"))
		b.WriteString(assetObj.HumanName())
		b.WriteRune('\n')

		if len(assetObj.PlatformIds) > 0 {
			b.WriteString(theme.Primary("platform-id:\t"))
			for j := range assetObj.PlatformIds {
				b.WriteString("  " + assetObj.PlatformIds[j])
			}
		}

		b.WriteRune('\n')
		b.WriteRune('\n')
	}

	return b.String()
}
