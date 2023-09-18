// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/providers"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/shared"
	"go.mondoo.com/cnquery/shared/proto"
)

func init() {
	rootCmd.AddCommand(RunCmd)

	RunCmd.Flags().StringP("command", "c", "", "MQL query to executed in the shell.")
	RunCmd.Flags().Bool("parse", false, "Parse the query and return the logical structure.")
	RunCmd.Flags().Bool("ast", false, "Parse the query and return the abstract syntax tree (AST).")
	RunCmd.Flags().BoolP("json", "j", false, "Run the query and return the object in a JSON structure.")
	RunCmd.Flags().String("platform-id", "", "Select a specific target asset by providing its platform ID.")
}

var RunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run an MQL query.",
	Long:  `Run an MQL query on the CLI and displays its results.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("platform-id", cmd.Flags().Lookup("platform-id"))
	},
	// we have to initialize an empty run so it shows up as a runnable command in --help
	Run: func(cmd *cobra.Command, args []string) {},
}

var RunCmdRun = func(cmd *cobra.Command, runtime *providers.Runtime, cliRes *plugin.ParseCLIRes) {
	conf := proto.RunQueryConfig{}

	conf.Command, _ = cmd.Flags().GetString("command")
	conf.DoAst, _ = cmd.Flags().GetBool("ast")
	conf.DoParse, _ = cmd.Flags().GetBool("parse")
	if doJSON, _ := cmd.Flags().GetBool("json"); doJSON {
		conf.Format = "json"
	}
	conf.PlatformId, _ = cmd.Flags().GetString("platform-id")
	conf.Inventory = &inventory.Inventory{
		Spec: &inventory.InventorySpec{
			Assets: []*inventory.Asset{cliRes.Asset},
		},
	}

	x := cnqueryPlugin{}
	w := shared.IOWriter{Writer: os.Stdout}
	err := x.RunQuery(&conf, runtime, &w)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to run query")
	}
}
