// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"context"
	"os"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v10/cli/config"
	cli_errors "go.mondoo.com/cnquery/v10/cli/errors"
	cnquery_providers "go.mondoo.com/cnquery/v10/providers"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/sysinfo"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream"
	"sigs.k8s.io/yaml"
)

func init() {
	rootCmd.AddCommand(LogoutCmd)
	LogoutCmd.Flags().Bool("force", false, "Force re-authentication")
}

var LogoutCmd = &cobra.Command{
	Use:     "logout",
	Aliases: []string{"unregister"},
	Short:   "Log out from Mondoo Platform",
	Long: `
This process also revokes the Mondoo Platform service account to 
ensure the credentials cannot be used in the future.
`,
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("force", cmd.Flags().Lookup("force"))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		defer cnquery_providers.GlobalCoordinator.Shutdown()
		var err error

		// its perfectly fine not to have a config here, therefore we ignore errors
		opts, optsErr := config.Read()
		if optsErr != nil {
			return errors.Wrap(optsErr, "could not load configuration")
		}

		// print the used config to the user
		config.DisplayUsedConfig()

		// determine information about the client
		sysInfo, err := sysinfo.Get()
		if err != nil {
			return errors.Wrap(err, "could not gather client information")
		}

		// check valid client authentication
		serviceAccount := opts.GetServiceCredential()
		if serviceAccount == nil {
			return cli_errors.NewCommandError(errors.Wrap(err, "could not initialize client authentication"), ConfigurationErrorCode)
		}

		plugins := defaultRangerPlugins(sysInfo, opts.GetFeatures())
		certAuth, err := upstream.NewServiceAccountRangerPlugin(serviceAccount)
		if err != nil {
			log.Error().Err(err).Msg("could not initialize client authentication")
			return cli_errors.NewCommandError(nil, ConfigurationErrorCode)
		}
		plugins = append(plugins, certAuth)

		httpClient, err := opts.GetHttpClient()
		if err != nil {
			return cli_errors.NewCommandError(errors.Wrap(err, "error while creating Mondoo API client"), 1)
		}
		client, err := upstream.NewAgentManagerClient(opts.UpstreamApiEndpoint(), httpClient, plugins...)
		if err != nil {
			log.Error().Err(err).Msg("could not initialize connection to Mondoo Platform")
			cli_errors.NewCommandError(nil, ConfigurationErrorCode)
		}

		if !viper.GetBool("force") {
			log.Info().Msg("are you sure you want to revoke client access to Mondoo Platform? Use --force if you are sure")
			return cli_errors.NewCommandError(errors.New("--force is required to logout"), ConfigurationErrorCode)
		}

		// try to load config into credentials struct
		credentials := opts.GetServiceCredential()

		// if we have credentials, we are going to self-destroy
		ctx := context.Background()
		if credentials != nil && len(credentials.Mrn) > 0 {
			_, err = client.PingPong(ctx, &upstream.Ping{})

			if err == nil {
				log.Info().Msgf("client %s authenticated successfully", credentials.Mrn)

				// un-register the agent
				_, err = client.UnRegisterAgent(ctx, &upstream.Mrn{
					Mrn: opts.AgentMrn,
				})
				if err != nil {
					log.Error().Err(err).Msg("failed to unregister client")
				}
			} else {
				log.Error().Err(err).Msg("communication with Mondoo Platform failed")
			}
		}

		// delete config if it exists
		path := viper.ConfigFileUsed()
		fi, err := os.Stat(path)
		if err == nil {
			log.Debug().Str("path", path).Msg("remove client information from config")

			opts.AgentMrn = ""

			data, err := yaml.Marshal(opts)
			if err != nil {
				log.Error().Err(err).Msg("could not update Mondoo config")
			}
			err = os.WriteFile(path, data, fi.Mode())
			if err != nil {
				log.Error().Err(err).Msg("could not update Mondoo config")
			}
		}

		log.Info().Msgf("Bye bye, space cat. Client %s unregistered successfully", credentials.Mrn)
		return nil
	},
}
