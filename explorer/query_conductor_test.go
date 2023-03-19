package explorer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchFilters(t *testing.T) {
	t.Run("one matching filter", func(t *testing.T) {
		filters := NewFilters("true", "false")
		filters.Compile("//owner")

		matching := []*Mquery{{Mql: "true"}}
		ChecksumFilters(matching)

		res, err := MatchFilters("assetMrn", matching, []*QueryPack{{ComputedFilters: filters}})
		require.NoError(t, err)
		assert.Equal(t, "6rm6AihK9P0=", res)
	})

	t.Run("no matching filter (matching is provided)", func(t *testing.T) {
		filters := NewFilters("true", "false")
		filters.Compile("//owner")

		matching := []*Mquery{{Mql: "0"}}
		ChecksumFilters(matching)

		_, err := MatchFilters("assetMrn", matching, []*QueryPack{{ComputedFilters: filters}})
		assert.EqualError(t, err,
			"rpc error: code = InvalidArgument desc = asset isn't supported by any querypacks\n"+
				"querypacks support: false, true\n"+
				"asset supports: 0\n")
	})

	t.Run("no matching filter (matching is empty)", func(t *testing.T) {
		filters := NewFilters("true", "false")
		filters.Compile("//owner")

		_, err := MatchFilters("assetMrn", []*Mquery{}, []*QueryPack{{ComputedFilters: filters}})
		assert.EqualError(t, err,
			"rpc error: code = InvalidArgument desc = asset doesn't support any querypacks")
	})
}
