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

func TestParseSysvServices(t *testing.T) {
	mock, err := mock.New(0, "./testdata/amzn1.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "amazonlinux",
			Family: []string{"linux"},
		},
	})
	require.NoError(t, err)

	sysv := SysVServiceManager{conn: mock}
	services, err := sysv.services()
	require.NoError(t, err)
	assert.Equal(t, 4, len(services), "detected the right amount of services")
}

func TestParseSysvServicesRunlevel(t *testing.T) {
	mock, err := mock.New(0, "./testdata/amzn1.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "amazonlinux",
			Family: []string{"linux"},
		},
	})
	require.NoError(t, err)

	sysv := SysVServiceManager{conn: mock}
	level, err := sysv.serviceRunLevel()
	require.NoError(t, err)
	assert.Equal(t, 3, len(level), "detected the right amount of services")
	assert.Equal(t, 4, len(level["sshd"]))
}

func TestParseSysvServicesRunning(t *testing.T) {
	mock, err := mock.New(0, "./testdata/amzn1.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "amazonlinux",
			Family: []string{"linux"},
		},
	})
	require.NoError(t, err)

	sysv := SysVServiceManager{conn: mock}
	// iterate over services and check if they are running
	running, err := sysv.running([]string{"sshd", "ntpd", "acpid"})
	require.NoError(t, err)
	assert.Equal(t, 3, len(running), "detected the right amount of services")
	assert.Equal(t, false, running["acpid"])
	assert.Equal(t, true, running["sshd"])
}
