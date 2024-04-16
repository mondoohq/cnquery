// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package llx

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/types"
)

func TestArrayFlat(t *testing.T) {
	t.Run("empty array with missing type info", func(t *testing.T) {
		res, ref, err := arrayFlat(nil, &RawData{
			Type:  types.ArrayLike,
			Value: []any{},
		}, nil, 0)
		require.NoError(t, err)
		require.Equal(t, uint64(0), ref)
		require.Equal(t, ArrayData(nil, types.Any), res)
	})
}
