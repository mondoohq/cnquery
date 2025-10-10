// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "generates Go code and documentation from an LR schema file",
	Long:  `parse an LR file and convert it to Go, then generates documentation from the LR file. This is the equivalent of running the 'go', 'docs yaml' and 'docs json' commands one after another.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dist, err := cmd.Flags().GetString("dist")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to get dist flag")
		}

		if dist == "" {
			log.Fatal().Msg("dist flag is required")
		}

		docsFile, err := cmd.Flags().GetString("docs-file")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to get docs-file flag")
		}

		if docsFile == "" {
			log.Fatal().Msg("docs-file flag is required")
		}

		lrFile := args[0]
		headerFile := ""
		runGoCmd(lrFile, dist, headerFile, false)
		runDocsYamlCmd(lrFile, headerFile, defaultVersionField, docsFile)
		runDocsJsonCmd(docsFile, dist)
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.Flags().String("dist", "", "folder for output LR and docs generation")
	generateCmd.MarkFlagRequired("dist")
	generateCmd.Flags().String("docs-file", "", "path to the docs file")
	generateCmd.MarkFlagRequired("docs-file")
}
