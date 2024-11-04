// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"fmt"

	"go.mondoo.com/cnquery/v11/cli/components"
	"go.mondoo.com/cnquery/v11/cli/theme"
)

type CustomString string

func (s CustomString) PrintableKeys() []string {
	return []string{"string"}
}
func (s CustomString) PrintableValue(_ int) string {
	return string(s)
}

func main() {
	customStrings := []CustomString{"first", "second", "third"}
	list := components.List(theme.OperatingSystemTheme, "string", customStrings)
	fmt.Printf(list)
}
