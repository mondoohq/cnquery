// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetString(t *testing.T) {
	m := map[string]any{
		"general.architecture": "llama",
		"general.parameter":    float64(42),
		"general.nil":          nil,
	}
	assert.Equal(t, "llama", getString(m, "general.architecture"))
	assert.Equal(t, "", getString(m, "missing"), "missing key returns empty string")
	assert.Equal(t, "", getString(m, "general.parameter"), "non-string value returns empty string")
	assert.Equal(t, "", getString(m, "general.nil"), "nil value returns empty string")
}

func TestGetInt(t *testing.T) {
	// JSON numbers decode into map[string]any as float64, which is the real
	// wire type from the ollama client; int64 is covered for completeness.
	m := map[string]any{
		"float": float64(8030261248),
		"int64": int64(4096),
		"str":   "128",
		"nil":   nil,
	}
	assert.Equal(t, int64(8030261248), getInt(m, "float"), "large float64 preserved without precision loss")
	assert.Equal(t, int64(4096), getInt(m, "int64"))
	assert.Equal(t, int64(0), getInt(m, "missing"), "missing key returns zero")
	assert.Equal(t, int64(0), getInt(m, "str"), "non-numeric value returns zero")
	assert.Equal(t, int64(0), getInt(m, "nil"), "nil value returns zero")
}

func TestGetArchInt(t *testing.T) {
	m := map[string]any{
		"llama.context_length":       float64(131072),
		"llama.attention.head_count": float64(32),
		"qwen2.embedding_length":     float64(3584),
	}
	assert.Equal(t, int64(131072), getArchInt(m, "llama", "context_length"))
	assert.Equal(t, int64(32), getArchInt(m, "llama", "attention.head_count"))
	assert.Equal(t, int64(3584), getArchInt(m, "qwen2", "embedding_length"))
	assert.Equal(t, int64(0), getArchInt(m, "llama", "embedding_length"), "wrong arch prefix returns zero")
	assert.Equal(t, int64(0), getArchInt(m, "", "context_length"), "empty arch returns zero, not a panic")
}

func TestGetStringSlice(t *testing.T) {
	m := map[string]any{
		"general.languages": []any{"en", "de", "fr"},
		"mixed":             []any{"en", float64(1), "de"},
		"scalar":            "en",
		"nil":               nil,
		"empty":             []any{},
	}
	assert.Equal(t, []interface{}{"en", "de", "fr"}, getStringSlice(m, "general.languages"))
	assert.Equal(t, []interface{}{"en", "de"}, getStringSlice(m, "mixed"), "non-string elements are skipped")
	assert.Equal(t, []interface{}{}, getStringSlice(m, "scalar"), "scalar (non-array) value yields empty slice")
	assert.Equal(t, []interface{}{}, getStringSlice(m, "missing"), "missing key yields empty slice")
	assert.Equal(t, []interface{}{}, getStringSlice(m, "nil"), "nil value yields empty slice")
	assert.Equal(t, []interface{}{}, getStringSlice(m, "empty"))
}
