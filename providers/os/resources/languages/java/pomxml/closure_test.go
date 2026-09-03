// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package pomxml

import (
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pom builds a POM declaring the given dependency XML, for the many fixtures
// below that differ only in their <dependencies> block.
func pom(groupID, artifactID, version, deps string) string {
	return `<project>
  <groupId>` + groupID + `</groupId>
  <artifactId>` + artifactID + `</artifactId>
  <version>` + version + `</version>
  <dependencies>` + deps + `</dependencies>
</project>`
}

func dep(groupID, artifactID, version, extra string) string {
	v := ""
	if version != "" {
		v = "<version>" + version + "</version>"
	}
	return `<dependency><groupId>` + groupID + `</groupId><artifactId>` + artifactID +
		`</artifactId>` + v + extra + `</dependency>`
}

// names returns the artifacts in the resolved tree, sorted, so a test states
// what the tree contains without depending on walk order.
func names(t *testing.T, p *pomProject) []string {
	t.Helper()
	var out []string
	for _, pkg := range p.Transitive() {
		out = append(out, pkg.Name)
	}
	sort.Strings(out)
	return out
}

func versionIn(t *testing.T, p *pomProject, name string) string {
	t.Helper()
	for _, pkg := range p.Transitive() {
		if pkg.Name == name {
			return pkg.Version
		}
	}
	return "<absent>"
}

func scopeIn(t *testing.T, p *pomProject, name string) string {
	t.Helper()
	for _, pkg := range p.Transitive() {
		if pkg.Name == name {
			return pkg.Scope
		}
	}
	return "<absent>"
}

// The whole point: a project's dependencies bring their own, and reading only
// the pom.xml reports none of them. Without this, an advisory against a
// transitively-shipped artifact correlates against an inventory that does not
// contain it, and the scan reports clean.
func TestClosureResolvesTransitiveDependencies(t *testing.T) {
	r := &mapResolver{poms: map[string]string{
		"org.example:lib:1.0.0":  pom("org.example", "lib", "1.0.0", dep("org.example", "deep", "2.0.0", "")),
		"org.example:deep:2.0.0": pom("org.example", "deep", "2.0.0", ""),
	}}
	got := parseWith(t, pom("com.example", "app", "1.0.0", dep("org.example", "lib", "1.0.0", "")), r)

	assert.Equal(t, []string{"org.example:deep", "org.example:lib"}, names(t, got))
	assert.Equal(t, "2.0.0", versionIn(t, got, "org.example:deep"))
}

// With no resolver the parser stays hermetic and reports exactly the declared
// list, which is what every existing caller gets. This is the guarantee that
// makes the closure safe to add at all.
func TestClosureAbsentWithoutResolver(t *testing.T) {
	e := &Extractor{}
	bom, err := e.Parse(strings.NewReader(pom("com.example", "app", "1.0.0",
		dep("org.example", "lib", "1.0.0", ""))), "pom.xml")
	require.NoError(t, err)

	assert.Equal(t, []string{"org.example:lib"}, names(t, bom.(*pomProject)),
		"no resolver means no tree to walk; the declared list is the answer")
}

// An artifact whose POM cannot be read ends its subtree. It is a GAP, not an
// assertion that it has no dependencies -- and crucially the artifact itself
// still appears, because the project does ship it.
func TestClosureUnreadablePomEndsTheSubtree(t *testing.T) {
	r := &mapResolver{poms: map[string]string{}} // nothing resolves
	got := parseWith(t, pom("com.example", "app", "1.0.0", dep("org.example", "lib", "1.0.0", "")), r)

	assert.Equal(t, []string{"org.example:lib"}, names(t, got),
		"the declared artifact is shipped and must be reported even when its POM is missing")
}

// Maven's transitive-scope table, in the two directions that matter.
//
// A `test` dependency of a dependency is not YOUR dependency: it exists for that
// artifact's own test run and is absent from what you ship. Same for `provided`,
// which the container supplies to that artifact, not to you. Walking them anyway
// is how a naive closure reports several times the artifacts a project has, and
// every one of those is a finding a user cannot act on.
func TestClosureDropsTestAndProvidedTransitives(t *testing.T) {
	libDeps := dep("org.example", "runtime-dep", "1.0.0", "<scope>runtime</scope>") +
		dep("org.example", "test-dep", "1.0.0", "<scope>test</scope>") +
		dep("org.example", "provided-dep", "1.0.0", "<scope>provided</scope>")
	r := &mapResolver{poms: map[string]string{
		"org.example:lib:1.0.0": pom("org.example", "lib", "1.0.0", libDeps),
	}}
	got := parseWith(t, pom("com.example", "app", "1.0.0", dep("org.example", "lib", "1.0.0", "")), r)

	assert.Equal(t, []string{"org.example:lib", "org.example:runtime-dep"}, names(t, got))
}

// The root's OWN test dependencies are a different case and must be kept: they
// are on the project's test classpath, so a vulnerability in one is real for
// whoever runs the build. They are reported as dev so a consumer can rank them
// differently, which is not the same as dropping them.
func TestClosureKeepsTheProjectsOwnTestDependencies(t *testing.T) {
	r := &mapResolver{poms: map[string]string{}}
	got := parseWith(t, pom("com.example", "app", "1.0.0",
		dep("org.junit.jupiter", "junit-jupiter", "5.10.0", "<scope>test</scope>")), r)

	assert.Equal(t, "dev", scopeIn(t, got, "org.junit.jupiter:junit-jupiter"))
}

// `provided` is production, not dev. The container supplies it at runtime, so a
// deployed application calls into it on every request -- marking it dev tells a
// consumer to care less about a vulnerability that is live in their deployment.
func TestClosureKeepsProvidedAsProduction(t *testing.T) {
	r := &mapResolver{poms: map[string]string{}}
	got := parseWith(t, pom("com.example", "app", "1.0.0",
		dep("jakarta.servlet", "jakarta.servlet-api", "6.0.0", "<scope>provided</scope>")), r)

	assert.Equal(t, "prod", scopeIn(t, got, "jakarta.servlet:jakarta.servlet-api"))
}

// Version mediation: Maven takes the version on the path NEAREST the root, not
// the highest, not the last seen, and not the one a deeper path reached first.
//
// This fixture isolates depth from DECLARATION ORDER: `far` is declared first
// and reaches shared@9.9.9 in two hops, `near` is declared second and reaches
// shared@1.0.0 in one. An implementation that simply took the first path it
// enumerated would report 9.9.9, which is a different artifact than Maven
// resolves and therefore a different set of advisories.
//
// It does NOT pin breadth-first on its own -- a depth-first walk happens to
// agree here. TestClosureWalkIsBreadthFirst below is the fixture that separates
// those two, and both are needed.
func TestClosureNearestVersionWins(t *testing.T) {
	r := &mapResolver{poms: map[string]string{
		"org.example:far:1.0.0":    pom("org.example", "far", "1.0.0", dep("org.example", "mid", "1.0.0", "")),
		"org.example:mid:1.0.0":    pom("org.example", "mid", "1.0.0", dep("org.example", "shared", "9.9.9", "")),
		"org.example:near:1.0.0":   pom("org.example", "near", "1.0.0", dep("org.example", "shared", "1.0.0", "")),
		"org.example:shared:1.0.0": pom("org.example", "shared", "1.0.0", ""),
		"org.example:shared:9.9.9": pom("org.example", "shared", "9.9.9", ""),
	}}
	got := parseWith(t, pom("com.example", "app", "1.0.0",
		dep("org.example", "far", "1.0.0", "")+dep("org.example", "near", "1.0.0", "")), r)

	assert.Equal(t, "1.0.0", versionIn(t, got, "org.example:shared"),
		"shared is one hop through near and two through far; the nearer path decides")
}

// The walk order itself, which is what makes mediation come out right.
//
// Declaration order here favours `near` too, so this fixture does not separate
// depth from order -- that is the test above. What it does separate is
// BREADTH-first from DEPTH-first: pop the queue from the back instead of the
// front and the walk descends far -> mid -> shared@9.9.9 before it ever looks
// at near's one-hop shared@1.0.0, and reports the version Maven does not pick.
func TestClosureWalkIsBreadthFirst(t *testing.T) {
	r := &mapResolver{poms: map[string]string{
		"org.example:near:1.0.0":   pom("org.example", "near", "1.0.0", dep("org.example", "shared", "1.0.0", "")),
		"org.example:far:1.0.0":    pom("org.example", "far", "1.0.0", dep("org.example", "mid", "1.0.0", "")),
		"org.example:mid:1.0.0":    pom("org.example", "mid", "1.0.0", dep("org.example", "shared", "9.9.9", "")),
		"org.example:shared:1.0.0": pom("org.example", "shared", "1.0.0", ""),
		"org.example:shared:9.9.9": pom("org.example", "shared", "9.9.9", ""),
	}}
	got := parseWith(t, pom("com.example", "app", "1.0.0",
		dep("org.example", "near", "1.0.0", "")+dep("org.example", "far", "1.0.0", "")), r)

	assert.Equal(t, "1.0.0", versionIn(t, got, "org.example:shared"),
		"a depth-first walk reaches shared@9.9.9 through far/mid before near's one-hop 1.0.0")
}

// The other half of mediation: a version the project declares ITSELF is at
// depth zero and beats anything the tree reaches.
func TestClosureDirectDeclarationBeatsTheTree(t *testing.T) {
	r := &mapResolver{poms: map[string]string{
		"org.example:lib:1.0.0":    pom("org.example", "lib", "1.0.0", dep("org.example", "shared", "9.9.9", "")),
		"org.example:shared:1.0.0": pom("org.example", "shared", "1.0.0", ""),
	}}
	got := parseWith(t, pom("com.example", "app", "1.0.0",
		dep("org.example", "lib", "1.0.0", "")+dep("org.example", "shared", "1.0.0", "")), r)

	assert.Equal(t, "1.0.0", versionIn(t, got, "org.example:shared"))
}

// The root's dependencyManagement overrides a version stated anywhere in the
// tree. This is how a project pins a transitive artifact it does not declare --
// the usual reason being that the pinned version is the one without the CVE, so
// reporting the overridden version would report a vulnerability the project has
// already fixed.
func TestClosureRootManagementOverridesTransitiveVersion(t *testing.T) {
	root := `<project>
  <groupId>com.example</groupId>
  <artifactId>app</artifactId>
  <version>1.0.0</version>
  <dependencyManagement>
    <dependencies>` + dep("org.example", "shared", "2.0.0", "") + `</dependencies>
  </dependencyManagement>
  <dependencies>` + dep("org.example", "lib", "1.0.0", "") + `</dependencies>
</project>`
	r := &mapResolver{poms: map[string]string{
		"org.example:lib:1.0.0": pom("org.example", "lib", "1.0.0", dep("org.example", "shared", "1.0.0", "")),
	}}
	got := parseWith(t, root, r)

	assert.Equal(t, "2.0.0", versionIn(t, got, "org.example:shared"),
		"the project pinned it; the tree's own version must not win")
}

// An <exclusion> applies to the WHOLE subtree beneath the dependency carrying
// it, not just its immediate children. A project excluding a logging backend
// two levels down would otherwise still be told it ships one.
func TestClosureExclusionsApplyToTheWholeSubtree(t *testing.T) {
	excl := `<exclusions><exclusion><groupId>org.example</groupId><artifactId>deep</artifactId></exclusion></exclusions>`
	r := &mapResolver{poms: map[string]string{
		"org.example:lib:1.0.0": pom("org.example", "lib", "1.0.0", dep("org.example", "mid", "1.0.0", "")),
		"org.example:mid:1.0.0": pom("org.example", "mid", "1.0.0", dep("org.example", "deep", "1.0.0", "")),
	}}
	got := parseWith(t, pom("com.example", "app", "1.0.0", dep("org.example", "lib", "1.0.0", excl)), r)

	assert.Equal(t, []string{"org.example:lib", "org.example:mid"}, names(t, got),
		"deep is two levels below the exclusion and must still be excluded")
}

// An <optional> dependency is one its own author says consumers must declare
// themselves. It is not dragged in, so it never enters a consumer's tree.
func TestClosureSkipsOptionalTransitives(t *testing.T) {
	libDeps := dep("org.example", "wanted", "1.0.0", "") +
		dep("org.example", "opt", "1.0.0", "<optional>true</optional>")
	r := &mapResolver{poms: map[string]string{
		"org.example:lib:1.0.0": pom("org.example", "lib", "1.0.0", libDeps),
	}}
	got := parseWith(t, pom("com.example", "app", "1.0.0", dep("org.example", "lib", "1.0.0", "")), r)

	assert.Equal(t, []string{"org.example:lib", "org.example:wanted"}, names(t, got))
}

// A cycle in the tree terminates. Maven trees are not supposed to have them,
// but a local repository can hold anything, and a walk that trusts the data
// hangs the scan rather than reporting one.
func TestClosureTerminatesOnACycle(t *testing.T) {
	r := &mapResolver{poms: map[string]string{
		"org.example:a:1.0.0": pom("org.example", "a", "1.0.0", dep("org.example", "b", "1.0.0", "")),
		"org.example:b:1.0.0": pom("org.example", "b", "1.0.0", dep("org.example", "a", "1.0.0", "")),
	}}
	got := parseWith(t, pom("com.example", "app", "1.0.0", dep("org.example", "a", "1.0.0", "")), r)

	assert.Equal(t, []string{"org.example:a", "org.example:b"}, names(t, got))
}

// A transitive artifact declared without a version resolves it from ITS OWN
// parent chain, which is the standard Spring layout one level down. Reading the
// artifact's POM alone yields a versionless dependency, and a purl with no
// version matches no advisory -- the exact failure inherit.go exists to prevent,
// reappearing inside the tree.
func TestClosureResolvesTransitiveVersionsFromTheirOwnParent(t *testing.T) {
	libPom := `<project>
  <parent>
    <groupId>org.example</groupId>
    <artifactId>lib-parent</artifactId>
    <version>1.0.0</version>
  </parent>
  <groupId>org.example</groupId>
  <artifactId>lib</artifactId>
  <version>1.0.0</version>
  <dependencies>` + dep("org.example", "managed", "", "") + `</dependencies>
</project>`
	libParent := `<project>
  <groupId>org.example</groupId>
  <artifactId>lib-parent</artifactId>
  <version>1.0.0</version>
  <dependencyManagement>
    <dependencies>` + dep("org.example", "managed", "3.1.4", "") + `</dependencies>
  </dependencyManagement>
</project>`
	r := &mapResolver{poms: map[string]string{
		"org.example:lib:1.0.0":        libPom,
		"org.example:lib-parent:1.0.0": libParent,
	}}
	got := parseWith(t, pom("com.example", "app", "1.0.0", dep("org.example", "lib", "1.0.0", "")), r)

	assert.Equal(t, "3.1.4", versionIn(t, got, "org.example:managed"),
		"a transitive artifact inherits its versions the same way the project does")
}

// The tree's edges are recorded, not just its nodes. A consumer that knows only
// the flat set cannot tell which of its own dependencies pulled a vulnerable
// artifact in, which is the first thing anyone asks on seeing the finding.
func TestClosureRecordsDependencyEdges(t *testing.T) {
	r := &mapResolver{poms: map[string]string{
		"org.example:lib:1.0.0":  pom("org.example", "lib", "1.0.0", dep("org.example", "deep", "2.0.0", "")),
		"org.example:deep:2.0.0": pom("org.example", "deep", "2.0.0", ""),
	}}
	got := parseWith(t, pom("com.example", "app", "1.0.0", dep("org.example", "lib", "1.0.0", "")), r)

	for _, pkg := range got.Transitive() {
		if pkg.Name == "org.example:lib" {
			assert.Equal(t, []string{"pkg:maven/org.example/deep@2.0.0"}, pkg.DependsOn,
				"lib is what pulls deep in, and the edge is how a user finds that out")
			return
		}
	}
	t.Fatal("org.example:lib absent from the tree")
}

// Direct() is unchanged by the closure: it answers "what did this project ask
// for", which is a different question from "what does it ship".
func TestClosureLeavesDirectAlone(t *testing.T) {
	r := &mapResolver{poms: map[string]string{
		"org.example:lib:1.0.0": pom("org.example", "lib", "1.0.0", dep("org.example", "deep", "2.0.0", "")),
	}}
	got := parseWith(t, pom("com.example", "app", "1.0.0", dep("org.example", "lib", "1.0.0", "")), r)

	var direct []string
	for _, pkg := range got.Direct() {
		direct = append(direct, pkg.Name)
	}
	assert.Equal(t, []string{"org.example:lib"}, direct,
		"deep is shipped but not declared; Direct must not report it")
}

// A dependency whose version never resolves is reported without one rather than
// guessed at, and its subtree is not walked -- there is no artifact to look up.
// An invented version would correlate the project against a stranger's
// advisories, which is worse than an admitted gap.
func TestClosureNeverGuessesAnUnresolvableVersion(t *testing.T) {
	r := &mapResolver{poms: map[string]string{}}
	got := parseWith(t, pom("com.example", "app", "1.0.0",
		dep("org.example", "lib", "${missing.version}", "")), r)

	assert.Equal(t, "", versionIn(t, got, "org.example:lib"))
	assert.Empty(t, r.calls, "there is no coordinate to look up without a version")
}

// A dependency the PROJECT declares optional is still declared -- it is in the
// project's own POM and on its classpath -- so it is reported, with the flag.
// The closure builds its packages itself rather than through depToPackage, so
// this is the path where the flag is easiest to drop silently.
func TestClosureKeepsTheOptionalFlagOnDeclaredDependencies(t *testing.T) {
	r := &mapResolver{poms: map[string]string{}}
	got := parseWith(t, pom("com.example", "app", "1.0.0",
		dep("org.example", "opt", "1.0.0", "<optional>true</optional>")+
			dep("org.example", "plain", "1.0.0", "")), r)

	for _, pkg := range got.Transitive() {
		switch pkg.Name {
		case "org.example:opt":
			assert.True(t, pkg.Optional, "declared <optional>true</optional>")
		case "org.example:plain":
			assert.False(t, pkg.Optional)
		}
	}
}

// Scope mediation takes the STRONGEST scope any path reaches an artifact under,
// not the first one enumerated.
//
// This is the spring-petclinic failure that made the closure worth measuring
// rather than reasoning about: `shared` is reachable both through a test-scoped
// dependency and through a compile-scoped one. Taking the first scope reported
// eight artifacts the application ships -- spring-beans and slf4j-api among
// them -- as test-only, which tells a user to care less about a vulnerability
// that is live in their deployment. The test-scoped path is declared first here
// precisely so that mistake fails.
func TestClosureTakesTheStrongestScope(t *testing.T) {
	r := &mapResolver{poms: map[string]string{
		"org.example:testlib:1.0.0": pom("org.example", "testlib", "1.0.0", dep("org.example", "shared", "1.0.0", "")),
		"org.example:prodlib:1.0.0": pom("org.example", "prodlib", "1.0.0", dep("org.example", "shared", "1.0.0", "")),
		"org.example:shared:1.0.0":  pom("org.example", "shared", "1.0.0", ""),
	}}
	got := parseWith(t, pom("com.example", "app", "1.0.0",
		dep("org.example", "testlib", "1.0.0", "<scope>test</scope>")+
			dep("org.example", "prodlib", "1.0.0", "")), r)

	assert.Equal(t, "prod", scopeIn(t, got, "org.example:shared"),
		"shared is on the compile classpath through prodlib, whatever else also reaches it")
	assert.Equal(t, "dev", scopeIn(t, got, "org.example:testlib"))
}

// Strengthening has to re-walk the subtree, not just relabel the node. The
// artifact below a test-scoped path was pruned under the weaker scope; if the
// upgrade stops at the node itself, everything under it stays mislabelled.
func TestClosureStrengtheningReachesTheWholeSubtree(t *testing.T) {
	r := &mapResolver{poms: map[string]string{
		"org.example:testlib:1.0.0": pom("org.example", "testlib", "1.0.0", dep("org.example", "mid", "1.0.0", "")),
		"org.example:prodlib:1.0.0": pom("org.example", "prodlib", "1.0.0", dep("org.example", "mid", "1.0.0", "")),
		"org.example:mid:1.0.0":     pom("org.example", "mid", "1.0.0", dep("org.example", "deep", "1.0.0", "")),
		"org.example:deep:1.0.0":    pom("org.example", "deep", "1.0.0", ""),
	}}
	got := parseWith(t, pom("com.example", "app", "1.0.0",
		dep("org.example", "testlib", "1.0.0", "<scope>test</scope>")+
			dep("org.example", "prodlib", "1.0.0", "")), r)

	assert.Equal(t, "prod", scopeIn(t, got, "org.example:mid"))
	assert.Equal(t, "prod", scopeIn(t, got, "org.example:deep"),
		"deep is two levels under the strengthened path and is shipped just the same")
}
