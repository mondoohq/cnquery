// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.Flags().String("dist", "", "folder for output LR and docs generation")
	generateCmd.MarkFlagRequired("dist") // nolint:errcheck
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "generates Go code and versions from an LR schema file",
	Long:  `parse an LR file and convert it to Go, then generates or updates the .lr.versions file. This is the equivalent of running the 'go' and 'versions' commands one after another.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dist, err := cmd.Flags().GetString("dist")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to get dist flag")
		}

		if dist == "" {
			log.Fatal().Msg("dist flag is required")
		}

		lrFile := args[0]
		headerFile := ""
		versionsFile := strings.TrimSuffix(lrFile, ".lr") + ".lr.versions"
		runGoCmd(lrFile, dist, headerFile, false)
		runVersionsCmd(lrFile, headerFile, defaultVersionField, versionsFile)
	},
}
