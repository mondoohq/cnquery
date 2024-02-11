// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/mock"
)

func TestParseUpstartServicesRunning(t *testing.T) {
	mock, err := mock.New(0, "./testdata/ubuntu1404.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "ubuntu",
			Family: []string{"linux", "ubuntu"},
		},
	})
	require.NoError(t, err)

	upstart := UpstartServiceManager{SysVServiceManager{conn: mock}}

	// iterate over services and check if they are running
	services, err := upstart.List()
	require.NoError(t, err)
	assert.Equal(t, 9, len(services), "detected the right amount of services")
}
