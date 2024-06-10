// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"io"
	"log"
	"os"

	"github.com/snowflakedb/gosnowflake"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/snowflake/provider"
)

func main() {
	// Set environment variables to control the behavior of the provider
	// This is required for the snowflake api that is used by the provider
	os.Setenv("SF_TF_NO_INSTRUMENTED_SQL", "1")
	os.Setenv("SF_TF_GOSNOWFLAKE_LOG_LEVEL", "error")
	// The following line changes the log level to debug
	_ = gosnowflake.GetLogger().SetLogLevel("error")
	log.SetOutput(io.Discard)
	plugin.Start(os.Args, provider.Init())
}
