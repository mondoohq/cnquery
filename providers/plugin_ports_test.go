// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolvePluginPortRangeFrom(t *testing.T) {
	t.Run("default when nothing is configured", func(t *testing.T) {
		r, err := resolvePluginPortRangeFrom("", "", "")
		require.NoError(t, err)
		// The whole point of the change: the default must stay out of
		// go-plugin's own 10000-25000, and inside the IANA dynamic range.
		assert.Equal(t, pluginPortRange{Min: 50000, Max: 65535}, r)
	})

	t.Run("config value wins", func(t *testing.T) {
		r, err := resolvePluginPortRangeFrom("50000-50100", "", "")
		require.NoError(t, err)
		assert.Equal(t, pluginPortRange{Min: 50000, Max: 50100}, r)
	})

	t.Run("config value tolerates whitespace", func(t *testing.T) {
		r, err := resolvePluginPortRangeFrom(" 50000 - 50100 ", "", "")
		require.NoError(t, err)
		assert.Equal(t, pluginPortRange{Min: 50000, Max: 50100}, r)
	})

	t.Run("single port range is allowed", func(t *testing.T) {
		r, err := resolvePluginPortRangeFrom("50000-50000", "", "")
		require.NoError(t, err)
		assert.Equal(t, pluginPortRange{Min: 50000, Max: 50000}, r)
	})

	t.Run("config value overrides go-plugin environment", func(t *testing.T) {
		r, err := resolvePluginPortRangeFrom("50000-50100", "20000", "21000")
		require.NoError(t, err)
		assert.Equal(t, pluginPortRange{Min: 50000, Max: 50100}, r)
	})

	t.Run("malformed config value is an error even when the environment is valid", func(t *testing.T) {
		_, err := resolvePluginPortRangeFrom("fifty-thousand", "20000", "21000")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "provider_port_range")
	})

	t.Run("go-plugin environment is honored", func(t *testing.T) {
		r, err := resolvePluginPortRangeFrom("", "20000", "21000")
		require.NoError(t, err)
		assert.Equal(t, pluginPortRange{Min: 20000, Max: 21000}, r)
	})

	t.Run("half-set go-plugin environment is an error", func(t *testing.T) {
		_, err := resolvePluginPortRangeFrom("", "20000", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be set together")

		_, err = resolvePluginPortRangeFrom("", "", "21000")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be set together")
	})

	t.Run("malformed go-plugin environment is an error", func(t *testing.T) {
		_, err := resolvePluginPortRangeFrom("", "21000", "20000")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "PLUGIN_MIN_PORT")
		assert.Contains(t, err.Error(), "greater than")
	})
}

func TestParsePluginPortRange(t *testing.T) {
	bad := map[string]string{
		"no dash":         "50000",
		"empty min":       "-50000",
		"empty max":       "50000-",
		"min above max":   "50100-50000",
		"port zero min":   "0-100",
		"port zero max":   "0-0",
		"above 65535":     "50000-70000",
		"negative":        "-1-50000",
		"not a number":    "abc-def",
		"float":           "50000.5-50100",
		"extra separator": "50000-50100-50200",
	}
	for name, in := range bad {
		t.Run(name, func(t *testing.T) {
			_, err := parsePluginPortRange(in)
			assert.Error(t, err, in)
		})
	}

	good := map[string]pluginPortRange{
		"1-65535":     {Min: 1, Max: 65535},
		"50000-50100": {Min: 50000, Max: 50100},
		"8080-8080":   {Min: 8080, Max: 8080},
	}
	for in, want := range good {
		t.Run(in, func(t *testing.T) {
			r, err := parsePluginPortRange(in)
			require.NoError(t, err)
			assert.Equal(t, want, r)
		})
	}
}
