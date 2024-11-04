// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package components

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/cli/theme"
)

// Object is the interface that a list need to implement so we can display its objects.
type Object interface {
	// PrintableKeys returns the list of keys that will be printed.
	PrintableKeys() []string

	// PrintableValue returns the key value based of the provided index.
	PrintableValue(index int) string
}

// List is a non-interactive function that lists objects to the user.
//
// e.g.
// ```go
//
//	type CustomString string
//
//	func (s CustomString) PrintableKeys() []string {
//		return []string{"string"}
//	}
//	func (s CustomString) PrintableValue(_ int) string {
//		return string(s)
//	}
//
//	func main() {
//		customStrings := []CustomString{"first", "second", "third"}
//		list := components.List(theme.OperatingSystemTheme, "string", customStrings)
//		fmt.Printf(list)
//	}
//
// ```
func List[O Object](theme *theme.Theme, listType string, list []O) string {
	b := &strings.Builder{}
	w := tabwriter.NewWriter(b, 1, 1, 1, ' ', tabwriter.TabIndent)

	log.Debug().Msgf("discovered %d %s(s)", len(list), listType)

	for i := range list {
		assetObj := list[i]

		for i, key := range assetObj.PrintableKeys() {
			fmt.Fprint(w, theme.Primary(key, ":\t"))
			fmt.Fprintln(w, assetObj.PrintableValue(i))
		}

		fmt.Fprintln(w, "")
	}

	w.Flush()

	return b.String()
}
