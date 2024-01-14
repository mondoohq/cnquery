// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v10"
	"go.mondoo.com/cnquery/v10/cli/components"
	"go.mondoo.com/cnquery/v10/cli/config"
	"go.mondoo.com/cnquery/v10/cli/shell"
	"go.mondoo.com/cnquery/v10/cli/theme"
	"go.mondoo.com/cnquery/v10/providers"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory/manager"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream"
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
	// we have to initialize an empty run so it shows up as a runnable command in --help
	Run: func(cmd *cobra.Command, args []string) {},
}

var shellRun = func(cmd *cobra.Command, runtime *providers.Runtime, cliRes *plugin.ParseCLIRes) {
	shellConf := ParseShellConfig(cmd, cliRes)
	if err := StartShell(runtime, shellConf); err != nil {
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
	UpstreamConfig *upstream.UpstreamConfig
}

func ParseShellConfig(cmd *cobra.Command, cliRes *plugin.ParseCLIRes) *ShellConfig {
	conf, err := config.Read()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	config.DisplayUsedConfig()

	var upstreamConfig *upstream.UpstreamConfig
	serviceAccount := conf.GetServiceCredential()
	if serviceAccount != nil {
		upstreamConfig = &upstream.UpstreamConfig{
			// AssetMrn: not necessary right now, especially since incognito
			SpaceMrn:    conf.GetParentMrn(),
			ApiEndpoint: conf.UpstreamApiEndpoint(),
			ApiProxy:    conf.APIProxy,
			Incognito:   viper.GetBool("incognito"),
			Creds:       conf.GetServiceCredential(),
		}
	}

	shellConf := ShellConfig{
		Features:       config.Features,
		PlatformID:     viper.GetString("platform-id"),
		Asset:          cliRes.Asset,
		UpstreamConfig: upstreamConfig,
	}

	shellConf.Command, _ = cmd.Flags().GetString("command")
	return &shellConf
}

// StartShell will start an interactive CLI shell
func StartShell(runtime *providers.Runtime, conf *ShellConfig) error {
	// we go through inventory resolution to resolve credentials properly for the passed-in asset
	im, err := manager.NewManager(manager.WithInventory(inventory.New(inventory.WithAssets(conf.Asset)), runtime))
	if err != nil {
		return errors.New("failed to resolve inventory for connection")
	}
	resolvedAsset, err := im.ResolveAsset(conf.Asset)
	if err != nil {
		return err
	}
	res, err := runtime.Provider.Instance.Plugin.Connect(&plugin.ConnectReq{
		Features: conf.Features,
		Asset:    resolvedAsset,
		Upstream: conf.UpstreamConfig,
	}, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("could not load asset information")
	}

	assets, err := providers.ProcessAssetCandidates(runtime, res, conf.UpstreamConfig, conf.PlatformID)
	if err != nil {
		log.Fatal().Err(err).Msg("could not process assets")
	}
	if len(assets) == 0 {
		log.Fatal().Msg("could not find an asset that we can connect to")
	}

	connectAsset := assets[0]
	if len(assets) > 1 {
		isTTY := isatty.IsTerminal(os.Stdout.Fd())
		if isTTY {
			connectAsset = components.AssetSelect(assets)
		} else {
			fmt.Println(components.AssetList(theme.OperatingSystemTheme, assets))
			log.Fatal().Msg("cannot connect to more than one asset, use --platform-id to select a specific asset")
		}
	}

	if connectAsset == nil {
		log.Fatal().Msg("no asset selected")
	}

	err = runtime.Connect(&plugin.ConnectReq{
		Features: conf.Features,
		Asset:    connectAsset,
		Upstream: conf.UpstreamConfig,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to asset")
	}
	log.Info().Msgf("connected to %s", runtime.Provider.Connection.Asset.Platform.Title)

	// when we close the shell, we need to close the backend and store the recording
	onCloseHandler := func() {
		runtime.Close()
		providers.Coordinator.Shutdown()
	}

	shellOptions := []shell.ShellOption{}
	shellOptions = append(shellOptions, shell.WithOnCloseListener(onCloseHandler))
	shellOptions = append(shellOptions, shell.WithFeatures(conf.Features))
	shellOptions = append(shellOptions, shell.WithUpstreamConfig(conf.UpstreamConfig))

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
