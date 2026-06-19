// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package llx

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/types"
)

// A null array receiver (e.g. a missing map key resolving to a typed null
// array, like secpol.privilegerights["SeMissing"]) must not error when the
// all/any/none/one assertion builtins are called on it. It propagates as a
// null bool so the check fails cleanly instead of crashing the scan.
func TestArrayAssertions_NullReceiver(t *testing.T) {
	cases := []struct {
		name string
		fn   func(*blockExecutor, *RawData, *Chunk, uint64) (*RawData, uint64, error)
	}{
		{"all", arrayAllV2},
		{"any", arrayAnyV2},
		{"none", arrayNoneV2},
		{"one", arrayOneV2},
	}
	for _, c := range cases {
		t.Run(c.name+" on typed null array returns null bool, no error", func(t *testing.T) {
			res, ref, err := c.fn(nil, &RawData{Type: types.Array(types.String), Value: nil}, nil, 0)
			require.NoError(t, err)
			require.Equal(t, uint64(0), ref)
			require.NotNil(t, res)
			require.Equal(t, types.Bool, res.Type)
			require.Nil(t, res.Value)
			require.NoError(t, res.Error)
		})

		t.Run(c.name+" preserves a genuine upstream error", func(t *testing.T) {
			boom := errors.New("upstream boom")
			res, _, err := c.fn(nil, &RawData{Type: types.Array(types.String), Value: nil, Error: boom}, nil, 0)
			require.NoError(t, err)
			require.NotNil(t, res)
			require.Equal(t, boom, res.Error)
		})
	}
}

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
