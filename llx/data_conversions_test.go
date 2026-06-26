// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package llx_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
)

func TestVersion_Conversions(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		sv := llx.StringPrimitive("1.2.3")
		sv.Type = string(types.Version)
		rd := sv.RawData()
		require.NoError(t, rd.Error, "no error converting version to raw data")
		require.Equal(t, "1.2.3", rd.Value, "version to raw data is the same")
	})
}

func TestResultNotPanicsOnBareTypes(t *testing.T) {
	t.Run("bare ArrayLike with elements", func(t *testing.T) {
		rd := &llx.RawData{Type: types.ArrayLike, Value: []any{"hello"}}
		require.NotPanics(t, func() { rd.Result() })
	})

	t.Run("bare ArrayLike empty", func(t *testing.T) {
		rd := &llx.RawData{Type: types.ArrayLike, Value: []any{}}
		require.NotPanics(t, func() { rd.Result() })
	})

	t.Run("bare MapLike with entries", func(t *testing.T) {
		rd := &llx.RawData{Type: types.MapLike, Value: map[string]any{"k": "v"}}
		require.NotPanics(t, func() { rd.Result() })
	})

	t.Run("Primitive round-trip with bare ArrayLike", func(t *testing.T) {
		p := &llx.Primitive{
			Type:  string(types.ArrayLike),
			Array: []*llx.Primitive{llx.StringPrimitive("hello")},
		}
		require.NotPanics(t, func() {
			raw := p.RawData()
			raw.Result()
		})
	})
}

func TestResultRawConversions(t *testing.T) {
	tests := []struct {
		raw *llx.RawData
	}{
		{raw: llx.VersionData("1.2.3")},
		{raw: llx.IPData(llx.ParseIP("192.168.0.1/27"))},
	}
	for i := range tests {
		cur := tests[i]
		t.Run(cur.raw.String(), func(t *testing.T) {
			require.NotContains(t, cur.raw.String(), llx.UNKNOWN_VALUE, fmt.Sprintf("implement String() for %#v", cur.raw))

			res := cur.raw.Result()
			require.NotNil(t, res)
			raw := res.RawData()
			require.NotNil(t, raw)
			assert.Equal(t, cur.raw.Type, raw.Type)
			assert.Equal(t, cur.raw.Value, raw.Value)
			res2 := raw.Result()
			require.NotNil(t, res2)
			assert.Equal(t, res, res2)
		})
	}
}

// TestEmptyTypePrimitiveConvertsToNil ensures an empty (untyped) primitive is
// converted to a Nil value instead of an error. An empty primitive represents
// an unset/null value; erroring here used to abort conversion of the whole
// surrounding array/map (see TestEmptyTypePrimitiveKeepsCollection).
func TestEmptyTypePrimitiveConvertsToNil(t *testing.T) {
	rd := (&llx.Primitive{}).RawData()
	require.NoError(t, rd.Error)
	assert.Equal(t, types.Nil, rd.Type)
	assert.Nil(t, rd.Value)
}

// TestEmptyTypePrimitiveKeepsCollection is a regression test for empty
// assessments: a single untyped nested field (e.g. an unset sub-field of a
// resource's @context block) must not discard the entire surrounding array.
// Previously RawData() errored on the untyped field and primitive2array /
// primitive2rawdataMapV2 propagated that error, returning an empty slice — so
// `list.all(...)` over @context resources rendered no failing resources at all.
func TestEmptyTypePrimitiveKeepsCollection(t *testing.T) {
	// A resource block whose nested context block carries an unset (untyped) field.
	contextBlock := &llx.Primitive{
		Type: string(types.Block),
		Map: map[string]*llx.Primitive{
			"path":  llx.StringPrimitive("main.tf"),
			"range": {}, // untyped/null sub-field — the trigger
		},
	}
	element := &llx.Primitive{
		Type: string(types.Block),
		Map: map[string]*llx.Primitive{
			"name":    llx.StringPrimitive("resource"),
			"context": contextBlock,
		},
	}
	arr := llx.ArrayPrimitive([]*llx.Primitive{element, element, element}, types.Block)

	rd := arr.RawData()
	require.NoError(t, rd.Error)
	got, ok := rd.Value.([]any)
	require.True(t, ok, "expected the array to survive conversion")
	assert.Len(t, got, 3, "the whole collection must be preserved, not emptied")
}
