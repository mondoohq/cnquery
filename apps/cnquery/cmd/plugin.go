package cmd

import (
	"context"
	"encoding/json"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/go-plugin"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnquery/cli/printer"
	"go.mondoo.com/cnquery/cli/shell"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/logger"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery"
	"go.mondoo.com/cnquery/motor/inventory"
	provider_resolver "go.mondoo.com/cnquery/motor/providers/resolver"
	"go.mondoo.com/cnquery/mqlc"
	"go.mondoo.com/cnquery/mqlc/parser"
	"go.mondoo.com/cnquery/resources/packs/os/info"
	"go.mondoo.com/cnquery/shared"
	"go.mondoo.com/cnquery/shared/proto"
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

func (c *cnqueryPlugin) RunQuery(conf *proto.RunQueryConfig, out shared.OutputHelper) error {
	if conf.Command == "" {
		return errors.New("No command provided, nothing to do.")
	}

	ctx := discovery.InitCtx(context.Background())

	if conf.DoParse {
		ast, err := parser.Parse(conf.Command)
		if err != nil {
			return errors.Wrap(err, "failed to parse command")
		}
		out.WriteString(logger.PrettyJSON(ast))
		return nil
	}

	if conf.DoAst {
		b, err := mqlc.Compile(conf.Command, info.Registry.Schema(), conf.Features, nil)
		if err != nil {
			return errors.Wrap(err, "failed to compile command")
		}

		out.WriteString(logger.PrettyJSON((b)) + "\n" + printer.DefaultPrinter.CodeBundle(b))

		return nil
	}

	log.Info().Msgf("discover related assets for %d asset(s)", len(conf.Inventory.Spec.Assets))
	im, err := inventory.New(inventory.WithInventory(conf.Inventory))
	if err != nil {
		return errors.Wrap(err, "could not load asset information")
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

	for i := range filteredAssets {
		connectAsset := filteredAssets[i]
		m, err := provider_resolver.OpenAssetConnection(ctx, connectAsset, im.GetCredential, conf.DoRecord)
		if err != nil {
			return errors.New("could not connect to asset")
		}

		// when we close the shell, we need to close the backend and store the recording
		onCloseHandler := func() {
			storeRecording(m)
			m.Close()
		}

		shellOptions := []shell.ShellOption{}
		shellOptions = append(shellOptions, shell.WithOnCloseListener(onCloseHandler))
		shellOptions = append(shellOptions, shell.WithFeatures(conf.Features))
		shellOptions = append(shellOptions, shell.WithOutput(out))

		sh, err := shell.New(m, shellOptions...)
		if err != nil {
			return errors.Wrap(err, "failed to initialize the shell")
		}
		defer sh.Close()

		code, results, err := sh.RunOnce(conf.Command)
		if err != nil {
			return errors.Wrap(err, "failed to run")
		}

		if conf.Format != "json" {
			sh.PrintResults(code, results)
		} else {
			renderJson(code, results, out)
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

func renderJson(code *llx.CodeBundle, results map[string]*llx.RawResult, out shared.OutputHelper) error {
	var checksums []string
	eps := code.CodeV2.Entrypoints()
	checksums = make([]string, len(eps))
	for i, ref := range eps {
		checksums[i] = code.CodeV2.Checksums[ref]
	}

	// since we iterate over checksums, we run into the situation that this could be a slice
	// eg. cnquery run k8s --all-namespaces --query "platform { name } k8s.pod.name" --json

	renderError := func(err error) {
		data, jErr := json.Marshal(struct {
			Error string `json:"error"`
		}{Error: err.Error()})
		if jErr == nil {
			out.Write(data)
		} else {
			// this should never happen :-)
			log.Warn().Err(err).Send()
		}
	}

	if len(checksums) > 1 {
		out.WriteString("[")
	}

	for j, checksum := range checksums {
		result := results[checksum]
		if result == nil {
			renderError(errors.New("cannot find result for this query"))
		} else if result.Data.Error != nil {
			renderError(result.Data.Error)
		} else {
			jsonData := result.Data.JSON(checksum, code)
			out.Write(append(jsonData, '\n'))
		}

		if len(checksums) != j+1 {
			out.WriteString(",")
		}
	}

	if len(checksums) > 1 {
		out.WriteString("]")
	}
	return nil
}
