// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The HuggingFace API returns "gated" either as a bool (false) or as a string
// ("auto"/"manual"); both must decode without error.
func TestGatedValueUnmarshal(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		gated   bool
		mode    string
		wantErr bool
	}{
		{"bool false", `false`, false, "false", false},
		{"bool true", `true`, true, "true", false},
		{"string auto", `"auto"`, true, "auto", false},
		{"string manual", `"manual"`, true, "manual", false},
		{"string false", `"false"`, false, "false", false},
		{"string empty", `""`, false, "", false},
		{"unexpected number", `5`, false, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var g GatedValue
			err := json.Unmarshal([]byte(tt.json), &g)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.gated, g.IsGated)
			assert.Equal(t, tt.mode, g.Mode)
		})
	}
}

func TestModelListUnmarshal(t *testing.T) {
	// Bare array form (the normal list response).
	var arr ModelList
	require.NoError(t, json.Unmarshal([]byte(`[{"id":"a/1"},{"id":"a/2"}]`), &arr))
	require.Len(t, arr.Models, 2)
	assert.Equal(t, "a/1", arr.Models[0].ID)

	// Object-wrapped form.
	var obj ModelList
	require.NoError(t, json.Unmarshal([]byte(`{"models":[{"id":"a/1"}]}`), &obj))
	require.Len(t, obj.Models, 1)

	// A "gated" string inside a list entry must not break array decoding.
	var gated ModelList
	require.NoError(t, json.Unmarshal([]byte(`[{"id":"a/1","gated":"manual"}]`), &gated))
	require.Len(t, gated.Models, 1)
	assert.True(t, gated.Models[0].Gated.IsGated)
}

func TestDatasetAndSpaceListUnmarshal(t *testing.T) {
	var dl DatasetList
	require.NoError(t, json.Unmarshal([]byte(`[{"id":"a/d1"}]`), &dl))
	require.Len(t, dl.Datasets, 1)

	var sl SpaceList
	require.NoError(t, json.Unmarshal([]byte(`[{"id":"a/s1"}]`), &sl))
	require.Len(t, sl.Spaces, 1)
}

func TestWebhookListUnmarshal(t *testing.T) {
	// Bare array form.
	var arr WebhookList
	require.NoError(t, json.Unmarshal([]byte(`[{"id":"w1"},{"id":"w2"}]`), &arr))
	require.Len(t, arr, 2)

	// Object-wrapped form.
	var obj WebhookList
	require.NoError(t, json.Unmarshal([]byte(`{"webhooks":[{"id":"w1"}]}`), &obj))
	require.Len(t, obj, 1)
	assert.Equal(t, "w1", obj[0].ID)
}
