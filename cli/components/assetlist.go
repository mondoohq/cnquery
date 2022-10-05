package components

import (
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/cli/theme"
	"go.mondoo.com/cnquery/motor/asset"
)

func AssetList(theme *theme.Theme, assetList []*asset.Asset) string {
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
