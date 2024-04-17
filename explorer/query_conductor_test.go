// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package explorer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11"
	"go.mondoo.com/cnquery/v11/mqlc"
	"go.mondoo.com/cnquery/v11/providers"
)

func TestLocalServicesInterface(t *testing.T) {
	localServices := &LocalServices{}
	// LocalServices should be a QueryConductor
	var conductor QueryConductor = localServices
	assert.NotNil(t, conductor)
}

func TestMatchFilters(t *testing.T) {
	schema := providers.DefaultRuntime().Schema()
	conf := mqlc.NewConfig(schema, cnquery.DefaultFeatures)

	t.Run("one matching filter", func(t *testing.T) {
		filters := NewFilters("true", "false")
		err := filters.Compile("//owner", conf)
		require.NoError(t, err)

		matching := []*Mquery{{Mql: "true"}}
		_, err = ChecksumFilters(matching, conf)
		require.NoError(t, err)

		res, err := MatchFilters("assetMrn", matching, []*QueryPack{{ComputedFilters: filters}}, schema)
		require.NoError(t, err)
		assert.Equal(t, "6rm6AihK9P0=", res)
	})

	t.Run("no matching filter (matching is provided)", func(t *testing.T) {
		filters := NewFilters("true", "false")
		err := filters.Compile("//owner", conf)
		require.NoError(t, err)

		matching := []*Mquery{{Mql: "0"}}
		_, err = ChecksumFilters(matching, conf)
		require.NoError(t, err)

		_, err = MatchFilters("assetMrn", matching, []*QueryPack{{ComputedFilters: filters}}, schema)
		assert.EqualError(t, err,
			"rpc error: code = InvalidArgument desc = asset isn't supported by any querypacks\n"+
				"querypacks support: false, true\n"+
				"asset supports: 0\n")
	})

	t.Run("no matching filter (matching is empty)", func(t *testing.T) {
		filters := NewFilters("true", "false")
		err := filters.Compile("//owner", conf)
		require.NoError(t, err)

		_, err = MatchFilters("assetMrn", []*Mquery{}, []*QueryPack{{ComputedFilters: filters}}, schema)
		assert.EqualError(t, err,
			"rpc error: code = InvalidArgument desc = asset doesn't support any querypacks")
	})
}
