package cmd

import (
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/cli/components"
	"go.mondoo.com/cnquery/cli/config"
	"go.mondoo.com/cnquery/cli/shell"
	"go.mondoo.com/cnquery/cli/theme"
	"go.mondoo.com/cnquery/providers"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory/manager"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/upstream"
	"go.mondoo.com/ranger-rpc"
)

func init() {
	rootCmd.AddCommand(shellCmd)

	shellCmd.Flags().StringP("command", "c", "", "MQL query to executed in the shell.")
	shellCmd.Flags().String("platform-id", "", "Select a specific target asset by providing its platform ID.")
}

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Interactive query shell for MQL.",
	Long:  `Allows the interactive exploration of MQL queries.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("platform-id", cmd.Flags().Lookup("platform-id"))
	},
}

var shellRun = func(cmd *cobra.Command, runtime *providers.Runtime, cliRes *plugin.ParseCLIRes) {
	conf, err := config.Read()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	config.DisplayUsedConfig()

	shellConf := ShellConfig{
		Features:   config.Features,
		PlatformID: viper.GetString("platform-id"),
		Asset:      cliRes.Asset,
	}

	shellConf.Command, _ = cmd.Flags().GetString("command")

	serviceAccount := conf.GetServiceCredential()
	if serviceAccount != nil {
		certAuth, err := upstream.NewServiceAccountRangerPlugin(serviceAccount)
		if err != nil {
			log.Error().Err(err).Msg("could not initialize client authentication")
			os.Exit(ConfigurationErrorCode)
		}

		shellConf.Upstream = &plugin.UpstreamClient{
			UpstreamConfig: plugin.UpstreamConfig{
				SpaceMrn:    conf.GetParentMrn(),
				ApiEndpoint: conf.UpstreamApiEndpoint(),
				// we do not use opts here since we want to ensure the result is not stored when users use the shell
				Incognito: true,
			},
			Plugins: []ranger.ClientPlugin{certAuth},
		}
	}

	// set up the http client to include proxy config
	httpClient, err := conf.GetHttpClient()
	if err != nil {
		log.Error().Err(err).Msg("error while setting up httpclient")
		os.Exit(ConfigurationErrorCode)
	}
	if shellConf.Upstream == nil {
		shellConf.Upstream = &plugin.UpstreamClient{}
	}
	shellConf.Upstream.HttpClient = httpClient

	err = StartShell(runtime, &shellConf)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to run query")
	}
}

// ShellConfig is the shared configuration for running a shell given all
// commandline and config inputs.
// TODO: the config is a shared structure, which should be moved to proto
type ShellConfig struct {
	Command        string
	Asset          *inventory.Asset
	Features       cnquery.Features
	PlatformID     string
	WelcomeMessage string

	Upstream *plugin.UpstreamClient
}

// StartShell will start an interactive CLI shell
func StartShell(runtime *providers.Runtime, conf *ShellConfig) error {
	im, err := manager.NewManager(manager.WithInventory(&inventory.Inventory{
		Spec: &inventory.InventorySpec{
			Assets: []*inventory.Asset{conf.Asset},
		},
	}, runtime))
	if err != nil {
		log.Fatal().Err(err).Msg("could not load asset information")
	}

	assetList := im.GetAssets()
	log.Debug().Msgf("resolved %d assets", len(assetList))

	if len(assetList) == 0 {
		log.Fatal().Msg("could not find an asset that we can connect to")
	}

	var connectAsset *inventory.Asset
	if len(assetList) == 1 {
		connectAsset = assetList[0]
	} else if len(assetList) > 1 && conf.PlatformID != "" {
		connectAsset, err = filterAssetByPlatformID(assetList, conf.PlatformID)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
	} else if len(assetList) > 1 {
		isTTY := isatty.IsTerminal(os.Stdout.Fd())
		if isTTY {
			connectAsset = components.AssetSelect(assetList)
		} else {
			fmt.Println(components.AssetList(theme.OperatingSystemTheme, assetList))
			log.Fatal().Msg("cannot connect to more than one asset, use --platform-id to select a specific asset")
		}
	}

	if connectAsset == nil {
		log.Fatal().Msg("no asset selected")
	}

	err = runtime.Connect(&plugin.ConnectReq{
		Features: conf.Features,
		Asset:    conf.Asset,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to asset")
	}

	log.Info().Msgf("connected to %s", runtime.Provider.Connection.Asset.Platform.Title)

	// when we close the shell, we need to close the backend and store the recording
	onCloseHandler := func() {
		runtime.Close()
	}

	shellOptions := []shell.ShellOption{}
	shellOptions = append(shellOptions, shell.WithOnCloseListener(onCloseHandler))
	shellOptions = append(shellOptions, shell.WithFeatures(conf.Features))

	if conf.Upstream != nil {
		shellOptions = append(shellOptions, shell.WithUpstreamConfig(&conf.Upstream.UpstreamConfig))
	}

	sh, err := shell.New(runtime, shellOptions...)
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize interactive shell")
	}
	if conf.WelcomeMessage != "" {
		sh.Theme.Welcome = conf.WelcomeMessage
	}
	sh.RunInteractive(conf.Command)

	return nil
}
