package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/apps/cnquery/cmd/builder"
	"go.mondoo.com/cnquery/cli/assetlist"
	"go.mondoo.com/cnquery/cli/inventoryloader"
	"go.mondoo.com/cnquery/cli/printer"
	"go.mondoo.com/cnquery/cli/shell"
	"go.mondoo.com/cnquery/logger"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery"
	"go.mondoo.com/cnquery/motor/inventory"
	v1 "go.mondoo.com/cnquery/motor/inventory/v1"
	"go.mondoo.com/cnquery/motor/providers"
	provider_resolver "go.mondoo.com/cnquery/motor/providers/resolver"
	"go.mondoo.com/cnquery/mqlc"
	"go.mondoo.com/cnquery/mqlc/parser"
	"go.mondoo.com/cnquery/resources/packs/all/info"
)

func init() {
	rootCmd.AddCommand(execCmd)
}

var execCmd = builder.NewProviderCommand(builder.CommandOpts{
	Use:   "run",
	Short: "Run a MQL query",
	Long:  `Run a MQL query on the CLI and displays its results.`,
	CommonFlags: func(cmd *cobra.Command) {
		cmd.Flags().Bool("parse", false, "Parse the query and return the logical structure")
		cmd.Flags().Bool("ast", false, "Parse the query and return the Abstract Syntax Tree (AST)")
		cmd.Flags().BoolP("json", "j", false, "Run the query and return the object in a JSON structure")
		cmd.Flags().String("query", "", "MQL query to be executed")
		cmd.Flags().MarkHidden("query")
		cmd.Flags().StringP("command", "c", "", "MQL query to be executed")

		cmd.Flags().StringP("password", "p", "", "connection password e.g. for ssh/winrm")
		cmd.Flags().Bool("ask-pass", false, "ask for connection password")
		cmd.Flags().StringP("identity-file", "i", "", "Selects a file from which the identity (private key) for public key authentication is read.")
		cmd.Flags().Bool("insecure", false, "disables TLS/SSL checks or SSH hostkey config")
		cmd.Flags().Bool("sudo", false, "runs with sudo")
		cmd.Flags().String("platform-id", "", "select an specific asset by providing the platform id for the target")
		cmd.Flags().Bool("instances", false, "also scan instances (only applies to api targets like aws, azure or gcp)")
		cmd.Flags().Bool("host-machines", false, "also scan host machines like ESXi server")

		cmd.Flags().Bool("record", false, "records provider calls (only works for operating system providers)")
		cmd.Flags().MarkHidden("record")

		cmd.Flags().String("record-file", "", "file path to for the recorded provider calls (only works for operating system providers)")
		cmd.Flags().MarkHidden("record-file")

		cmd.Flags().String("path", "", "path to a local file or directory that the connection should use")
		cmd.Flags().StringToString("option", nil, "addition connection options, multiple options can be passed in via --option key=value")
		cmd.Flags().String("discover", "", "enables the discovery of nested assets. Supported are 'all|instances|host-instances|host-machines|container|container-images|pods|cronjobs|statefulsets|deployments|jobs|replicasets|daemonsets'")
		cmd.Flags().StringToString("discover-filter", nil, "additional filter for asset discovery")
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
		ctx := discovery.InitCtx(context.Background())

		// check if the user used --password without a value
		askPass, err := cmd.Flags().GetBool("ask-pass")
		if err == nil && askPass {
			askForPassword("Enter password: ", cmd.Flags())
		}

		flagAsset := builder.ParseTargetAsset(cmd, args, provider, assetType)

		command, _ := cmd.Flags().GetString("command")
		// fallback to --query
		if command == "" {
			command, _ = cmd.Flags().GetString("query")
		}

		// determine the scan config from pipe or args
		v1Inventory, err := getInventory(flagAsset, viper.GetBool("insecure"))
		if err != nil {
			log.Fatal().Err(err).Msg("could not load configuration")
		}
		features := cnquery.DefaultFeatures

		doParse, err := cmd.Flags().GetBool("parse")
		if err != nil {
			log.Fatal().Err(err).Msg("could not load parse setting")
		}
		if doParse {
			ast, err := parser.Parse(command)
			fmt.Println(logger.PrettyJSON(ast))

			if err != nil {
				log.Error().Err(err).Msg("failed to parse command")
			}
			return
		}

		doAST, err := cmd.Flags().GetBool("ast")
		if err != nil {
			log.Fatal().Err(err).Msg("could not load AST setting")
		}
		if doAST {
			b, err := mqlc.Compile(command, info.Registry.Schema(), features, nil)

			fmt.Println(logger.PrettyJSON(b))
			res := printer.DefaultPrinter.CodeBundle(b)
			fmt.Println(res)

			if err != nil {
				log.Error().Err(err).Msg("failed to compile")
			}
			return
		}

		doJSON, err := cmd.Flags().GetBool("json")
		if err != nil {
			log.Fatal().Err(err).Msg("could not load json export setting")
		}

		log.Info().Msgf("discover related assets for %d asset(s)", len(v1Inventory.Spec.Assets))
		im, err := inventory.New(inventory.WithInventory(v1Inventory))
		if err != nil {
			log.Fatal().Err(err).Msg("could not load asset information")
		}
		assetErrors := im.Resolve(ctx)
		if len(assetErrors) > 0 {
			for a := range assetErrors {
				log.Error().Err(assetErrors[a]).Str("asset", a.Name).Msg("could not connect to asset")
			}
		}

		assetList := im.GetAssets()
		if len(assetList) == 0 {
			log.Fatal().Msg("could not find an asset that we can connect to")
		}

		var connectAsset *asset.Asset
		selectedPlatformID := viper.GetString("platform-id")

		if len(assetList) == 1 {
			connectAsset = assetList[0]
		} else if len(assetList) > 1 && selectedPlatformID != "" {
			connectAsset, err = filterAssetByPlatformID(assetList, selectedPlatformID)
			if err != nil {
				log.Fatal().Err(err).Send()
			}
		} else if len(assetList) > 1 {
			r := &assetlist.SimpleRender{}
			fmt.Println(r.Render(assetList))
			log.Fatal().Msg("cannot connect to more than one asset, use --platform-id to select a specific asset")
		}

		record := viper.GetBool("record")
		if record {
			log.Info().Msg("enable recording of platform calls")
		}

		m, err := provider_resolver.OpenAssetConnection(ctx, connectAsset, im.GetCredential, record)
		if err != nil {
			log.Fatal().Err(err).Msg("could not connect to asset")
		}

		// when we close the shell, we need to close the backend and store the recording
		onCloseHandler := func() {
			storeRecording(m)

			// close backend connection
			m.Close()
		}

		shellOptions := []shell.ShellOption{}
		shellOptions = append(shellOptions, shell.WithOnCloseListener(onCloseHandler))
		shellOptions = append(shellOptions, shell.WithFeatures(features))

		sh, err := shell.New(m, shellOptions...)
		if err != nil {
			log.Error().Err(err).Msg("failed to initialize Mondoo Shell")
		}
		defer sh.Close()

		code, results, err := sh.RunOnce(command)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to run")
		}

		if !doJSON {
			sh.PrintResults(code, results)
			return
		}

		var checksums []string
		eps := code.CodeV2.Entrypoints()
		checksums = make([]string, len(eps))
		for i, ref := range eps {
			checksums[i] = code.CodeV2.Checksums[ref]
		}

		for _, checksum := range checksums {
			result := results[checksum]
			if result == nil {
				log.Fatal().Msg("cannot find result for this query")
			}

			if result.Data.Error != nil {
				log.Fatal().Err(result.Data.Error).Msg("received an error:")
			}

			j := result.Data.JSON(checksum, code)
			fmt.Println(string(j))
		}
	},
})

// TODO: consider moving this to inventoryloader package
func getInventory(cliAsset *asset.Asset, insecure bool) (*v1.Inventory, error) {
	var v1inventory *v1.Inventory
	var err error

	// parses optional inventory file if inventory was not piped already
	if v1inventory == nil {
		v1inventory, err = inventoryloader.Parse()
		if err != nil {
			return nil, errors.Wrap(err, "could not parse inventory")
		}
	}

	// add asset from cli to inventory
	if (len(v1inventory.Spec.GetAssets()) == 0) && cliAsset != nil {
		v1inventory.AddAssets(cliAsset)
	}

	// if the --insecure flag is set, we overwrite the individual setting for the asset
	if insecure == true {
		v1inventory.MarkConnectionsInsecure()
	}

	return v1inventory, nil
}

func filterAssetByPlatformID(assetList []*asset.Asset, selectionID string) (*asset.Asset, error) {
	var foundAsset *asset.Asset
	for i := range assetList {
		assetObj := assetList[i]
		for j := range assetObj.PlatformIds {
			if assetObj.PlatformIds[j] == selectionID {
				return assetObj, nil
			}
		}
	}

	if foundAsset == nil {
		return nil, errors.New("could not find an asset with the provided identifer: " + selectionID)
	}
	return foundAsset, nil
}

// storeRecording stores tracked commands and files into the recording file
func storeRecording(m *motor.Motor) {
	if m.IsRecording() {
		filename := viper.GetString("record-file")
		if filename == "" {
			filename = "recording-" + time.Now().Format("20060102150405") + ".toml"
		}
		log.Info().Str("filename", filename).Msg("store recordings")
		data := m.Recording()
		os.WriteFile(filename, data, 0o700)
	}
}
