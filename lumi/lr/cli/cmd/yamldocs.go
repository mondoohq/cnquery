package cmd

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.io/mondoo/lumi/lr"
	"io/ioutil"
	"os"
	"sigs.k8s.io/yaml"
	"sort"
)

func init() {
	yamlDocsCmd.Flags().String("file", "", "optional file path to write content to a file")
	rootCmd.AddCommand(yamlDocsCmd)
}

type LrDocs struct {
	Resources map[string]*LrDocsEntry `json:"resources,omitempty"`
}

type LrDocsEntry struct {
	// TODO: this is just an indicator, we may want to replace this with native lumi resource platform information
	Platforms []string        `json:"platforms,omitempty"`
	Snippets  []LrDocsSnippet `json:"snippets,omitempty"`
}

type LrDocsSnippet struct {
	Query string `json:"query,omitempty"`
	Title string `json:"title,omitempty"`
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

		data, err := yaml.Marshal(docs)
		if err != nil {
			log.Fatal().Err(err).Msg("could not marshal docs")
		}

		log.Info().Str("file", filepath).Msg("write file")
		err = ioutil.WriteFile(filepath, data, 0700)
		if err != nil {
			log.Fatal().Err(err).Msg("could not write docs file")
		}
	},
}
