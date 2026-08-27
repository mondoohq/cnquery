// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package pomxml

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/providers/os/resources/languages"
	"go.mondoo.com/mql/sbom"
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

// A version written as ${jackson.version} is how a Maven project keeps a family
// of artifacts on one version, and it is extremely common. Before properties
// were resolved, the dependency's version was the literal property reference —
// which produces a purl that matches no package registry and no advisory, so
// the dependency was silently exempt from vulnerability correlation and from
// license lookup alike.
func TestPomXmlResolvesPropertyVersions(t *testing.T) {
	f, err := os.Open("./testdata/properties.pom.xml")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&Extractor{}).Parse(f, "pom.xml")
	require.NoError(t, err)

	byName := map[string]*languages.Package{}
	for _, p := range info.Transitive() {
		byName[p.Name] = p
	}

	t.Run("a declared property is substituted", func(t *testing.T) {
		p := byName["org.springframework:spring-core"]
		require.NotNil(t, p)
		assert.Equal(t, "5.3.18", p.Version)
		assert.Equal(t, "pkg:maven/org.springframework/spring-core@5.3.18", p.Purl)
	})

	t.Run("a property whose value is another property resolves through", func(t *testing.T) {
		p := byName["com.fasterxml.jackson.core:jackson-databind"]
		require.NotNil(t, p)
		assert.Equal(t, "2.13.0", p.Version)
		assert.Equal(t, "pkg:maven/com.fasterxml.jackson.core/jackson-databind@2.13.0", p.Purl)
	})

	t.Run("project.version is a built-in", func(t *testing.T) {
		p := byName["com.example:myapp-core"]
		require.NotNil(t, p)
		assert.Equal(t, "2.5.0", p.Version)
	})

	t.Run("dependencyManagement supplies an omitted version", func(t *testing.T) {
		p := byName["com.google.guava:guava"]
		require.NotNil(t, p)
		assert.Equal(t, "31.1-jre", p.Version)
		assert.Equal(t, "pkg:maven/com.google.guava/guava@31.1-jre", p.Purl)
	})

	// An unresolvable reference stays as written. Substituting a guess would
	// report the dependency at some other version's identity, matching that
	// version's advisories and that version's license; leaving the reference
	// intact keeps the gap visible to whatever reads it.
	t.Run("an undeclared property is not guessed at", func(t *testing.T) {
		p := byName["org.example:mystery"]
		require.NotNil(t, p)
		assert.Equal(t, "${undeclared.version}", p.Version)
	})
}

// A property may refer to itself, directly or through a cycle. Resolution must
// terminate rather than spin, and must not invent a value.
func TestPomXmlPropertyCycleTerminates(t *testing.T) {
	pom := `<project xmlns="http://maven.apache.org/POM/4.0.0">
  <groupId>g</groupId><artifactId>a</artifactId><version>1.0</version>
  <properties><a.version>${b.version}</a.version><b.version>${a.version}</b.version></properties>
  <dependencies>
    <dependency><groupId>g</groupId><artifactId>dep</artifactId><version>${a.version}</version></dependency>
  </dependencies>
</project>`
	done := make(chan *languages.Package, 1)
	go func() {
		info, err := (&Extractor{}).Parse(strings.NewReader(pom), "pom.xml")
		if err != nil {
			done <- nil
			return
		}
		for _, p := range info.Transitive() {
			if p.Name == "g:dep" {
				done <- p
				return
			}
		}
		done <- nil
	}()
	select {
	case p := <-done:
		require.NotNil(t, p)
		// Whatever it settles on, it must still look unresolved rather than
		// having been given a fabricated version.
		assert.Contains(t, p.Version, "${")
	case <-time.After(10 * time.Second):
		t.Fatal("property resolution did not terminate on a cyclic declaration")
	}
}
