// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package components

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
)

func AssetSelect(assetList []*inventory.Asset) *inventory.Asset {
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

	selection := -1 // make sure we have an invalid index
	model := NewListModel("Available assets", list, func(s int) {
		selection = s
	})
	_, err := tea.NewProgram(model, tea.WithInputTTY()).Run()
	if err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	if selection == -1 {
		return nil
	}
	selected := assetList[selection]
	log.Info().Int("selection", selection).Str("asset", selected.Name).Msg("selected asset")
	return selected
}
