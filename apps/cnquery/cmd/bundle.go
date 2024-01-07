// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"strconv"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v9/cli/config"
	cli_errors "go.mondoo.com/cnquery/v9/cli/errors"
	"go.mondoo.com/cnquery/v9/explorer"
	"go.mondoo.com/cnquery/v9/providers"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/v9/utils/stringx"
)

func init() {
	// bundle init
	packBundlesCmd.AddCommand(queryPackInitCmd)

	// bundle lint
	packBundlesCmd.AddCommand(queryPackLintCmd)

	// publish
	queryPackPublishCmd.Flags().String("pack-version", "", "Override the version of each pack in the bundle")
	packBundlesCmd.AddCommand(queryPackPublishCmd)

	rootCmd.AddCommand(packBundlesCmd)
}

var packBundlesCmd = &cobra.Command{
	Use:     "bundle",
	Aliases: []string{"pack"},
	Short:   "Create, upload, and validate query packs.",
}

//go:embed bundle_querypack-example.mql.yaml
var embedQueryPackTemplate []byte

var queryPackInitCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Create an example query pack that you can use as a starting point. If you don't provide a filename, cnquery uses `example-pack.mql.yaml`.",
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
	schema := providers.DefaultRuntime().Schema()
	_, err := queryPackBundle.Compile(context.Background(), schema)
	if err != nil {
		errors = append(errors, "could not compile the query pack bundle", err.Error())
	}

	return errors
}

// ensureProviders ensures that all providers are locally installed
func ensureProviders() error {
	for _, v := range providers.DefaultProviders {
		if _, err := providers.EnsureProvider(providers.ProviderLookup{ID: v.ID}, true, nil); err != nil {
			return err
		}
	}
	return nil
}

var queryPackLintCmd = &cobra.Command{
	Use:     "lint [path]",
	Aliases: []string{"validate"},
	Short:   "Apply style formatting to a query pack.",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info().Str("file", args[0]).Msg("lint query pack")
		err := ensureProviders()
		if err != nil {
			log.Warn().Err(err).Msg("could not ensure all providers are installed")
		}

		queryPackBundle, err := explorer.BundleFromPaths(args[0])
		if err != nil {
			return cli_errors.NewCommandError(errors.Wrap(err, "could not load query pack"), 1)
		}

		errors := validate(queryPackBundle)
		if len(errors) > 0 {
			log.Error().Msg("could not validate query pack")
			for i := range errors {
				fmt.Fprintf(os.Stderr, stringx.Indent(2, errors[i]))
			}
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return cli_errors.ExitCode1WithoutError
		}

		log.Info().Msg("valid query pack")
		return nil
	},
}

var queryPackPublishCmd = &cobra.Command{
	Use:     "publish [path]",
	Aliases: []string{"upload"},
	Short:   "Add a user-owned query pack to the Mondoo Security Registry.",
	Args:    cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("pack-version", cmd.Flags().Lookup("pack-version"))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		opts, optsErr := config.Read()
		if optsErr != nil {
			return cli_errors.NewCommandError(errors.Wrap(optsErr, "could not load configuration"), ConfigurationErrorCode)
		}
		config.DisplayUsedConfig()

		filename := args[0]
		log.Info().Str("file", filename).Msg("load query pack bundle")
		queryPackBundle, err := explorer.BundleFromPaths(filename)
		if err != nil {
			return cli_errors.NewCommandError(errors.Wrap(err, "could not load query pack bundle"), 1)
		}

		bundleErrors := validate(queryPackBundle)
		if len(bundleErrors) > 0 {
			log.Error().Msg("could not validate query pack")
			for i := range bundleErrors {
				fmt.Fprintf(os.Stderr, stringx.Indent(2, bundleErrors[i]))
			}
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return cli_errors.ExitCode1WithoutError
		}
		log.Info().Msg("valid query pack")

		// compile manipulates the bundle, therefore we read it again
		queryPackBundle, err = explorer.BundleFromPaths(filename)
		if err != nil {
			return cli_errors.NewCommandError(errors.Wrap(err, "could not load query pack bundle"), 1)
		}

		log.Info().Str("space", opts.SpaceMrn).Msg("add query pack bundle to space")
		overrideVersionFlag := false
		overrideVersion := viper.GetString("pack-version")
		if len(overrideVersion) > 0 {
			overrideVersionFlag = true
		}

		serviceAccount := opts.GetServiceCredential()
		if serviceAccount == nil {
			return cli_errors.NewCommandError(errors.New("cnquery has no credentials. Log in with `cnquery login`"), 1)
		}

		certAuth, err := upstream.NewServiceAccountRangerPlugin(serviceAccount)
		if err != nil {
			log.Error().Err(err).Msg("could not initialize client authentication")
			return cli_errors.NewCommandError(nil, ConfigurationErrorCode)
		}
		httpClient, err := opts.GetHttpClient()
		if err != nil {
			return cli_errors.NewCommandError(errors.Wrap(err, "error while creating Mondoo API client"), 1)
		}
		queryHubServices, err := explorer.NewQueryHubClient(opts.UpstreamApiEndpoint(), httpClient, certAuth)
		if err != nil {
			return cli_errors.NewCommandError(errors.Wrap(err, "could not connect to the Mondoo Security Registry"), 1)
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
			return cli_errors.NewCommandError(errors.Wrap(err, "could not add query packs"), 1)
		}

		log.Info().Msg("successfully added query packs")
		return nil
	},
}
