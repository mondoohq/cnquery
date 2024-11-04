// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package components

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog/log"
)

// Selector is the interface that items need to implement so that we can select them.
type Selector interface {
	HumanName() string
}

// Select is an interactive prompt that displays the provided message and displays a
// list of items to be selected.
//
// e.g.
// ```go
//
//	type CustomString string
//
//	func (s CustomString) HumanName() string {
//		return string(s)
//	}
//
//	func main() {
//		customStrings := []CustomString{"first", "second", "third"}
//		selected := components.Select("Choose a string", customStrings)
//		fmt.Printf("You chose the %s string.\n", customStrings[selected])
//	}
//
// ```
func Select[S Selector](msg string, items []S) int {
	list := make([]string, len(items))

	for i := range items {
		list[i] = items[i].HumanName()
	}

	selection := -1 // make sure we have an invalid index
	model := NewListModel(msg, list, func(s int) {
		selection = s
	})
	_, err := tea.NewProgram(model, tea.WithInputTTY()).Run()
	if err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	if selection == -1 {
		return -1
	}
	selected := items[selection]
	log.Debug().
		Int("selection", selection).
		Str("asset", selected.HumanName()).
		Msg("selected")
	return selection
}
