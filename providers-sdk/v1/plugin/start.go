// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package plugin

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"go.mondoo.com/mql/v13/logger"
	"go.mondoo.com/mql/v13/profiling"
	inventory "go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
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
	AssetUrlTrees      []*inventory.AssetUrlBranch
	Maturity           string `json:",omitempty"`
}

type Connector struct {
	Name      string
	Use       string   `json:",omitempty"`
	Short     string   `json:",omitempty"`
	Long      string   `json:",omitempty"`
	MinArgs   uint     `json:",omitempty"`
	MaxArgs   uint     `json:",omitempty"`
	IsHidden  bool     `json:",omitempty"`
	Flags     []Flag   `json:",omitempty"`
	Aliases   []string `json:",omitempty"`
	Discovery []string `json:",omitempty"`
	Maturity  string   `json:",omitempty"`
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
		Name: "mql-plugin",
		// Level: hclog.LevelFromString("DEBUG"),
		Level:  hclog.Info,
		Output: io.Discard,
	})

	providerName := providerNameFromArgs(args)
	profiler, err := profiling.Start("mql-provider-"+providerName, map[string]string{
		"provider": providerName,
	})
	if err != nil {
		log.Warn().Err(err).Msg("Pyroscope profiling not started")
	}
	defer func() { _ = profiler.Stop() }()

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

// providerNameFromArgs returns the binary name (without extension) of the
// running provider, used as the `provider` profiling tag. Provider binaries
// are named after the provider directory (e.g. providers/aws/dist/aws), so
// the basename of argv[0] is the provider name. Falls back to "unknown" if
// argv[0] is empty (defensive — it never should be).
func providerNameFromArgs(args []string) string {
	if len(args) == 0 || args[0] == "" {
		if exe, err := os.Executable(); err == nil {
			args = []string{exe}
		} else {
			return "unknown"
		}
	}
	name := filepath.Base(args[0])
	name = strings.TrimSuffix(name, filepath.Ext(name))
	if name == "" || name == "." {
		return "unknown"
	}
	return name
}
