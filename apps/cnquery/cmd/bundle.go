package cmd

import (
	"context"
	_ "embed"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnquery/explorer"
)

func init() {
	// bundle init
	packBundlesCmd.AddCommand(queryPackInitCmd)

	// bundle validate
	packBundlesCmd.AddCommand(queryPackValidateCmd)

	rootCmd.AddCommand(packBundlesCmd)
}

var packBundlesCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Manage query packs",
}

//go:embed bundle_querypack-example.mql.yaml
var embedQueryPackTemplate []byte

var queryPackInitCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Creates an example query pack that can be used as a starting point. If no filename is provided, `example-pack.mql.yaml` us used",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := "example-pack.mql.yaml"
		if len(args) == 1 {
			name = args[0]
		}

		_, err := os.Stat(name)
		if err == nil {
			log.Fatal().Msgf("Query Pack '%s' already exists", name)
		}

		err = os.WriteFile(name, embedQueryPackTemplate, 0o640)
		if err != nil {
			log.Fatal().Err(err).Msgf("Could not write '%s'", name)
		}
		log.Info().Msgf("Example query pack file written to %s", name)
	},
}

var queryPackValidateCmd = &cobra.Command{
	Use:   "validate [path]",
	Short: "Validates a query pack",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		log.Info().Str("file", args[0]).Msg("validate query pack")
		pack, err := explorer.BundleFromPaths(args[0])
		if err != nil {
			log.Fatal().Err(err).Msg("could not load query pack")
		}

		_, err = pack.Compile(context.Background())
		if err != nil {
			log.Fatal().Err(err).Msg("could not validate query pack")
		}
		log.Info().Msg("valid query pack")
	},
}
