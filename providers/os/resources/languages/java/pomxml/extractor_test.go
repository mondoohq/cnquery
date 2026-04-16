// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package pomxml

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/sbom"
)

func TestPomXmlExtractorSimple(t *testing.T) {
	f, err := os.Open("./testdata/simple.pom.xml")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&Extractor{}).Parse(f, "path/to/pom.xml")
	require.NoError(t, err)

	// Root project
	root := info.Root()
	require.NotNil(t, root)
	assert.Equal(t, "com.example:myapp", root.Name)
	assert.Equal(t, "1.0.0", root.Version)
	assert.Equal(t, "pkg:maven/com.example/myapp@1.0.0", root.Purl)
	assert.Equal(t, []*sbom.Evidence{{Type: sbom.EvidenceType_EVIDENCE_TYPE_FILE, Value: "path/to/pom.xml"}}, root.EvidenceList)

	// Direct deps (excludes test and provided)
	direct := info.Direct()
	assert.Equal(t, 2, len(direct))

	p := direct.Find("org.apache.commons:commons-lang3")
	require.NotNil(t, p)
	assert.Equal(t, "3.12.0", p.Version)
	assert.Equal(t, "pkg:maven/org.apache.commons/commons-lang3@3.12.0", p.Purl)

	p = direct.Find("com.google.guava:guava")
	require.NotNil(t, p)
	assert.Equal(t, "31.1-jre", p.Version)

	// Transitive includes all 4 deps (including test and provided)
	transitive := info.Transitive()
	assert.Equal(t, 4, len(transitive))

	p = transitive.Find("junit:junit")
	require.NotNil(t, p)
	assert.Equal(t, "4.13.2", p.Version)

	p = transitive.Find("javax.servlet:javax.servlet-api")
	require.NotNil(t, p)
	assert.Equal(t, "4.0.1", p.Version)
}
