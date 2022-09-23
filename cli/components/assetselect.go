package components

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
)

func AssetSelect(assetList []*asset.Asset) *asset.Asset {
	list := make([]string, len(assetList))

	// map asset name to list
	for i := range assetList {
		a := assetList[i]
		name := a.Name
		if a.Platform != nil {
			name = fmt.Sprintf("%s (%s)", a.Name, a.Platform.Name)
		}
		list[i] = name
	}

	var selection int
	err := tea.NewProgram(NewListModel("Available assets", list, func(s int) {
		selection = s
	})).Start()
	if err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	selected := assetList[selection]
	log.Info().Int("selection", selection).Str("asset", selected.Name).Msg("selected asset")
	return selected
}
