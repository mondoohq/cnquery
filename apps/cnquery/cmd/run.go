package cmd

import (
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/providers"
	"go.mondoo.com/cnquery/providers/proto"
	"go.mondoo.com/cnquery/shared"
	run "go.mondoo.com/cnquery/shared/proto"
)

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringP("command", "c", "", "MQL query to executed in the shell.")
	runCmd.Flags().Bool("parse", false, "Parse the query and return the logical structure.")
	runCmd.Flags().Bool("ast", false, "Parse the query and return the abstract syntax tree (AST).")
	runCmd.Flags().BoolP("json", "j", false, "Run the query and return the object in a JSON structure.")
	runCmd.Flags().String("platform-id", "", "Select a specific target asset by providing its platform ID.")
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run an MQL query.",
	Long:  `Run an MQL query on the CLI and displays its results.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("platform-id", cmd.Flags().Lookup("platform-id"))
	},
}

var runcmdRun = func(cmd *cobra.Command, runtime *providers.Runtime, cliRes *proto.ParseCLIRes) {
	conf := run.RunQueryConfig{}

	conf.Command, _ = cmd.Flags().GetString("command")
	conf.DoAst, _ = cmd.Flags().GetBool("ast")
	conf.DoParse, _ = cmd.Flags().GetBool("parse")
	conf.DoRecord, _ = cmd.Flags().GetBool("record")
	if doJSON, _ := cmd.Flags().GetBool("json"); doJSON {
		conf.Format = "json"
	}
	conf.PlatformId, _ = cmd.Flags().GetString("platform-id")
	conf.Inventory = cliRes.Inventory

	x := cnqueryPlugin{}
	w := shared.IOWriter{Writer: os.Stdout}
	err := x.RunQuery(&conf, runtime, &w)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to run query")
	}
}
