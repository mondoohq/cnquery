// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package conanlock

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers/os/resources/languages/conan"
	"go.mondoo.com/mql/v13/sbom"
)

func TestConanLockV2(t *testing.T) {
	f, err := os.Open("./testdata/v2.conan.lock")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&Extractor{}).Parse(f, "path/to/conan.lock")
	require.NoError(t, err)

	// No root in v2
	assert.Nil(t, info.Root())

	// Direct = requires only
	direct := info.Direct()
	assert.Equal(t, 3, len(direct))

	p := direct.Find("zlib")
	require.NotNil(t, p)
	assert.Equal(t, "1.3.1", p.Version)
	assert.Equal(t, "pkg:conan/zlib@1.3.1", p.Purl)
	assert.Equal(t, []*sbom.Evidence{{Type: sbom.EvidenceType_EVIDENCE_TYPE_FILE, Value: "path/to/conan.lock"}}, p.EvidenceList)

	p = direct.Find("openssl")
	require.NotNil(t, p)
	assert.Equal(t, "3.2.1", p.Version)

	// Build requires should NOT be in direct
	assert.Nil(t, direct.Find("cmake"))

	// Transitive = requires + build_requires
	transitive := info.Transitive()
	assert.Equal(t, 5, len(transitive))
	assert.NotNil(t, transitive.Find("cmake"))
	assert.NotNil(t, transitive.Find("ninja"))
}

func TestConanLockV1(t *testing.T) {
	f, err := os.Open("./testdata/v1.conan.lock")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&Extractor{}).Parse(f, "conan.lock")
	require.NoError(t, err)

	// V1 has a root (node 0 with path)
	root := info.Root()
	require.NotNil(t, root)
	assert.Equal(t, "myproject", root.Name)
	assert.Equal(t, "1.0", root.Version)

	// V1 doesn't distinguish direct from transitive
	assert.Nil(t, info.Direct())

	// Transitive = all non-root nodes
	transitive := info.Transitive()
	assert.Equal(t, 3, len(transitive))

	p := transitive.Find("zlib")
	require.NotNil(t, p)
	assert.Equal(t, "1.3.1", p.Version)
	assert.Equal(t, "pkg:conan/zlib@1.3.1", p.Purl)

	// Root should not be in transitive
	assert.Nil(t, transitive.Find("myproject"))
}

func TestParseConanRef(t *testing.T) {
	tests := []struct {
		ref     string
		name    string
		version string
	}{
		{"zlib/1.3.1#abc123", "zlib", "1.3.1"},
		{"openssl/3.2.1", "openssl", "3.2.1"},
		{"boost/1.84.0#deadbeef", "boost", "1.84.0"},
		{"mylib", "mylib", ""},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			name, version := conan.ParseConanRef(tt.ref)
			assert.Equal(t, tt.name, name)
			assert.Equal(t, tt.version, version)
		})
	}
}
