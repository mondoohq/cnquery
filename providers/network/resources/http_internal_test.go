// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

func TestHttpHeaderServer(t *testing.T) {
	t.Run("returns the Server header value", func(t *testing.T) {
		h := &mqlHttpHeader{
			Params: plugin.TValue[map[string]any]{
				Data:  map[string]any{"Server": []any{"nginx"}},
				State: plugin.StateIsSet,
			},
		}
		v, err := h.server()
		require.NoError(t, err)
		assert.Equal(t, "nginx", v)
	})

	t.Run("is null when the Server header is absent", func(t *testing.T) {
		h := &mqlHttpHeader{
			Params: plugin.TValue[map[string]any]{
				Data:  map[string]any{},
				State: plugin.StateIsSet,
			},
		}
		v, err := h.server()
		require.NoError(t, err)
		assert.Equal(t, "", v)
		assert.NotEqual(t, 0, h.Server.State&plugin.StateIsNull)
	})
}

func TestParseXssProtectionDirectives(t *testing.T) {
	t.Run("parses enabled, mode, and report", func(t *testing.T) {
		enabled, mode, report := parseXssProtectionDirectives([]any{"1; mode=block; report=https://example.com/r"})
		assert.Equal(t, llx.BoolTrue, enabled)
		assert.Equal(t, llx.StringData("block"), mode)
		assert.Equal(t, llx.StringData("https://example.com/r"), report)
	})

	t.Run("parses a disabled header", func(t *testing.T) {
		enabled, mode, report := parseXssProtectionDirectives([]any{"0"})
		assert.Equal(t, llx.BoolFalse, enabled)
		assert.Equal(t, llx.NilData, mode)
		assert.Equal(t, llx.NilData, report)
	})

	t.Run("ignores directives that are not part of the header syntax", func(t *testing.T) {
		enabled, _, report := parseXssProtectionDirectives([]any{"1; max-age=99"})
		assert.Equal(t, llx.BoolTrue, enabled)
		assert.Equal(t, llx.NilData, report)
	})
}
