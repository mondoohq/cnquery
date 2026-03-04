// Copyright (c) Mondoo, Inc.
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
	t.Run("client build 26100 supported", func(t *testing.T) {
		pf := &inventory.Platform{
			Version: "26100",
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
