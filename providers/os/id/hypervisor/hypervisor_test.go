// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package hypervisor_test

import (
	"testing"

	subject "go.mondoo.com/mql/v13/providers/os/id/hypervisor"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers/os/connection/mock"
	"go.mondoo.com/mql/v13/providers/os/detector"
)

func TestHypervisorDarwinMachdepCpuFeatures(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/macos_machdep_cpu_features.toml"))
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	hypervisor, ok := subject.Hypervisor(conn, platform)
	require.True(t, ok)

	assert.Equal(t, "VMware", hypervisor)
}

func TestHypervisorDarwinKernHvVMMPresent(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/macos_apple_virtualization_like_tart.toml"))
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	hypervisor, ok := subject.Hypervisor(conn, platform)
	require.True(t, ok)

	assert.Equal(t, "Apple Virtualization", hypervisor)
}

func TestHypervisorDarwinSystemProfiler(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/macos_system_profiler.toml"))
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	hypervisor, ok := subject.Hypervisor(conn, platform)
	require.True(t, ok)

	assert.Equal(t, "VirtualBox", hypervisor)
}

func TestHypervisorWindowsWin32Manufacturer(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/windows_ciminstance_win32_computersystem_manufacturer.toml"))
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	hypervisor, ok := subject.Hypervisor(conn, platform)
	require.True(t, ok)

	assert.Equal(t, "VMware", hypervisor)
}

func TestHypervisorWindowsWmicGetModel(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/windows_wmic_computersystem_get_model.toml"))
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	hypervisor, ok := subject.Hypervisor(conn, platform)
	require.True(t, ok)

	assert.Equal(t, "VirtualBox", hypervisor)
}

func TestHypervisorWindowsServer2022SMBIOSBIOSVersion(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/windows_serer_2022_running_hyper_v.toml"))
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	hypervisor, ok := subject.Hypervisor(conn, platform)
	require.True(t, ok)

	assert.Equal(t, "Hyper-V", hypervisor)
}

func TestHypervisorLinuxDmidecode(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/linux_dmidecode.toml"))
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	hypervisor, ok := subject.Hypervisor(conn, platform)
	require.True(t, ok)

	assert.Equal(t, "RHEV Hypervisor", hypervisor)
}

func TestHypervisorLinuxSystemdDetectVirt(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/linux_systemd_detect_virt.toml"))
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	hypervisor, ok := subject.Hypervisor(conn, platform)
	require.True(t, ok)

	assert.Equal(t, "KVM", hypervisor)
}

func TestHypervisorLinuxDMIProductName(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/linux_dmi_product_name.toml"))
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	hypervisor, ok := subject.Hypervisor(conn, platform)
	require.True(t, ok)

	assert.Equal(t, "VMware", hypervisor)
}

func TestHypervisorLinuxOpenShiftVirtualization(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/linux_openshift_virtualization.toml"))
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	hypervisor, ok := subject.Hypervisor(conn, platform)
	require.True(t, ok)

	assert.Equal(t, "OpenShift Virtualization", hypervisor)
}
