// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v11"
	"go.mondoo.com/cnquery/v11/cli/config"
	cli_errors "go.mondoo.com/cnquery/v11/cli/errors"
	"go.mondoo.com/cnquery/v11/cli/theme"
	"go.mondoo.com/cnquery/v11/providers"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/sysinfo"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/upstream/health"
	"go.mondoo.com/ranger-rpc"
	"sigs.k8s.io/yaml"
)

func init() {
	StatusCmd.Flags().StringP("output", "o", "", "Set output format. Accepts json or yaml.")
	rootCmd.AddCommand(StatusCmd)
}

// StatusCmd represents the version command
var StatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Verify access to Mondoo Platform",
	Long: `
Status sends a ping to Mondoo Platform to verify the credentials.
	`,
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("output", cmd.Flags().Lookup("output"))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		defer providers.Coordinator.Shutdown()

		config.DisplayUsedConfig()

		s, err := checkStatus()
		if err != nil {
			return err
		}

		switch strings.ToLower(viper.GetString("output")) {
		case "yaml":
			s.RenderYaml()
		case "json":
			s.RenderJson()
		default:
			s.RenderCliStatus()
		}

		if !s.Client.Registered || s.Client.PingPongError != nil {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return cli_errors.ExitCode1WithoutError
		}
		return nil
	},
}

func checkStatus() (Status, error) {
	s := Status{
		Client: ClientStatus{
			Timestamp: time.Now().Format(time.RFC3339),
			Version:   cnquery.GetVersion(),
			Build:     cnquery.GetBuild(),
		},
	}

	opts, optsErr := config.Read()
	if optsErr != nil {
		return s, cli_errors.NewCommandError(errors.Wrap(optsErr, "could not load configuration"), 1)
	}

	httpClient, err := opts.GetHttpClient()
	if err != nil {
		return s, cli_errors.NewCommandError(errors.Wrap(err, "failed to set up Mondoo API client"), 1)
	}

	sysInfo, err := sysinfo.Get()
	if err == nil {
		s.Client.Platform = sysInfo.Platform
		s.Client.Hostname = sysInfo.Hostname
		s.Client.IP = sysInfo.IP
	}

	// check server health and clock skew
	upstreamStatus, err := health.CheckApiHealth(httpClient, opts.UpstreamApiEndpoint())
	if err != nil {
		log.Error().Err(err).Msg("could not check upstream health")
	}
	s.Upstream = upstreamStatus

	latestVersion, err := cnquery.GetLatestVersion(httpClient)
	if err != nil {
		return s, cli_errors.NewCommandError(errors.Wrap(err, "failed to get latest version"), 1)
	}

	s.Client.LatestVersion = latestVersion

	// check valid agent authentication
	plugins := []ranger.ClientPlugin{}

	// try to load config into credentials struct
	credentials := opts.GetServiceCredential()
	if credentials != nil && len(credentials.Mrn) > 0 {
		s.Client.ParentMrn = credentials.GetParentMrn()
		s.Client.Registered = true
		s.Client.ServiceAccount = credentials.Mrn
		s.Client.Mrn = opts.AgentMrn
		if s.Client.Mrn == "" {
			s.Client.Mrn = "no managed client"
		}

		certAuth, err := upstream.NewServiceAccountRangerPlugin(credentials)
		if err != nil {
			return s, cli_errors.NewCommandError(errors.Wrap(err, "invalid credentials"), ConfigurationErrorCode)
		}
		plugins = append(plugins, certAuth)

		// try to ping the server
		client, err := upstream.NewAgentManagerClient(s.Upstream.API.Endpoint, httpClient, plugins...)
		if err == nil {
			_, err = client.PingPong(context.Background(), &upstream.Ping{})
			if err != nil {
				s.Client.PingPongError = err
			}
		} else {
			s.Client.PingPongError = err
		}
	}

	return s, nil
}

type Status struct {
	Client   ClientStatus  `json:"client"`
	Upstream health.Status `json:"upstream"`
}

type ClientStatus struct {
	Timestamp      string              `json:"timestamp,omitempty"`
	Mrn            string              `json:"mrn,omitempty"`
	ServiceAccount string              `json:"service_account,omitempty"`
	ParentMrn      string              `json:"parentMrn,omitempty"`
	Version        string              `json:"version,omitempty"`
	LatestVersion  string              `json:"latest_version,omitempty"`
	Build          string              `json:"build,omitempty"`
	Labels         map[string]string   `json:"labels,omitempty"`
	Platform       *inventory.Platform `json:"platform,omitempty"`
	IP             string              `json:"ip,omitempty"`
	Hostname       string              `json:"hostname,omitempty"`
	Registered     bool                `json:"registered,omitempty"`
	PingPongError  error               `json:"pingPongError,omitempty"`
}

func (s Status) RenderCliStatus() {
	if s.Client.Platform != nil {
		agent := s.Client
		log.Info().Msg("Platform:\t\t" + agent.Platform.Name)
		log.Info().Msg("Version:\t\t" + agent.Platform.Version)
		log.Info().Msg("Hostname:\t\t" + agent.Hostname)
		log.Info().Msg("IP:\t\t\t" + agent.IP)
	} else {
		log.Warn().Msg("could not determine client platform information")
	}

	log.Info().Msg("Time:\t\t\t" + s.Client.Timestamp)
	log.Info().Msg("Version:\t\t" + cnquery.GetVersion() + " (API Version: " + cnquery.APIVersion() + ")")

	if s.Client.LatestVersion != "" {
		log.Info().Msg("Latest Version:\t" + s.Client.LatestVersion)

		if cnquery.GetVersion() != s.Client.LatestVersion && cnquery.GetVersion() != "unstable" {
			log.Warn().Msg("A newer version is available")
		}
	}

	installed, outdated, err := getProviders()
	if err != nil {
		log.Warn().Err(err).Msg("failed to get provider info")
	}
	log.Info().Msg("Installed Providers:\t" + strings.Join(installed, " | "))

	if len(outdated) > 0 {
		log.Info().Msg("Outdated Providers:\t" + strings.Join(outdated, " | "))
	}

	log.Info().Msg("API ConnectionConfig:\t" + s.Upstream.API.Endpoint)
	log.Info().Msg("API Status:\t\t" + s.Upstream.API.Status)
	log.Info().Msg("API Time:\t\t" + s.Upstream.API.Timestamp)
	log.Info().Msg("API Version:\t\t" + s.Upstream.API.Version)

	if s.Upstream.API.Version != cnquery.APIVersion() {
		log.Warn().Msg("API versions do not match, please update the client")
	}

	if len(s.Upstream.Features) > 0 {
		log.Info().Msg("Features:\t\t" + strings.Join(s.Upstream.Features, ","))
	}

	if s.Client.ParentMrn != "" {
		log.Info().Msg("Owner:\t\t" + s.Client.ParentMrn)
	}

	if s.Client.Registered {
		log.Info().Msg("Client:\t\t" + s.Client.Mrn)
		log.Info().Msg("Service Account:\t\t" + s.Client.ServiceAccount)
		log.Info().Msg(theme.DefaultTheme.Success("client is registered"))
	} else {
		log.Error().Msg("client is not registered")
	}

	if s.Client.Registered && s.Client.PingPongError == nil {
		log.Info().Msg(theme.DefaultTheme.Success("client authenticated successfully"))
	} else if s.Client.PingPongError != nil {
		log.Error().Err(s.Client.PingPongError).
			Msgf("The Mondoo Platform credentials provided at %s didn't successfully authenticate with Mondoo Platform. Please re-authenticate with Mondoo Platform. To learn how, read https://mondoo.com/docs/cnspec/cnspec-adv-install/registration/.",
				viper.ConfigFileUsed())
	}

	for i := range s.Upstream.Warnings {
		log.Warn().Msg(s.Upstream.Warnings[i])
	}
}

func (s Status) RenderJson() {
	output, err := json.Marshal(s)
	if err != nil {
		log.Error().Err(err).Msg("could not generate json")
	}
	os.Stdout.Write(output)
}

func (s Status) RenderYaml() {
	output, err := yaml.Marshal(s)
	if err != nil {
		log.Error().Err(err).Msg("could not generate yaml")
	}
	os.Stdout.Write(output)
}

func getProviders() ([]string, []string, error) {
	var installed []string
	var outdated []string

	allProviders, err := providers.ListActive()
	if err != nil {
		return nil, nil, err
	}
	for _, provider := range allProviders {
		installed = append(installed, provider.Name)
		latestVersion, err := providers.LatestVersion(provider.Name)
		if err != nil {
			continue
		}
		if latestVersion != provider.Version && provider.Name != "core" {
			outdated = append(outdated, provider.Name)
		}
	}

	return installed, outdated, nil
}
