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
)

//go:generate mockgen -source=./coordinator.go -destination=./mock_coordinator.go -package=providers
//go:generate mockgen -source=../providers-sdk/v1/plugin/interface.go -destination=./mock_plugin_interface.go -package=providers

type ProvidersCoordinator interface {
	NewRuntime() *Runtime
	NewRuntimeFrom(parent *Runtime) *Runtime
	RuntimeFor(asset *inventory.Asset, parent *Runtime) (*Runtime, error)
	RemoveRuntime(runtime *Runtime)
	GetRunningProvider(id string, update UpdateProvidersConfig) (*RunningProvider, error)
	SetProviders(providers Providers)
	Providers() Providers
	LoadSchema(name string) (*resources.Schema, error)
	Shutdown()
}

var BuiltinCoreID = coreconf.Config.ID

var Coordinator ProvidersCoordinator = &coordinator{
	runningByID: map[string]*RunningProvider{},
	runtimes:    map[string]*Runtime{},
}

type coordinator struct {
	providers   Providers
	runningByID map[string]*RunningProvider

	unprocessedRuntimes []*Runtime
	runtimes            map[string]*Runtime
	runtimeCnt          int
	mutex               sync.Mutex
}

type builtinProvider struct {
	Runtime *RunningProvider
	Config  *pp.Provider
}

type UpdateProvidersConfig struct {
	// if true, will try to update providers when new versions are available
	Enabled bool
	// seconds until we try to refresh the providers version again
	RefreshInterval int
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

func (c *coordinator) NewRuntime() *Runtime {
	return c.newRuntime()
}

func (c *coordinator) newRuntime() *Runtime {
	res := &Runtime{
		coordinator:     c,
		providers:       map[string]*ConnectedProvider{},
		schema:          newExtensibleSchema(),
		recording:       NullRecording{},
		shutdownTimeout: defaultShutdownTimeout,
	}
	res.schema.runtime = res

	// TODO: do this dynamically in the future
	// Once these calls are removed, please remember to update mock.go to explicitly
	// load all schemas on startup.
	res.schema.unsafeLoadAll()
	// TODO: this step too should be optional only, even when loading all.
	// It is executed when the we connect via a provider, so doing it here is
	// overkill.
	res.schema.unsafeRefresh()

	c.mutex.Lock()
	c.unprocessedRuntimes = append(c.unprocessedRuntimes, res)
	c.runtimeCnt++
	cnt := c.runtimeCnt
	c.mutex.Unlock()
	log.Debug().Msg("Started a new runtime (" + strconv.Itoa(cnt) + " total)")

	return res
}

func (c *coordinator) NewRuntimeFrom(parent *Runtime) *Runtime {
	res := c.NewRuntime()
	res.UpstreamConfig = parent.UpstreamConfig
	res.recording = parent.Recording()
	for k, v := range parent.providers {
		res.providers[k] = v
	}
	return res
}

// RuntimeFor an asset will return a new or existing runtime for a given asset.
// If a runtime for this asset already exists, it will re-use it. If the runtime
// is new, it will create it and detect the provider.
// The asset and parent must be defined.
func (c *coordinator) RuntimeFor(asset *inventory.Asset, parent *Runtime) (*Runtime, error) {
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

// Only call this with a mutex lock around it!
func (c *coordinator) unsafeRefreshRuntimes() {
	var remaining []*Runtime
	for i := range c.unprocessedRuntimes {
		rt := c.unprocessedRuntimes[i]
		if asset := rt.asset(); asset == nil || !c.unsafeSetAssetRuntime(asset, rt) {
			remaining = append(remaining, rt)
		}
	}
	c.unprocessedRuntimes = remaining
}

func (c *coordinator) unsafeGetAssetRuntime(asset *inventory.Asset) *Runtime {
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
func (c *coordinator) unsafeSetAssetRuntime(asset *inventory.Asset, runtime *Runtime) bool {
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

// RemoveRuntime will remove a runtime from the coordinator. This can potentially
// shutdown a running provider if it's not referenced by another runtime.
func (c *coordinator) RemoveRuntime(runtime *Runtime) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.unsafeRefreshRuntimes()

	if runtime.asset() != nil {
		delete(c.runtimes, runtime.asset().Mrn)
		for _, id := range runtime.asset().PlatformIds {
			delete(c.runtimes, id)
		}
	}

	// Analyze the providers that are still being used by active runtimes
	usedProviders := map[string]struct{}{}
	for _, r := range c.runtimes {
		for _, p := range r.providers {
			usedProviders[p.Instance.ID] = struct{}{}
		}
	}

	// Shutdown any providers that are not being used anymore
	for id, p := range c.runningByID {
		if _, ok := usedProviders[id]; !ok {
			log.Debug().Msg("shutting down unused provider " + p.Name)
			if err := c.stop(p); err != nil {
				log.Warn().Err(err).Str("provider", p.Name).Msg("failed to shut down provider")
			}
		}
	}
}

func (c *coordinator) GetRunningProvider(id string, update UpdateProvidersConfig) (*RunningProvider, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	running := c.runningByID[id]
	if running == nil {
		var err error
		running, err = c.unsafeStartProvider(id, update)
		if err != nil {
			return nil, err
		}
	}
	return running, nil
}

// unsafeStartProvider will start a provider and add it to the list of running providers. Must be called
// with a mutex lock around it.
func (c *coordinator) unsafeStartProvider(id string, update UpdateProvidersConfig) (*RunningProvider, error) {
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

	if c.providers == nil {
		var err error
		c.providers, err = ListActive()
		if err != nil {
			return nil, err
		}
	}

	provider, ok := c.providers[id]
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

	c.runningByID[res.ID] = res
	return res, nil
}

func (c *coordinator) SetProviders(providers Providers) {
	c.providers = providers
}

func (c *coordinator) Providers() Providers {
	return c.providers
}

// stop will stop a provider and remove it from the list of running providers. Must be called
// with a mutex lock around it.
func (c *coordinator) stop(provider *RunningProvider) error {
	found := c.runningByID[provider.ID]
	if found != nil {
		delete(c.runningByID, provider.ID)
	}
	return provider.Shutdown()
}

func (c *coordinator) Shutdown() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, provider := range c.runningByID {
		if err := provider.Shutdown(); err != nil {
			log.Warn().Err(err).Str("provider", provider.Name).Msg("failed to shut down provider")
		}
		provider.Client.Kill()
	}
	c.runningByID = map[string]*RunningProvider{}
	c.runtimes = map[string]*Runtime{}
	c.runtimeCnt = 0
	c.unprocessedRuntimes = []*Runtime{}
}

// LoadSchema for a given provider. Providers also cache their Schemas, so
// calling this with the same provider multiple times will use the loaded
// cached schema after the first call.
func (c *coordinator) LoadSchema(name string) (*resources.Schema, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if x, ok := builtinProviders[name]; ok {
		return x.Runtime.Schema, nil
	}

	provider, ok := c.providers[name]
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
