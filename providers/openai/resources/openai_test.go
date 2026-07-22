// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

func TestUnixToTime(t *testing.T) {
	// A zero unix timestamp is how the OpenAI API surfaces a null/absent time
	// (e.g. a never-used API key). It must map to the Go zero time so callers
	// can detect it and emit null instead of a year-1 timestamp.
	assert.True(t, unixToTime(0).IsZero())

	ts := int64(1719878400) // 2024-07-02T00:00:00Z
	got := unixToTime(ts)
	assert.False(t, got.IsZero())
	assert.Equal(t, ts, got.Unix())
}

func newTestModel(id string) *mqlOpenaiModel {
	return &mqlOpenaiModel{Id: plugin.TValue[string]{Data: id}}
}

func TestOpenaiModelIsFineTuned(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"gpt-4o", false},
		{"o3-mini", false},
		{"ft:gpt-4o-mini:my-org:custom:abc123", true},
		{"", false},
	}
	for _, tc := range tests {
		got, err := newTestModel(tc.id).isFineTuned()
		assert.NoError(t, err)
		assert.Equal(t, tc.want, got, tc.id)
	}
}

func TestOpenaiModelBaseModel(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"gpt-4o", ""}, // base model, not fine-tuned
		{"ft:gpt-4o-mini:my-org:custom:abc123", "gpt-4o-mini"}, // full fine-tuned form
		{"ft:gpt-3.5-turbo:acme::xyz", "gpt-3.5-turbo"},        // base name unaffected by later colons
		{"ft:", ""}, // degenerate, no base
	}
	for _, tc := range tests {
		got, err := newTestModel(tc.id).baseModel()
		assert.NoError(t, err)
		assert.Equal(t, tc.want, got, tc.id)
	}
}
