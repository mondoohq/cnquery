// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package vmware_test

import (
	"testing"

	subject "go.mondoo.com/cnquery/v11/providers/os/id/vmware"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v11/providers/os/detector"
	"go.mondoo.com/cnquery/v11/providers/os/resources/smbios"
)

func TestDetectLinuxInstance(t *testing.T) {
	conn, err := mock.New(0, "./testdata/instance_linux.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	mgr, err := smbios.ResolveManager(conn, platform)
	require.NoError(t, err)

	identifier, name, relatedIdentifiers := subject.Detect(conn, platform, mgr)

	assert.Equal(t,
		"//platformid.api.mondoo.app/runtime/vmware/uuid/5c4c1142-a38a-b604-dfde-60730c109bac",
		identifier,
	)
	assert.Equal(t, "", name)
	require.Len(t, relatedIdentifiers, 1)
	assert.Equal(t,
		"//platformid.api.mondoo.app/runtime/vcenter/moid/vm-6143",
		relatedIdentifiers[0],
	)
}
func TestDetectLinuxInstanceWithoutVmtoolsd(t *testing.T) {
	conn, err := mock.New(0, "./testdata/instance_linux_no_vmtoolsd.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	mgr, err := smbios.ResolveManager(conn, platform)
	require.NoError(t, err)

	identifier, name, relatedIdentifiers := subject.Detect(conn, platform, mgr)

	assert.Equal(t,
		"//platformid.api.mondoo.app/runtime/vmware/uuid/5c4c1142-a38a-b604-dfde-60730c109bac",
		identifier,
	)
	assert.Equal(t, "", name)
	require.Len(t, relatedIdentifiers, 0)
}

func TestDetectWindowsInstance(t *testing.T) {
	conn, err := mock.New(0, "./testdata/instance_windows.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	mgr, err := smbios.ResolveManager(conn, platform)
	require.NoError(t, err)

	identifier, name, relatedIdentifiers := subject.Detect(conn, platform, mgr)

	assert.Equal(t,
		"//platformid.api.mondoo.app/runtime/vmware/uuid/5c4c1142-a38a-b604-dfde-60730c109bab",
		identifier,
	)
	assert.Equal(t, "", name)
	require.Len(t, relatedIdentifiers, 1)
	assert.Equal(t,
		"//platformid.api.mondoo.app/runtime/vcenter/moid/vm-6143",
		relatedIdentifiers[0],
	)
}

func TestDetectWindowsInstanceWithoutVmtoolsd(t *testing.T) {
	conn, err := mock.New(0, "./testdata/instance_windows_no_vmtoolsd.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	mgr, err := smbios.ResolveManager(conn, platform)
	require.NoError(t, err)

	identifier, name, relatedIdentifiers := subject.Detect(conn, platform, mgr)

	assert.Equal(t,
		"//platformid.api.mondoo.app/runtime/vmware/uuid/5c4c1142-a38a-b604-dfde-60730c109bab",
		identifier,
	)
	assert.Equal(t, "", name)
	require.Len(t, relatedIdentifiers, 0)
}

func TestNoMatch(t *testing.T) {
	conn, err := mock.New(0, "./testdata/no_vmware_instance.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	mgr, err := smbios.ResolveManager(conn, platform)
	require.NoError(t, err)

	identifier, name, relatedIdentifiers := subject.Detect(conn, platform, mgr)

	assert.Empty(t, identifier)
	assert.Empty(t, name)
	assert.Empty(t, relatedIdentifiers)
}
