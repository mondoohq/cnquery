// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"os"

	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/nmap/provider"
)

func main() {
	plugin.Start(os.Args, provider.Init())
}
