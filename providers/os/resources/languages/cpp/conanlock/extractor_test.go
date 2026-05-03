// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package conanlock

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseV1(t *testing.T) {
	f, err := os.Open("testdata/v1.json")
	require.NoError(t, err)
	defer f.Close()

	e := &Extractor{}
	bom, err := e.Parse(f, "testdata/v1.json")
	require.NoError(t, err)

	assert.Nil(t, bom.Root())
	assert.Nil(t, bom.Direct())

	pkgs := bom.Transitive()
	assert.Len(t, pkgs, 3) // myproject skipped (has path)

	boost := pkgs.Find("boost")
	require.NotNil(t, boost)
	assert.Equal(t, "1.84.0", boost.Version)
	assert.Equal(t, "pkg:conan/boost@1.84.0", boost.Purl)

	zlib := pkgs.Find("zlib")
	require.NotNil(t, zlib)
	assert.Equal(t, "1.3.1", zlib.Version)

	fmtPkg := pkgs.Find("fmt")
	require.NotNil(t, fmtPkg)
	assert.Equal(t, "10.2.1", fmtPkg.Version)
}

func TestParseV2(t *testing.T) {
	f, err := os.Open("testdata/v2.json")
	require.NoError(t, err)
	defer f.Close()

	e := &Extractor{}
	bom, err := e.Parse(f, "testdata/v2.json")
	require.NoError(t, err)

	pkgs := bom.Transitive()
	assert.Len(t, pkgs, 4) // 2 requires + 1 build_requires + 1 python_requires

	boost := pkgs.Find("boost")
	require.NotNil(t, boost)
	assert.Equal(t, "1.84.0", boost.Version)
	assert.Equal(t, "pkg:conan/boost@1.84.0", boost.Purl)

	cmake := pkgs.Find("cmake")
	require.NotNil(t, cmake)
	assert.Equal(t, "3.28.1", cmake.Version)

	conanTools := pkgs.Find("conan-tools")
	require.NotNil(t, conanTools)
	assert.Equal(t, "1.0.0", conanTools.Version)
}

func TestParseConanReference(t *testing.T) {
	tests := []struct {
		ref     string
		name    string
		version string
		ok      bool
	}{
		{"boost/1.84.0", "boost", "1.84.0", true},
		{"boost/1.84.0#abc123", "boost", "1.84.0", true},
		{"boost/1.84.0@user/channel", "boost", "1.84.0", true},
		{"boost/1.84.0@user/channel#rev", "boost", "1.84.0", true},
		{"", "", "", false},
		{"noversion", "", "", false},
	}

	for _, tt := range tests {
		ref, ok := parseConanReference(tt.ref)
		assert.Equal(t, tt.ok, ok, "ref: %s", tt.ref)
		if ok {
			assert.Equal(t, tt.name, ref.Name, "ref: %s", tt.ref)
			assert.Equal(t, tt.version, ref.Version, "ref: %s", tt.ref)
		}
	}
}

func TestName(t *testing.T) {
	e := &Extractor{}
	assert.Equal(t, "conanlock", e.Name())
}
