// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"context"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v9"
	"go.mondoo.com/cnquery/v9/cli/config"
	cli_errors "go.mondoo.com/cnquery/v9/cli/errors"
	cnquery_providers "go.mondoo.com/cnquery/v9/providers"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/sysinfo"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/upstream"
	"go.mondoo.com/ranger-rpc"
	"go.mondoo.com/ranger-rpc/plugins/authentication/statictoken"
)

func init() {
	rootCmd.AddCommand(LoginCmd)
	LoginCmd.Flags().StringP("token", "t", "", "Set a client registration token.")
	LoginCmd.Flags().StringToString("annotation", nil, "Set the client annotations.")
	LoginCmd.Flags().String("name", "", "Set asset name.")
	LoginCmd.Flags().String("api-endpoint", "", "Set the Mondoo API endpoint.")
}

var LoginCmd = &cobra.Command{
	Use:     "login",
	Aliases: []string{"register"},
	Short:   "Register with Mondoo Platform.",
	Long: `
Log in to Mondoo Platform using a registration token. To pass in the token, use 
the '--token' flag.

You can generate a new registration token on the Mondoo Dashboard. Go to
https://console.mondoo.com -> Space -> Settings -> Registration Token. Copy the token and pass it in 
using the '--token' argument.

You remain logged in until you explicitly log out using the 'logout' subcommand.
	`,
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("api_endpoint", cmd.Flags().Lookup("api-endpoint"))
		viper.BindPFlag("name", cmd.Flags().Lookup("name"))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		defer cnquery_providers.Coordinator.Shutdown()
		token, _ := cmd.Flags().GetString("token")
		annotations, _ := cmd.Flags().GetStringToString("annotation")
		return register(token, annotations)
	},
}

func register(token string, annotations map[string]string) error {
	var err error
	var credential *upstream.ServiceAccountCredentials

	// determine information about the client
	sysInfo, err := sysinfo.Get()
	if err != nil {
		return cli_errors.NewCommandError(errors.Wrap(err, "could not gather client information"), 1)
	}
	defaultPlugins := defaultRangerPlugins(sysInfo, cnquery.DefaultFeatures)

	apiEndpoint := viper.GetString("api_endpoint")
	token = strings.TrimSpace(token)

	// NOTE: login is special because we do not have a config yet
	proxy, err := config.GetAPIProxy()
	if err != nil {
		return cli_errors.NewCommandError(errors.Wrap(err, "could not parse proxy URL"), 1)
	}
	httpClient := ranger.NewHttpClient(ranger.WithProxy(proxy))

	// we handle three cases here:
	// 1. user has a token provided
	// 2. user has no token provided, but has a service account file is already there
	//
	if token != "" {
		// print token details
		claims, err := upstream.ExtractTokenClaims(token)
		if err != nil {
			log.Warn().Err(err).Msg("could not read the token")
		} else {
			if len(claims.Description) > 0 {
				log.Info().Msg("token description: " + claims.Description)
			}
			if claims.IsExpired() {
				log.Warn().Msg("token is expired")
			} else {
				log.Info().Msg("token will expire at " + claims.Claims.Expiry.Time().Format(time.RFC1123))
			}

			if apiEndpoint == "" {
				apiEndpoint = claims.ApiEndpoint
			}
		}

		// gather service account
		plugins := []ranger.ClientPlugin{}
		plugins = append(plugins, defaultPlugins...)
		plugins = append(plugins, statictoken.NewRangerPlugin(token))

		client, err := upstream.NewAgentManagerClient(apiEndpoint, httpClient, plugins...)
		if err != nil {
			return cli_errors.NewCommandError(errors.Wrap(err, "could not connect to mondoo platform"), 1)
		}

		name := viper.GetString("name")
		if name == "" {
			name = sysInfo.Hostname
		}

		confirmation, err := client.RegisterAgent(context.Background(), &upstream.AgentRegistrationRequest{
			Token: token,
			Name:  name,
			AgentInfo: &upstream.AgentInfo{
				Mrn:              "",
				Version:          sysInfo.Version,
				Build:            sysInfo.Build,
				PlatformName:     sysInfo.Platform.Name,
				PlatformRelease:  sysInfo.Platform.Version,
				PlatformArch:     sysInfo.Platform.Arch,
				PlatformIp:       sysInfo.IP,
				PlatformHostname: sysInfo.Hostname,
				Labels:           nil,
				PlatformId:       sysInfo.PlatformId,
			},
		})
		if err != nil {
			return cli_errors.NewCommandError(errors.Wrap(err, "failed to log in client"), 1)
		}

		log.Debug().Msg("store configuration")
		// overwrite force, otherwise it will be stored
		viper.Set("force", false)

		// update configuration file, api-endpoint is set automatically
		viper.Set("agent_mrn", confirmation.AgentMrn)
		viper.Set("api_endpoint", confirmation.Credential.ApiEndpoint)
		viper.Set("space_mrn", confirmation.Credential.GetParentMrn())
		viper.Set("mrn", confirmation.Credential.Mrn)
		viper.Set("private_key", confirmation.Credential.PrivateKey)
		viper.Set("certificate", confirmation.Credential.Certificate)
		viper.Set("annotations", annotations)
		credential = confirmation.Credential
	} else {
		// try to read local options
		opts, optsErr := config.Read()
		if optsErr != nil {
			log.Warn().Msg("could not load configuration, please use --token or --config with the appropriate values")
			return cli_errors.ExitCode1WithoutError
		}
		// print the used config to the user
		config.DisplayUsedConfig()

		httpClient, err = opts.GetHttpClient()
		if err != nil {
			log.Warn().Err(err).Msg("could not create http client")
			return cli_errors.ExitCode1WithoutError
		}

		if opts.AgentMrn != "" {
			// already authenticated
			log.Info().Msg("client is already logged in, skipping")
			credential = opts.GetServiceCredential()
		} else {
			credential = opts.GetServiceCredential()

			// run ping pong
			plugins := []ranger.ClientPlugin{}
			plugins = append(plugins, defaultPlugins...)
			certAuth, err := upstream.NewServiceAccountRangerPlugin(credential)
			if err != nil {
				log.Warn().Err(err).Msg("could not initialize certificate authentication")
				return cli_errors.ExitCode1WithoutError
			}
			plugins = append(plugins, certAuth)

			client, err := upstream.NewAgentManagerClient(apiEndpoint, httpClient, plugins...)
			if err != nil {
				log.Warn().Err(err).Msg("could not connect to Mondoo Platform")
				return cli_errors.ExitCode1WithoutError
			}

			name := viper.GetString("name")
			if name == "" {
				name = sysInfo.Hostname
			}

			confirmation, err := client.RegisterAgent(context.Background(), &upstream.AgentRegistrationRequest{
				Name: name,
				AgentInfo: &upstream.AgentInfo{
					Mrn:              opts.AgentMrn,
					Version:          sysInfo.Version,
					Build:            sysInfo.Build,
					PlatformName:     sysInfo.Platform.Name,
					PlatformRelease:  sysInfo.Platform.Version,
					PlatformArch:     sysInfo.Platform.Arch,
					PlatformIp:       sysInfo.IP,
					PlatformHostname: sysInfo.Hostname,
					Labels:           opts.Labels,
					PlatformId:       sysInfo.PlatformId,
				},
			})
			if err != nil {
				return cli_errors.NewCommandError(errors.Wrap(err, "failed to log in client"), 1)
			}

			// update configuration file, api-endpoint is set automatically
			// NOTE: we ignore the credentials from confirmation since the service never returns the credentials again
			viper.Set("agent_mrn", confirmation.AgentMrn)
		}
	}

	err = config.StoreConfig()
	if err != nil {
		log.Warn().Err(err).Msg("could not write mondoo configuration")
		return cli_errors.ExitCode1WithoutError
	}

	// run ping pong to validate the service account
	plugins := []ranger.ClientPlugin{}
	plugins = append(plugins, defaultPlugins...)
	certAuth, err := upstream.NewServiceAccountRangerPlugin(credential)
	if err != nil {
		log.Warn().Err(err).Msg("could not initialize certificate authentication")
	}
	plugins = append(plugins, certAuth)
	client, err := upstream.NewAgentManagerClient(apiEndpoint, httpClient, plugins...)
	if err != nil {
		log.Warn().Err(err).Msg("could not connect to mondoo platform")
		return cli_errors.ExitCode1WithoutError
	}

	_, err = client.PingPong(context.Background(), &upstream.Ping{})
	if err != nil {
		log.Warn().Msg(err.Error())
		return cli_errors.ExitCode1WithoutError
	}

	log.Info().Msgf("client %s has logged in successfully", viper.Get("agent_mrn"))
	return nil
}
