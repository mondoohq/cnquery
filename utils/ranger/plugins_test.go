// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package ranger

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13"
	"go.mondoo.com/ranger-rpc/plugins/scope"
)

func TestSysInfoHeader(t *testing.T) {
	t.Run("nil sysinfo defaults PN to GOOS", func(t *testing.T) {
		plugin := sysInfoHeader(mql.Features{}, nil)
		h := plugin.GetHeader(nil)

		parsed, err := scope.ParseXInfoHeader(h.Get("User-Agent"))
		require.NoError(t, err)

		assert.Equal(t, mql.Version, parsed["mql"])
		assert.Equal(t, mql.Build, parsed["build"])
		assert.Equal(t, runtime.GOOS, parsed["PN"])
	})

	t.Run("populates all sysinfo fields", func(t *testing.T) {
		si := &ClientSysInfo{
			PlatformName:    "ubuntu",
			PlatformVersion: "22.04",
			PlatformArch:    "amd64",
			IP:              "10.0.0.1",
			Hostname:        "myhost",
			PlatformID:      "platform-123",
			Product: Product{
				Name:    "cnspec",
				Version: "9.1.0",
				Build:   "deadbeef",
			},
		}
		plugin := sysInfoHeader(mql.Features{}, si)
		h := plugin.GetHeader(nil)

		parsed, err := scope.ParseXInfoHeader(h.Get("User-Agent"))
		require.NoError(t, err)

		assert.Equal(t, "ubuntu", parsed["PN"])
		assert.Equal(t, "22.04", parsed["PR"])
		assert.Equal(t, "amd64", parsed["PA"])
		assert.Equal(t, "10.0.0.1", parsed["IP"])
		assert.Equal(t, "myhost", parsed["HN"])
		assert.Equal(t, mql.Version, parsed["mql"])
		assert.Equal(t, "9.1.0", parsed["cnspec"])
		assert.Equal(t, "deadbeef", parsed["build"])
		assert.Equal(t, "platform-123", h.Get("Mondoo-PlatformID"))
	})

	t.Run("product name without version is ignored", func(t *testing.T) {
		si := &ClientSysInfo{
			PlatformName: "debian",
			Product:      Product{Name: "cnspec"},
		}
		plugin := sysInfoHeader(mql.Features{}, si)
		h := plugin.GetHeader(nil)

		parsed, err := scope.ParseXInfoHeader(h.Get("User-Agent"))
		require.NoError(t, err)

		_, hasProduct := parsed["cnspec"]
		assert.False(t, hasProduct, "product should not be added when version is empty")
		assert.Equal(t, mql.Version, parsed["mql"], "mql version should still be present")
	})

	t.Run("product version without name is ignored", func(t *testing.T) {
		si := &ClientSysInfo{
			PlatformName: "debian",
			Product:      Product{Version: "9.1.0"},
		}
		plugin := sysInfoHeader(mql.Features{}, si)
		h := plugin.GetHeader(nil)

		parsed, err := scope.ParseXInfoHeader(h.Get("User-Agent"))
		require.NoError(t, err)

		_, hasEmpty := parsed[""]
		assert.False(t, hasEmpty, "empty product name should not be added as a key")
		assert.Equal(t, mql.Version, parsed["mql"], "mql version should still be present")
	})

	t.Run("empty platform name defaults to GOOS", func(t *testing.T) {
		si := &ClientSysInfo{PlatformName: ""}
		plugin := sysInfoHeader(mql.Features{}, si)
		h := plugin.GetHeader(nil)

		parsed, err := scope.ParseXInfoHeader(h.Get("User-Agent"))
		require.NoError(t, err)

		assert.Equal(t, runtime.GOOS, parsed["PN"])
	})
}
