// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
)

func TestFamilyFromNames(t *testing.T) {
	tests := []struct {
		name string
		id   string
		root string
		want string
	}{
		{"base mistral", "mistral-large-latest", "", "Mistral"},
		{"open mistral nemo prefers Nemo over Mistral", "open-mistral-nemo", "", "Nemo"},
		{"codestral before mistral", "codestral-2405", "", "Codestral"},
		{"mixtral not mistral", "open-mixtral-8x22b", "", "Mixtral"},
		{"pixtral", "pixtral-12b-2409", "", "Pixtral"},
		{"ministral", "ministral-8b-latest", "", "Ministral"},
		{"magistral", "magistral-medium-latest", "", "Magistral"},
		{"mathstral", "mathstral-7b", "", "Mathstral"},
		{"devstral", "devstral-small-2505", "", "Devstral"},
		{"embed", "mistral-embed", "", "Embed"},
		{"moderation", "mistral-moderation-latest", "", "Moderation"},
		{"case-insensitive", "MISTRAL-LARGE", "", "Mistral"},
		{"unknown returns empty", "gpt-4o", "", ""},
		{"fine-tuned falls back to root", "ft:abc123", "codestral-2405", "Codestral"},
		{"id wins over root", "mixtral-8x7b", "mistral-large", "Mixtral"},
		{"empty root not consulted", "unknown-model", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, familyFromNames(tc.id, tc.root))
		})
	}
}

func TestParseParameterSize(t *testing.T) {
	tests := []struct {
		name string
		id   string
		root string
		want string
	}{
		{"simple b suffix", "ministral-8b-latest", "", "8B"},
		{"mixture of experts", "open-mixtral-8x22b", "", "8x22B"},
		{"small moe", "open-mixtral-8x7b", "", "8x7B"},
		{"7b", "open-mistral-7b", "", "7B"},
		{"decimal size", "some-model-3.8b-instruct", "", "3.8B"},
		{"already uppercase unit", "some-model-7B", "", "7B"},
		{"m suffix", "tiny-500m-model", "", "500M"},
		{"no size token", "mistral-large-latest", "", ""},
		{"date is not a size", "codestral-2405", "", ""},
		{"fine-tuned falls back to root", "ft:custom", "open-mistral-7b", "7B"},
		{"id size wins over root", "my-13b-tune", "open-mistral-7b", "13B"},
		{"empty when neither matches", "custom", "base", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, parseParameterSize(tc.id, tc.root))
		})
	}
}

func TestTimeFromUnix(t *testing.T) {
	assert.Nil(t, timeFromUnix(0), "zero timestamp maps to null")

	got := timeFromUnix(1700000000)
	if assert.NotNil(t, got) {
		assert.True(t, got.Equal(time.Unix(1700000000, 0)))
	}
}

func TestTimeFromUnixPtr(t *testing.T) {
	assert.Nil(t, timeFromUnixPtr(nil), "nil maps to null")

	zero := int64(0)
	assert.Nil(t, timeFromUnixPtr(&zero), "zero maps to null")

	ts := int64(1700000000)
	got := timeFromUnixPtr(&ts)
	if assert.NotNil(t, got) {
		assert.True(t, got.Equal(time.Unix(ts, 0)))
	}
}

func TestFloatDataPtr(t *testing.T) {
	assert.Same(t, llx.NilData, floatDataPtr(nil), "nil float maps to MQL null")

	v := 0.0
	got := floatDataPtr(&v)
	assert.Equal(t, types.Float, got.Type)
	assert.Equal(t, 0.0, got.Value, "a real zero is preserved, not treated as null")

	v = 0.7
	got = floatDataPtr(&v)
	assert.Equal(t, 0.7, got.Value)
}
