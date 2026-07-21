// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package registry

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildSubKey(t *testing.T) {
	require.Equal(t, "TMPREG_T", buildSubKeyPath("T"))
}

func TestGetRegistryPath(t *testing.T) {
	t.Run("get registry path for an registry that has not been loaded yet", func(t *testing.T) {
		rh := NewRegistryHandler()
		_, err := rh.getRegistryPath("TMPREG_T")
		require.Error(t, err)
	})
	t.Run("get registry path for an registry that has been loaded", func(t *testing.T) {
		rh := NewRegistryHandler()
		rh.registries["SOFTWARE"] = "TMPREG_SOFTWARE"
		path, err := rh.getRegistryPath("SOFTWARE")
		require.NoError(t, err)
		require.Equal(t, "HKLM\\TMPREG_SOFTWARE", path)
	})
}

func TestGetRegistryKeyPath(t *testing.T) {
	t.Run("get registry key path for an registry that has not been loaded yet", func(t *testing.T) {
		rh := NewRegistryHandler()
		_, err := rh.getRegistryKeyPath("TMPREG_T", "Microsoft\\Windows")
		require.Error(t, err)
	})
	t.Run("get registry key path for an registry that has been loaded", func(t *testing.T) {
		rh := NewRegistryHandler()
		rh.registries["SOFTWARE"] = "TMPREG_SOFTWARE"
		path, err := rh.getRegistryKeyPath("SOFTWARE", "Microsoft\\Windows")
		require.NoError(t, err)
		require.Equal(t, "HKLM\\TMPREG_SOFTWARE\\Microsoft\\Windows", path)
	})
}

func TestUserRegistryID(t *testing.T) {
	// per-user ids are namespaced so they never collide with KnownRegistryFiles
	require.Equal(t, "USER_S-1-5-21-1-2-3-1001", userRegistryID("S-1-5-21-1-2-3-1001"))
	require.Equal(t, "TMPREG_USER_S-1-5-21-1-2-3-1001", buildSubKeyPath(userRegistryID("S-1-5-21-1-2-3-1001")))
}

func TestUserHiveKeyPath(t *testing.T) {
	sid := "S-1-5-21-1-2-3-1001"

	t.Run("hive not loaded", func(t *testing.T) {
		rh := NewRegistryHandler()
		_, err := rh.userHiveKeyPath(sid, "Software\\Policies")
		require.Error(t, err)
	})

	t.Run("hive loaded", func(t *testing.T) {
		rh := NewRegistryHandler()
		rh.registries[userRegistryID(sid)] = buildSubKeyPath(userRegistryID(sid))

		// sub-path under the hive
		path, err := rh.userHiveKeyPath(sid, "Software\\Policies\\Microsoft")
		require.NoError(t, err)
		require.Equal(t, "HKLM\\TMPREG_USER_"+sid+"\\Software\\Policies\\Microsoft", path)

		// leading/trailing backslashes are trimmed
		path, err = rh.userHiveKeyPath(sid, "\\Software\\")
		require.NoError(t, err)
		require.Equal(t, "HKLM\\TMPREG_USER_"+sid+"\\Software", path)

		// empty sub-path resolves to the hive root
		path, err = rh.userHiveKeyPath(sid, "")
		require.NoError(t, err)
		require.Equal(t, "HKLM\\TMPREG_USER_"+sid, path)
	})
}

func TestLoadUserHive_Validation(t *testing.T) {
	rh := NewRegistryHandler()
	require.Error(t, rh.LoadUserHive("", "C:\\Users\\x\\NTUSER.DAT"), "empty sid must error")
	require.Error(t, rh.LoadUserHive("S-1-5-21-1-2-3-1001", ""), "empty NTUSER.DAT path must error")
}
