package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.io/mondoo/lumi/lr"
	"sigs.k8s.io/yaml"
)

func init() {
	yamlDocsCmd.Flags().String("file", "", "optional file path to write content to a file")
	rootCmd.AddCommand(yamlDocsCmd)
}

type LrDocs struct {
	Resources map[string]*LrDocsEntry `json:"resources,omitempty"`
}

type LrDocsEntry struct {
	// Maturity of the resource: experimental, preview, public, deprecated
	// default maturity is public if nothing is provided
	Maturity string `json:"maturity,omitempty"`
	// this is just an indicator, we may want to replace this with native lumi resource platform information
	Platform  *LrDocsPlatform      `json:"platform,omitempty"`
	Docs      *LrDocsDocumentation `json:"docs,omitempty"`
	Resources []LrDocsRefs         `json:"resources ,omitempty"`
	Refs      []LrDocsRefs         `json:"refs,omitempty"`
	Snippets  []LrDocsSnippet      `json:"snippets,omitempty"`
}

type LrDocsPlatform struct {
	Name    []string `json:"name,omitempty"`
	Familiy []string `json:"family,omitempty"`
}

type LrDocsDocumentation struct {
	Description string `json:"desc,omitempty"`
}

type LrDocsRefs struct {
	Title string `json:"title,omitempty"`
	Url   string `json:"url,omitempty"`
}

type LrDocsSnippet struct {
	Title string `json:"title,omitempty"`
	Query string `json:"query,omitempty"`
}

var yamlDocsCmd = &cobra.Command{
	Use:   "yamldocs",
	Short: "generates yaml docs skeleton file",
	Long:  `parse an LR file and generates a yaml file structure for additional documentation.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		raw, err := ioutil.ReadFile(args[0])
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

		filepath, err := cmd.Flags().GetString("file")
		if err != nil {
			log.Fatal().Err(err).Msg("invalid argument for `file`")
		}

		docs := LrDocs{
			Resources: map[string]*LrDocsEntry{},
		}

		for i := range res.Resources {
			id := res.Resources[i].ID
			docs.Resources[id] = nil
		}

		// default behaviour is to output the result on cli
		if filepath == "" {
			data, err := yaml.Marshal(docs)
			if err != nil {
				log.Fatal().Err(err).Msg("could not marshal docs")
			}

			fmt.Println(string(data))
			return
		}

		// if an file was provided, we check if the file exist and merge existing content with the new resources
		// to ensure that existing documentation stays available
		var existingData LrDocs
		_, err = os.Stat(filepath)
		if err == nil {
			log.Info().Msg("load existing data")
			content, err := ioutil.ReadFile(filepath)
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
				docs.Resources[k] = v
			}
		}

		// ensure default values are set
		for k := range docs.Resources {
			docs.Resources[k] = ensureDefaults(k, docs.Resources[k])
		}

		data, err := yaml.Marshal(docs)
		if err != nil {
			log.Fatal().Err(err).Msg("could not marshal docs")
		}

		log.Info().Str("file", filepath).Msg("write file")
		err = ioutil.WriteFile(filepath, data, 0o700)
		if err != nil {
			log.Fatal().Err(err).Msg("could not write docs file")
		}
	},
}

var platformMapping = map[string][]string{
	"aws":     {"aws"},
	"gcp":     {"gcloud"},
	"k8s":     {"kubernetes"},
	"azure":   {"azure"},
	"azurerm": {"azure"},
	"arista":  {"arista-eos"},
	"equinix": {"equinix"},
	"ms365":   {"microsoft365"},
	"msgraph": {"microsoft365"},
	"vsphere": {"vmware-esxi", "vmware-vsphere"},
	"esxi":    {"vmware-esxi", "vmware-vsphere"},
}

func ensureDefaults(id string, entry *LrDocsEntry) *LrDocsEntry {
	for k := range platformMapping {
		if strings.HasPrefix(id, k) {
			if entry == nil {
				entry = &LrDocsEntry{}
			}

			entry.Platform = &LrDocsPlatform{
				Name: platformMapping[k],
			}
		}
	}

	return entry
}
