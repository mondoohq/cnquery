// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
package cmd

import (
	"github.com/spf13/cobra"
	"go.mondoo.com/cnquery/v12/cli/lsp"
)

func init() {
	rootCmd.AddCommand(LspCmd)
}

var LspCmd = &cobra.Command{
	Use:   "lsp",
	Short: "Launch the MQL Language Server",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create the LSP server
		return lsp.RunStdio()
	},
}
