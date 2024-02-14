// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-plugin"
	"github.com/stretchr/testify/assert"
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
