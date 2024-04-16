// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package explorer_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/explorer"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/testutils"
)

func TestNewFilters(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		f := explorer.NewFilters()
		require.NotNil(t, f)
		require.Empty(t, f.Items)
	})

	t.Run("two filters", func(t *testing.T) {
		f := explorer.NewFilters("true", "false")
		require.NotNil(t, f)
		assert.Equal(t, map[string]*explorer.Mquery{
			"0": {Mql: "true"},
			"1": {Mql: "false"},
		}, f.Items)
	})
}

func TestSummarize(t *testing.T) {
	t.Run("with NewFilters initialization", func(t *testing.T) {
		f := explorer.NewFilters("true", "asset.platform != ''")
		assert.Equal(t, "asset.platform != '', true", f.Summarize())
	})

	t.Run("with mixed filters", func(t *testing.T) {
		f := &explorer.Filters{
			Items: map[string]*explorer.Mquery{
				"one": {Mql: "asset.name"},
				"two": {Title: "filter 2"},
			},
		}

		assert.Equal(t, "asset.name, filter 2", f.Summarize())
	})
}

func TestBundleAssetFilter(t *testing.T) {
	// load the raw bundle
	tester := testutils.InitTester(testutils.LinuxMock())
	bundle, err := explorer.BundleFromPaths("../examples/os.mql.yaml")
	require.NoError(t, err)
	assert.Equal(t, 1, len(bundle.Packs))
	assert.Equal(t, "asset.family.contains(\"unix\")", bundle.Packs[0].Filters.Items["0"].Mql)
	assert.Equal(t, (*explorer.Filters)(nil), bundle.Packs[0].ComputedFilters)

	// check that the computed asset filters are set
	pbm, err := bundle.Compile(context.Background(), tester.Runtime.Schema())
	require.NoError(t, err)
	assert.Equal(t, "asset.family.contains(\"unix\")", pbm.Packs["//local.cnquery.io/run/local-execution/querypacks/linux-mixed-queries"].ComputedFilters.Summarize())
}
