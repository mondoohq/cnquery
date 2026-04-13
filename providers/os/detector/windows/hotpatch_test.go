// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

func TestParseWinRegistryHotpatch(t *testing.T) {

	t.Run("parse hptpatching settings correctly", func(t *testing.T) {
		data := `{
			"Name":  "Hotpatch Enrollment Package",
			"HotPatchTableSize": "4096",
			"EnableVirtualizationBasedSecurity": "1"
		}`

		m, err := ParseWinRegistryHotpatch(strings.NewReader(data))
		assert.Nil(t, err)
		assert.True(t, m)
	})

	t.Run("parse missing table size", func(t *testing.T) {
		data := `{
			"Name":  "Hotpatch Enrollment Package",
			"HotPatchTableSize": "0",
			"EnableVirtualizationBasedSecurity": "1"
		}`

		m, err := ParseWinRegistryHotpatch(strings.NewReader(data))
		assert.Nil(t, err)
		assert.False(t, m)
	})

	t.Run("parse missing name", func(t *testing.T) {
		data := `{
			"Name":  "",
			"HotPatchTableSize": "4096",
			"EnableVirtualizationBasedSecurity": "1"
		}`

		m, err := ParseWinRegistryHotpatch(strings.NewReader(data))
		assert.Nil(t, err)
		assert.False(t, m)
	})

	t.Run("parse missing VBS", func(t *testing.T) {
		data := `{
			"Name":  "Hotpatch Enrollment Package",
			"HotPatchTableSize": "1",
			"EnableVirtualizationBasedSecurity": "0"
		}`

		m, err := ParseWinRegistryHotpatch(strings.NewReader(data))
		assert.Nil(t, err)
		assert.False(t, m)
	})

	t.Run("parse empty JSON", func(t *testing.T) {
		data := `{
			"Name":  "",
			"HotPatchTableSize": "0",
			"EnableVirtualizationBasedSecurity": "0"
		}`

		m, err := ParseWinRegistryHotpatch(strings.NewReader(data))
		assert.Nil(t, err)
		assert.False(t, m)
	})
}

func TestParseWinRegistryClientHotpatch(t *testing.T) {
	t.Run("client hotpatch enabled", func(t *testing.T) {
		data := `{
			"AllowRebootlessUpdates": "1",
			"EnableVirtualizationBasedSecurity": "1"
		}`

		m, err := ParseWinRegistryClientHotpatch(strings.NewReader(data))
		assert.Nil(t, err)
		assert.True(t, m)
	})

	t.Run("client no rebootless updates", func(t *testing.T) {
		data := `{
			"AllowRebootlessUpdates": "0",
			"EnableVirtualizationBasedSecurity": "1"
		}`

		m, err := ParseWinRegistryClientHotpatch(strings.NewReader(data))
		assert.Nil(t, err)
		assert.False(t, m)
	})

	t.Run("client no VBS", func(t *testing.T) {
		data := `{
			"AllowRebootlessUpdates": "1",
			"EnableVirtualizationBasedSecurity": "0"
		}`

		m, err := ParseWinRegistryClientHotpatch(strings.NewReader(data))
		assert.Nil(t, err)
		assert.False(t, m)
	})

	t.Run("client empty values", func(t *testing.T) {
		data := `{
			"AllowRebootlessUpdates": "",
			"EnableVirtualizationBasedSecurity": ""
		}`

		m, err := ParseWinRegistryClientHotpatch(strings.NewReader(data))
		assert.Nil(t, err)
		assert.False(t, m)
	})
}

func TestHotpatchSupported(t *testing.T) {
	t.Run("client amd64 build 26100 with sufficient UBR", func(t *testing.T) {
		pf := &inventory.Platform{
			Version: "26100",
			Build:   "5000",
			Arch:    "AMD64",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "1"},
		}
		assert.True(t, hotpatchSupported(pf))
	})

	t.Run("client amd64 build 26100 with exact minimum UBR", func(t *testing.T) {
		pf := &inventory.Platform{
			Version: "26100",
			Build:   "2033",
			Arch:    "AMD64",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "1"},
		}
		assert.True(t, hotpatchSupported(pf))
	})

	t.Run("client amd64 build 26100 with UBR below minimum", func(t *testing.T) {
		pf := &inventory.Platform{
			Version: "26100",
			Build:   "2000",
			Arch:    "AMD64",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "1"},
		}
		assert.False(t, hotpatchSupported(pf))
	})

	t.Run("client arm64 build 26100 with sufficient UBR", func(t *testing.T) {
		pf := &inventory.Platform{
			Version: "26100",
			Build:   "5000",
			Arch:    "ARM64",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "1"},
		}
		assert.True(t, hotpatchSupported(pf))
	})

	t.Run("client arm64 build 26100 with exact minimum UBR", func(t *testing.T) {
		pf := &inventory.Platform{
			Version: "26100",
			Build:   "4929",
			Arch:    "ARM64",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "1"},
		}
		assert.True(t, hotpatchSupported(pf))
	})

	t.Run("client arm64 build 26100 with UBR below arm64 minimum", func(t *testing.T) {
		pf := &inventory.Platform{
			Version: "26100",
			Build:   "3775",
			Arch:    "ARM64",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "1"},
		}
		assert.False(t, hotpatchSupported(pf))
	})

	t.Run("client build 26100 with empty UBR", func(t *testing.T) {
		pf := &inventory.Platform{
			Version: "26100",
			Build:   "",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "1"},
		}
		assert.False(t, hotpatchSupported(pf))
	})

	t.Run("client build above 26100 always supported", func(t *testing.T) {
		pf := &inventory.Platform{
			Version: "27000",
			Build:   "100",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "1"},
		}
		assert.True(t, hotpatchSupported(pf))
	})

	t.Run("client build 22000 not supported", func(t *testing.T) {
		pf := &inventory.Platform{
			Version: "22000",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "1"},
		}
		assert.False(t, hotpatchSupported(pf))
	})

	t.Run("server build 20348 supported", func(t *testing.T) {
		pf := &inventory.Platform{
			Version: "20348",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "3"},
		}
		assert.True(t, hotpatchSupported(pf))
	})

	t.Run("server build 19041 not supported", func(t *testing.T) {
		pf := &inventory.Platform{
			Version: "19041",
			Labels:  map[string]string{"windows.mondoo.com/product-type": "3"},
		}
		assert.False(t, hotpatchSupported(pf))
	})
}
