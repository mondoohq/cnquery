package plugin

import (
	"io"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/rs/zerolog"
	"go.mondoo.com/cnquery/logger"
)

type Provider struct {
	Name       string
	ID         string
	Version    string
	Connectors []Connector
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

type FlagType byte

const (
	FlagType_Bool FlagType = 1 + iota
	FlagType_Int
	FlagType_String
	FlagType_List
	FlagType_KeyValue
)

type FlagOption byte

const (
	FlagOption_Hidden FlagOption = 0x1 << iota
	FlagOption_Deprecated
	FlagOption_Required
	FlagOption_Password
	// max: 8 options!
)

type Flag struct {
	Long    string     `json:",omitempty"`
	Short   string     `json:",omitempty"`
	Default string     `json:",omitempty"`
	Desc    string     `json:",omitempty"`
	Type    FlagType   `json:",omitempty"`
	Option  FlagOption `json:",omitempty"`
	// ConfigEntry that is used for this flag:
	// "" = use the same as Long
	// "some.other" = map to some.other field
	// "-" = do not read this from config
	ConfigEntry string `json:",omitempty"`
}

func Start(args []string, impl ProviderPlugin) {
	logger.CliCompactLogger(logger.LogOutputWriter)
	zerolog.SetGlobalLevel(zerolog.WarnLevel)

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
