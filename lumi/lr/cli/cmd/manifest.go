package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.io/mondoo/lumi/lr"
	"sigs.k8s.io/yaml"
)

func init() {
	manifestCmd.Flags().String("file", "", "optional file path to write content to a file")
	manifestCmd.Flags().String("version", "latest", "optional version to mark resource, default is latest")
	rootCmd.AddCommand(manifestCmd)
}

type LrInventory struct {
	Resources map[string]string `json:"resources,omitempty"`
}

var manifestCmd = &cobra.Command{
	Use:   "manifest",
	Short: "generates yaml manifest of resources",
	Long:  `parse an LR file and generates a yaml file structure for resource versioning inventory`,
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

		filepath, err := cmd.Flags().GetString("file")
		if err != nil {
			log.Fatal().Err(err).Msg("invalid argument for `file`")
		}

		docs := LrInventory{
			Resources: map[string]string{},
		}

		// to ensure we generate the same markdown, we sort the resources first
		sort.SliceStable(res.Resources, func(i, j int) bool {
			return res.Resources[i].ID < res.Resources[j].ID
		})

		version, err := cmd.Flags().GetString("version")
		if err != nil {
			log.Fatal().Err(err).Msg("invalid argument for `version`")
		}

		for _, r := range res.Resources {
			docs.Resources[r.ID] = version
			for _, f := range r.Body.Fields {
				key := r.ID + "." + f.ID
				docs.Resources[key] = version
			}
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
		var existingData LrInventory
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
				//keep older version information
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
