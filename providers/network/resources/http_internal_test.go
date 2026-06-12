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

	t.Run("matches directive names case-insensitively", func(t *testing.T) {
		_, mode, report := parseXssProtectionDirectives([]any{"1; Mode=Block; Report=https://example.com/r"})
		assert.Equal(t, llx.StringData("Block"), mode)
		assert.Equal(t, llx.StringData("https://example.com/r"), report)
	})
}

func TestParseStsDirectives(t *testing.T) {
	t.Run("parses max-age, includeSubDomains, and preload", func(t *testing.T) {
		preload, includeSubDomains, maxAge := parseStsDirectives([]any{"max-age=31536000; includeSubDomains; preload"})
		assert.True(t, preload)
		assert.True(t, includeSubDomains)
		assert.Equal(t, llx.TimeData(llx.DurationToTime(31536000)), maxAge)
	})

	t.Run("matches directive names case-insensitively", func(t *testing.T) {
		preload, includeSubDomains, maxAge := parseStsDirectives([]any{"Max-Age=31536000; includeSubdomains; Preload"})
		assert.True(t, preload)
		assert.True(t, includeSubDomains)
		assert.Equal(t, llx.TimeData(llx.DurationToTime(31536000)), maxAge)
	})

	t.Run("reports an invalid max-age value", func(t *testing.T) {
		preload, includeSubDomains, maxAge := parseStsDirectives([]any{"max-age=oops"})
		assert.False(t, preload)
		assert.False(t, includeSubDomains)
		require.Error(t, maxAge.Error)
		assert.Contains(t, maxAge.Error.Error(), "oops")
	})

	t.Run("leaves absent directives unset", func(t *testing.T) {
		preload, includeSubDomains, maxAge := parseStsDirectives([]any{"max-age=600"})
		assert.False(t, preload)
		assert.False(t, includeSubDomains)
		assert.Equal(t, llx.TimeData(llx.DurationToTime(600)), maxAge)
	})
}

func TestParseContentTypeDirectives(t *testing.T) {
	t.Run("parses the media type and parameters", func(t *testing.T) {
		typ, params := parseContentTypeDirectives([]any{"text/html; charset=UTF-8"})
		assert.Equal(t, llx.StringData("text/html"), typ)
		assert.Equal(t, map[string]any{"charset": "UTF-8"}, params.Value)
	})

	t.Run("normalizes the media type and parameter names, keeps values as sent", func(t *testing.T) {
		typ, params := parseContentTypeDirectives([]any{"Text/HTML; CHARSET=UTF-8"})
		assert.Equal(t, llx.StringData("text/html"), typ)
		assert.Equal(t, map[string]any{"charset": "UTF-8"}, params.Value)
	})

	t.Run("has no parameters when none are sent", func(t *testing.T) {
		typ, params := parseContentTypeDirectives([]any{"application/json"})
		assert.Equal(t, llx.StringData("application/json"), typ)
		assert.Equal(t, llx.NilData, params)
	})
}

func TestParseSetCookieDirectives(t *testing.T) {
	t.Run("parses the cookie name, value, and attributes", func(t *testing.T) {
		name, value, params := parseSetCookieDirectives([]any{"sid=abc123; Secure; HttpOnly; Max-Age=100"})
		assert.Equal(t, llx.StringData("sid"), name)
		assert.Equal(t, llx.StringData("abc123"), value)
		assert.Equal(t, map[string]any{"secure": "", "httponly": "", "max-age": "100"}, params.Value)
	})

	t.Run("keeps cookie name and value casing, normalizes attribute names", func(t *testing.T) {
		name, value, params := parseSetCookieDirectives([]any{"SessionId=AbC123; SECURE; httpOnly"})
		assert.Equal(t, llx.StringData("SessionId"), name)
		assert.Equal(t, llx.StringData("AbC123"), value)
		assert.Equal(t, map[string]any{"secure": "", "httponly": ""}, params.Value)
	})

	t.Run("keeps attribute value casing", func(t *testing.T) {
		_, _, params := parseSetCookieDirectives([]any{"sid=1; Domain=Example.COM; SameSite=Strict"})
		assert.Equal(t, map[string]any{"domain": "Example.COM", "samesite": "Strict"}, params.Value)
	})
}

func TestParseCspDirectives(t *testing.T) {
	t.Run("parses directives into a map", func(t *testing.T) {
		m := parseCspDirectives([]any{"default-src 'self'; script-src 'none'"})
		assert.Equal(t, map[string]any{"default-src": "'self'", "script-src": "'none'"}, m)
	})

	t.Run("normalizes directive names, keeps values as sent", func(t *testing.T) {
		m := parseCspDirectives([]any{"Default-Src 'self'; SCRIPT-SRC https://CDN.example.com"})
		assert.Equal(t, map[string]any{"default-src": "'self'", "script-src": "https://CDN.example.com"}, m)
	})
}
