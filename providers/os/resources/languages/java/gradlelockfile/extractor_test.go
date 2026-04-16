// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package gradlelockfile

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/sbom"
)

func TestGradleLockfileExtractor(t *testing.T) {
	f, err := os.Open("./testdata/simple.gradle.lockfile")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&Extractor{}).Parse(f, "path/to/gradle.lockfile")
	require.NoError(t, err)

	// No root
	assert.Nil(t, info.Root())

	// No direct/transitive distinction
	assert.Nil(t, info.Direct())

	transitive := info.Transitive()
	assert.Equal(t, 6, len(transitive))

	p := transitive.Find("com.google.guava:guava")
	require.NotNil(t, p)
	assert.Equal(t, "31.1-jre", p.Version)
	assert.Equal(t, "pkg:maven/com.google.guava/guava@31.1-jre", p.Purl)
	assert.Equal(t, []*sbom.Evidence{{Type: sbom.EvidenceType_EVIDENCE_TYPE_FILE, Value: "path/to/gradle.lockfile"}}, p.EvidenceList)

	p = transitive.Find("org.apache.commons:commons-lang3")
	require.NotNil(t, p)
	assert.Equal(t, "3.12.0", p.Version)
	assert.Equal(t, "pkg:maven/org.apache.commons/commons-lang3@3.12.0", p.Purl)

	// Test deps are included in transitive
	p = transitive.Find("junit:junit")
	require.NotNil(t, p)
	assert.Equal(t, "4.13.2", p.Version)
}

func TestIsTestOnly(t *testing.T) {
	assert.True(t, isTestOnly([]string{"testCompileClasspath", "testRuntimeClasspath"}))
	assert.False(t, isTestOnly([]string{"compileClasspath", "runtimeClasspath"}))
	assert.False(t, isTestOnly([]string{"compileClasspath", "testRuntimeClasspath"}))
	assert.False(t, isTestOnly([]string{}))
}
