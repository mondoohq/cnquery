// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"context"
	"os"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/go-plugin"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnquery/v11/cli/config"
	"go.mondoo.com/cnquery/v11/cli/printer"
	"go.mondoo.com/cnquery/v11/cli/reporter"
	"go.mondoo.com/cnquery/v11/cli/shell"
	"go.mondoo.com/cnquery/v11/explorer/scan"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/logger"
	"go.mondoo.com/cnquery/v11/mqlc"
	"go.mondoo.com/cnquery/v11/mqlc/parser"
	"go.mondoo.com/cnquery/v11/providers"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/recording"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/v11/shared"
	run "go.mondoo.com/cnquery/v11/shared/proto"
	"go.mondoo.com/cnquery/v11/utils/iox"
	"google.golang.org/protobuf/proto"
)

// pluginCmd represents the version command
var pluginCmd = &cobra.Command{
	Use:    "run_as_plugin",
	Hidden: true,
	Short:  "Run as a plugin",
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

func (c *cnqueryPlugin) RunQuery(conf *run.RunQueryConfig, runtime *providers.Runtime, out iox.OutputHelper) error {
	if conf.Command == "" && conf.Input == "" {
		return errors.New("No command provided, nothing to do.")
	}

	opts, optsErr := config.Read()
	if optsErr != nil {
		log.Fatal().Err(optsErr).Msg("could not load configuration")
	}
	conf.Features = opts.GetFeatures()

	config.DisplayUsedConfig()

	if conf.DoParse {
		ast, err := parser.Parse(conf.Command)
		if err != nil {
			return errors.Wrap(err, "failed to parse command")
		}
		out.WriteString(logger.PrettyJSON(ast))
		return nil
	}

	if conf.DoAst {
		b, err := mqlc.Compile(conf.Command, nil, mqlc.NewConfig(runtime.Schema(), conf.Features))
		if err != nil {
			return errors.Wrap(err, "failed to compile command")
		}

		out.WriteString(logger.PrettyJSON((b)) + "\n" + printer.DefaultPrinter.CodeBundle(b))
		return nil
	}

	if conf.DoInfo {
		ast, err := parser.Parse(mqlc.Dedent(conf.Command))
		if ast == nil {
			return errors.Wrap(err, "failed to parse command")
		}

		conf := mqlc.NewConfig(runtime.Schema(), conf.Features)
		conf.EnableStats()
		_, err = mqlc.CompileAST(ast, nil, conf)
		if err != nil {
			return errors.Wrap(err, "failed to compile command")
		}

		out.WriteString(printer.DefaultPrinter.CompilerStats(conf.Stats))
		return nil
	}

	var upstreamConfig *upstream.UpstreamConfig
	serviceAccount := opts.GetServiceCredential()
	if serviceAccount != nil {
		upstreamConfig = &upstream.UpstreamConfig{
			SpaceMrn:    opts.GetParentMrn(),
			ApiEndpoint: opts.UpstreamApiEndpoint(),
			ApiProxy:    opts.APIProxy,
			Incognito:   conf.Incognito,
			Creds:       serviceAccount,
		}
	}

	ctx := context.Background()
	discoveredAssets, err := scan.DiscoverAssets(ctx, conf.Inventory, upstreamConfig, runtime.Recording())
	if err != nil {
		return err
	}
	if conf.Format == "json" {
		out.WriteString("[")
	}

	// anyResultFailed is a flag that will be switched on if any query result failed,
	// if the flag `exit-1-on-failure` is provided and anyResultFailed is true, we
	// will exit the program with the exit code `1`
	anyResultFailed := false
	// we defer this check since we want it to be the last thing to be evaluated
	defer func() {
		if conf.GetExit_1OnFailure() && anyResultFailed {
			os.Exit(1)
		}
	}()

	for i := range discoveredAssets.Assets {
		asset := discoveredAssets.Assets[i]

		if asset.Asset.Connections[0].DelayDiscovery {
			discoveredAsset, err := scan.HandleDelayedDiscovery(ctx, asset.Asset, asset.Runtime, nil, "")
			if err != nil {
				log.Error().Err(err).Str("asset", asset.Asset.Name).Msg("failed to handle delayed discovery for asset")
				continue
			}
			asset.Asset = discoveredAsset
		}

		// when we close the shell, we need to close the backend and store the recording
		onCloseHandler := func() {
			// FIXME: store recording
			// m.StoreRecording(viper.GetString("record-file"))
		}

		shellOptions := []shell.ShellOption{}
		shellOptions = append(shellOptions, shell.WithOnCloseListener(onCloseHandler))
		shellOptions = append(shellOptions, shell.WithFeatures(conf.Features))
		shellOptions = append(shellOptions, shell.WithOutput(out))

		if upstreamConfig != nil {
			shellOptions = append(shellOptions, shell.WithUpstreamConfig(upstreamConfig))
		}

		sh, err := shell.New(asset.Runtime, shellOptions...)
		if err != nil {
			return errors.Wrap(err, "failed to initialize the shell")
		}
		defer func() {
			// prevent the recording from being closed multiple times
			err = asset.Runtime.SetRecording(recording.Null{})
			if err != nil {
				log.Error().Err(err).Msg("failed to set the recording layer to null")
			}
			sh.Close()
		}()

		var code *llx.CodeBundle
		var results map[string]*llx.RawResult
		if conf.Input != "" {
			var raw []byte
			raw, err = os.ReadFile(conf.Input)
			if err != nil {
				return errors.Wrap(err, "failed to read code bundle from file")
			}
			var b llx.CodeBundle
			if err = proto.Unmarshal(raw, &b); err != nil {
				return errors.Wrap(err, "failed to unmarshal code bundle")
			}
			code = &b
			results, err = sh.RunOnceBundle(code)
		} else {
			code, results, err = sh.RunOnce(conf.Command)
		}
		if err != nil {
			return errors.Wrap(err, "failed to run")
		}

		// check if any result failed
		for _, result := range results {
			if result == nil || result.Data == nil {
				continue
			}

			if truthy, ok := result.Data.IsTruthy(); ok && !truthy {
				anyResultFailed = true
			}
		}

		if conf.Format == "llx" && conf.Output != "" {
			out, err := code.MarshalVT()
			if err != nil {
				return errors.Wrap(err, "failed to marshal code bundle")
			}
			err = os.WriteFile(conf.Output, out, 0o644)
			if err != nil {
				return errors.Wrap(err, "failed to save code bundle")
			}
			return nil
		}

		if conf.Format != "json" {
			sh.PrintResults(code, results)
		} else {
			_ = reporter.CodeBundleToJSON(code, results, out)
			if len(discoveredAssets.Assets) != i+1 {
				out.WriteString(",")
			}
		}
	}

	if conf.Format == "json" {
		out.WriteString("]")
	}

	return nil
}
