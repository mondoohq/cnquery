package cmd

import (
	"os"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/apps/cnquery/cmd/builder"
	"go.mondoo.com/cnquery/cli/components"
	"go.mondoo.com/cnquery/cli/config"
	"go.mondoo.com/cnquery/cli/inventoryloader"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/shared"
	"go.mondoo.com/cnquery/shared/proto"
)

func init() {
	rootCmd.AddCommand(execCmd)
}

var execCmd = builder.NewProviderCommand(builder.CommandOpts{
	Use:   "run",
	Short: "Run an MQL query",
	Long:  `Run an MQL query on the CLI and displays its results.`,
	CommonFlags: func(cmd *cobra.Command) {
		cmd.Flags().Bool("parse", false, "Parse the query and return the logical structure.")
		cmd.Flags().Bool("ast", false, "Parse the query and return the abstract syntax tree (AST).")
		cmd.Flags().BoolP("json", "j", false, "Run the query and return the object in a JSON structure.")
		cmd.Flags().String("query", "", "MQL query to execute.")
		cmd.Flags().MarkHidden("query")
		cmd.Flags().StringP("command", "c", "", "MQL query to execute.")

		cmd.Flags().StringP("password", "p", "", "Connection password, such as for SSH/WinRM.")
		cmd.Flags().Bool("ask-pass", false, "Prompt for connection password.")
		cmd.Flags().StringP("identity-file", "i", "", "Select a file from which to read the identity (private key) for public key authentication.")
		cmd.Flags().Bool("insecure", false, "Disable TLS/SSL checks or SSH hostkey config.")
		cmd.Flags().Bool("sudo", false, "Elevate privileges with sudo.")
		cmd.Flags().String("platform-id", "", "Select a specific target asset by providing its platform ID.")
		cmd.Flags().Bool("instances", false, "Also scan instances. This only applies to API targets like AWS, Azure or GCP.)")
		cmd.Flags().Bool("host-machines", false, "Also scan host machines like ESXi server.")

		cmd.Flags().Bool("record", false, "Record provider calls. This only works for operating system providers.)")
		cmd.Flags().MarkHidden("record")

		cmd.Flags().String("record-file", "", "File path for the recorded provider calls. This only works for operating system providers.)")
		cmd.Flags().MarkHidden("record-file")

		cmd.Flags().String("path", "", "Path to a local file or directory for the connection to use.")
		cmd.Flags().StringToString("option", nil, "Additional connection options. You can pass multiple options using `--option key=value`.")
		cmd.Flags().String("discover", common.DiscoveryAuto, "Enable the discovery of nested assets. Supported: 'all|auto|instances|host-instances|host-machines|container|container-images|pods|cronjobs|statefulsets|deployments|jobs|replicasets|daemonsets'")
		cmd.Flags().StringToString("discover-filter", nil, "Additional filter for asset discovery.")
	},
	CommonPreRun: func(cmd *cobra.Command, args []string) {
		// for all assets
		viper.BindPFlag("insecure", cmd.Flags().Lookup("insecure"))
		viper.BindPFlag("sudo.active", cmd.Flags().Lookup("sudo"))

		viper.BindPFlag("output", cmd.Flags().Lookup("output"))

		viper.BindPFlag("vault.name", cmd.Flags().Lookup("vault"))
		viper.BindPFlag("platform-id", cmd.Flags().Lookup("platform-id"))
		viper.BindPFlag("query", cmd.Flags().Lookup("query"))
		viper.BindPFlag("command", cmd.Flags().Lookup("command"))

		viper.BindPFlag("record", cmd.Flags().Lookup("record"))
		viper.BindPFlag("record-file", cmd.Flags().Lookup("record-file"))
	},
	Run: func(cmd *cobra.Command, args []string, provider providers.ProviderType, assetType builder.AssetType) {
		conf, err := GetCobraRunConfig(cmd, args, provider, assetType)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to prepare config")
		}

		x := cnqueryPlugin{}
		w := shared.IOWriter{Writer: os.Stdout}
		err = x.RunQuery(conf, &w)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to run query")
		}
	},
})

// GetCobraRunConfig parses cobra and viper flags targeted at a "run" call
// and translates them into a config for the runner.
func GetCobraRunConfig(cmd *cobra.Command, args []string, provider providers.ProviderType, assetType builder.AssetType) (*proto.RunQueryConfig, error) {
	conf := proto.RunQueryConfig{
		Features: config.Features,
	}

	// check if the user used --password without a value
	askPass, err := cmd.Flags().GetBool("ask-pass")
	if err == nil && askPass {
		pass, err := components.AskPassword("Enter password: ")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to get password")
		}
		cmd.Flags().Set("password", pass)
	}

	conf.Command, _ = cmd.Flags().GetString("command")
	// fallback to --query
	if conf.Command == "" {
		conf.Command, _ = cmd.Flags().GetString("query")
	}

	conf.DoParse, err = cmd.Flags().GetBool("parse")
	if err != nil {
		return nil, errors.New("could not load parse setting")
	}
	if conf.DoParse {
		return &conf, nil
	}

	conf.DoAst, err = cmd.Flags().GetBool("ast")
	if err != nil {
		return nil, errors.New("could not load AST setting")
	}
	if conf.DoAst {
		return &conf, nil
	}

	doJSON, err := cmd.Flags().GetBool("json")
	if err != nil {
		return nil, errors.New("could not load JSON export setting")
	}
	if doJSON {
		conf.Format = "json"
	}

	conf.DoRecord = viper.GetBool("record")

	// determine the scan config from pipe or args
	flagAsset := builder.ParseTargetAsset(cmd, args, provider, assetType)
	conf.Inventory, err = inventoryloader.ParseOrUse(flagAsset, viper.GetBool("insecure"))
	if err != nil {
		return nil, errors.Wrap(err, "could not load configuration")
	}

	conf.PlatformId = viper.GetString("platform-id")

	return &conf, nil
}
