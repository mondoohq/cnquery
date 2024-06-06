// Copyright (c) Mondoo, Inc.
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
