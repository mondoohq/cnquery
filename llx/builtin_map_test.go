// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package llx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/types"
)

func newTestBlockExecutor() *blockExecutor {
	return &blockExecutor{
		ctx: &MQLExecutorV2{code: &CodeV2{}},
	}
}

func newStringKeyChunk() *Chunk {
	return &Chunk{
		Function: &Function{
			Args: []*Primitive{{Value: []byte("key"), Type: string(types.String)}},
		},
	}
}

func runIndexHandler(t *testing.T, bind *RawData, operator string) (*RawData, uint64, error) {
	t.Helper()

	handler, err := BuiltinFunctionV2(bind.Type, operator)
	require.NoError(t, err)

	return handler.f(newTestBlockExecutor(), bind, newStringKeyChunk(), 0)
}

func TestDictGetIndex_NilValue(t *testing.T) {
	for _, operator := range []string{"[]", "[]?"} {
		t.Run(operator+" returns typed null when parent dict is nil", func(t *testing.T) {
			bind := &RawData{Type: types.Dict, Value: nil}

			res, ref, err := runIndexHandler(t, bind, operator)
			require.NoError(t, err)
			assert.Equal(t, uint64(0), ref)
			assert.Equal(t, types.Dict, res.Type)
			assert.Nil(t, res.Value, "null dict access should propagate null, not error")
		})
	}
}

func newArrayArgChunk(elems []*Primitive, elemType types.Type) *Chunk {
	return &Chunk{
		Function: &Function{
			Args: []*Primitive{ArrayPrimitive(elems, elemType)},
		},
	}
}

func runDictInHandler(t *testing.T, bind *RawData, chunk *Chunk, operator string) (*RawData, uint64, error) {
	t.Helper()

	handler, err := BuiltinFunctionV2(bind.Type, operator)
	require.NoError(t, err)

	return handler.f(newTestBlockExecutor(), bind, chunk, 0)
}

func TestDictIn_ScalarValues(t *testing.T) {
	intArr := newArrayArgChunk([]*Primitive{IntPrimitive(1), IntPrimitive(2)}, types.Int)
	strArr := newArrayArgChunk([]*Primitive{StringPrimitive("1"), StringPrimitive("2")}, types.String)

	cases := []struct {
		name  string
		bind  any
		chunk *Chunk
		op    string
		want  *RawData
	}{
		// CIS-style: DWORD (int64) against numeric array — used to error
		{"int64 bind in int array match", int64(2), intArr, "in", BoolTrue},
		{"int64 bind in int array miss", int64(3), intArr, "in", BoolFalse},
		{"int64 bind notIn int array match", int64(2), intArr, "notIn", BoolFalse},
		{"int64 bind notIn int array miss", int64(3), intArr, "notIn", BoolTrue},

		// Cross-type: DWORD against string array (matches the literal CIS check shape)
		{"int64 bind in string array match", int64(2), strArr, "in", BoolTrue},
		{"int64 bind in string array miss", int64(3), strArr, "in", BoolFalse},

		// Existing string-bind path still works through the unified helper
		{"string bind in string array match", "1", strArr, "in", BoolTrue},
		{"string bind in string array miss", "9", strArr, "in", BoolFalse},

		// Bool bind
		{"bool bind in bool array",
			true,
			newArrayArgChunk([]*Primitive{BoolPrimitive(false), BoolPrimitive(true)}, types.Bool),
			"in", BoolTrue},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bind := &RawData{Type: types.Dict, Value: tc.bind}
			res, ref, err := runDictInHandler(t, bind, tc.chunk, tc.op)
			require.NoError(t, err)
			assert.Equal(t, uint64(0), ref)
			assert.Equal(t, tc.want, res)
		})
	}
}

func TestDictIn_NilBindAndArg(t *testing.T) {
	intArr := newArrayArgChunk([]*Primitive{IntPrimitive(1)}, types.Int)

	t.Run("nil dict bind", func(t *testing.T) {
		bind := &RawData{Type: types.Dict, Value: nil}
		res, _, err := runDictInHandler(t, bind, intArr, "in")
		require.NoError(t, err)
		assert.Equal(t, BoolFalse, res)
	})
}

func TestMapGetIndex_NilValue(t *testing.T) {
	for _, operator := range []string{"[]", "[]?"} {
		t.Run(operator+" returns typed null when parent map is nil", func(t *testing.T) {
			mapType := types.Map(types.String, types.String)
			bind := &RawData{Type: mapType, Value: nil}

			res, ref, err := runIndexHandler(t, bind, operator)
			require.NoError(t, err)
			assert.Equal(t, uint64(0), ref)
			assert.Equal(t, types.String, res.Type)
			assert.Nil(t, res.Value, "null map access should propagate null, not error")
		})
	}
}
