// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package vcpkg

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	f, err := os.Open("testdata/vcpkg.json")
	require.NoError(t, err)
	defer f.Close()

	e := &Extractor{}
	bom, err := e.Parse(f, "testdata/vcpkg.json")
	require.NoError(t, err)

	// The manifest's own port is the root project, not a dependency.
	root := bom.Root()
	require.NotNil(t, root)
	assert.Equal(t, "my-app", root.Name)
	assert.Equal(t, "1.2.0", root.Version)
	assert.Equal(t, "pkg:vcpkg/my-app@1.2.0", root.Purl)

	assert.Nil(t, bom.Transitive())

	deps := bom.Direct()
	// fmt (deduped from its two occurrences), zlib, boost-system.
	assert.Len(t, deps, 3)

	fmtPkg := deps.Find("fmt")
	require.NotNil(t, fmtPkg)
	// No override → no manifest-stated version.
	assert.Equal(t, "", fmtPkg.Version)
	assert.Equal(t, "pkg:vcpkg/fmt", fmtPkg.Purl)

	zlib := deps.Find("zlib")
	require.NotNil(t, zlib)
	// Version pinned by an override.
	assert.Equal(t, "1.3.1", zlib.Version)
	assert.Equal(t, "pkg:vcpkg/zlib@1.3.1", zlib.Purl)

	// Object-form dependency resolves to its name.
	boost := deps.Find("boost-system")
	require.NotNil(t, boost)
	assert.Equal(t, "pkg:vcpkg/boost-system", boost.Purl)
}

func TestName(t *testing.T) {
	e := &Extractor{}
	assert.Equal(t, "vcpkg", e.Name())
}
