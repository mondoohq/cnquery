package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/mattn/go-isatty"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/cli/components"
	"go.mondoo.com/cnquery/cli/shell"
	"go.mondoo.com/cnquery/cli/theme"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery"
	"go.mondoo.com/cnquery/motor/inventory"
	v1 "go.mondoo.com/cnquery/motor/inventory/v1"
	provider_resolver "go.mondoo.com/cnquery/motor/providers/resolver"
	"go.mondoo.com/cnquery/resources"
)

// ShellConfig is the shared configuration for running a shell given all
// commandline and config inputs.
// TODO: the config is a shared structure, which should be moved to proto
type ShellConfig struct {
	Command    string
	Inventory  *v1.Inventory
	Features   cnquery.Features
	PlatformID string

	DoRecord       bool
	WelcomeMessage string

	UpstreamConfig *resources.UpstreamConfig
}

// StartShell will start an interactive CLI shell
func StartShell(conf *ShellConfig) error {
	ctx := discovery.InitCtx(context.Background())

	log.Info().Msgf("discover related assets for %d asset(s)", len(conf.Inventory.Spec.Assets))
	im, err := inventory.New(inventory.WithInventory(conf.Inventory))
	if err != nil {
		log.Fatal().Err(err).Msg("could not load asset information")
	}
	assetErrors := im.Resolve(ctx)
	if len(assetErrors) > 0 {
		for a := range assetErrors {
			log.Error().Err(assetErrors[a]).Str("asset", a.Name).Msg("could not connect to asset")
		}
		log.Fatal().Msg("could not resolve assets")
	}

	assetList := im.GetAssets()
	log.Debug().Msgf("resolved %d assets", len(assetList))

	if len(assetList) == 0 {
		log.Fatal().Msg("could not find an asset that we can connect to")
	}

	var connectAsset *asset.Asset
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
			fmt.Println(components.AssetList(theme.OperatingSytemTheme, assetList))
			log.Fatal().Msg("cannot connect to more than one asset, use --platform-id to select a specific asset")
		}
	}

	if connectAsset == nil {
		log.Fatal().Msg("no asset selected")
	}

	m, err := provider_resolver.OpenAssetConnection(ctx, connectAsset, im.GetCredential, conf.DoRecord)
	if err != nil {
		log.Fatal().Err(err).Msg("could not connect to asset")
	}

	// when we close the shell, we need to close the backend and store the recording
	onCloseHandler := func() {
		// store tracked commands and files
		storeRecording(m)

		// close backend connection
		m.Close()
	}

	shellOptions := []shell.ShellOption{}
	shellOptions = append(shellOptions, shell.WithOnCloseListener(onCloseHandler))
	shellOptions = append(shellOptions, shell.WithFeatures(conf.Features))

	if conf.UpstreamConfig != nil {
		shellOptions = append(shellOptions, shell.WithUpstreamConfig(conf.UpstreamConfig))
	}

	sh, err := shell.New(m, shellOptions...)
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize interactive shell")
	}
	if conf.WelcomeMessage != "" {
		sh.Theme.Welcome = conf.WelcomeMessage
	}
	sh.RunInteractive(conf.Command)

	return nil
}
