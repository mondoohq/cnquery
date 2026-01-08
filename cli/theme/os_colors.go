// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows

package theme

import (
	"strings"

	"github.com/muesli/termenv"
	"go.mondoo.com/cnquery/v12/cli/printer"
	"go.mondoo.com/cnquery/v12/cli/theme/colors"
)

// OperatingSystemTheme for unix shell
var OperatingSystemTheme = &Theme{
	Colors: colors.DefaultColorTheme,
	List: func(items ...string) string {
		var w strings.Builder
		for i := range items {
			w.WriteString("- " + items[i] + "\n")
		}
		res := w.String()
		return res[0 : len(res)-1]
	},
	Landing:       termenv.String(logo).Foreground(colors.DefaultColorTheme.Primary).String(),
	Welcome:       logo + " interactive shell\n",
	Prefix:        "cnquery> ",
	PolicyPrinter: printer.DefaultPrinter,
}
