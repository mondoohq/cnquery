// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package shell

import (
	"strings"

	"go.mondoo.com/cnquery/v12/cli/theme"
	"go.mondoo.com/cnquery/v12/llx"
)

func formatSuggestions(suggestions []*llx.Documentation, theme *theme.Theme) string {
	var res strings.Builder
	res.WriteString(theme.Secondary("\nsuggestions: \n"))
	for i := range suggestions {
		s := suggestions[i]
		res.WriteString(theme.List(s.Field+": "+s.Title) + "\n")
	}
	return res.String()
}
