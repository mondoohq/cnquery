package providers

import (
	"io/ioutil"
	"os"
	"os/exec"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/muesli/termenv"
	"github.com/rs/zerolog/log"
	pp "go.mondoo.com/cnquery/providers/plugin"
)

type coordinator struct {
	Providers Providers
	Running   []*ProviderRuntime
	mutex     sync.Mutex
}

var Coordinator = coordinator{
	Running: []*ProviderRuntime{},
}

func (c *coordinator) Start(name string) (*ProviderRuntime, error) {
	if x, ok := builtinProviders[name]; ok {
		log.Warn().Msg("using builtin provider for " + name)
		return x.Runtime, nil
	}

	if c.Providers == nil {
		var err error
		c.Providers, err = List()
		if err != nil {
			return nil, err
		}
	}

	provider, ok := c.Providers[name]
	if !ok {
		return nil, errors.New("cannot find provider " + name)
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

	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, errors.Wrap(err, "failed to initialize plugin client")
	}

	// Request the plugin
	pluginName := "provider"
	raw, err := rpcClient.Dispense(pluginName)
	if err != nil {
		client.Kill()
		return nil, errors.Wrap(err, "failed to call "+pluginName+" plugin")
	}

	res := &ProviderRuntime{
		Name:   name,
		Plugin: raw.(pp.ProviderPlugin),
		Client: client,
	}

	c.mutex.Lock()
	c.Running = append(c.Running, res)
	c.mutex.Unlock()

	return res, nil
}

type ProviderRuntime struct {
	Name   string
	Plugin pp.ProviderPlugin
	Client *plugin.Client

	isClosed bool
}

func (c *coordinator) Close(p *ProviderRuntime) {
	if !p.isClosed {
		p.isClosed = true
		if p.Client != nil {
			p.Client.Kill()
		}
	}

	c.mutex.Lock()
	for i := range c.Running {
		if c.Running[i] == p {
			c.Running = append(c.Running[0:i], c.Running[i+1:]...)
			break
		}
	}
	c.mutex.Unlock()
}

func (c *coordinator) Shutdown() {
	c.mutex.Lock()
	for i := range c.Running {
		cur := c.Running[i]
		cur.isClosed = true
		cur.Client.Kill()
	}
	c.mutex.Unlock()
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
