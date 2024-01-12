// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"go.mondoo.com/cnquery/v10"
	"go.mondoo.com/cnquery/v10/apps/cnquery/cmd"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream/health"
)

func main() {
	defer health.ReportPanic("cnquery", cnquery.Version, cnquery.Build)
	cmd.Execute()
}
