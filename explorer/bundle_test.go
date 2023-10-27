// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package explorer

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/testutils"
)

var mock = testutils.LinuxMock()

func TestBundleLoad(t *testing.T) {
	t.Run("load bundle from file", func(t *testing.T) {
		bundle, err := BundleFromPaths("../examples/os.mql.yaml")
		require.NoError(t, err)
		assert.Equal(t, 1, len(bundle.Packs))
		assert.Equal(t, 3, len(bundle.Packs[0].Queries))

		// ensure that the uid is generated
		assert.True(t, len(bundle.Packs[0].Queries[0].Uid) > 0)
	})

	t.Run("compile complex bundle", func(t *testing.T) {
		bundle, err := BundleFromPaths("../examples/complex.mql.yaml")
		require.NoError(t, err)
		assert.Equal(t, 1, len(bundle.Packs))
		assert.Equal(t, 4, len(bundle.Queries))

		mock := testutils.LinuxMock()
		m, err := bundle.Compile(context.Background(), mock.Schema())
		require.NoError(t, err)
		require.NotNil(t, m)
		assert.Len(t, m.Queries, 6)
	})

	t.Run("load bundle from memory", func(t *testing.T) {
		data, err := os.ReadFile("../examples/os.mql.yaml")
		require.NoError(t, err)
		bundle, err := BundleFromYAML(data)
		require.NoError(t, err)
		assert.Equal(t, 1, len(bundle.Packs))
		assert.Equal(t, 3, len(bundle.Packs[0].Queries))

		// ensure that the uid is generated
		assert.True(t, len(bundle.Packs[0].Queries[0].Uid) > 0)
	})
}

func TestFilterQueriesWontCompile(t *testing.T) {
	b2, err := BundleFromYAML([]byte(failingVariant))
	require.NoError(t, err)
	_, err2 := b2.CompileExt(context.Background(), BundleCompileConf{
		Schema:        mock.Schema(),
		RemoveFailing: false,
	})
	require.Error(t, err2)
}

func TestFilterQueriesIgnoreError(t *testing.T) {
	b, err := BundleFromYAML([]byte(failingVariant))
	require.NoError(t, err)
	bmap, err := b.CompileExt(context.Background(), BundleCompileConf{
		Schema:        mock.Schema(),
		RemoveFailing: true,
	})
	require.NoError(t, err)
	require.NotNil(t, bmap)
	assert.Len(t, bmap.Queries, 4)
}

const failingVariant = `
packs:
- uid: mondoo-soc2-inventory
  queries:
  - uid: failing-pack_embed
    mql: not_me_i_wont
  - uid: pack-variant_embed
    variants:
    - uid: variant-ok
    - uid: variant-nok
  - uid: failing-pack-ref
  groups:
  - title: Main
    filters: "true"
    queries:
    - uid: variant-root
    - uid: failing-group-embed
      mql: embed_wont_work_for_this
    - uid: group-variant_embed
      variants:
      - uid: variant-ok
      - uid: variant-nok
    - uid: failing-group-ref
queries:
- uid: failing-pack-ref
  mql: i_wont_compile
- uid: failing-group-ref
  mql: i_also_wont_compile
- uid: variant-root
  variants:
  - uid: variant-ok
  - uid: variant-nok
- uid: variant-ok
  mql: asset.name
- uid: variant-nok
  mql: definitely_not_in_here
`
