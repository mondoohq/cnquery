// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package llx

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/types"
)

func TestArrayFlat(t *testing.T) {
	t.Run("empty array with missing type info", func(t *testing.T) {
		res, ref, err := arrayFlat(nil, &RawData{
			Type:  types.ArrayLike,
			Value: []any{},
		}, nil, 0)
		require.NoError(t, err)
		require.Equal(t, uint64(0), ref)
		require.Equal(t, ArrayData([]any{}, types.Any), res)
	})
}

// Ensure internal array helpers return empty slices (not nil) so that downstream
// operations like array concatenation do not hit "cannot add arrays to null".
func TestArrayHelpers_EmptyNotNil(t *testing.T) {
	t.Run("flatten of empty array returns empty slice", func(t *testing.T) {
		res := flatten([]any{})
		require.NotNil(t, res)
		require.Equal(t, []any{}, res)
	})

	t.Run("_arraySample with zero count returns empty slice", func(t *testing.T) {
		res := _arraySample([]any{1, 2, 3}, 0)
		require.NotNil(t, res)
		require.Equal(t, []any{}, res)
	})

	t.Run("_arraySample with empty array returns empty slice", func(t *testing.T) {
		res := _arraySample([]any{}, 5)
		require.NotNil(t, res)
		require.Equal(t, []any{}, res)
	})
}
