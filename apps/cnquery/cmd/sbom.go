// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v10/cli/reporter"
	"go.mondoo.com/cnquery/v10/logger"
	"go.mondoo.com/cnquery/v10/providers"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	sbom "go.mondoo.com/cnquery/v10/sbom"
	"go.mondoo.com/cnquery/v10/shared"
)

func init() {
	rootCmd.AddCommand(sbomCmd)
	sbomCmd.Flags().String("asset-name", "", "User-override for the asset name.")
	sbomCmd.Flags().StringToString("annotation", nil, "Add an annotation to the asset.") // user-added, editable
	sbomCmd.Flags().StringP("output", "o", "list", "Set output format: "+sbom.AllFormats())
	sbomCmd.Flags().String("output-target", "", "Set output target to which the sbom report will be written.")
	sbomCmd.Flags().Bool("with-evidence", false, "Display evidence for each component")
}

var sbomCmd = &cobra.Command{
	Use:   "sbom",
	Short: "Experimental: Generate a software bill of materials (SBOM) for a given asset",
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

		err = viper.BindPFlag("output-target", cmd.Flags().Lookup("output-target"))
		if err != nil {
			log.Fatal().Err(err).Msg("failed to bind output-target flag")
		}

		err = viper.BindPFlag("with-evidence", cmd.Flags().Lookup("with-evidence"))
		if err != nil {
			log.Fatal().Err(err).Msg("failed to bind with-evidence flag")
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

	conf, err := getCobraScanConfig(cmd, runtime, cliRes)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get scan config")
	}

	conf.QueryPackNames = nil
	conf.QueryPackPaths = nil
	conf.Bundle = pb
	conf.IsIncognito = true

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

	boms, err := sbom.NewBom(buf.Bytes())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse bom")
	}

	var exporter sbom.Exporter
	output := viper.GetString("output")
	exporter = sbom.NewExporter(output)
	if exporter == nil {
		log.Fatal().Err(err).Msg("failed to get exporter for output format: " + output)
	}

	if viper.GetBool("with-evidence") {
		x, ok := exporter.(*sbom.TextList)
		if ok {
			x.ApplyOptions(sbom.WithEvidence())
		}
	}

	outputTarget := viper.GetString("output-target")
	for i := range boms {
		bom := boms[i]
		output := bytes.Buffer{}
		err := exporter.Render(&output, &bom)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to render SBOM")
		}

		if outputTarget != "" {
			filename := outputTarget
			if len(boms) > 1 {
				filename = fmt.Sprintf("%s-%d.%s", path.Base(outputTarget), i, path.Ext(outputTarget))
			}
			err := os.WriteFile(filename, output.Bytes(), 0o600)
			if err != nil {
				log.Fatal().Err(err).Msg("failed to write SBOM to file")
			}
		} else {
			fmt.Println(output.String())
		}
	}
}
