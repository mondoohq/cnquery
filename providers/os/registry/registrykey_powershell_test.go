// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package registry

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWindowsRegistryKeyItemParser(t *testing.T) {
	r, err := os.Open("./testdata/registrykey.json")
	require.NoError(t, err)

	items, err := ParsePowershellRegistryKeyItems(r)
	assert.Nil(t, err)
	assert.Equal(t, 10, len(items))
	assert.Equal(t, "ConsentPromptBehaviorAdmin", items[0].Key)
	assert.Equal(t, 4, items[0].Value.Kind)
	assert.Equal(t, int64(5), items[0].Value.Number)
	assert.Equal(t, int64(5), items[0].GetRawValue())
	assert.Equal(t, "5", items[0].String())
}

func TestWindowsRegistryKeyChildParser(t *testing.T) {
	r, err := os.Open("./testdata/registrykey-children.json")
	require.NoError(t, err)

	items, err := ParsePowershellRegistryKeyChildren(r)
	assert.Nil(t, err)
	assert.Equal(t, 5, len(items))
}

func TestWindowsRegistryKeyMultiStringParser(t *testing.T) {
	r, err := os.Open("./testdata/registrykey_multistring.json")
	require.NoError(t, err)

	items, err := ParsePowershellRegistryKeyItems(r)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(items))
	assert.Equal(t, "Machine", items[0].Key)
	assert.Equal(t, 7, items[0].Value.Kind)
	assert.Equal(t, []any{
		"Software\\Microsoft\\Windows NT\\CurrentVersion\\Print",
		"Software\\Microsoft\\Windows NT\\CurrentVersion\\Windows",
		"System\\CurrentControlSet\\Control\\Print\\Printers",
	}, items[0].GetRawValue())
}

func TestNormalizeMultiSz(t *testing.T) {
	t.Run("empty MULTI_SZ artifact is normalized to empty slice", func(t *testing.T) {
		// Windows API returns [""] for empty REG_MULTI_SZ (\0\0 bytes)
		result := normalizeMultiSz([]string{""})
		assert.Equal(t, []string{}, result)
	})
	t.Run("nil input stays nil", func(t *testing.T) {
		assert.Nil(t, normalizeMultiSz(nil))
	})
	t.Run("non-empty values are preserved", func(t *testing.T) {
		assert.Equal(t, []string{"foo"}, normalizeMultiSz([]string{"foo"}))
		assert.Equal(t, []string{"foo", "bar"}, normalizeMultiSz([]string{"foo", "bar"}))
	})
	t.Run("multiple empty strings are preserved", func(t *testing.T) {
		// Two empty strings is distinct from the single-empty artifact
		assert.Equal(t, []string{"", ""}, normalizeMultiSz([]string{"", ""}))
	})
}
