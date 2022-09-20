package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/cli/assetlist"
	"go.mondoo.com/cnquery/cli/printer"
	"go.mondoo.com/cnquery/cli/shell"
	"go.mondoo.com/cnquery/logger"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery"
	"go.mondoo.com/cnquery/motor/inventory"
	v1 "go.mondoo.com/cnquery/motor/inventory/v1"
	provider_resolver "go.mondoo.com/cnquery/motor/providers/resolver"
	"go.mondoo.com/cnquery/mqlc"
	"go.mondoo.com/cnquery/mqlc/parser"
	"go.mondoo.com/cnquery/resources/packs/os/info"
)

type RunQueryConfig struct {
	Command   string
	Inventory *v1.Inventory
	Features  cnquery.Features

	DoParse    bool
	DoAST      bool
	DoRecord   bool
	Format     string
	PlatformID string
}

func RunQuery(conf *RunQueryConfig, out io.Writer) error {
	if conf.Command == "" {
		return errors.New("No command provided, nothing to do.")
	}

	ctx := discovery.InitCtx(context.Background())

	if conf.DoParse {
		ast, err := parser.Parse(conf.Command)
		if err != nil {
			return errors.Wrap(err, "failed to parse command")
		}
		out.Write([]byte(logger.PrettyJSON(ast)))
		return nil
	}

	if conf.DoAST {
		b, err := mqlc.Compile(conf.Command, info.Registry.Schema(), conf.Features, nil)
		if err != nil {
			return errors.Wrap(err, "failed to compile command")
		}

		out.Write([]byte(logger.PrettyJSON((b))))
		out.Write([]byte{'\n'})
		out.Write([]byte(printer.DefaultPrinter.CodeBundle(b)))

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

	var connectAsset *asset.Asset

	if len(assetList) == 1 {
		connectAsset = assetList[0]
	} else if len(assetList) > 1 && conf.PlatformID != "" {
		connectAsset, err = filterAssetByPlatformID(assetList, conf.PlatformID)
		if err != nil {
			return err
		}
	} else if len(assetList) > 1 {
		r := &assetlist.SimpleRender{}
		out.Write([]byte(r.Render(assetList)))
		out.Write([]byte{'\n'})
		return errors.New("cannot connect to more than one asset, use --platform-id to select a specific asset")
	}

	if conf.DoRecord {
		log.Info().Msg("enable recording of platform calls")
	}

	m, err := provider_resolver.OpenAssetConnection(ctx, connectAsset, im.GetCredential, conf.DoRecord)
	if err != nil {
		return errors.New("could not connect to asset")
	}

	// when we close the shell, we need to close the backend and store the recording
	onCloseHandler := func() {
		storeRecording(m)

		// close backend connection
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

	fmt.Printf("%#v\n", conf.Command)
	code, results, err := sh.RunOnce(conf.Command)
	fmt.Printf("%#v\n", code)
	if err != nil {
		return errors.Wrap(err, "failed to run")
	}

	if conf.Format != "json" {
		sh.PrintResults(code, results)
		return nil
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
			return errors.New("cannot find result for this query")
		}

		if result.Data.Error != nil {
			return result.Data.Error
		}

		j := result.Data.JSON(checksum, code)
		out.Write(j)
		out.Write([]byte{'\n'})
	}

	return nil
}
