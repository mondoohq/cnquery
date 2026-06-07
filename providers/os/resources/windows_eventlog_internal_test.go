// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/mql/v13/providers/os/registry"
)

func dwordItem(key string, n int64) registry.RegistryKeyItem {
	return registry.RegistryKeyItem{Key: key, Value: registry.RegistryKeyValue{Kind: registry.DWORD, Number: n}}
}

func szItem(key, s string) registry.RegistryKeyItem {
	return registry.RegistryKeyItem{Key: key, Value: registry.RegistryKeyValue{Kind: registry.SZ, String: s}}
}

func TestRegistryItemInt(t *testing.T) {
	t.Run("DWORD value", func(t *testing.T) {
		n, ok := registryItemInt(dwordItem("MaxSize", 196608))
		assert.True(t, ok)
		assert.Equal(t, int64(196608), n)
	})

	t.Run("string decimal (Group Policy form)", func(t *testing.T) {
		n, ok := registryItemInt(szItem("Retention", "0"))
		assert.True(t, ok)
		assert.Equal(t, int64(0), n)
	})

	t.Run("string hex 0xFFFFFFFF", func(t *testing.T) {
		n, ok := registryItemInt(szItem("Retention", "0xFFFFFFFF"))
		assert.True(t, ok)
		assert.Equal(t, int64(0xFFFFFFFF), n)
	})

	t.Run("string -1", func(t *testing.T) {
		n, ok := registryItemInt(szItem("Retention", "-1"))
		assert.True(t, ok)
		assert.Equal(t, int64(-1), n)
	})

	t.Run("non-numeric string", func(t *testing.T) {
		_, ok := registryItemInt(szItem("Retention", "nope"))
		assert.False(t, ok)
	})
}

func TestDecodeRetention(t *testing.T) {
	assert.Equal(t, retentionOverwriteAsNeeded, decodeRetention(0))
	assert.Equal(t, retentionNeverOverwrite, decodeRetention(0xFFFFFFFF))
	assert.Equal(t, retentionNeverOverwrite, decodeRetention(-1))
	// a positive seconds value (legacy "overwrite by days")
	assert.Equal(t, retentionOverwriteByDays, decodeRetention(604800))
}

func TestLookupInt(t *testing.T) {
	items := map[string]registry.RegistryKeyItem{
		"maxsize":   dwordItem("MaxSize", 32768),
		"retention": szItem("Retention", "0"),
	}

	n, ok := lookupInt(items, "maxsize")
	assert.True(t, ok)
	assert.Equal(t, int64(32768), n)

	_, ok = lookupInt(items, "missing")
	assert.False(t, ok)
}
