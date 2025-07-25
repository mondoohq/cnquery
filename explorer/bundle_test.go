// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package explorer

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11"
	"go.mondoo.com/cnquery/v11/mqlc"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/testutils"
)

var (
	mock = testutils.LinuxMock()
	conf = mqlc.NewConfig(mock.Schema(), cnquery.DefaultFeatures)
)

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

func testBundleCompiles(t *testing.T, raw string) error {
	b, err := BundleFromYAML([]byte(raw))
	require.NoError(t, err)
	_, err2 := b.CompileExt(context.Background(), BundleCompileConf{
		CompilerConfig: conf,
		RemoveFailing:  false,
	})
	return err2
}

func TestPackWithoutQueries(t *testing.T) {
	err := testBundleCompiles(t, packWithoutQueries)
	assert.NoError(t, err)
}

func TestMissingUidMrnWontCompile(t *testing.T) {
	t.Run("missing pack UID/MRN", func(t *testing.T) {
		err := testBundleCompiles(t, missingPackUidMrn)
		assert.Equal(t, "failed to refresh mrn for querypack hello world: cannot refresh MRN with an empty UID", err.Error())
	})

	t.Run("missing query UID/MRN", func(t *testing.T) {
		err := testBundleCompiles(t, missingQueryUidMrn)
		assert.Equal(t, "failed to refresh mrn for query query-title: cannot refresh MRN with an empty UID", err.Error())
	})
}

const missingPackUidMrn = `
packs:
- name: hello world
`

const missingQueryUidMrn = `
packs:
- name: hello world
  uid: test-pack
  queries:
  - mql: return true
    title: query-title
`

const packWithoutQueries = `
packs:
- name: hello world
  uid: test-pack
`

func TestFilterQueriesWontCompile(t *testing.T) {
	err := testBundleCompiles(t, failingVariant)
	assert.Error(t, err)
}

func TestFilterQueriesIgnoreError(t *testing.T) {
	b, err := BundleFromYAML([]byte(failingVariant))
	require.NoError(t, err)
	bmap, err := b.CompileExt(context.Background(), BundleCompileConf{
		CompilerConfig: conf,
		RemoveFailing:  true,
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

func TestProps_QueryPropsLifted(t *testing.T) {
	// In this test, we expect that 3 properties are lifted to the pack:
	// home, homeDir, and user.
	// These must reference the queries prop through the for field
	bundleYaml := `
packs:
  - uid: example1
    name: Example policy 1
    groups:
      - title: group1
        filters: return true
        queries:
          - uid: variant-1
          - uid: variant-2
          - uid: variant-3
          - uid: variant-4
queries:
  - uid: variant-check
    title: Variant check
    variants:
      - uid: variant-1
      - uid: variant-2
      - uid: variant-3

  - uid: variant-1
    mql: props.home + " on 1"
    props:
      - uid: home
        mql: return "p1"

  - uid: variant-2
    mql: props.home + " on 2"
    props:
      - uid: home
        mql: return "p2"

  - uid: variant-3
    mql: props.homeDir + " on 3"
    props:
      - uid: homeDir
        mql: return "p3"
  
  - uid: variant-4
    mql: props.user + " is the user"
    props:
      - uid: user
        mql: return "ada"`

	b, err := BundleFromYAML([]byte(bundleYaml))
	require.NoError(t, err)
	_, err = b.CompileExt(context.Background(), BundleCompileConf{
		CompilerConfig: conf,
		RemoveFailing:  true,
	})
	require.NoError(t, err)

	require.Len(t, b.Packs[0].Props, 3)
	require.Len(t, b.Packs[0].Props[0].For, 2)
	require.NotEmpty(t, b.Packs[0].Props[0].For[0].Mrn)
	require.NotEmpty(t, b.Packs[0].Props[0].For[1].Mrn)
	require.Equal(t, b.Queries[1].Props[0].Mrn, b.Packs[0].Props[0].For[0].Mrn)
	require.Equal(t, b.Queries[2].Props[0].Mrn, b.Packs[0].Props[0].For[1].Mrn)
	require.Len(t, b.Packs[0].Props[1].For, 1)
	require.Equal(t, b.Queries[3].Props[0].Mrn, b.Packs[0].Props[1].For[0].Mrn)
	require.Len(t, b.Packs[0].Props[2].For, 1)
	require.Equal(t, b.Queries[4].Props[0].Mrn, b.Packs[0].Props[2].For[0].Mrn)
}

func TestProps_QueryPropMrnsResolved(t *testing.T) {
	// In this test, we expect that the property mrns are resolved correctly
	// and that the for field is set to the correct query mrn.
	bundleYaml := `
packs:
  - uid: example1
    name: Example policy 1
    version: "1.0.0"
    authors:
      - name: Mondoo
        email: hello@mondoo.com
    groups:
      - title: group1
        filters: return true
        queries:
          - uid: variant-1
          - uid: variant-2
          - uid: variant-3
          - uid: variant-4
    props:
      - uid: userHome
        for:
          - uid: home
          - uid: homeDir
        mql: return "ex"

queries:
  - uid: variant-check
    title: Variant check
    variants:
      - uid: variant-1
      - uid: variant-2
      - uid: variant-3

  - uid: variant-1
    mql: props.home + " on 1"
    props:
      - uid: home
        mql: return "p1"

  - uid: variant-2
    mql: props.home + " on 2"
    props:
      - uid: home
        mql: return "p2"

  - uid: variant-3
    mql: props.homeDir + " on 3"
    props:
      - uid: homeDir
        mql: return "p3"
  
  - uid: variant-4
    mql: props.user + " is the user"
    props:
      - uid: user
        mql: return "ada"`

	b, err := BundleFromYAML([]byte(bundleYaml))
	require.NoError(t, err)
	_, err = b.CompileExt(context.Background(), BundleCompileConf{
		CompilerConfig: conf,
		RemoveFailing:  true,
	})
	require.NoError(t, err)

	require.Len(t, b.Packs[0].Props, 1)
	require.Len(t, b.Packs[0].Props[0].For, 3)
	require.NotEmpty(t, b.Packs[0].Props[0].For[0].Mrn)
	require.NotEmpty(t, b.Packs[0].Props[0].For[1].Mrn)
	require.NotEmpty(t, b.Packs[0].Props[0].For[2].Mrn)
	require.Equal(t, b.Queries[1].Props[0].Mrn, b.Packs[0].Props[0].For[0].Mrn)
	require.Equal(t, b.Queries[2].Props[0].Mrn, b.Packs[0].Props[0].For[1].Mrn)
	require.Equal(t, b.Queries[3].Props[0].Mrn, b.Packs[0].Props[0].For[2].Mrn)
}
