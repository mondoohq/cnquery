// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/mqlr/lrcore"
	"sigs.k8s.io/yaml"
)

func init() {
	docsYamlCmd.Flags().String("docs-file", "", "optional file path to write content to a file")
	docsYamlCmd.Flags().String("version", defaultVersionField, "optional version to mark resource, default is latest")
	docsYamlCmd.Flags().String("license-header-file", "", "optional file path to read license header from")
	docsCmd.AddCommand(docsYamlCmd)
	docsJsonCmd.Flags().String("dist", "", "folder for output json generation")
	docsCmd.AddCommand(docsJsonCmd)
	rootCmd.AddCommand(docsCmd)
}

const defaultVersionField = "9.0.0"

var docsCmd = &cobra.Command{
	Use: "docs",
}

var docsYamlCmd = &cobra.Command{
	Use:   "yaml",
	Short: "generates yaml docs skeleton file and merges it into existing definition",
	Long:  `parse an LR file and generates a yaml file structure for additional documentation.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		lrFile := args[0]

		docsFilePath, err := cmd.Flags().GetString("docs-file")
		if err != nil {
			log.Fatal().Err(err).Msg("invalid argument for `docs-file`")
		}

		version, err := cmd.Flags().GetString("version")
		if err != nil {
			log.Fatal().Err(err).Msg("invalid argument for `version`")
		}

		headerFile, _ := cmd.Flags().GetString("license-header-file")

		runDocsYamlCmd(lrFile, headerFile, version, docsFilePath)
	},
}

var docsJsonCmd = &cobra.Command{
	Use:   "json",
	Short: "convert yaml docs manifest into json",
	Long:  `convert a yaml-based docs manifest into its json description, ready for loading`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		file := args[0]

		dist, err := cmd.Flags().GetString("dist")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to get dist flag")
		}

		runDocsJsonCmd(file, dist)
	},
}

func runDocsYamlCmd(lrFile string, headerFile string, version string, docsFilePath string) {
	raw, err := os.ReadFile(lrFile)
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}

	res, err := lrcore.Parse(string(raw))
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}

	// if an file was provided, we check if the file exist and merge existing content with the new resources
	// to ensure that existing documentation stays available
	var existingData lrcore.LrDocs
	_, err = os.Stat(docsFilePath)
	if err == nil {
		log.Info().Msg("load existing data")
		content, err := os.ReadFile(docsFilePath)
		if err != nil {
			log.Fatal().Err(err).Msg("could not read file " + docsFilePath)
		}
		err = yaml.Unmarshal(content, &existingData)
		if err != nil {
			log.Fatal().Err(err).Msg("could not load yaml data")
		}
	}

	docs, err := res.GenerateDocs(version, defaultVersionField, existingData)
	if err != nil {
		log.Fatal().Err(err).Msg("could not generate docs")
	}
	// default behaviour is to output the result on cli
	if docsFilePath == "" {
		data, err := yaml.Marshal(docs)
		if err != nil {
			log.Fatal().Err(err).Msg("could not marshal docs")
		}

		fmt.Println(string(data))
		return
	}

	// generate content
	data, err := yaml.Marshal(docs)
	if err != nil {
		log.Fatal().Err(err).Msg("could not marshal docs")
	}
	// add license header
	var headerTpl *template.Template
	if headerFile != "" {
		headerRaw, err := os.ReadFile(headerFile)
		if err != nil {
			log.Fatal().Err(err).Msg("could not read license header file")
		}
		headerTpl, err = template.New("license_header").Parse(string(headerRaw))
		if err != nil {
			log.Fatal().Err(err).Msg("could not parse license header template")
		}
	}

	header, err := lrcore.LicenseHeader(headerTpl, lrcore.LicenseHeaderOptions{LineStarter: "#"})
	if err != nil {
		log.Fatal().Err(err).Msg("could not generate license header")
	}
	data = append([]byte(header), data...)

	log.Info().Str("file", docsFilePath).Msg("write file")
	err = os.WriteFile(docsFilePath, data, 0o700)
	if err != nil {
		log.Fatal().Err(err).Msg("could not write docs file")
	}
}

func runDocsJsonCmd(yamlDocsFile string, dist string) {
	// without dist we want the file to be put alongside the original
	if dist == "" {
		src, err := filepath.Abs(yamlDocsFile)
		if err != nil {
			log.Fatal().Err(err).Msg("cannot figure out the absolute path for the source file")
		}
		dist = filepath.Dir(src)
	}

	raw, err := os.ReadFile(yamlDocsFile)
	if err != nil {
		log.Fatal().Err(err)
	}

	var lrDocsData lrcore.LrDocs
	err = yaml.Unmarshal(raw, &lrDocsData)
	if err != nil {
		log.Fatal().Err(err).Msg("could not load yaml data")
	}

	out, err := json.Marshal(&lrDocsData)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to convert yaml to json")
	}

	if err = os.MkdirAll(dist, 0o755); err != nil {
		log.Fatal().Err(err).Msg("failed to create dist folder")
	}
	infoFile := path.Join(dist, strings.TrimSuffix(path.Base(yamlDocsFile), ".yaml")+".json")
	err = os.WriteFile(infoFile, []byte(out), 0o644)
	if err != nil {
		log.Fatal().Err(err).Str("path", infoFile).Msg("failed to write to json file")
	}
}
