// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers/os/registry"
)

// stringItems builds a name->item map mirroring readPowershellKey's lower-cased
// keys for string registry values.
func stringItems(kv map[string]string) map[string]registry.RegistryKeyItem {
	items := map[string]registry.RegistryKeyItem{}
	for k, v := range kv {
		items[k] = registry.RegistryKeyItem{
			Key:   k,
			Value: registry.RegistryKeyValue{String: v},
		}
	}
	return items
}

func TestPowershellIntPtr(t *testing.T) {
	t.Run("present and 1 returns pointer to 1", func(t *testing.T) {
		items := dwordItems(map[string]int64{"enablescriptblocklogging": 1})
		got := powershellIntPtr(items, "EnableScriptBlockLogging")
		require.NotNil(t, got)
		assert.Equal(t, int64(1), *got)
	})

	t.Run("present and explicit 0 returns pointer to 0 (not nil)", func(t *testing.T) {
		// nullable correctness: an explicit 0 must be distinguishable from absent
		items := dwordItems(map[string]int64{"enablescriptblocklogging": 0})
		got := powershellIntPtr(items, "EnableScriptBlockLogging")
		require.NotNil(t, got)
		assert.Equal(t, int64(0), *got)
	})

	t.Run("absent returns nil", func(t *testing.T) {
		assert.Nil(t, powershellIntPtr(dwordItems(nil), "EnableScriptBlockLogging"))
		assert.Nil(t, powershellIntPtr(nil, "EnableTranscripting"))
	})

	t.Run("value name matching is case insensitive", func(t *testing.T) {
		items := dwordItems(map[string]int64{"enabletranscripting": 1})
		got := powershellIntPtr(items, "EnableTranscripting")
		require.NotNil(t, got)
		assert.Equal(t, int64(1), *got)
	})

	t.Run("unrelated value names do not match", func(t *testing.T) {
		items := dwordItems(map[string]int64{"someothervalue": 1})
		assert.Nil(t, powershellIntPtr(items, "EnableScriptBlockLogging"))
	})
}

func TestPowershellStringPtr(t *testing.T) {
	t.Run("present returns pointer to value", func(t *testing.T) {
		items := stringItems(map[string]string{"executionpolicy": "AllSigned"})
		got := powershellStringPtr(items, "ExecutionPolicy")
		require.NotNil(t, got)
		assert.Equal(t, "AllSigned", *got)
	})

	t.Run("present and explicit empty string returns pointer (not nil)", func(t *testing.T) {
		// nullable correctness: an explicit "" must be distinguishable from absent
		items := stringItems(map[string]string{"executionpolicy": ""})
		got := powershellStringPtr(items, "ExecutionPolicy")
		require.NotNil(t, got)
		assert.Equal(t, "", *got)
	})

	t.Run("absent returns nil", func(t *testing.T) {
		assert.Nil(t, powershellStringPtr(stringItems(nil), "ExecutionPolicy"))
		assert.Nil(t, powershellStringPtr(nil, "ExecutionPolicy"))
	})

	t.Run("value name matching is case insensitive", func(t *testing.T) {
		items := stringItems(map[string]string{"executionpolicy": "RemoteSigned"})
		got := powershellStringPtr(items, "ExecutionPolicy")
		require.NotNil(t, got)
		assert.Equal(t, "RemoteSigned", *got)
	})
}
