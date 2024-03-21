// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v10/providers"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/shared"
	"go.mondoo.com/cnquery/v10/shared/proto"
)

func init() {
	rootCmd.AddCommand(RunCmd)

	RunCmd.Flags().StringP("command", "c", "", "MQL query to executed in the shell.")
	RunCmd.Flags().Bool("parse", false, "Parse the query and return the logical structure.")
	RunCmd.Flags().Bool("ast", false, "Parse the query and return the abstract syntax tree (AST).")
	RunCmd.Flags().Bool("info", false, "Parse the query and provide information about it.")
	RunCmd.Flags().BoolP("json", "j", false, "Run the query and return the object in a JSON structure.")
	RunCmd.Flags().String("platform-id", "", "Select a specific target asset by providing its platform ID.")

	RunCmd.Flags().String("llx", "", "Generate the executable code bundle and save it to the specified file.")
	RunCmd.Flags().MarkHidden("llx")
	RunCmd.Flags().String("use-llx", "", "Run the code specified in the code bundle on disk")
	RunCmd.Flags().MarkHidden("use-llx")
	RunCmd.Flags().StringToString("annotations", nil, "Specify annotations for this run")
	RunCmd.Flags().MarkHidden("annotations")
}

var RunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run an MQL query",
	Long:  `Run an MQL query on the CLI and displays its results.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		_ = viper.BindPFlag("platform-id", cmd.Flags().Lookup("platform-id"))
		_ = viper.BindPFlag("annotations", cmd.Flags().Lookup("annotations"))
	},
	// we have to initialize an empty run so it shows up as a runnable command in --help
	Run: func(cmd *cobra.Command, args []string) {},
}

var RunCmdRun = func(cmd *cobra.Command, runtime *providers.Runtime, cliRes *plugin.ParseCLIRes) {
	conf := proto.RunQueryConfig{}

	conf.Command, _ = cmd.Flags().GetString("command")
	conf.DoAst, _ = cmd.Flags().GetBool("ast")
	conf.DoInfo, _ = cmd.Flags().GetBool("info")
	conf.DoParse, _ = cmd.Flags().GetBool("parse")
	if doJSON, _ := cmd.Flags().GetBool("json"); doJSON {
		conf.Format = "json"
	}
	if llx, _ := cmd.Flags().GetString("llx"); llx != "" {
		conf.Format = "llx"
		conf.Output = llx
	}
	if llx, _ := cmd.Flags().GetString("use-llx"); llx != "" {
		conf.Input = llx
	}

	conf.PlatformId, _ = cmd.Flags().GetString("platform-id")
	annotations, _ := cmd.Flags().GetStringToString("annotations")
	cliRes.Asset.AddAnnotations(annotations)

	in := &inventory.Inventory{
		Spec: &inventory.InventorySpec{
			Assets: []*inventory.Asset{cliRes.Asset},
		},
	}
	err := in.PreProcess() // required to resolve secrets
	if err != nil {
		log.Fatal().Err(err).Msg("failed to resolve inventory")
	}

	conf.Inventory = in
	conf.Incognito, _ = cmd.Flags().GetBool("incognito")

	x := cnqueryPlugin{}
	w := shared.IOWriter{Writer: os.Stdout}
	err = x.RunQuery(&conf, runtime, &w)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to run query")
	}
}
