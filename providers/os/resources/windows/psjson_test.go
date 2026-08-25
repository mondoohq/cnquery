// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPSInt64Array(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  PSInt64Array
	}{
		{name: "bare array", input: `[1,2]`, want: PSInt64Array{1, 2}},
		{name: "bare and empty", input: `[]`, want: PSInt64Array{}},
		// The shape a Select-Object calculated property produces. A plain
		// []int64 tag decodes it to empty, which reports "no security
		// services running" on a host running Credential Guard.
		{name: "wrapped by Count", input: `{"value":[1,2],"Count":2}`, want: PSInt64Array{1, 2}},
		{name: "wrapped and empty", input: `{"value":[],"Count":0}`, want: PSInt64Array{}},
		{name: "wrapped with capital Value", input: `{"Value":[3],"Count":1}`, want: PSInt64Array{3}},
		// A one-element list PowerShell flattened out of its array.
		{name: "single flattened element", input: `2`, want: PSInt64Array{2}},
		{name: "absent", input: `null`, want: nil},
		// A calculated property yielding nothing serializes as {} rather than
		// as null.
		{name: "empty object", input: `{}`, want: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got PSInt64Array
			require.NoError(t, json.Unmarshal([]byte(tc.input), &got))
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestPSInt64ArrayDistinguishesAbsentFromEmpty(t *testing.T) {
	// An absent property has to stay distinguishable from a present but empty
	// list: the first is "unknown", the second is "none configured", and an
	// audit reads them differently.
	var absent PSInt64Array
	require.NoError(t, json.Unmarshal([]byte(`null`), &absent))
	assert.Nil(t, absent)

	var empty PSInt64Array
	require.NoError(t, json.Unmarshal([]byte(`[]`), &empty))
	assert.NotNil(t, empty)
	assert.Len(t, empty, 0)
}

func TestPSUnwrapListRejectsGarbage(t *testing.T) {
	var got PSInt64Array
	assert.Error(t, json.Unmarshal([]byte(`{"value":[1,`), &got))
}
