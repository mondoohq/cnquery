// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"github.com/cockroachdb/errors"
	"github.com/hashicorp/go-plugin"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnquery/cli/config"
	"go.mondoo.com/cnquery/cli/printer"
	"go.mondoo.com/cnquery/cli/reporter"
	"go.mondoo.com/cnquery/cli/shell"
	"go.mondoo.com/cnquery/logger"
	"go.mondoo.com/cnquery/mqlc"
	"go.mondoo.com/cnquery/mqlc/parser"
	"go.mondoo.com/cnquery/providers"
	pp "go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/shared"
	run "go.mondoo.com/cnquery/shared/proto"
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

func (c *cnqueryPlugin) RunQuery(conf *run.RunQueryConfig, runtime *providers.Runtime, out shared.OutputHelper) error {
	if conf.Command == "" {
		return errors.New("No command provided, nothing to do.")
	}

	opts, optsErr := config.Read()
	if optsErr != nil {
		log.Fatal().Err(optsErr).Msg("could not load configuration")
	}

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

	var upstreamConfig *upstream.UpstreamConfig
	serviceAccount := opts.GetServiceCredential()
	if serviceAccount != nil {
		upstreamConfig = &upstream.UpstreamConfig{
			SpaceMrn:    opts.GetParentMrn(),
			ApiEndpoint: opts.UpstreamApiEndpoint(),
			Incognito:   conf.Incognito,
			Creds:       serviceAccount,
		}
	}

	err := runtime.Connect(&pp.ConnectReq{
		Features: config.Features,
		Asset:    conf.Inventory.Spec.Assets[0],
		Upstream: upstreamConfig,
	})
	if err != nil {
		return err
	}

	if conf.Format == "json" {
		out.WriteString("[")
	}

	// FIXME: workaround for gcp-snapshot
	// For a gcp-snapshot asset, we start with a GCP connection.
	// This get's overridden by a filesystem connection. The filesystem connection is what we need for the scan
	// But later, we need the GCP runtime to cleanup the snapshot disk
	if runtime.Provider.Instance.Name == "gcp" && runtime.Provider.Connection.Name == "filesystem" {
		defer runtime.Close()
	}

	assets, err := providers.ProcessAssetCandidates(runtime, runtime.Provider.Connection, upstreamConfig, conf.PlatformId)
	if err != nil {
		return err
	}

	for i := range assets {
		connectAsset := assets[i]
		connectAssetRuntime := providers.Coordinator.NewRuntime()
		connectAssetRuntime.Recording = runtime.Recording

		if err := connectAssetRuntime.DetectProvider(connectAsset); err != nil {
			return err
		}
		err := connectAssetRuntime.Connect(&pp.ConnectReq{
			Features: config.Features,
			Asset:    connectAsset,
			Upstream: upstreamConfig,
		})
		if err != nil {
			return err
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

		sh, err := shell.New(connectAssetRuntime, shellOptions...)
		if err != nil {
			return errors.Wrap(err, "failed to initialize the shell")
		}
		defer func() {
			// prevent the recording from being closed multiple times
			connectAssetRuntime.Recording = providers.NullRecording{}
			sh.Close()
		}()

		code, results, err := sh.RunOnce(conf.Command)
		if err != nil {
			return errors.Wrap(err, "failed to run")
		}

		if conf.Format != "json" {
			sh.PrintResults(code, results)
		} else {
			reporter.BundleResultsToJSON(code, results, out)
			if len(assets) != i+1 {
				out.WriteString(",")
			}
		}

	}

	if conf.Format == "json" {
		out.WriteString("]")
	}

	return nil
}
