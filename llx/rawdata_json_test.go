// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package llx

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/types"
)

func TestRawDataJson_removeUnderscoreKeys(t *testing.T) {
	tests := map[string]struct {
		input []string
		want  []string
	}{
		"no underscores": {
			input: []string{"this", "that"},
			want:  []string{"this", "that"},
		},
		"trailing underscore": {
			input: []string{"this", "that", "_"},
			want:  []string{"this", "that"},
		},
		"leading underscore": {
			input: []string{"_", "this", "that"},
			want:  []string{"this", "that"},
		},
		"alternating underscores": {
			input: []string{"_", "this", "_", "that", "_"},
			want:  []string{"this", "that"},
		},
		"all underscores": {
			input: []string{"_", "_", "_"},
			want:  []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := removeUnderscoreKeys(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRawDataJson_nevertime(t *testing.T) {
	never := NeverPastTime
	var res bytes.Buffer
	require.NoError(t, rawDataJSON(types.Time, &never, "blfbjef", &CodeBundle{}, &res))
	require.Equal(t, res.String(), "\"Never\"")
	require.True(t, json.Valid(res.Bytes()))
}

func TestRawDataJson_duration(t *testing.T) {
	const mins = 60         // 1 minute in seconds
	const hours = 60 * mins // 1 hour in seconds
	const days = 24 * hours // 24 hours in seconds
	dur := DurationToTime(4*days + 13*hours + 42*mins)
	var res bytes.Buffer
	require.NoError(t, rawDataJSON(types.Time, &dur, "", &CodeBundle{}, &res))
	require.Equal(t, res.String(), "\"4 days 13 hours 42 minutes\"")
	require.True(t, json.Valid(res.Bytes()))
}

func TestRawDataJson_Umlauts(t *testing.T) {
	var res bytes.Buffer
	require.NoError(t, rawDataJSON(types.String, "Systemintegrit\x84t", "blfbjef", &CodeBundle{}, &res))
	require.Equal(t, res.String(), "\"Systemintegrit\\ufffdt\"")
	require.True(t, json.Valid(res.Bytes()))
}

// Two block entries can resolve to the same human-readable label (e.g.
// `package("apparmor").installed` and `package("apparmor-utils").installed`
// both label as `package.installed`). The JSON output must still parse — and
// in particular must not contain duplicate map keys, which downstream
// consumers reject when unmarshaling into proto map fields.
func TestRawDataJson_refMapJSON_collidingLabels(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		data   map[string]any
	}{
		{
			name: "two colliding",
			labels: map[string]string{
				"sum-a": "package.installed",
				"sum-b": "package.installed",
			},
			data: map[string]any{
				"sum-a": BoolData(true),
				"sum-b": BoolData(false),
			},
		},
		{
			name: "three colliding",
			labels: map[string]string{
				"sum-a": "package.installed",
				"sum-b": "package.installed",
				"sum-c": "package.installed",
			},
			data: map[string]any{
				"sum-a": BoolData(true),
				"sum-b": BoolData(false),
				"sum-c": BoolData(true),
			},
		},
		{
			name: "mixed collisions and unique",
			labels: map[string]string{
				"sum-a": "package.installed",
				"sum-b": "package.installed",
				"sum-c": "asset.family",
			},
			data: map[string]any{
				"sum-a": BoolData(true),
				"sum-b": BoolData(false),
				"sum-c": StringData("debian"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bundle := &CodeBundle{Labels: &Labels{Labels: tc.labels}}
			var buf bytes.Buffer
			require.NoError(t, refMapJSON(types.Block, tc.data, "", bundle, &buf))
			require.True(t, json.Valid(buf.Bytes()), "output is not valid JSON: %s", buf.String())

			// json.Unmarshal silently drops duplicate keys, so walk the raw
			// token stream to assert all top-level keys are unique.
			dec := json.NewDecoder(bytes.NewReader(buf.Bytes()))
			seen := map[string]struct{}{}
			var depth int
			for {
				tok, err := dec.Token()
				if err != nil {
					break
				}
				switch v := tok.(type) {
				case json.Delim:
					if v == '{' {
						depth++
					}
					if v == '}' {
						depth--
					}
				case string:
					if depth == 1 && dec.More() {
						if _, dup := seen[v]; dup {
							t.Fatalf("duplicate JSON key %q in %s", v, buf.String())
						}
						seen[v] = struct{}{}
					}
				}
			}
			require.Equal(t, len(tc.data), len(seen), "every entry must be present in JSON: %s", buf.String())
		})
	}
}
