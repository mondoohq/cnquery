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

func TestDictGetIndex_NilValue(t *testing.T) {
	t.Run("returns typed null when parent dict is nil", func(t *testing.T) {
		e := newTestBlockExecutor()
		bind := &RawData{Type: types.Dict, Value: nil}

		res, ref, err := dictGetIndex(e, bind, newStringKeyChunk(), 0)
		require.NoError(t, err)
		assert.Equal(t, uint64(0), ref)
		assert.Equal(t, types.Dict, res.Type)
		assert.Nil(t, res.Value, "null dict access should propagate null, not error")
	})

	t.Run("conditional index also returns typed null", func(t *testing.T) {
		e := newTestBlockExecutor()
		bind := &RawData{Type: types.Dict, Value: nil}

		res, ref, err := dictGetConditionalIndex(e, bind, newStringKeyChunk(), 0)
		require.NoError(t, err)
		assert.Equal(t, uint64(0), ref)
		assert.Equal(t, types.Dict, res.Type)
		assert.Nil(t, res.Value, "conditional null dict access should propagate null")
	})
}

func TestMapGetIndex_NilValue(t *testing.T) {
	t.Run("returns typed null when parent map is nil", func(t *testing.T) {
		e := newTestBlockExecutor()
		mapType := types.Map(types.String, types.String)
		bind := &RawData{Type: mapType, Value: nil}

		res, ref, err := mapGetIndex(e, bind, newStringKeyChunk(), 0)
		require.NoError(t, err)
		assert.Equal(t, uint64(0), ref)
		assert.Nil(t, res.Value, "null map access should propagate null, not error")
	})

	t.Run("conditional index also returns typed null", func(t *testing.T) {
		e := newTestBlockExecutor()
		mapType := types.Map(types.String, types.String)
		bind := &RawData{Type: mapType, Value: nil}

		res, ref, err := mapGetConditionalIndex(e, bind, newStringKeyChunk(), 0)
		require.NoError(t, err)
		assert.Equal(t, uint64(0), ref)
		assert.Nil(t, res.Value, "conditional null map access should propagate null")
	})
}
