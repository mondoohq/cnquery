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

// Regression: a RawData that reaches Result() with a malformed type (e.g. a
// bare types.ArrayLike with no element type) must not panic. It should surface
// as a serialization error on the Result, letting the scan continue.
func TestRawDataResult_MalformedArrayType(t *testing.T) {
	t.Run("bare ArrayLike with values", func(t *testing.T) {
		raw := &llx.RawData{
			Type:  types.ArrayLike,
			Value: []any{"a", "b"},
		}
		require.NotPanics(t, func() {
			res := raw.Result()
			require.NotNil(t, res)
			require.NotEmpty(t, res.Error)
		})
	})

	t.Run("empty type with non-nil value", func(t *testing.T) {
		raw := &llx.RawData{
			Type:  types.Type(""),
			Value: "hello",
		}
		require.NotPanics(t, func() {
			res := raw.Result()
			require.NotNil(t, res)
			require.NotEmpty(t, res.Error)
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
