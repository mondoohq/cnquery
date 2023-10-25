// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package llx

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v9/types"
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
	rawDataJSON(types.Time, &never, "blfbjef", &CodeBundle{}, &res)
	require.Equal(t, res.String(), "\"Never\"")
	require.True(t, json.Valid(res.Bytes()))
}
