// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plugin

import (
	"io"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"go.mondoo.com/cnquery/v10/logger"
)

type Provider struct {
	Name            string
	ID              string
	Version         string
	ConnectionTypes []string
	// CrossProviderTypes are asset providers that already
	// have a primary provider set, but which may need to use
	// resources from a different provider. For example:
	// The primary provider of an asset may be the "os" provider.
	// However, it now wants to use resources from the "network" provider.
	// The "network" provider can indicate that it also supports
	// assets from the "os" provider.
	// TODO: This is only a hotfix and will be solved by
	// each provider creating an asset object when it tries to
	// call out.
	CrossProviderTypes []string
	Connectors         []Connector
}

type Connector struct {
	Name      string
	Use       string   `json:",omitempty"`
	Short     string   `json:",omitempty"`
	Long      string   `json:",omitempty"`
	MinArgs   uint     `json:",omitempty"`
	MaxArgs   uint     `json:",omitempty"`
	Flags     []Flag   `json:",omitempty"`
	Aliases   []string `json:",omitempty"`
	Discovery []string `json:",omitempty"`
}

func Start(args []string, impl ProviderPlugin) {
	logger.CliCompactLogger(logger.LogOutputWriter)

	var logLevel string
	pflag.StringVar(&logLevel, "log-level", "warn", "Log level")
	pflag.Parse()

	ll, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		log.Warn().Msgf("Failed parsing log level: %s", logLevel)
	} else {
		zerolog.SetGlobalLevel(ll)
	}
	log.Debug().Msgf("Log level set to %s", ll)

	// disable the plugin's logs
	pluginLogger := hclog.New(&hclog.LoggerOptions{
		Name: "cnquery-plugin",
		// Level: hclog.LevelFromString("DEBUG"),
		Level:  hclog.Info,
		Output: io.Discard,
	})

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]plugin.Plugin{
			"provider": &ProviderPluginImpl{Impl: impl},
		},
		Logger: pluginLogger,

		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
