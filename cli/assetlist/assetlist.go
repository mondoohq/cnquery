package assetlist

import (
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/cli/theme"
	"go.mondoo.com/cnquery/motor/asset"
)

func NewSimpleRenderer(theme *theme.Theme) *simpleRender {
	return &simpleRender{
		theme: theme,
	}
}

type simpleRender struct {
	theme *theme.Theme
}

func (a *simpleRender) Render(assetList []*asset.Asset) string {
	b := strings.Builder{}

	log.Info().Msgf("discovered %d asset(s)", len(assetList))

	for i := range assetList {
		assetObj := assetList[i]

		b.WriteString(a.theme.Primary("name:\t\t"))
		b.WriteString(assetObj.HumanName())
		b.WriteRune('\n')

		if len(assetObj.PlatformIds) > 0 {
			b.WriteString(a.theme.Primary("platform-id:\t"))
			for j := range assetObj.PlatformIds {
				b.WriteString("  " + assetObj.PlatformIds[j])
			}
		}

		b.WriteRune('\n')
		b.WriteRune('\n')
	}

	return b.String()
}
