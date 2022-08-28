package cmd

import (
	"encoding/json"
	"os"
	"path"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnquery/resources/lr"
)

var printStdout = false

var goCmd = &cobra.Command{
	Use:   "go",
	Short: "convert LR file to go",
	Long:  `parse an LR file and convert it to go, saving it in the same location with the suffix .go`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		file := args[0]
		folder := path.Dir(file)
		packageName := path.Base(folder)

		res, err := lr.Resolve(file, func(path string) ([]byte, error) {
			return os.ReadFile(path)
		})
		if err != nil {
			log.Error().Err(err)
			return
		}

		collector := lr.NewCollector(args[0])
		godata, err := lr.Go(packageName, res, collector)
		if err != nil {
			log.Error().Err(err).Msg("failed to compile go code")
		}

		err = os.WriteFile(args[0]+".go", []byte(godata), 0o644)
		if err != nil {
			log.Error().Err(err).Msg("failed to write to go file")
		}

		schema, err := lr.Schema(res)
		if err != nil {
			log.Error().Err(err).Msg("failed to generate schema")
		}

		schemaData, err := json.Marshal(schema)
		if err != nil {
			log.Error().Err(err).Msg("failed to generate schema json")
		}

		infoFolder := path.Join(folder, "info")
		if err = os.MkdirAll(infoFolder, 0o755); err != nil {
			log.Fatal().Err(err).Str("folder", infoFolder).Msg("failed to ensure info folder")
		}

		infoFile := path.Join(infoFolder, args[0]+".json")
		err = os.WriteFile(infoFile, []byte(schemaData), 0o644)
		if err != nil {
			log.Error().Err(err).Msg("failed to write schema json")
		}
	},
}

func init() {
	rootCmd.AddCommand(goCmd)
}
