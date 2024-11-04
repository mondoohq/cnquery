// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"fmt"

	"go.mondoo.com/cnquery/v11/cli/components"
)

type CustomString string

func (s CustomString) HumanName() string {
	return string(s)
}

func main() {
	customStrings := []CustomString{"first", "second", "third"}
	selected := components.Select("Choose a string", customStrings)
	fmt.Printf("You chose the %s string.\n", customStrings[selected])
}
