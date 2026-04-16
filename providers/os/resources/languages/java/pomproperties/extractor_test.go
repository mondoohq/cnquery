// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package pomproperties

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/sbom"
)

func TestPomPropertiesExtractorSimple(t *testing.T) {
	f, err := os.Open("./testdata/simple.pom.properties")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&Extractor{}).Parse(f, "META-INF/maven/org.apache.commons/commons-lang3/pom.properties")
	require.NoError(t, err)

	root := info.Root()
	require.NotNil(t, root)
	assert.Equal(t, "org.apache.commons:commons-lang3", root.Name)
	assert.Equal(t, "3.12.0", root.Version)
	assert.Equal(t, "pkg:maven/org.apache.commons/commons-lang3@3.12.0", root.Purl)
	assert.Equal(t, []*sbom.Evidence{{Type: sbom.EvidenceType_EVIDENCE_TYPE_FILE, Value: "META-INF/maven/org.apache.commons/commons-lang3/pom.properties"}}, root.EvidenceList)

	// pom.properties has no direct deps
	assert.Nil(t, info.Direct())

	// Transitive returns the single package
	transitive := info.Transitive()
	assert.Equal(t, 1, len(transitive))
	assert.Equal(t, "org.apache.commons:commons-lang3", transitive[0].Name)
}

func TestPomPropertiesExtractorGuava(t *testing.T) {
	f, err := os.Open("./testdata/guava.pom.properties")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&Extractor{}).Parse(f, "path/to/pom.properties")
	require.NoError(t, err)

	root := info.Root()
	require.NotNil(t, root)
	assert.Equal(t, "com.google.guava:guava", root.Name)
	assert.Equal(t, "31.1-jre", root.Version)
	assert.Equal(t, "pkg:maven/com.google.guava/guava@31.1-jre", root.Purl)
}
