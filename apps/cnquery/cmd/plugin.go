package cmd

import (
	"context"
	"github.com/spf13/viper"
	"os"

	"errors"
	"github.com/hashicorp/go-plugin"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	cnquery_config "go.mondoo.com/cnquery/apps/cnquery/cmd/config"
	"go.mondoo.com/cnquery/cli/config"
	"go.mondoo.com/cnquery/cli/printer"
	"go.mondoo.com/cnquery/cli/reporter"
	"go.mondoo.com/cnquery/cli/shell"
	"go.mondoo.com/cnquery/logger"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery"
	"go.mondoo.com/cnquery/motor/inventory"
	provider_resolver "go.mondoo.com/cnquery/motor/providers/resolver"
	"go.mondoo.com/cnquery/mqlc"
	"go.mondoo.com/cnquery/mqlc/parser"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/all/info"
	"go.mondoo.com/cnquery/shared"
	"go.mondoo.com/cnquery/shared/proto"
	"go.mondoo.com/cnquery/upstream"
	"go.mondoo.com/ranger-rpc"
)

// pluginCmd represents the version command
var pluginCmd = &cobra.Command{
	Use:    "run_as_plugin",
	Hidden: true,
	Short:  "Run as a plugin.",
	Run: func(cmd *cobra.Command, args []string) {
		plugin.Serve(&plugin.ServeConfig{
			HandshakeConfig: shared.Handshake,
			Plugins: map[string]plugin.Plugin{
				"counter": &shared.CNQueryPlugin{Impl: &cnqueryPlugin{}},
			},

			// A non-nil value here enables gRPC serving for this plugin...
			GRPCServer: plugin.DefaultGRPCServer,
		})
	},
}

func init() {
	rootCmd.AddCommand(pluginCmd)
}

type cnqueryPlugin struct{}

func (c *cnqueryPlugin) RunQuery(conf *proto.RunQueryConfig, out shared.OutputHelper) error {
	if conf.Command == "" {
		return errors.New("No command provided, nothing to do.")
	}

	opts, optsErr := cnquery_config.ReadConfig()
	if optsErr != nil {
		log.Fatal().Err(optsErr).Msg("could not load configuration")
	}

	config.DisplayUsedConfig()

	ctx := discovery.InitCtx(context.Background())

	if conf.DoParse {
		ast, err := parser.Parse(conf.Command)
		if err != nil {
			return errors.Join(err, errors.New("failed to parse command"))
		}
		out.WriteString(logger.PrettyJSON(ast))
		return nil
	}

	if conf.DoAst {
		b, err := mqlc.Compile(conf.Command, nil, mqlc.NewConfig(info.Registry.Schema(), conf.Features))
		if err != nil {
			return errors.Join(err, errors.New("failed to compile command"))
		}

		out.WriteString(logger.PrettyJSON((b)) + "\n" + printer.DefaultPrinter.CodeBundle(b))

		return nil
	}

	log.Info().Msgf("discover related assets for %d asset(s)", len(conf.Inventory.Spec.Assets))
	im, err := inventory.New(inventory.WithInventory(conf.Inventory))
	if err != nil {
		return errors.Join(err, errors.New("could not load asset information"))
	}
	assetErrors := im.Resolve(ctx)
	if len(assetErrors) > 0 {
		for a := range assetErrors {
			log.Error().Err(assetErrors[a]).Str("asset", a.Name).Msg("could not resolve asset")
		}
	}

	assetList := im.GetAssets()
	if len(assetList) == 0 {
		return errors.New("could not find an asset that we can connect to")
	}

	filteredAssets := []*asset.Asset{}
	if len(assetList) > 1 && conf.PlatformId != "" {
		filteredAsset, err := filterAssetByPlatformID(assetList, conf.PlatformId)
		if err != nil {
			return err
		}
		filteredAssets = append(filteredAssets, filteredAsset)
	} else {
		filteredAssets = assetList
	}

	if conf.DoRecord {
		log.Info().Msg("enable recording of platform calls")
	}

	if conf.Format == "json" {
		out.WriteString("[")
	}

	var upstreamConfig *resources.UpstreamConfig
	serviceAccount := opts.GetServiceCredential()
	if serviceAccount != nil {
		certAuth, err := upstream.NewServiceAccountRangerPlugin(serviceAccount)
		if err != nil {
			log.Error().Err(err).Msg("could not initialize client authentication")
			os.Exit(ConfigurationErrorCode)
		}

		upstreamConfig = &resources.UpstreamConfig{
			// we currently do not expose incognito to the plugin/run command
			Incognito:   true,
			SpaceMrn:    opts.GetParentMrn(),
			ApiEndpoint: opts.UpstreamApiEndpoint(),
			Plugins:     []ranger.ClientPlugin{certAuth},
			HttpClient:  ranger.DefaultHttpClient(),
		}
	}

	for i := range filteredAssets {
		connectAsset := filteredAssets[i]
		m, err := provider_resolver.OpenAssetConnection(ctx, connectAsset, im.GetCredsResolver(), conf.DoRecord)
		if err != nil {
			return errors.New("could not connect to asset")
		}

		// when we close the shell, we need to close the backend and store the recording
		onCloseHandler := func() {
			m.StoreRecording(viper.GetString("record-file"))
		}

		shellOptions := []shell.ShellOption{}
		shellOptions = append(shellOptions, shell.WithOnCloseListener(onCloseHandler))
		shellOptions = append(shellOptions, shell.WithFeatures(conf.Features))
		shellOptions = append(shellOptions, shell.WithOutput(out))

		if upstreamConfig != nil {
			shellOptions = append(shellOptions, shell.WithUpstreamConfig(upstreamConfig))
		}

		sh, err := shell.New(m, shellOptions...)
		if err != nil {
			return errors.Join(err, errors.New("failed to initialize the shell"))
		}
		defer sh.Close()

		code, results, err := sh.RunOnce(conf.Command)
		if err != nil {
			return errors.Join(err, errors.New("failed to run"))
		}

		if conf.Format != "json" {
			sh.PrintResults(code, results)
		} else {
			reporter.BundleResultsToJSON(code, results, out)
			if len(filteredAssets) != i+1 {
				out.WriteString(",")
			}
		}

	}

	if conf.Format == "json" {
		out.WriteString("]")
	}

	return nil
}
