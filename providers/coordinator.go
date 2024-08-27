// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"math"
	"os"
	"os/exec"
	"strconv"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/muesli/termenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	pp "go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/recording"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/resources"
	coreconf "go.mondoo.com/cnquery/v11/providers/core/config"
)

//go:generate mockgen -source=./coordinator.go -destination=./mock_coordinator.go -package=providers
//go:generate mockgen -source=../providers-sdk/v1/plugin/interface.go -destination=./mock_plugin_interface.go -package=providers
//go:generate mockgen -source=../providers-sdk/v1/resources/schema.go -destination=./mock_schema.go -package=providers

type ProvidersCoordinator interface {
	NextConnectionId() uint32
	NewRuntime() *Runtime
	NewRuntimeFrom(parent *Runtime) *Runtime
	RuntimeFor(asset *inventory.Asset, parent *Runtime) (*Runtime, error)
	RemoveRuntime(runtime *Runtime)
	GetRunningProvider(id string, update UpdateProvidersConfig) (*RunningProvider, error)
	SetProviders(providers Providers)
	Providers() Providers
	DeactivateProviderDiscovery()
	LoadSchema(name string) (resources.ResourcesSchema, error)
	Schema() resources.ResourcesSchema
	Shutdown()
}

var BuiltinCoreID = coreconf.Config.ID

var Coordinator ProvidersCoordinator

func newCoordinator() *coordinator {
	c := &coordinator{
		runningByID: map[string]*RunningProvider{},
		runtimes:    map[string]*Runtime{},
		schema:      newExtensibleSchema(),
	}
	c.schema.coordinator = c
	return c
}

type coordinator struct {
	lastConnectionID uint32
	connectionsLock  sync.Mutex

	providers   Providers
	runningByID map[string]*RunningProvider

	unprocessedRuntimes []*Runtime
	runtimes            map[string]*Runtime
	runtimeCnt          int
	mutex               sync.Mutex
	schema              extensibleSchema
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

func (c *coordinator) NextConnectionId() uint32 {
	c.connectionsLock.Lock()
	defer c.connectionsLock.Unlock()
	c.lastConnectionID++
	return c.lastConnectionID
}

func (c *coordinator) NewRuntime() *Runtime {
	return c.newRuntime()
}

func (c *coordinator) newRuntime() *Runtime {
	res := &Runtime{
		coordinator:     c,
		providers:       map[string]*ConnectedProvider{},
		recording:       recording.Null{},
		shutdownTimeout: defaultShutdownTimeout,
	}

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
		if rt.isClosed {
			continue
		}
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

	// Shutdown any providers that are not being used anymore.
	// We have runtimes that are used for initialising a scan, but are not
	// used for the actual scan. They reference no providers, so shouldn't affect
	// the shutdown of providers.
	uprocessedRuntimeWithProviders := false
	for _, rt := range c.unprocessedRuntimes {
		if rt.Provider != nil {
			uprocessedRuntimeWithProviders = true
		}
	}
	if len(c.runtimes) == 0 && !uprocessedRuntimeWithProviders {
		for _, p := range c.runningByID {
			log.Debug().Msg("shutting down unused provider " + p.Name)
			if err := c.stop(p); err != nil {
				log.Warn().Err(err).Str("provider", p.Name).Msg("failed to shut down provider")
			}
		}
	} else {
		// Check for killed/crashed providers and remove them from the list of running providers
		for _, p := range c.runningByID {
			if p.isCloseOrShutdown() {
				log.Warn().Str("provider", p.Name).Msg("removing closed provider")
				delete(c.runningByID, p.ID)
			}
		}
	}

	// If all providers have been killed, reset the connection IDs back to 0
	if len(c.runningByID) == 0 {
		c.connectionsLock.Lock()
		defer c.connectionsLock.Unlock()
		c.lastConnectionID = 0
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
		if id != BuiltinCoreID && id != mockProvider.ID && id != sbomProvider.ID {
			log.Warn().Msg("using builtin provider for " + x.Config.Name)
		}
		if id == mockProvider.ID {
			mp := x.Runtime.Plugin.(*mockProviderService)
			mp.Init(x.Runtime)
		}
		if id == sbomProvider.ID {
			mp := x.Runtime.Plugin.(*sbomProviderService)
			mp.Init(x.Runtime)
		}
		c.schema.Add(id, x.Runtime.Schema)
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
		updated, err := TryProviderUpdate(provider, update)
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

	connectFunc := func() (pp.ProviderPlugin, *plugin.Client, error) {
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
			return nil, nil, errors.Wrap(err, "failed to initialize plugin client")
		}

		// Request the plugin
		pluginName := "provider"
		raw, err := rpcClient.Dispense(pluginName)
		if err != nil {
			client.Kill()
			return nil, nil, errors.Wrap(err, "failed to call "+pluginName+" plugin")
		}

		return raw.(pp.ProviderPlugin), client, nil
	}

	plug, client, err := connectFunc()
	if err != nil {
		return nil, err
	}

	c.schema.Add(provider.ID, provider.Schema)

	res, err := SupervisedRunningProivder(provider.Name, provider.ID, plug, client, provider.Schema, connectFunc)
	if err != nil {
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
		provider.KillClient()
	}
	c.runningByID = map[string]*RunningProvider{}
	c.runtimes = map[string]*Runtime{}
	c.runtimeCnt = 0
	c.unprocessedRuntimes = []*Runtime{}
	c.schema.Close()
}

func (c *coordinator) DeactivateProviderDiscovery() {
	// Setting this to the max int means this value will always be larger than
	// any real timestamp for the last installation time of a provider.
	c.schema.lastRefreshed = math.MaxInt64
}

func (c *coordinator) Schema() resources.ResourcesSchema {
	return &c.schema
}

// LoadSchema for a given provider. Providers also cache their Schemas, so
// calling this with the same provider multiple times will use the loaded
// cached schema after the first call.
func (c *coordinator) LoadSchema(name string) (resources.ResourcesSchema, error) {
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
