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
