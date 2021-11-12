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
	"go.mondoo.io/mondoo/lumi/lr/docs"
	"sigs.k8s.io/yaml"
)

func init() {
	docsYamlCmd.Flags().String("file", "", "optional file path to write content to a file")
	docsCmd.AddCommand(docsYamlCmd)
	docsCmd.AddCommand(docsGoCmd)
	rootCmd.AddCommand(docsCmd)
}

var docsCmd = &cobra.Command{
	Use: "docs",
}

var docsYamlCmd = &cobra.Command{
	Use:   "yaml",
	Short: "generates yaml docs skeleton file and merges it into existing defintion",
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

		d := docs.LrDocs{
			Resources: map[string]*docs.LrDocsEntry{},
		}

		for i := range res.Resources {
			id := res.Resources[i].ID
			d.Resources[id] = nil
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
				d.Resources[k] = v
			}
		}

		// ensure default values are set
		for k := range d.Resources {
			d.Resources[k] = ensureDefaults(k, d.Resources[k])
		}

		// generate content
		data, err := yaml.Marshal(d)
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

func ensureDefaults(id string, entry *docs.LrDocsEntry) *docs.LrDocsEntry {
	for k := range platformMapping {
		if strings.HasPrefix(id, k) {
			if entry == nil {
				entry = &docs.LrDocsEntry{}
			}

			entry.Platform = &docs.LrDocsPlatform{
				Name: platformMapping[k],
			}
		}
	}

	return entry
}

var docsGoCmd = &cobra.Command{
	Use:   "go",
	Short: "convert yaml docs file to go",
	Long:  `parse an yaml docs file and convert it to go, saving it in the same location with the suffix .go`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		raw, err := ioutil.ReadFile(args[0])
		if err != nil {
			log.Error().Err(err)
			return
		}

		var lrDocsData docs.LrDocs
		err = yaml.Unmarshal(raw, &lrDocsData)
		if err != nil {
			log.Fatal().Err(err).Msg("could not load yaml data")
		}

		godata := docs.Go(lrDocsData)

		if printStdout {
			fmt.Println(godata)
		} else {
			filename := strings.TrimSuffix(args[0], ".yaml") + ".go"
			err = ioutil.WriteFile(filename, []byte(godata), 0o644)
			if err != nil {
				log.Error().Err(err).Msg("failed to write to go file")
			}
		}
	},
}
