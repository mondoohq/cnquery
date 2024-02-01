// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/muesli/termenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	pp "go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/resources"
	coreconf "go.mondoo.com/cnquery/v10/providers/core/config"
	"go.mondoo.com/cnquery/v10/providers/core/resources/versions/semver"
	"google.golang.org/grpc/status"
)

var BuiltinCoreID = coreconf.Config.ID

// var Coordinator = coordinator{
// 	RunningByID:      map[string]*RunningProvider{},
// 	RunningEphemeral: map[*RunningProvider]struct{}{},
// 	runtimes:         map[string]*Runtime{},
// }

func NewCoordinator() *Coordinator {
	return &Coordinator{
		RunningByID:      map[string]*RunningProvider{},
		RunningEphemeral: map[*RunningProvider]struct{}{},
		runtimes:         map[string]*Runtime{},
	}
}

var AvailableProviders Providers

type Coordinator struct {
	RunningByID      map[string]*RunningProvider
	RunningEphemeral map[*RunningProvider]struct{}

	unprocessedRuntimes []*Runtime
	runtimes            map[string]*Runtime
	runtimeCnt          int
	mutex               sync.Mutex
}

type builtinProvider struct {
	Runtime *RunningProvider
	Config  *pp.Provider
}

type RunningProvider struct {
	Name   string
	ID     string
	Plugin pp.ProviderPlugin
	Client *plugin.Client
	Schema *resources.Schema

	// isClosed is true for any provider that is not running anymore,
	// either via shutdown or via crash
	isClosed bool
	// isShutdown is only used once during provider shutdown
	isShutdown bool
	// provider errors which are evaluated and printed during shutdown of the provider
	err          error
	lock         sync.Mutex
	shutdownLock sync.Mutex
	interval     time.Duration
	gracePeriod  time.Duration
}

// initialize the heartbeat with the provider
func (p *RunningProvider) heartbeat() error {
	if err := p.doOneHeartbeat(p.interval + p.gracePeriod); err != nil {
		p.Shutdown()
		return err
	}

	go func() {
		for !p.isCloseOrShutdown() {
			if err := p.doOneHeartbeat(p.interval + p.gracePeriod); err != nil {
				p.Shutdown()
				break
			}

			time.Sleep(p.interval)
		}
	}()

	return nil
}

func (p *RunningProvider) doOneHeartbeat(t time.Duration) error {
	_, err := p.Plugin.Heartbeat(&pp.HeartbeatReq{
		Interval: uint64(t),
	})
	if err != nil {
		if status, ok := status.FromError(err); ok {
			if status.Code() == 12 {
				return errors.New("please update the provider plugin for " + p.Name)
			}
		}
		return errors.New("cannot establish heartbeat with the provider plugin for " + p.Name)
	}
	return nil
}

func (p *RunningProvider) isCloseOrShutdown() bool {
	p.shutdownLock.Lock()
	defer p.shutdownLock.Unlock()
	return p.isClosed || p.isShutdown
}

func (p *RunningProvider) Shutdown() error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.isShutdown {
		return nil
	}

	// This is an error that happened earlier, so we print it directly.
	// The error this function returns is about failing to shutdown.
	if p.err != nil {
		log.Error().Msg(p.err.Error())
	}

	var err error
	if !p.isClosed {
		_, err = p.Plugin.Shutdown(&pp.ShutdownReq{})
		if err != nil {
			log.Debug().Err(err).Str("plugin", p.Name).Msg("error in plugin shutdown")
		}

		// If the plugin was not in active use, we may not have a client at this
		// point. Since all of this is run within a sync-lock, we can check the
		// client and if it exists use it to send the kill signal.
		if p.Client != nil {
			p.Client.Kill()
		}
		p.shutdownLock.Lock()
		p.isClosed = true
		p.isShutdown = true
		p.shutdownLock.Unlock()
	} else {
		p.shutdownLock.Lock()
		p.isShutdown = true
		p.shutdownLock.Unlock()
	}

	return err
}

type UpdateProvidersConfig struct {
	// if true, will try to update providers when new versions are available
	Enabled bool
	// seconds until we try to refresh the providers version again
	RefreshInterval int
}

func (c *Coordinator) Start(id string, isEphemeral bool, update UpdateProvidersConfig) (*RunningProvider, error) {
	if x, ok := builtinProviders[id]; ok {
		// We don't warn for core providers, which are the only providers
		// built into the binary (for now).
		if id != BuiltinCoreID && id != mockProvider.ID {
			log.Warn().Msg("using builtin provider for " + x.Config.Name)
		}
		if id == mockProvider.ID {
			mp := x.Runtime.Plugin.(*mockProviderService)
			mp.Init(x.Runtime)
		}
		return x.Runtime, nil
	}

	if AvailableProviders == nil {
		var err error
		AvailableProviders, err = ListActive()
		if err != nil {
			return nil, err
		}
	}

	provider, ok := AvailableProviders[id]
	if !ok {
		return nil, errors.New("cannot find provider " + id)
	}

	if update.Enabled {
		// We do not stop on failed updates. Up until some other errors happens,
		// things are still functional. We want to consider failure, possibly
		// with a config entry in the future.
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

	pluginCmd := exec.Command(provider.binPath(), []string{"run_as_plugin", "--log-level", zerolog.GlobalLevel().String()}...)

	addColorConfig(pluginCmd)

	pluginLogger := &hclogger{Logger: log.Logger}
	pluginLogger.SetLevel(hclog.Warn)
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

	res := &RunningProvider{
		Name:        provider.Name,
		ID:          provider.ID,
		Plugin:      raw.(pp.ProviderPlugin),
		Client:      client,
		Schema:      provider.Schema,
		interval:    2 * time.Second,
		gracePeriod: 3 * time.Second,
	}

	if err := res.heartbeat(); err != nil {
		return nil, err
	}

	c.mutex.Lock()
	if isEphemeral {
		c.RunningEphemeral[res] = struct{}{}
	} else {
		if c.RunningByID[res.ID] != nil {
			log.Error().Msg("overriding a running provider")
		}
		c.RunningByID[res.ID] = res
		log.Warn().Msgf("starting provider %s", res.Name)
	}
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

func (c *Coordinator) tryProviderUpdate(provider *Provider, update UpdateProvidersConfig) (*Provider, error) {
	if provider.Path == "" {
		return nil, errors.New("cannot determine installation path for provider")
	}

	statPath := provider.confJSONPath()
	stat, err := os.Stat(statPath)
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
		log.Warn().Msg(err.Error())
		// we can just continue with the existing provider, no need to error up,
		// the warning is enough since we are still functional
		return provider, nil
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
	provider, err = installVersion(provider.Name, latest)
	if err != nil {
		return nil, err
	}
	PrintInstallResults([]*Provider{provider})
	now := time.Now()
	if err := os.Chtimes(statPath, now, now); err != nil {
		log.Warn().
			Str("provider", provider.Name).
			Msg("failed to update refresh time on provider")
	}

	return provider, nil
}

func (c *Coordinator) NewRuntime() *Runtime {
	return c.newRuntime(false)
}

func (c *Coordinator) newRuntime(isEphemeral bool) *Runtime {
	res := &Runtime{
		Coordinator:     c,
		providers:       map[string]*ConnectedProvider{},
		schema:          newExtensibleSchema(),
		recording:       NullRecording{},
		shutdownTimeout: defaultShutdownTimeout,
		isEphemeral:     isEphemeral,
	}
	res.schema.runtime = res

	// TODO: do this dynamically in the future
	// Once these calls are removed, please remember to update mock.go to explicitly
	// load all schemas on startup.
	res.schema.unsafeLoadAll()
	// TODO: this step too shouild be optional only, even when loading all.
	// It is executed when the we connect via a provider, so doing it here is
	// overkill.
	res.schema.unsafeRefresh()

	if !isEphemeral {
		c.mutex.Lock()
		c.unprocessedRuntimes = append(c.unprocessedRuntimes, res)
		c.runtimeCnt++
		cnt := c.runtimeCnt
		c.mutex.Unlock()
		log.Debug().Msg("Started a new runtime (" + strconv.Itoa(cnt) + " total)")
	}

	return res
}

func (c *Coordinator) NewRuntimeFrom(parent *Runtime) *Runtime {
	res := c.NewRuntime()
	res.recording = parent.Recording()
	for k, v := range parent.providers {
		res.providers[k] = v
	}
	return res
}

// RuntimFor an asset will return a new or existing runtime for a given asset.
// If a runtime for this asset already exists, it will re-use it. If the runtime
// is new, it will create it and detect the provider.
// The asset and parent must be defined.
func (c *Coordinator) RuntimeFor(asset *inventory.Asset, parent *Runtime) (*Runtime, error) {
	c.mutex.Lock()
	c.unsafeRefreshRuntimes()
	res := c.unsafeGetAssetRuntime(asset)
	c.mutex.Unlock()

	if res != nil {
		return res, nil
	}

	res = c.NewRuntimeFrom(parent)
	return res, res.DetectProvider(asset)
}

// EphemeralRuntimeFor an asset, creates a new ephemeral runtime and connectors.
// These are designed to be thrown away at the end of their use.
// Note: at the time of writing they may still share auxiliary providers with
// other runtimes, e.g. if provider X spawns another provider Y, the latter
// may be a shared provider. The majority of memory load should be on the
// primary provider (eg X in this case) so that it can effectively clear
// its memory at the end of its use.
func (c *Coordinator) EphemeralRuntimeFor(asset *inventory.Asset) (*Runtime, error) {
	res := c.newRuntime(true)
	return res, res.DetectProvider(asset)
}

// Only call this with a mutex lock around it!
func (c *Coordinator) unsafeRefreshRuntimes() {
	var remaining []*Runtime
	for i := range c.unprocessedRuntimes {
		rt := c.unprocessedRuntimes[i]
		if asset := rt.asset(); asset == nil || !c.unsafeSetAssetRuntime(asset, rt) {
			remaining = append(remaining, rt)
		}
	}
	c.unprocessedRuntimes = remaining
}

func (c *Coordinator) unsafeGetAssetRuntime(asset *inventory.Asset) *Runtime {
	if asset.Mrn != "" {
		if rt := c.runtimes[asset.Mrn]; rt != nil {
			return rt
		}
	}
	for _, id := range asset.PlatformIds {
		if rt := c.runtimes[id]; rt != nil {
			return rt
		}
	}
	return nil
}

// Returns true if we were able to set the runtime index for this asset,
// i.e. if either the MRN and/or its platform IDs were identified.
func (c *Coordinator) unsafeSetAssetRuntime(asset *inventory.Asset, runtime *Runtime) bool {
	found := false
	if asset.Mrn != "" {
		c.runtimes[asset.Mrn] = runtime
		found = true
	}
	for _, id := range asset.PlatformIds {
		c.runtimes[id] = runtime
		found = true
	}
	return found
}

func (c *Coordinator) Stop(provider *RunningProvider, isEphemeral bool) error {
	if provider == nil {
		return nil
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if isEphemeral {
		delete(c.RunningEphemeral, provider)
	} else {
		found := c.RunningByID[provider.ID]
		if found != nil {
			delete(c.RunningByID, provider.ID)
		}
	}

	return provider.Shutdown()
}

func (c *Coordinator) Shutdown() {
	c.mutex.Lock()

	for cur := range c.RunningEphemeral {
		if err := cur.Shutdown(); err != nil {
			log.Warn().Err(err).Str("provider", cur.Name).Msg("failed to shut down provider")
		}
		cur.isClosed = true
		cur.Client.Kill()
	}
	c.RunningEphemeral = map[*RunningProvider]struct{}{}

	for _, runtime := range c.RunningByID {
		if err := runtime.Shutdown(); err != nil {
			log.Warn().Err(err).Str("provider", runtime.Name).Msg("failed to shut down provider")
		}
		runtime.isClosed = true
		runtime.Client.Kill()
	}
	c.RunningByID = map[string]*RunningProvider{}
	c.runtimes = map[string]*Runtime{}
	c.runtimeCnt = 0
	c.unprocessedRuntimes = []*Runtime{}

	c.mutex.Unlock()
}

// LoadSchema for a given provider. Providers also cache their Schemas, so
// calling this with the same provider multiple times will use the loaded
// cached schema after the first call.
func (c *Coordinator) LoadSchema(name string) (*resources.Schema, error) {
	if x, ok := builtinProviders[name]; ok {
		return x.Runtime.Schema, nil
	}

	provider, ok := AvailableProviders[name]
	if !ok {
		return nil, errors.New("cannot find provider '" + name + "'")
	}

	if provider.Schema == nil {
		if err := provider.LoadResources(); err != nil {
			return nil, errors.Wrap(err, "failed to load provider '"+name+"' resources info")
		}
	}

	return provider.Schema, nil
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
