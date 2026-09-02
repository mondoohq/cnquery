// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package registry

import (
	"os"
	"strings"
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

func TestWindowsRegistryKeyQwordParser(t *testing.T) {
	// kind 11 == REG_QWORD; PowerShell emits the value as a JSON number
	const data = `[{
		"key": "LowMemoryThreshold",
		"value": { "kind": 11, "data": 4294967296 }
	}]`

	items, err := ParsePowershellRegistryKeyItems(strings.NewReader(data))
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, 11, items[0].Value.Kind)
	// a QWORD must surface as the integer value, not the float bit-pattern
	assert.Equal(t, int64(4294967296), items[0].Value.Number)
	assert.Equal(t, int64(4294967296), items[0].GetRawValue())
	assert.Equal(t, "4294967296", items[0].String())
}

// A REG_QWORD above 2^53 must survive parsing exactly. Decoding through float64
// silently rounds these: 9007199254740993 lands on 9007199254740992, and a
// FILETIME-shaped value loses its low digits. PowerShell types REG_QWORD as
// System.Int64, so the whole int64 range has to round trip.
func TestWindowsRegistryKeyQwordPrecision(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		want int64
	}{
		{"2^53 + 1, the first integer float64 cannot hold", "9007199254740993", 9007199254740993},
		{"FILETIME with significant low digits", "133712345678901234", 133712345678901234},
		{"int64 max", "9223372036854775807", 9223372036854775807},
		{"int64 min", "-9223372036854775808", -9223372036854775808},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := `[{"key":"T","value":{"kind":11,"data":` + tc.raw + `}}]`

			items, err := ParsePowershellRegistryKeyItems(strings.NewReader(data))
			require.NoError(t, err)
			require.Len(t, items, 1)

			assert.Equal(t, tc.want, items[0].Value.Number)
			assert.Equal(t, tc.want, items[0].GetRawValue())
			// the string fallback must agree with the number, digit for digit
			assert.Equal(t, tc.raw, items[0].String())
		})
	}
}

// A REG_BINARY still parses once numbers arrive as json.Number rather than
// float64, since its elements are decoded through the same path.
func TestWindowsRegistryKeyBinaryStillParses(t *testing.T) {
	const data = `[{"key":"B","value":{"kind":3,"data":[0,1,127,255]}}]`

	items, err := ParsePowershellRegistryKeyItems(strings.NewReader(data))
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, []byte{0, 1, 127, 255}, items[0].Value.Binary)
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

// TestWindowsRegistryEmptyListParsers covers the shape PowerShell produces for
// a key that exists but has nothing in it. `ConvertTo-Json` writes nothing at
// all for an empty array, so the collection scripts return an empty stream
// rather than `[]`. Decoding that used to fail with "unexpected end of JSON
// input", which turned "this key is present and configures nothing" into an
// unreadable key. The SCHANNEL subtree hits this routinely: the Protocols key
// holds only subkeys and no values, and a protocol's Client subkey can exist
// with neither Enabled nor DisabledByDefault set.
func TestWindowsRegistryEmptyListParsers(t *testing.T) {
	for _, in := range []string{"", "\r\n", "   \n"} {
		items, err := ParsePowershellRegistryKeyItems(strings.NewReader(in))
		require.NoError(t, err)
		assert.Empty(t, items)

		children, err := ParsePowershellRegistryKeyChildren(strings.NewReader(in))
		require.NoError(t, err)
		assert.Empty(t, children)
	}

	// malformed output is still an error, not silently an empty key
	_, err := ParsePowershellRegistryKeyItems(strings.NewReader("{not json"))
	assert.Error(t, err)
	_, err = ParsePowershellRegistryKeyChildren(strings.NewReader("{not json"))
	assert.Error(t, err)
}
