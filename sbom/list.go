// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"github.com/muesli/termenv"
	"go.mondoo.com/cnquery/v10/cli/theme/colors"
	"io"
	"sort"
	"strings"
)

type CliList struct {
}

func (s *CliList) Render(w io.Writer, bom *Sbom) error {
	sort.SliceStable(bom.Packages, func(i, j int) bool {
		if bom.Packages[i].Name != bom.Packages[j].Name {
			return bom.Packages[i].Name < bom.Packages[j].Name
		}

		return bom.Packages[i].Name < bom.Packages[j].Name
	})

	for i := range bom.Packages {
		pkg := bom.Packages[i]

		sb := strings.Builder{}

		// rpm/libxxhash0/0.8.0-2 arm64
		if pkg.Type != "" {
			sb.WriteString(termenv.String(pkg.Type).Foreground(colors.DefaultColorTheme.Disabled).String())
			sb.WriteString(termenv.String("/").Foreground(colors.DefaultColorTheme.Disabled).String())
		}
		sb.WriteString(termenv.String(pkg.Name).Foreground(colors.DefaultColorTheme.Primary).String())
		sb.WriteString(termenv.String("/").Foreground(colors.DefaultColorTheme.Disabled).String())
		sb.WriteString(pkg.Version)

		if pkg.Architecture != "" {
			sb.WriteString(" ")
			sb.WriteString(pkg.Architecture)
		}

		if pkg.Location != "" {
			sb.WriteString(" ")
			sb.WriteString(termenv.String(pkg.Location).Foreground(colors.DefaultColorTheme.Disabled).String())
		}

		sb.WriteString("\n")

		_, err := w.Write([]byte(sb.String()))
		if err != nil {
			return err
		}
	}
	return nil
}
