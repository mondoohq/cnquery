// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"bytes"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v10/cli/config"
	"go.mondoo.com/cnquery/v10/cli/execruntime"
	"go.mondoo.com/cnquery/v10/cli/inventoryloader"
	"go.mondoo.com/cnquery/v10/cli/reporter"
	"go.mondoo.com/cnquery/v10/logger"
	"go.mondoo.com/cnquery/v10/providers"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream"
	sbom "go.mondoo.com/cnquery/v10/sbom"
	"go.mondoo.com/cnquery/v10/shared"
)

func init() {
	rootCmd.AddCommand(sbomCmd)
	sbomCmd.Flags().String("asset-name", "", "User-override for the asset name")
	sbomCmd.Flags().StringToString("annotation", nil, "Add an annotation to the asset.") // user-added, editable
	sbomCmd.Flags().StringP("output", "o", "list", "Set output format: "+sbom.AllFormats())
}

var sbomCmd = &cobra.Command{
	Use:   "sbom",
	Short: "Experimental: Generate a software bill of materials (SBOM) for a given asset.",
	Long: `Generate a software bill of materials (SBOM) for a given asset. The SBOM
is a representation of the asset's software components and their dependencies.

The following formats are supported:
- list (default)
- cnquery-json
- cyclonedx-json
- cyclonedx-xml
- spdx-json
- spdx-tag-value

Note this command is experimental and may change in the future.
`,
	PreRun: func(cmd *cobra.Command, args []string) {
		err := viper.BindPFlag("output", cmd.Flags().Lookup("output"))
		if err != nil {
			log.Fatal().Err(err).Msg("failed to bind output flag")
		}
	},
	// we have to initialize an empty run so it shows up as a runnable command in --help
	Run: func(cmd *cobra.Command, args []string) {},
}

var sbomCmdRun = func(cmd *cobra.Command, runtime *providers.Runtime, cliRes *plugin.ParseCLIRes) {
	log.Info().Msg("This command is experimental. Please report any issues to https://github.com/mondoohq/cnquery.")
	pb, err := sbom.QueryPack()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load query pack")
	}

	conf, err := getSbomScanConfig(cmd, runtime, cliRes)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get scan config")
	}

	conf.QueryPackNames = nil
	conf.QueryPackPaths = nil
	conf.Bundle = pb

	report, err := RunScan(conf)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to run scan")
	}

	buf := bytes.Buffer{}
	w := shared.IOWriter{Writer: &buf}
	err = reporter.ReportCollectionToJSON(report, &w)
	if err == nil {
		logger.DebugDumpJSON("mondoo-sbom-report", buf.Bytes())
	}

	jr, err := sbom.NewReportCollectionJson(buf.Bytes())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse report collection")
	}

	boms, err := sbom.GenerateBom(jr)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to generate SBOM")
	}

	var exporter sbom.Exporter
	output := viper.GetString("output")
	exporter = sbom.NewExporter(output)
	if exporter == nil {
		log.Fatal().Err(err).Msg("failed to get exporter for output format: " + output)
	}

	for _, bom := range boms {
		output := bytes.Buffer{}
		err := exporter.Render(&output, &bom)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to render SBOM")
		}
		fmt.Println(output.String())
	}
}

// TODO: harmonize with getCobraScanConfig
func getSbomScanConfig(cmd *cobra.Command, runtime *providers.Runtime, cliRes *plugin.ParseCLIRes) (*scanConfig, error) {
	opts, err := config.Read()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	config.DisplayUsedConfig()

	annotations, err := cmd.Flags().GetStringToString("annotation")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse annotations")
	}

	// merge the config and the user-provided annotations with the latter having precedence
	optAnnotations := opts.Annotations
	if optAnnotations == nil {
		optAnnotations = map[string]string{}
	}
	for k, v := range annotations {
		optAnnotations[k] = v
	}

	assetName, err := cmd.Flags().GetString("asset-name")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse asset-name")
	}
	if assetName != "" && cliRes.Asset != nil {
		cliRes.Asset.Name = assetName
	}

	inv, err := inventoryloader.ParseOrUse(cliRes.Asset, viper.GetBool("insecure"), optAnnotations)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse inventory")
	}

	conf := scanConfig{
		Features:    opts.GetFeatures(),
		IsIncognito: true,
		Inventory:   inv,
		runtime:     runtime,
	}

	// detect CI/CD runs and read labels from runtime and apply them to all assets in the inventory
	runtimeEnv := execruntime.Detect()
	if opts.AutoDetectCICDCategory && runtimeEnv.IsAutomatedEnv() || opts.Category == "cicd" {
		log.Info().Msg("detected ci-cd environment")
		// NOTE: we only apply those runtime environment labels for CI/CD runs to ensure other assets from the
		// inventory are not touched, we may consider to add the data to the flagAsset
		if runtimeEnv != nil {
			runtimeLabels := runtimeEnv.Labels()
			conf.Inventory.ApplyLabels(runtimeLabels)
		}
		conf.Inventory.ApplyCategory(inventory.AssetCategory_CATEGORY_CICD)
	}

	var serviceAccount *upstream.ServiceAccountCredentials
	if !conf.IsIncognito {
		serviceAccount = opts.GetServiceCredential()
		if serviceAccount != nil {
			log.Info().Msg("using service account credentials")
			conf.runtime.UpstreamConfig = &upstream.UpstreamConfig{
				SpaceMrn:    opts.GetParentMrn(),
				ApiEndpoint: opts.UpstreamApiEndpoint(),
				ApiProxy:    opts.APIProxy,
				Incognito:   conf.IsIncognito,
				Creds:       serviceAccount,
			}
		} else {
			log.Warn().Msg("No credentials provided. Switching to --incognito mode.")
			conf.IsIncognito = true
		}
	}

	return &conf, nil
}
