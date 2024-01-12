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

func TestRhelKernelLatest(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/redhat_kernel_reboot.toml")
	mock, err := mock.New(filepath, &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "redhat",
			Family: []string{"linux", "redhat"},
		},
	})
	require.NoError(t, err)

	lb := RpmNewestKernel{conn: mock}
	required, err := lb.RebootPending()
	require.NoError(t, err)
	assert.Equal(t, true, required)
}

func TestAmznContainerWithoutKernel(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/amzn_kernel_container.toml")
	mock, err := mock.New(filepath, &inventory.Asset{
		Platform: &inventory.Platform{
			Name:    "amazonlinux",
			Version: "2018.03",
			Family:  []string{"linux"},
		},
	})
	require.NoError(t, err)

	lb := RpmNewestKernel{conn: mock}
	required, err := lb.RebootPending()
	require.NoError(t, err)

	assert.Equal(t, false, required)
}

func TestAmznEc2Kernel(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/amzn_kernel_ec2.toml")
	mock, err := mock.New(filepath, &inventory.Asset{
		Platform: &inventory.Platform{
			Name:    "amazonlinux",
			Version: "2018.03",
			Family:  []string{"linux"},
		},
	})
	require.NoError(t, err)

	lb := RpmNewestKernel{conn: mock}
	required, err := lb.RebootPending()
	require.NoError(t, err)

	assert.Equal(t, false, required)
}
