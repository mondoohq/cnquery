package providers

import (
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/go-plugin"
	"github.com/muesli/termenv"
	"github.com/rs/zerolog/log"
	pp "go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/resources"
	"go.mondoo.com/cnquery/providers/core/resources/versions/semver"
)

var Coordinator = coordinator{
	Running: []*RunningProvider{},
}

type coordinator struct {
	Providers Providers
	Running   []*RunningProvider
	mutex     sync.Mutex
}

type RunningProvider struct {
	Name   string
	ID     string
	Plugin pp.ProviderPlugin
	Client *plugin.Client
	Schema *resources.Schema

	isClosed bool
}

type UpdateProvidersConfig struct {
	// if true, will try to update providers when new versions are available
	Enabled bool
	// seconds until we try to refresh the providers version again
	RefreshInterval int
}

func (c *coordinator) Start(id string, update UpdateProvidersConfig) (*RunningProvider, error) {
	if x, ok := builtinProviders[id]; ok {
		// We don't warn for the core provider, which is the only provider expected
		// to be built into the binary for now.
		if id != BuiltinCoreID {
			log.Warn().Msg("using builtin provider for " + x.Config.Name)
		}
		return x.Runtime, nil
	}

	if c.Providers == nil {
		var err error
		c.Providers, err = ListActive()
		if err != nil {
			return nil, err
		}
	}

	provider, ok := c.Providers[id]
	if !ok {
		return nil, errors.New("cannot find provider " + id)
	}

	if update.Enabled {
		// We do not stop on failed updates. Up until some other errors happens,
		// things are still functional. We want to consider failure, possibly
		// with a config entry in the futuer.
		updated, err := c.tryProviderUpdate(provider, update)
		if err != nil {
			log.Error().
				Err(err).
				Str("provider", provider.Name).
				Msg("failed to update provider")
		} else {
			provider = updated
		}
	}

	if provider.Schema == nil {
		if err := provider.LoadResources(); err != nil {
			return nil, errors.Wrap(err, "failed to load provider "+id+" resources info")
		}
	}

	pluginCmd := exec.Command(provider.binPath(), "run_as_plugin")
	log.Debug().Str("path", pluginCmd.Path).Msg("running provider plugin")

	addColorConfig(pluginCmd)

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: pp.Handshake,
		Plugins:         pp.PluginMap,
		Cmd:             pluginCmd,
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolNetRPC, plugin.ProtocolGRPC,
		},
		Logger: &hclogger{},
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

	res := &RunningProvider{
		Name:   provider.Name,
		ID:     provider.ID,
		Plugin: raw.(pp.ProviderPlugin),
		Client: client,
		Schema: provider.Schema,
	}

	c.mutex.Lock()
	c.Running = append(c.Running, res)
	c.mutex.Unlock()

	return res, nil
}

type ProviderVersions struct {
	Providers []ProviderVersion `json:"providers"`
}

type ProviderVersion struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (c *coordinator) tryProviderUpdate(provider *Provider, update UpdateProvidersConfig) (*Provider, error) {
	if provider.Path == "" {
		return nil, errors.New("cannot determine installation path for provider")
	}

	binPath := provider.binPath()
	stat, err := os.Stat(binPath)
	if err != nil {
		return nil, err
	}

	if update.RefreshInterval > 0 {
		mtime := stat.ModTime()
		secs := time.Since(mtime).Seconds()
		if secs < float64(update.RefreshInterval) {
			lastRefresh := time.Since(mtime).String()
			log.Debug().
				Str("last-refresh", lastRefresh).
				Str("provider", provider.Name).
				Msg("no need to update provider")
			return provider, nil
		}
	}

	latest, err := LatestVersion(provider.Name)
	if err != nil {
		return nil, err
	}

	semver := semver.Parser{}
	diff, err := semver.Compare(provider.Version, latest)
	if err != nil {
		return nil, err
	}
	if diff >= 0 {
		return provider, nil
	}

	log.Info().
		Str("installed", provider.Version).
		Str("latest", latest).
		Msg("found a new version for '" + provider.Name + "' provider")
	provider, err = InstallVersion(provider.Name, latest)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if err := os.Chtimes(binPath, now, now); err != nil {
		log.Warn().
			Str("provider", provider.Name).
			Msg("failed to update refresh time on provider")
	}

	return provider, nil
}

func (c *coordinator) Close(p *RunningProvider) {
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

func (c *coordinator) LoadSchema(name string) *resources.Schema {
	if x, ok := builtinProviders[name]; ok {
		return x.Runtime.Schema
	}

	provider, ok := c.Providers[name]
	if !ok {
		return nil
	}
	return provider.Schema
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
