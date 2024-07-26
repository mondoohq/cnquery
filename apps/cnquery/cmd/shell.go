// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v11"
	"go.mondoo.com/cnquery/v11/cli/components"
	"go.mondoo.com/cnquery/v11/cli/config"
	"go.mondoo.com/cnquery/v11/cli/shell"
	"go.mondoo.com/cnquery/v11/cli/theme"
	"go.mondoo.com/cnquery/v11/explorer/scan"
	"go.mondoo.com/cnquery/v11/providers"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/upstream"
)

func init() {
	rootCmd.AddCommand(shellCmd)

	shellCmd.Flags().StringP("command", "c", "", "MQL query to execute in the shell")
	shellCmd.Flags().String("platform-id", "", "Select a specific target asset by providing its platform ID")
	shellCmd.Flags().StringToString("annotations", nil, "Specify annotations for this run")
	_ = shellCmd.Flags().MarkHidden("annotations")
}

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Interactive query shell for MQL",
	Long:  `Allows the interactive exploration of MQL queries`,
	PreRun: func(cmd *cobra.Command, args []string) {
		_ = viper.BindPFlag("platform-id", cmd.Flags().Lookup("platform-id"))
		_ = viper.BindPFlag("annotations", cmd.Flags().Lookup("annotations"))
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

	annotations, _ := cmd.Flags().GetStringToString("annotations")
	cliRes.Asset.AddAnnotations(annotations)

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
	ctx := context.Background()
	discoveredAssets, err := scan.DiscoverAssets(ctx,
		inventory.New(inventory.WithAssets(conf.Asset)),
		conf.UpstreamConfig,
		runtime.Recording())
	if err != nil {
		log.Fatal().Err(err).Msg("could not process assets")
	}
	filteredAssets := discoveredAssets.GetAssetsByPlatformID(conf.PlatformID)
	if len(filteredAssets) == 0 {
		log.Fatal().Msg("could not find an asset that we can connect to")
	}

	var connectAsset *scan.AssetWithRuntime
	if len(filteredAssets) == 1 {
		connectAsset = filteredAssets[0]
	} else if len(filteredAssets) > 1 {
		invAssets := make([]*inventory.Asset, 0, len(filteredAssets))
		for _, a := range filteredAssets {
			invAssets = append(invAssets, a.Asset)
		}

		isTTY := isatty.IsTerminal(os.Stdout.Fd())
		if isTTY {
			selectedAsset := components.AssetSelect(invAssets)
			if selectedAsset >= 0 {
				connectAsset = filteredAssets[selectedAsset]
			}
		} else {
			fmt.Println(components.AssetList(theme.OperatingSystemTheme, invAssets))
			log.Fatal().Msg("cannot connect to more than one asset, use --platform-id to select a specific asset")
		}
	}

	if connectAsset == nil {
		log.Error().Msg("no asset selected")
		os.Exit(1)
	}

	if connectAsset.Asset.Connections[0].DelayDiscovery {
		discoveredAsset, err := scan.HandleDelayedDiscovery(ctx, connectAsset.Asset, connectAsset.Runtime, nil, "")
		if err != nil {
			log.Error().Msg("no asset selected")
			os.Exit(1)
		}
		connectAsset.Asset = discoveredAsset
	}

	log.Info().Msgf("connected to %s", connectAsset.Runtime.Provider.Connection.Asset.Platform.Title)

	// when we close the shell, we need to close the backend and store the recording
	onCloseHandler := func() {
		connectAsset.Runtime.Close()
		providers.Coordinator.Shutdown()
	}

	shellOptions := []shell.ShellOption{}
	shellOptions = append(shellOptions, shell.WithOnCloseListener(onCloseHandler))
	shellOptions = append(shellOptions, shell.WithFeatures(conf.Features))
	shellOptions = append(shellOptions, shell.WithUpstreamConfig(conf.UpstreamConfig))

	sh, err := shell.New(connectAsset.Runtime, shellOptions...)
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize interactive shell")
	}
	if conf.WelcomeMessage != "" {
		sh.Theme.Welcome = conf.WelcomeMessage
	}
	sh.RunInteractive(conf.Command)

	return nil
}
