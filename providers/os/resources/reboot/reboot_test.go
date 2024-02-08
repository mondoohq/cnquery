// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reboot

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/mock"
)

func TestRebootOnUbuntu(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/ubuntu_reboot.toml")
	mock, err := mock.New(0, filepath, &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "ubuntu",
			Family: []string{"linux", "debian", "ubuntu"},
		},
	})
	require.NoError(t, err)

	lb, err := New(mock)
	require.NoError(t, err)

	required, err := lb.RebootPending()
	require.NoError(t, err)
	assert.Equal(t, true, required)
}

func TestRebootOnRhel(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/redhat_kernel_reboot.toml")
	mock, err := mock.New(0, filepath, &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "redhat",
			Family: []string{"linux", "redhat"},
		},
	})
	require.NoError(t, err)

	lb, err := New(mock)
	require.NoError(t, err)

	required, err := lb.RebootPending()
	require.NoError(t, err)

	assert.Equal(t, true, required)
}

func TestRebootOnWindows(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/windows_reboot.toml")
	mock, err := mock.New(0, filepath, &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "windows",
			Family: []string{"windows"},
		},
	})
	require.NoError(t, err)

	lb, err := New(mock)
	require.NoError(t, err)

	required, err := lb.RebootPending()
	require.NoError(t, err)
	assert.Equal(t, true, required)
}
