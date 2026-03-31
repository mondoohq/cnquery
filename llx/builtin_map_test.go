// Copyright (c) Mondoo, Inc.
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
