// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package packagelockjson

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers/os/resources/languages"
	"go.mondoo.com/mql/v13/sbom"
)

func TestPackageJsonLockExtractorWithPackages(t *testing.T) {
	f, err := os.Open("./testdata/lockfile-v2.json")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&Extractor{}).Parse(f, "path/to/package-lock.json")
	assert.Nil(t, err)

	root := info.Root()
	assert.Equal(t, &languages.Package{
		Name:         "npm",
		Version:      "7.0.0",
		Purl:         "pkg:npm/npm@7.0.0",
		Cpes:         []string{"cpe:2.3:a:npm:npm:7.0.0:*:*:*:*:*:*:*"},
		EvidenceList: []*sbom.Evidence{{Type: sbom.EvidenceType_EVIDENCE_TYPE_FILE, Value: "path/to/package-lock.json"}},
	}, root)

	list := info.Transitive()
	assert.Equal(t, 2, len(list))

	p := list.Find("@babel/code-frame")
	assert.Equal(t, &languages.Package{
		Name:         "@babel/code-frame",
		Version:      "7.10.4",
		Purl:         "pkg:npm/node-modules/%40babel@7.10.4",
		Cpes:         []string{"cpe:2.3:a:node_modules\\/\\@babel\\/code-frame:node_modules\\/\\@babel\\/code-frame:7.10.4:*:*:*:*:*:*:*"},
		EvidenceList: []*sbom.Evidence{{Type: sbom.EvidenceType_EVIDENCE_TYPE_FILE, Value: "path/to/package-lock.json"}},
	}, p)

}

// TestPackageJsonLockDependencyEdges verifies the package→package edges
// (Package.DependsOn) resolved from a lockfileVersion 2+ tree: each edge ref is
// the resolved target package's Purl, so the graph is self-consistent
// (edge ref == target node Purl). app→foo→bar.
func TestPackageJsonLockDependencyEdges(t *testing.T) {
	f, err := os.Open("./testdata/graph-lock.json")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&Extractor{}).Parse(f, "path/to/package-lock.json")
	require.NoError(t, err)

	list := info.Transitive()
	app := list.Find("app")
	foo := list.Find("foo")
	bar := list.Find("bar")
	require.NotNil(t, app)
	require.NotNil(t, foo)
	require.NotNil(t, bar)

	// edges point at the resolved target's Purl (ref == target node Purl).
	assert.Equal(t, []string{foo.Purl}, app.DependsOn, "app depends on foo")
	assert.Equal(t, []string{bar.Purl}, foo.DependsOn, "foo depends on bar")
	assert.Nil(t, bar.DependsOn, "bar has no dependencies")
}

// TestPackageJsonLockDirect verifies Direct() resolves the root's declared
// dependencies via their node_modules/<name> path (lockfileVersion 2+), matching
// the same package's Transitive() representation. Previously it looked deps up by
// bare name against the path-keyed `packages` map and returned nothing.
func TestPackageJsonLockDirect(t *testing.T) {
	f, err := os.Open("./testdata/graph-lock.json")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&Extractor{}).Parse(f, "path/to/package-lock.json")
	require.NoError(t, err)

	direct := info.Direct()
	require.Len(t, direct, 1, "root declares exactly one direct dependency (foo)")
	foo := direct.Find("foo")
	require.NotNil(t, foo)
	assert.Equal(t, "1.2.3", foo.Version)
	// Direct's foo matches Transitive's foo exactly (same purl + edges).
	assert.Equal(t, info.Transitive().Find("foo").Purl, foo.Purl)
	assert.Equal(t, info.Transitive().Find("foo").DependsOn, foo.DependsOn)
}

func TestPackageJsonLockExtractorWithDependencies(t *testing.T) {
	f, err := os.Open("./testdata/workbox-package-lock.json")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&Extractor{}).Parse(f, "path/to/package-lock.json")
	assert.Nil(t, err)

	root := info.Root()
	assert.Equal(t, &languages.Package{
		Name:         "workbox",
		Version:      "0.0.0",
		Purl:         "pkg:npm/workbox@0.0.0",
		Cpes:         []string{"cpe:2.3:a:workbox:workbox:0.0.0:*:*:*:*:*:*:*"},
		EvidenceList: []*sbom.Evidence{{Type: sbom.EvidenceType_EVIDENCE_TYPE_FILE, Value: "path/to/package-lock.json"}},
	}, root)

	list := info.Transitive()
	assert.Equal(t, 1299, len(list))

	p := list.Find("@babel/generator")
	assert.Equal(t, &languages.Package{
		Name:         "@babel/generator",
		Version:      "7.0.0",
		Purl:         "pkg:npm/%40babel/generator@7.0.0",
		Cpes:         []string{"cpe:2.3:a:\\@babel\\/generator:\\@babel\\/generator:7.0.0:*:*:*:*:*:*:*"},
		EvidenceList: []*sbom.Evidence{{Type: sbom.EvidenceType_EVIDENCE_TYPE_FILE, Value: "path/to/package-lock.json"}},
	}, p)

	p = list.Find("@lerna/changed")
	assert.Equal(t, &languages.Package{
		Name:         "@lerna/changed",
		Version:      "3.3.2",
		Purl:         "pkg:npm/%40lerna/changed@3.3.2",
		Cpes:         []string{"cpe:2.3:a:\\@lerna\\/changed:\\@lerna\\/changed:3.3.2:*:*:*:*:*:*:*"},
		EvidenceList: []*sbom.Evidence{{Type: sbom.EvidenceType_EVIDENCE_TYPE_FILE, Value: "path/to/package-lock.json"}},
	}, p)
}
