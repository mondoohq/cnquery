// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"os"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/snowflake/provider"
)

func main() {
	// Set environment variables to control the behavior of the provider
	// This is required for the snowflake api that is used by the provider
	os.Setenv("SF_TF_NO_INSTRUMENTED_SQL", "1")
	os.Setenv("SF_TF_GOSNOWFLAKE_LOG_LEVEL", "error")
	plugin.Start(os.Args, provider.Init())
}
