// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin/gen"
	"go.mondoo.com/cnquery/v11/providers/nmap/config"
)

func main() {
	gen.CLI(&config.Config)
}
