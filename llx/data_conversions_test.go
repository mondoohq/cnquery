// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package llx_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/types"
)

func TestVersion_Conversions(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		sv := llx.StringPrimitive("1.2.3")
		sv.Type = string(types.Version)
		rd := sv.RawData()
		require.NoError(t, rd.Error, "no error converting version to raw data")
		require.Equal(t, "1.2.3", rd.Value, "version to raw data is the same")
	})

	t.Run("raw and result conversions", func(t *testing.T) {
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
	})
}
