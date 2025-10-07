// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"encoding/json"
	"go/format"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/mqlr/lrcore"
	"sigs.k8s.io/yaml"
)

var goCmd = &cobra.Command{
	Use:   "go",
	Short: "convert LR file to go",
	Long:  `parse an LR file and convert it to go, saving it in the same location with the suffix .go`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dist, err := cmd.Flags().GetString("dist")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to get dist flag")
		}

		failOnDups, _ := cmd.Flags().GetBool("fail-on-duplicates")

		file := args[0]
		packageName := path.Base(path.Dir(file))

		res, err := lrcore.Resolve(file, func(path string) ([]byte, error) {
			return os.ReadFile(path)
		})
		if err != nil {
			log.Fatal().Err(err).Msg("failed to resolve")
			return
		}

		dups := res.GetDuplicates()
		if failOnDups && len(dups) > 0 {
			log.Fatal().Int("count", len(dups)).Strs("paths", dups).Msg("duplicate field paths detected, exiting")
		} else if len(dups) > 0 {
			log.Warn().Int("count", len(dups)).Strs("paths", dups).Msg("duplicate field paths detected")
		}

		// add license header
		var headerTpl *template.Template
		if headerFile, err := cmd.Flags().GetString("license-header-file"); err == nil && headerFile != "" {
			headerRaw, err := os.ReadFile(headerFile)
			if err != nil {
				log.Fatal().Err(err).Msg("could not read license header file")
			}
			headerTpl, err = template.New("license_header").Parse(string(headerRaw))
			if err != nil {
				log.Fatal().Err(err).Msg("could not parse license header template")
			}
		}

		collector := lrcore.NewCollector(args[0])
		goCode, err := lrcore.Go(packageName, res, collector, headerTpl)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to compile go code")
		}

		fmtGoData, err := format.Source([]byte(goCode))
		if err != nil {
			log.Fatal().Err(err).Msg("failed to format go code")
		}
		err = os.WriteFile(args[0]+".go", fmtGoData, 0o644)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to write to go file")
		}

		schema, err := lrcore.Schema(res)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to generate schema")
		}

		// we will attempt to auto-detect the manifest to inject some metadata
		// into the schema
		manifestPath := file + ".manifest.yaml"
		raw, err := os.ReadFile(manifestPath)
		if err == nil {
			var lrDocsData lrcore.LrDocs
			err = yaml.Unmarshal(raw, &lrDocsData)
			if err != nil {
				log.Fatal().Err(err).Msg("could not load yaml data")
			}

			lrcore.InjectMetadata(schema, &lrDocsData)
		} else if os.IsNotExist(err) {
			log.Info().Str("path", manifestPath).Msg("no manifest found, ignoring")
		} else {
			log.Fatal().Err(err).Str("path", manifestPath).Msg("failed to read manifest")
		}

		schemaData, err := json.Marshal(schema)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to generate schema json")
		}

		base := path.Base(args[0])
		base = strings.TrimSuffix(base, ".lr")

		dst := strings.TrimSuffix(file, ".lr") + ".resources.json"
		err = os.WriteFile(dst, []byte(schemaData), 0o644)
		if err != nil {
			log.Fatal().Err(err).Str("dst", dst).Msg("failed to write schema json")
		}

		if dist != "" {
			if err = os.MkdirAll(dist, 0o755); err != nil {
				log.Fatal().Err(err).Msg("failed to create dist folder")
			}
			infoFile := path.Join(dist, base+".resources.json")
			err = os.WriteFile(infoFile, []byte(schemaData), 0o644)
			if err != nil {
				log.Fatal().Err(err).Str("dst", infoFile).Msg("failed to write schema json")
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(goCmd)
	goCmd.Flags().Bool("fail-on-duplicates", false, "fail if duplicate LR field paths are detected")
	goCmd.Flags().String("dist", "", "folder for output json generation")
	goCmd.Flags().String("license-header-file", "", "optional file path to read license header from")
}
