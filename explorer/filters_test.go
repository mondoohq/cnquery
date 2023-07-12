package explorer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/providers"
)

func TestNewFilters(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		f := NewFilters()
		require.NotNil(t, f)
		require.Empty(t, f.Items)
	})

	t.Run("two filters", func(t *testing.T) {
		f := NewFilters("true", "false")
		require.NotNil(t, f)
		assert.Equal(t, map[string]*Mquery{
			"0": {Mql: "true"},
			"1": {Mql: "false"},
		}, f.Items)
	})
}

func TestSummarize(t *testing.T) {
	t.Run("with NewFilters initialization", func(t *testing.T) {
		f := NewFilters("true", "asset.platform != ''")
		assert.Equal(t, "asset.platform != '', true", f.Summarize())
	})

	t.Run("with mixed filters", func(t *testing.T) {
		f := &Filters{
			Items: map[string]*Mquery{
				"one": {Mql: "asset.name"},
				"two": {Title: "filter 2"},
			},
		}

		assert.Equal(t, "asset.name, filter 2", f.Summarize())
	})
}

func TestBundleAssetFilter(t *testing.T) {
	// load the raw bundle
	bundle, err := BundleFromPaths("../examples/os.mql.yaml")
	require.NoError(t, err)
	assert.Equal(t, 1, len(bundle.Packs))
	assert.Equal(t, "asset.family.contains(\"unix\")", bundle.Packs[0].Filters.Items["0"].Mql)
	assert.Equal(t, (*Filters)(nil), bundle.Packs[0].ComputedFilters)

	// check that the computed asset filters are set
	pbm, err := bundle.Compile(context.Background(), providers.DefaultRuntime().Schema())
	require.NoError(t, err)
	assert.Equal(t, "asset.family.contains(\"unix\")", pbm.Packs["//local.cnquery.io/run/local-execution/querypacks/linux-mixed-queries"].ComputedFilters.Summarize())
}
