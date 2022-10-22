package cmd

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"strconv"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	cnquery_config "go.mondoo.com/cnquery/apps/cnquery/cmd/config"
	"go.mondoo.com/cnquery/cli/config"
	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnquery/stringx"
	"go.mondoo.com/cnquery/upstream"
	"go.mondoo.com/ranger-rpc"
)

func init() {
	// bundle init
	packBundlesCmd.AddCommand(queryPackInitCmd)

	// bundle validate
	packBundlesCmd.AddCommand(queryPackValidateCmd)

	// bundle add
	queryPackUploadCmd.Flags().String("pack-version", "", "Override the version of each pack in the bundle")
	packBundlesCmd.AddCommand(queryPackUploadCmd)

	rootCmd.AddCommand(packBundlesCmd)
}

var packBundlesCmd = &cobra.Command{
	Use:     "bundle",
	Aliases: []string{"pack"},
	Short:   "Manage query packs",
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

func validate(queryPackBundle *explorer.Bundle) []string {
	errors := []string{}

	// check that we have uids for packs and queries
	for i := range queryPackBundle.Packs {
		pack := queryPackBundle.Packs[i]
		packId := strconv.Itoa(i)

		if pack.Uid == "" {
			errors = append(errors, fmt.Sprintf("pack %s does not define a uid", packId))
		} else {
			packId = pack.Uid
		}

		if pack.Name == "" {
			errors = append(errors, fmt.Sprintf("pack %s does not define a name", packId))
		}

		for j := range pack.Queries {
			query := pack.Queries[j]
			queryId := strconv.Itoa(j)
			if query.Uid == "" {
				errors = append(errors, fmt.Sprintf("query %s/%s does not define a uid", packId, queryId))
			} else {
				queryId = query.Uid
			}

			if query.Title == "" {
				errors = append(errors, fmt.Sprintf("query %s/%s does not define a name", packId, queryId))
			}
		}
	}

	// we compile after the checks because it removes the uids and replaces it with mrns
	_, err := queryPackBundle.Compile(context.Background())
	if err != nil {
		errors = append(errors, "could not compile the query pack bundle", err.Error())
	}

	return errors
}

var queryPackValidateCmd = &cobra.Command{
	Use:   "validate [path]",
	Short: "Validates a query pack",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		log.Info().Str("file", args[0]).Msg("validate query pack")
		queryPackBundle, err := explorer.BundleFromPaths(args[0])
		if err != nil {
			log.Fatal().Err(err).Msg("could not load query pack")
		}

		errors := validate(queryPackBundle)
		if len(errors) > 0 {
			log.Error().Msg("could not validate query pack")
			for i := range errors {
				fmt.Fprintf(os.Stderr, stringx.Indent(2, errors[i]))
			}
			os.Exit(1)
		}

		log.Info().Msg("valid query pack")
	},
}

var queryPackUploadCmd = &cobra.Command{
	Use:   "upload [path]",
	Short: "Adds a user-owned pack to Mondoo's Query Hub",
	Args:  cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("pack-version", cmd.Flags().Lookup("pack-version"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		opts, optsErr := cnquery_config.ReadConfig()
		if optsErr != nil {
			log.Fatal().Err(optsErr).Msg("could not load configuration")
		}
		config.DisplayUsedConfig()

		filename := args[0]
		log.Info().Str("file", filename).Msg("load query pack bundle")
		queryPackBundle, err := explorer.BundleFromPaths(filename)
		if err != nil {
			log.Fatal().Err(err).Msg("could not load query pack bundle")
		}

		errors := validate(queryPackBundle)
		if len(errors) > 0 {
			log.Error().Msg("could not validate query pack")
			for i := range errors {
				fmt.Fprintf(os.Stderr, stringx.Indent(2, errors[i]))
			}
			os.Exit(1)
		}
		log.Info().Msg("valid query pack")

		// compile manipulates the bundle, therefore we read it again
		queryPackBundle, err = explorer.BundleFromPaths(filename)
		if err != nil {
			log.Fatal().Err(err).Msg("could not load query pack bundle")
		}

		log.Info().Str("space", opts.SpaceMrn).Msg("add query pack bundle to space")
		overrideVersionFlag := false
		overrideVersion := viper.GetString("pack-version")
		if len(overrideVersion) > 0 {
			overrideVersionFlag = true
		}

		serviceAccount := opts.GetServiceCredential()
		if serviceAccount == nil {
			log.Fatal().Msg("cnquery has no credentials. Log in with `cnquery login`")
		}

		certAuth, _ := upstream.NewServiceAccountRangerPlugin(serviceAccount)
		queryHubServices, err := explorer.NewQueryHubClient(opts.UpstreamApiEndpoint(), ranger.DefaultHttpClient(), certAuth)
		if err != nil {
			log.Fatal().Err(err).Msg("could not connect to query hub")
		}

		// set the owner mrn for spaces
		queryPackBundle.OwnerMrn = opts.SpaceMrn
		ctx := context.Background()

		// override version and/or labels
		for i := range queryPackBundle.Packs {
			p := queryPackBundle.Packs[i]

			// override query pack version
			if overrideVersionFlag {
				p.Version = overrideVersion
			}
		}

		// send data upstream
		_, err = queryHubServices.SetBundle(ctx, queryPackBundle)
		if err != nil {
			log.Fatal().Err(err).Msg("could not add query packs")
		}

		log.Info().Msg("successfully added query packs")
	},
}
