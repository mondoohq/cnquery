// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-plugin"
	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	pp "go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	gomock "go.uber.org/mock/gomock"
)

func TestShutdown(t *testing.T) {
	ctrl := gomock.NewController(t)
	c := &coordinator{
		runningByID: map[string]*RunningProvider{},
		runtimes:    map[string]*Runtime{},
	}

	for i := range 10 {
		c.runtimes[fmt.Sprintf("runtime-%d", i)] = &Runtime{}

		id := fmt.Sprintf("test-%d", i)

		// Make sure Shutdown is called for all running providers
		mockPlugin := NewMockProviderPlugin(ctrl)
		mockPlugin.EXPECT().Shutdown(gomock.Any()).Times(1).Return(nil, nil)
		c.runningByID[id] = &RunningProvider{
			ID:     id,
			Plugin: mockPlugin,
			Client: &plugin.Client{},
		}
	}

	c.Shutdown()

	// Make sure all running providers and runtimes are removed
	assert.Empty(t, c.runningByID)
	assert.Empty(t, c.runtimes)
}

func TestRemoveRuntime_AssetMrn(t *testing.T) {
	mrn := "mrn1"
	r := &Runtime{
		Provider: &ConnectedProvider{
			Connection: &pp.ConnectRes{
				Asset: &inventory.Asset{Mrn: mrn},
			},
		},
	}

	c := &coordinator{
		runningByID: map[string]*RunningProvider{},
		runtimes: map[string]*Runtime{
			mrn:    r,
			"mrn2": r,
		},
	}

	c.RemoveRuntime(r)
	assert.NotContains(t, c.runtimes, mrn)
	assert.Contains(t, c.runtimes, "mrn2")
}

func TestRemoveRuntime_PlatformId(t *testing.T) {
	pId := "platformId1"
	r := &Runtime{
		Provider: &ConnectedProvider{
			Connection: &pp.ConnectRes{
				Asset: &inventory.Asset{PlatformIds: []string{pId}},
			},
		},
	}

	c := &coordinator{
		runningByID: map[string]*RunningProvider{},
		runtimes: map[string]*Runtime{
			pId:           r,
			"platformId2": r,
		},
	}

	c.RemoveRuntime(r)
	assert.NotContains(t, c.runtimes, pId)
	assert.Contains(t, c.runtimes, "platformId2")
}

func TestRemoveRuntime_StopUnusedProvider(t *testing.T) {
	ctrl := gomock.NewController(t)

	// Setup 1 provider with 1 runtime
	mockPlugin1 := NewMockProviderPlugin(ctrl)
	mockPlugin1.EXPECT().Shutdown(gomock.Any()).Times(1).Return(nil, nil)
	p1 := &RunningProvider{
		ID:     "provider1",
		Plugin: mockPlugin1,
	}
	r1 := &Runtime{
		providers: map[string]*ConnectedProvider{
			"provider1": {Instance: p1},
		},
		Provider: &ConnectedProvider{
			Instance: p1,
			Connection: &pp.ConnectRes{
				Asset: &inventory.Asset{PlatformIds: []string{"platformId1"}},
			},
		},
	}

	// Setup another provider with another runtime
	mockPlugin2 := NewMockProviderPlugin(ctrl)
	mockPlugin2.EXPECT().Shutdown(gomock.Any()).Times(1).Return(nil, nil)
	p2 := &RunningProvider{
		ID:     "provider2",
		Plugin: mockPlugin2,
	}
	r2 := &Runtime{
		providers: map[string]*ConnectedProvider{
			"provider2": {Instance: p2},
		},
		Provider: &ConnectedProvider{
			Instance: p2,
			Connection: &pp.ConnectRes{
				Asset: &inventory.Asset{PlatformIds: []string{"platformId2"}},
			},
		},
	}

	c := &coordinator{
		runningByID: map[string]*RunningProvider{
			"provider1": p1,
			"provider2": p2,
		},
		runtimes: map[string]*Runtime{
			"platformId1": r1,
			"platformId2": r2,
		},
	}

	// Remove all runtimes
	c.RemoveRuntime(r1)
	c.RemoveRuntime(r2)

	// Verify that all provider are stopped
	assert.Empty(t, c.runningByID)
}

func TestRemoveRuntime_RemoveDeadProvider(t *testing.T) {
	ctrl := gomock.NewController(t)

	// Setup 1 closed provider with 1 runtime
	mockPlugin1 := NewMockProviderPlugin(ctrl)
	p1 := &RunningProvider{
		ID:       "provider1",
		Plugin:   mockPlugin1,
		isClosed: true,
	}
	r1 := &Runtime{
		providers: map[string]*ConnectedProvider{
			"provider1": {Instance: p1},
		},
		Provider: &ConnectedProvider{
			Instance: p1,
			Connection: &pp.ConnectRes{
				Asset: &inventory.Asset{PlatformIds: []string{"platformId1"}},
			},
		},
	}

	// Setup another provider with another runtime
	r2 := &Runtime{
		providers: map[string]*ConnectedProvider{
			"provider2": {Instance: p1},
		},
		Provider: &ConnectedProvider{
			Instance: p1,
			Connection: &pp.ConnectRes{
				Asset: &inventory.Asset{PlatformIds: []string{"platformId2"}},
			},
		},
	}

	c := &coordinator{
		runningByID: map[string]*RunningProvider{
			"provider1": p1,
		},
		runtimes: map[string]*Runtime{
			"platformId1": r1,
			"platformId2": r2,
		},
	}

	// Remove 1 runtime
	c.RemoveRuntime(r1)

	// Verify that the provider has been removed because it crashed
	assert.Empty(t, c.runningByID)

	c.RemoveRuntime(r2)
}

func TestRemoveRuntime_UsedProvider(t *testing.T) {
	ctrl := gomock.NewController(t)

	// Setup 1 provider with 1 runtime
	mockPlugin1 := NewMockProviderPlugin(ctrl)
	p1 := &RunningProvider{
		ID:     "provider1",
		Plugin: mockPlugin1,
	}
	r1 := &Runtime{
		providers: map[string]*ConnectedProvider{
			"provider1": {Instance: p1},
		},
		Provider: &ConnectedProvider{
			Instance: p1,
			Connection: &pp.ConnectRes{
				Asset: &inventory.Asset{PlatformIds: []string{"platformId1"}},
			},
		},
	}

	// Setup another provider with the same runtime
	r2 := &Runtime{
		providers: map[string]*ConnectedProvider{
			"provider2": {Instance: p1},
		},
		Provider: &ConnectedProvider{
			Instance: p1,
			Connection: &pp.ConnectRes{
				Asset: &inventory.Asset{PlatformIds: []string{"platformId2"}},
			},
		},
	}

	c := &coordinator{
		runningByID: map[string]*RunningProvider{
			"provider1": p1,
		},
		runtimes: map[string]*Runtime{
			"platformId1": r1,
			"platformId2": r2,
		},
	}

	// Remove the first runtime
	c.RemoveRuntime(r1)

	// Verify that the first provider is stopped
	assert.Contains(t, c.runningByID, "provider1")
}
