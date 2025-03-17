// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package hypervisor_test

import (
	"testing"

	subject "go.mondoo.com/cnquery/v11/providers/os/id/hypervisor"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v11/providers/os/detector"
)

func TestHypervisorDarwinMachdepCpuFeatures(t *testing.T) {
	conn, err := mock.New(0, "./testdata/macos_machdep_cpu_features.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	hypervisor, ok := subject.Hypervisor(conn, platform)
	require.True(t, ok)

	assert.Equal(t, "VMware", hypervisor)
}

func TestHypervisorDarwinKernHvVMMPresent(t *testing.T) {
	conn, err := mock.New(0, "./testdata/macos_apple_virtualization_like_tart.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	hypervisor, ok := subject.Hypervisor(conn, platform)
	require.True(t, ok)

	assert.Equal(t, "Apple Virtualization", hypervisor)
}

func TestHypervisorDarwinSystemProfiler(t *testing.T) {
	conn, err := mock.New(0, "./testdata/macos_system_profiler.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	hypervisor, ok := subject.Hypervisor(conn, platform)
	require.True(t, ok)

	assert.Equal(t, "VirtualBox", hypervisor)
}

func TestHypervisorWindowsWin32Manufacturer(t *testing.T) {
	conn, err := mock.New(0, "./testdata/windows_ciminstance_win32_computersystem_manufacturer.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	hypervisor, ok := subject.Hypervisor(conn, platform)
	require.True(t, ok)

	assert.Equal(t, "VMware", hypervisor)
}

func TestHypervisorWindowsWmicGetModel(t *testing.T) {
	conn, err := mock.New(0, "./testdata/windows_wmic_computersystem_get_model.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	hypervisor, ok := subject.Hypervisor(conn, platform)
	require.True(t, ok)

	assert.Equal(t, "VirtualBox", hypervisor)
}

func TestHypervisorLinuxDmidecode(t *testing.T) {
	conn, err := mock.New(0, "./testdata/linux_dmidecode.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	hypervisor, ok := subject.Hypervisor(conn, platform)
	require.True(t, ok)

	assert.Equal(t, "RHEV Hypervisor", hypervisor)
}

func TestHypervisorLinuxSystemdDetectVirt(t *testing.T) {
	conn, err := mock.New(0, "./testdata/linux_systemd_detect_virt.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	hypervisor, ok := subject.Hypervisor(conn, platform)
	require.True(t, ok)

	assert.Equal(t, "KVM", hypervisor)
}

func TestHypervisorLinuxDMIProductName(t *testing.T) {
	conn, err := mock.New(0, "./testdata/linux_dmi_product_name.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	hypervisor, ok := subject.Hypervisor(conn, platform)
	require.True(t, ok)

	assert.Equal(t, "VMware", hypervisor)
}
