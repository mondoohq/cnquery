package cmd

import (
	"encoding/json"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.io/mondoo/lumi/lr"
)

var printStdout = false

var goCmd = &cobra.Command{
	Use:   "go",
	Short: "convert LR file to go",
	Long:  `parse an LR file and convert it to go, saving it in the same location with the suffix .go`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		file := args[0]
		raw, err := os.ReadFile(file)
		if err != nil {
			log.Error().Err(err)
			return
		}

		res, err := lr.Parse(string(raw))
		if err != nil {
			log.Error().Err(err).Msg("failed to parse LR code")
			return
		}

		collector := lr.NewCollector(args[0])
		godata, err := lr.Go(res, collector)
		if err != nil {
			log.Error().Err(err).Msg("failed to compile go code")
		}

		err = os.WriteFile(args[0]+".go", []byte(godata), 0o644)
		if err != nil {
			log.Error().Err(err).Msg("failed to write to go file")
		}

		schema, err := lr.Schema(res, collector)
		if err != nil {
			log.Error().Err(err).Msg("failed to generate schema")
		}

		schemaData, err := json.Marshal(schema)
		if err != nil {
			log.Error().Err(err).Msg("failed to generate schema json")
		}

		err = os.WriteFile(args[0]+".json", []byte(schemaData), 0o644)
		if err != nil {
			log.Error().Err(err).Msg("failed to write schema json")
		}
	},
}

func init() {
	rootCmd.AddCommand(goCmd)
}
