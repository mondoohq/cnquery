// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package theme

import (
	"strings"

	"github.com/muesli/termenv"
	"go.mondoo.com/cnquery/v12/cli/printer"
	"go.mondoo.com/cnquery/v12/cli/theme/colors"
)

// OperatingSystemTheme for windows shell
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
	Landing: termenv.String("cnquery™\n" + logo + "\n").Foreground(colors.DefaultColorTheme.Primary).String(),
	Welcome: "cnquery™\n" + logo + " interactive shell\n",
	// NOTE: this is important to be short for windows, otherwise the auto-complete will make strange be jumps
	// ENSURE YOU TEST A CHANGE BEFORE COMMIT ON WINDOWS
	Prefix:        "> ",
	PolicyPrinter: printer.DefaultPrinter,
}
