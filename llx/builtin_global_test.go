// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package llx

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/types"
)

func TestAvailableCallV2(t *testing.T) {
	t.Run("returns true when the argument resolves cleanly", func(t *testing.T) {
		ref := uint64(1)
		exec := newTestBlockExecutor()
		exec.cache = newCache()
		exec.cache.Store(ref, &stepCache{Result: StringData("ok")})

		res, dref, err := availableCallV2(exec, &Function{Args: []*Primitive{RefPrimitiveV2(ref)}}, 0)
		require.NoError(t, err)
		assert.Equal(t, uint64(0), dref)
		assert.Equal(t, BoolTrue, res)
	})

	t.Run("returns true when the argument resolves to null without error", func(t *testing.T) {
		ref := uint64(1)
		exec := newTestBlockExecutor()
		exec.cache = newCache()
		exec.cache.Store(ref, &stepCache{Result: &RawData{Type: types.String}})

		res, dref, err := availableCallV2(exec, &Function{Args: []*Primitive{RefPrimitiveV2(ref)}}, 0)
		require.NoError(t, err)
		assert.Equal(t, uint64(0), dref)
		assert.Equal(t, BoolTrue, res)
	})

	t.Run("returns false when the argument resolves with an error", func(t *testing.T) {
		ref := uint64(1)
		exec := newTestBlockExecutor()
		exec.cache = newCache()
		exec.cache.Store(ref, &stepCache{Result: &RawData{Type: types.String, Error: errors.New("boom")}})

		res, dref, err := availableCallV2(exec, &Function{Args: []*Primitive{RefPrimitiveV2(ref)}}, 0)
		require.NoError(t, err)
		assert.Equal(t, uint64(0), dref)
		assert.Equal(t, BoolFalse, res)
	})
}
