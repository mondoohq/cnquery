// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.mondoo.com/cnquery/v12/cli/config"
)

// featuresCmd represents the version command
var featuresCmd = &cobra.Command{
	Hidden: true,
	Use:    "features",
	Short:  "Display cnquery features",
	Run: func(cmd *cobra.Command, args []string) {
		// prerequisite: features must be initialized via config on the root command
		// otherwise config.Features won't contain anything useful
		fmt.Println("Active features: " + config.Features.String())
	},
}

func init() {
	rootCmd.AddCommand(featuresCmd)
}
