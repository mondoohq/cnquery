// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"sync"

	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/resources"
)

type localCoordinator struct {
	parent Coordinator

	runningByID      map[string]*RunningProvider
	runningEphemeral map[*RunningProvider]struct{}
	mutex            sync.Mutex
}

func NewLocalCoordinator(parent Coordinator) Coordinator {
	return &localCoordinator{
		parent:           parent,
		runningByID:      map[string]*RunningProvider{},
		runningEphemeral: map[*RunningProvider]struct{}{},
	}
}

func (lc *localCoordinator) Start(id string, isEphemeral bool, update UpdateProvidersConfig) (*RunningProvider, error) {
	// From the parent's perspective, all providers from its children are ephemeral
	provider, err := lc.parent.Start(id, true, update)
	if err != nil {
		return nil, err
	}

	lc.mutex.Lock()
	if isEphemeral {
		lc.runningEphemeral[provider] = struct{}{}
	} else {
		lc.runningByID[provider.ID] = provider
	}
	lc.mutex.Unlock()
	return provider, nil
}

func (lc *localCoordinator) Stop(provider *RunningProvider, isEphemeral bool) error {
	if provider == nil {
		return nil
	}

	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	if isEphemeral {
		delete(lc.runningEphemeral, provider)
	} else {
		found := lc.runningByID[provider.ID]
		if found != nil {
			delete(lc.runningByID, provider.ID)
		}
	}
	return lc.parent.Stop(provider, true)
}

func (lc *localCoordinator) NewRuntime() *Runtime {
	return lc.parent.NewRuntime()
}

func (lc *localCoordinator) RuntimeFor(asset *inventory.Asset, parent *Runtime) (*Runtime, error) {
	return lc.parent.RuntimeFor(asset, parent)
}

func (lc *localCoordinator) GetRunningProviderById(id string) *RunningProvider {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()
	return lc.runningByID[id]
}

func (lc *localCoordinator) GetProviders() Providers {
	return lc.parent.GetProviders()
}

func (lc *localCoordinator) SetProviders(providers Providers) {
	lc.parent.SetProviders(providers)
}

func (lc *localCoordinator) LoadSchema(name string) (*resources.Schema, error) {
	return lc.parent.LoadSchema(name)
}

func (lc *localCoordinator) Shutdown() {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	for cur := range lc.runningEphemeral {
		lc.parent.Stop(cur, true)
	}
	lc.runningEphemeral = map[*RunningProvider]struct{}{}

	for _, runtime := range lc.runningByID {
		lc.parent.Stop(runtime, true)
	}
	lc.runningByID = map[string]*RunningProvider{}
}
