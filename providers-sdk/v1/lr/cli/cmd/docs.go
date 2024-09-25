// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/lr"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/lr/docs"
	"sigs.k8s.io/yaml"
)

func init() {
	docsYamlCmd.Flags().String("docs-file", "", "optional file path to write content to a file")
	docsYamlCmd.Flags().String("version", defaultVersionField, "optional version to mark resource, default is latest")
	docsYamlCmd.Flags().String("license-header-file", "", "optional file path to read license header from")
	docsCmd.AddCommand(docsYamlCmd)
	docsJSONCmd.Flags().String("dist", "", "folder for output json generation")
	docsCmd.AddCommand(docsJSONCmd)
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
		raw, err := os.ReadFile(args[0])
		if err != nil {
			log.Error().Msg(err.Error())
			return
		}

		res, err := lr.Parse(string(raw))
		if err != nil {
			log.Error().Msg(err.Error())
			return
		}

		// to ensure we generate the same markdown, we sort the resources first
		sort.SliceStable(res.Resources, func(i, j int) bool {
			return res.Resources[i].ID < res.Resources[j].ID
		})

		filepath, err := cmd.Flags().GetString("docs-file")
		if err != nil {
			log.Fatal().Err(err).Msg("invalid argument for `file`")
		}

		version, err := cmd.Flags().GetString("version")
		if err != nil {
			log.Fatal().Err(err).Msg("invalid argument for `version`")
		}

		d := docs.LrDocs{
			Resources: map[string]*docs.LrDocsEntry{},
		}

		fields := map[string][]*lr.BasicField{}
		isPrivate := map[string]bool{}
		for i := range res.Resources {
			id := res.Resources[i].ID
			isPrivate[id] = res.Resources[i].IsPrivate
			d.Resources[id] = nil
			if res.Resources[i].Body != nil {
				basicFields := []*lr.BasicField{}
				for _, f := range res.Resources[i].Body.Fields {
					if f.BasicField != nil {
						basicFields = append(basicFields, f.BasicField)
					}
				}
				fields[id] = basicFields
			}
		}

		// default behaviour is to output the result on cli
		if filepath == "" {
			data, err := yaml.Marshal(d)
			if err != nil {
				log.Fatal().Err(err).Msg("could not marshal docs")
			}

			fmt.Println(string(data))
			return
		}

		// if an file was provided, we check if the file exist and merge existing content with the new resources
		// to ensure that existing documentation stays available
		var existingData docs.LrDocs
		_, err = os.Stat(filepath)
		if err == nil {
			log.Info().Msg("load existing data")
			content, err := os.ReadFile(filepath)
			if err != nil {
				log.Fatal().Err(err).Msg("could not read file " + filepath)
			}
			err = yaml.Unmarshal(content, &existingData)
			if err != nil {
				log.Fatal().Err(err).Msg("could not load yaml data")
			}

			log.Info().Msg("merge content")
			for k := range existingData.Resources {
				v := existingData.Resources[k]
				d.Resources[k] = v
			}
		}

		// ensure default values and fields are set
		for k := range d.Resources {
			d.Resources[k] = ensureDefaults(k, d.Resources[k], version)
			mergeFields(version, d.Resources[k], fields[k])
			// Merge in other doc fields from core.lr
			d.Resources[k].IsPrivate = isPrivate[k]
		}

		// generate content
		data, err := yaml.Marshal(d)
		if err != nil {
			log.Fatal().Err(err).Msg("could not marshal docs")
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

		header, err := lr.LicenseHeader(headerTpl, lr.LicenseHeaderOptions{LineStarter: "#"})
		if err != nil {
			log.Fatal().Err(err).Msg("could not generate license header")
		}
		data = append([]byte(header), data...)

		log.Info().Str("file", filepath).Msg("write file")
		err = os.WriteFile(filepath, data, 0o700)
		if err != nil {
			log.Fatal().Err(err).Msg("could not write docs file")
		}
	},
}

// required to be before more detail platform to ensure the right mapping
var platformMappingKeys = []string{
	"aws", "gcp", "k8s", "azure", "azurerm", "arista", "equinix", "ms365", "msgraph", "vsphere", "esxi", "terraform", "terraform.state", "terraform.plan",
}

var platformMapping = map[string][]string{
	"aws":             {"aws"},
	"gcp":             {"gcp"},
	"k8s":             {"kubernetes"},
	"azure":           {"azure"},
	"azurerm":         {"azure"},
	"arista":          {"arista-eos"},
	"equinix":         {"equinix"},
	"ms365":           {"microsoft365"},
	"msgraph":         {"microsoft365"},
	"vsphere":         {"vmware-esxi", "vmware-vsphere"},
	"esxi":            {"vmware-esxi", "vmware-vsphere"},
	"terraform":       {"terraform-hcl"},
	"terraform.state": {"terraform-state"},
	"terraform.plan":  {"terraform-plan"},
}

func ensureDefaults(id string, entry *docs.LrDocsEntry, version string) *docs.LrDocsEntry {
	for _, k := range platformMappingKeys {
		if entry == nil {
			entry = &docs.LrDocsEntry{}
		}
		if entry.MinMondooVersion == "" {
			entry.MinMondooVersion = version
		} else if entry.MinMondooVersion == defaultVersionField && version != defaultVersionField {
			// Update to specified version if previously set to default
			entry.MinMondooVersion = version
		}
		if strings.HasPrefix(id, k) {
			entry.Platform = &docs.LrDocsPlatform{
				Name: platformMapping[k],
			}
		}
	}
	return entry
}

func mergeFields(version string, entry *docs.LrDocsEntry, fields []*lr.BasicField) {
	if entry == nil && len(fields) > 0 {
		entry = &docs.LrDocsEntry{}
		entry.Fields = map[string]*docs.LrDocsField{}
	} else if entry == nil {
		return
	} else if entry.Fields == nil {
		entry.Fields = map[string]*docs.LrDocsField{}
	}
	docFields := entry.Fields
	for _, f := range fields {
		if docFields[f.ID] == nil {
			fDoc := &docs.LrDocsField{
				MinMondooVersion: version,
			}
			entry.Fields[f.ID] = fDoc
		} else if entry.Fields[f.ID].MinMondooVersion == defaultVersionField && version != defaultVersionField {
			entry.Fields[f.ID].MinMondooVersion = version
		}
		// Scrub field version if same as resource
		if entry.Fields[f.ID].MinMondooVersion == entry.MinMondooVersion {
			entry.Fields[f.ID].MinMondooVersion = ""
		}
	}
}

func extractComments(raw []string) (string, string) {
	if len(raw) == 0 {
		return "", ""
	}

	for i := range raw {
		raw[i] = strings.Trim(raw[i][2:], " \t\n")
	}

	title, rest := raw[0], raw[1:]
	desc := strings.Join(rest, " ")

	return title, desc
}

var docsJSONCmd = &cobra.Command{
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

		// without dist we want the file to be put alongside the original
		if dist == "" {
			src, err := filepath.Abs(file)
			if err != nil {
				log.Fatal().Err(err).Msg("cannot figure out the absolute path for the source file")
			}
			dist = filepath.Dir(src)
		}

		raw, err := os.ReadFile(file)
		if err != nil {
			log.Fatal().Err(err)
		}

		var lrDocsData docs.LrDocs
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
		infoFile := path.Join(dist, strings.TrimSuffix(path.Base(args[0]), ".yaml")+".json")
		err = os.WriteFile(infoFile, []byte(out), 0o644)
		if err != nil {
			log.Fatal().Err(err).Str("path", infoFile).Msg("failed to write to json file")
		}
	},
}
