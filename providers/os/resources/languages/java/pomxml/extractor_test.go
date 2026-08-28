// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package pomxml

import (
	"encoding/xml"
	"io"
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

// parsePom is a small helper for the POMs below, which are written inline
// because each one exists to hold a single shape.
func parsePom(t *testing.T, pom string) *pomProject {
	t.Helper()
	info, err := (&Extractor{}).Parse(strings.NewReader(pom), "pom.xml")
	require.NoError(t, err)
	p, ok := info.(*pomProject)
	require.True(t, ok, "Parse should return a *pomProject")
	return p
}

func packagesByName(pkgs languages.Packages) map[string]*languages.Package {
	out := map[string]*languages.Package{}
	for _, p := range pkgs {
		out[p.Name] = p
	}
	return out
}

// A <dependency> and the <dependencyManagement> entry that versions it need not
// spell their coordinates the same way: writing ${project.groupId} in one and
// the literal in the other is ordinary, and so is the reverse. Comparing them as
// written matches only by coincidence, and a miss is silent — the dependency
// comes out with no version at all, which is the outcome resolving versions was
// meant to prevent.
func TestPomXmlManagedVersionMatchesAcrossPropertyReferences(t *testing.T) {
	t.Run("the dependency uses a property, the management entry is literal", func(t *testing.T) {
		p := parsePom(t, `<project xmlns="http://maven.apache.org/POM/4.0.0">
  <groupId>com.example</groupId><artifactId>myapp</artifactId><version>2.5.0</version>
  <dependencyManagement><dependencies>
    <dependency><groupId>com.example</groupId><artifactId>myapp-core</artifactId><version>2.5.0</version></dependency>
  </dependencies></dependencyManagement>
  <dependencies>
    <dependency><groupId>${project.groupId}</groupId><artifactId>myapp-core</artifactId></dependency>
  </dependencies>
</project>`)

		got := packagesByName(p.Transitive())["com.example:myapp-core"]
		require.NotNil(t, got)
		assert.Equal(t, "2.5.0", got.Version)
		assert.Equal(t, "pkg:maven/com.example/myapp-core@2.5.0", got.Purl)
	})

	t.Run("the management entry uses a property, the dependency is literal", func(t *testing.T) {
		p := parsePom(t, `<project xmlns="http://maven.apache.org/POM/4.0.0">
  <groupId>com.example</groupId><artifactId>myapp</artifactId><version>2.5.0</version>
  <properties><guava.groupId>com.google.guava</guava.groupId></properties>
  <dependencyManagement><dependencies>
    <dependency><groupId>${guava.groupId}</groupId><artifactId>guava</artifactId><version>31.1-jre</version></dependency>
  </dependencies></dependencyManagement>
  <dependencies>
    <dependency><groupId>com.google.guava</groupId><artifactId>guava</artifactId></dependency>
  </dependencies>
</project>`)

		got := packagesByName(p.Transitive())["com.google.guava:guava"]
		require.NotNil(t, got)
		assert.Equal(t, "31.1-jre", got.Version)
		assert.Equal(t, "pkg:maven/com.google.guava/guava@31.1-jre", got.Purl)
	})
}

// Maven applies a managed scope exactly as it applies a managed version, so the
// two have to be read together. A dependency omitting both is a test dependency
// at the managed version; reading only the version promotes it into the
// production set with a real, matchable purl, and it arrives in an SBOM as
// something the application ships.
func TestPomXmlManagedScopeKeepsTestDependenciesOutOfDirect(t *testing.T) {
	p := parsePom(t, `<project xmlns="http://maven.apache.org/POM/4.0.0">
  <groupId>com.example</groupId><artifactId>myapp</artifactId><version>2.5.0</version>
  <dependencyManagement><dependencies>
    <dependency><groupId>junit</groupId><artifactId>junit</artifactId><version>4.13.2</version><scope>test</scope></dependency>
    <dependency><groupId>jakarta.servlet</groupId><artifactId>jakarta.servlet-api</artifactId><version>5.0.0</version><scope>provided</scope></dependency>
    <dependency><groupId>com.google.guava</groupId><artifactId>guava</artifactId><version>31.1-jre</version></dependency>
  </dependencies></dependencyManagement>
  <dependencies>
    <dependency><groupId>junit</groupId><artifactId>junit</artifactId></dependency>
    <dependency><groupId>jakarta.servlet</groupId><artifactId>jakarta.servlet-api</artifactId></dependency>
    <dependency><groupId>com.google.guava</groupId><artifactId>guava</artifactId></dependency>
  </dependencies>
</project>`)

	direct := packagesByName(p.Direct())
	assert.NotContains(t, direct, "junit:junit",
		"a managed test scope must keep the dependency out of the production set")
	assert.NotContains(t, direct, "jakarta.servlet:jakarta.servlet-api",
		"a managed provided scope must keep the dependency out of the production set")

	guava := direct["com.google.guava:guava"]
	require.NotNil(t, guava, "a managed dependency with no scope is still production")
	assert.Equal(t, "31.1-jre", guava.Version)

	// Transitive is every declared dependency regardless of scope, so the
	// managed version still has to reach them.
	all := packagesByName(p.Transitive())
	require.NotNil(t, all["junit:junit"])
	assert.Equal(t, "4.13.2", all["junit:junit"].Version)
}

// A dependency's own scope wins over the managed one, which is what lets a
// module opt a managed test dependency back into its production set.
func TestPomXmlDependencyScopeOverridesTheManagedScope(t *testing.T) {
	p := parsePom(t, `<project xmlns="http://maven.apache.org/POM/4.0.0">
  <groupId>com.example</groupId><artifactId>myapp</artifactId><version>2.5.0</version>
  <dependencyManagement><dependencies>
    <dependency><groupId>junit</groupId><artifactId>junit</artifactId><version>4.13.2</version><scope>test</scope></dependency>
  </dependencies></dependencyManagement>
  <dependencies>
    <dependency><groupId>junit</groupId><artifactId>junit</artifactId><scope>compile</scope></dependency>
  </dependencies>
</project>`)

	direct := packagesByName(p.Direct())
	require.Contains(t, direct, "junit:junit")
	assert.Equal(t, "4.13.2", direct["junit:junit"].Version)
}

// <properties> decodes through a custom UnmarshalXML, so what it does with a
// decoding failure is its own decision and has to be asserted on directly: at
// the Parse boundary the outer decoder fails on the same malformed document
// either way, which would make a test here pass whatever UnmarshalXML returned.
func TestPomPropertiesUnmarshal(t *testing.T) {
	decodeProperties := func(t *testing.T, doc string) (pomProperties, error) {
		t.Helper()
		d := xml.NewDecoder(strings.NewReader(doc))
		tok, err := d.Token()
		require.NoError(t, err)
		start, ok := tok.(xml.StartElement)
		require.True(t, ok)

		var p pomProperties
		return p, p.UnmarshalXML(d, start)
	}

	t.Run("a well-formed block decodes every property", func(t *testing.T) {
		props, err := decodeProperties(t,
			`<properties><spring.version>5.3.18</spring.version><jackson.version>2.13.0</jackson.version></properties>`)
		require.NoError(t, err)
		assert.Equal(t, pomProperties{"spring.version": "5.3.18", "jackson.version": "2.13.0"}, props)
	})

	// Swallowing this would leave every version referring to a property in the
	// block unresolvable, and an unresolvable version is a dependency no
	// advisory matches — a silent gap rather than a reported one.
	t.Run("a malformed block is an error", func(t *testing.T) {
		_, err := decodeProperties(t, `<properties><good>1.0</good><bad>&notanentity;</bad></properties>`)
		require.Error(t, err)
	})

	// An early end of document arrives as a syntax error rather than a bare
	// io.EOF, because the decoder is inside an open element. Singling io.EOF out
	// would be a branch that never runs.
	t.Run("a truncated block is an error too", func(t *testing.T) {
		_, err := decodeProperties(t, `<properties><good>1.0</good>`)
		require.Error(t, err)
		assert.NotErrorIs(t, err, io.EOF)
	})
}
