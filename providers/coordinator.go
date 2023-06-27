package providers

import (
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/muesli/termenv"
	"github.com/rs/zerolog/log"
	pp "go.mondoo.com/cnquery/providers/plugin"
)

type coordinator struct {
	Providers Providers
}

var Coordinator = coordinator{}

func (c *coordinator) Start(name string) error {
	if c.Providers == nil {
		var err error
		c.Providers, err = List()
		if err != nil {
			return err
		}
	}

	provider, ok := c.Providers[name]
	if !ok {
		return errors.New("cannot find provider " + name)
	}

	// disable the plugin's logs
	pluginLogger := hclog.New(&hclog.LoggerOptions{
		Name: "provider-plugin",
		// Level: hclog.LevelFromString("DEBUG"),
		Level:  hclog.Info,
		Output: ioutil.Discard,
	})

	pluginCmd := exec.Command(provider.Path, "run_as_plugin")
	log.Debug().Str("path", pluginCmd.Path).Msg("running provider plugin")

	addColorConfig(pluginCmd)

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: pp.Handshake,
		Plugins:         pp.PluginMap,
		Cmd:             pluginCmd,
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolNetRPC, plugin.ProtocolGRPC,
		},
		Logger: pluginLogger,
		Stderr: os.Stderr,
	})
	defer client.Kill()

	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		return errors.Wrap(err, "failed to initialize plugin client")
	}

	// Request the plugin
	pluginName := "provider"
	raw, err := rpcClient.Dispense(pluginName)
	if err != nil {
		return errors.Wrap(err, "failed to call "+pluginName+" plugin")
	}

	cnquery := raw.(pp.ProviderPlugin)

	// writer := shared.IOWriter{Writer: os.Stdout}
	// err = cnquery.RunQuery(conf, &writer)
	// if err != nil {
	// 	if status, ok := status.FromError(err); ok {
	// 		code := status.Code()
	// 		switch code {
	// 		case codes.Unavailable, codes.Internal:
	// 			return errors.New(pluginName + " plugin crashed, please report any stack trace you see with this error")
	// 		case codes.Unimplemented:
	// 			return errors.New(pluginName + " plugin failed, the call is not implemented, please report this error")
	// 		default:
	// 			return errors.New(pluginName + " plugin failed, error " + strconv.Itoa(int(code)) + ": " + status.Message())
	// 		}
	// 	}

	// 	return err
	// }

	panic("STH")
	_ = cnquery

	return nil
}

func addColorConfig(cmd *exec.Cmd) {
	switch termenv.EnvColorProfile() {
	case termenv.ANSI256, termenv.ANSI, termenv.TrueColor:
		cmd.Env = append(cmd.Env, "CLICOLOR_FORCE=1")
	default:
		// it will default to no-color, since it's run as a plugin
		// so there is nothing to do here
	}
}
