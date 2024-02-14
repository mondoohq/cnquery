// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-plugin"
	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	pp "go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
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
		mockPlugin := pp.NewMockProviderPlugin(ctrl)
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
