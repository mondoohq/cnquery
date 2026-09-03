// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package pomxml

import (
	"strings"

	"go.mondoo.com/mql/providers/os/resources/languages"
	"go.mondoo.com/mql/providers/os/resources/languages/java"
)

// The transitive closure: what a pom.xml means, rather than what it says.
//
// A pom.xml states the dependencies a project asked for. It does not state the
// ones those in turn drag in, and for a real application that is most of them --
// spring-petclinic declares 12 and ships an order of magnitude more. Reporting
// only the declared set is not a partial inventory in the harmless sense: every
// artifact the application actually ships is absent from it, so an advisory
// against any of them correlates against nothing and a caller who ran a
// vulnerability scan is told they are clean about code they never examined.
//
// This is the same problem inherit.go solved for versions, at the next level up,
// and it is solved the same way. The coordinates of each dependency are stated
// in the POM; on a machine that has built the project their POMs are already on
// disk; what is missing is permission to go and read them. So the caller
// supplies them through the SAME ParentResolver, and with none supplied nothing
// here runs and Transitive() reports exactly the declared list it always did.
//
// What is NOT done here matters as much, and it is the same rule inherit.go
// states. A POM that cannot be read ends that subtree: the dependencies below it
// are unknown, never asserted absent, and never guessed at from a directory
// listing or a "latest version wins" fallback. An inventory that invents an
// artifact the project does not ship is worse than one that admits a gap,
// because it correlates the project against a stranger's advisories.

const (
	// maxClosureDepth bounds the dependency tree's depth. Real trees are deep --
	// a Spring Boot starter reaches eight or nine levels -- but not unbounded,
	// and a cycle that survives the visited-set check terminates here.
	maxClosureDepth = 32
	// maxClosurePoms bounds the total artifacts read for one project's closure.
	// spring-petclinic reads roughly 200; the cap is set well above what a real
	// application needs so that hitting it means something is wrong, not that a
	// large project was silently truncated.
	maxClosurePoms = 4096
)

// closureNode is one artifact in the resolved tree: the package it renders as,
// and the artifacts it depends on in turn.
type closureNode struct {
	key       string // groupId:artifactId -- the identity mediation is keyed on
	pkg       *languages.Package
	scope     string   // the Maven scope this artifact was reached under
	dependsOn []string // purls of its resolved children
}

// closureWalker resolves a project's dependency tree breadth-first.
//
// Breadth-first is not an implementation detail, it is Maven's version
// mediation. When two paths reach the same artifact at different versions Maven
// takes the one NEAREST the root, and a breadth-first walk visits the nearest
// path first -- so the first time an artifact is seen is the version that wins,
// and later sightings only contribute edges. Ties at equal depth go to the
// earlier declaration, which is also Maven's rule and also falls out of the
// walk order.
type closureWalker struct {
	root     *pomProject
	resolver ParentResolver
	read     int
	picked   map[string]*closureNode
	order    []*closureNode
	// poms caches each artifact's parsed, parent-resolved POM. A diamond in the
	// tree is the normal shape, not the exception, and re-reading and
	// re-inheriting the same artifact for every path that reaches it is what
	// makes a naive walk quadratic.
	poms map[string]*pomProject
}

// pending is one edge of the tree still to be walked: the dependency as its
// parent declares it, the pom that declares it (whose properties resolve it),
// and the state accumulated along the path.
type pending struct {
	dep        pomDependency
	owner      *pomProject
	scope      string
	exclusions []pomExclusion
	depth      int
	parent     *closureNode
}

// resolveClosure walks the project's dependency tree and returns every artifact
// in it, the project's own declared dependencies first and in declaration order.
//
// A nil resolver returns nil, which is the caller's signal to report the
// declared list unchanged.
func (p *pomProject) resolveClosure(r ParentResolver) []*closureNode {
	if r == nil {
		return nil
	}
	w := &closureWalker{
		root:     p,
		resolver: r,
		picked:   map[string]*closureNode{},
		poms:     map[string]*pomProject{},
	}
	return w.walk()
}

func (w *closureWalker) walk() []*closureNode {
	// The project's own declared dependencies seed the walk, in the order the
	// POM writes them, so a root declaration always wins mediation against
	// anything deeper.
	var queue []pending
	for _, dep := range w.root.Dependencies {
		scope := scopeOrDefault(w.root.effectiveScope(dep))
		queue = append(queue, pending{
			dep:        dep,
			owner:      w.root,
			scope:      scope,
			exclusions: dep.Exclusions,
			depth:      0,
		})
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		node, isNew := w.admit(cur)
		if node == nil {
			continue
		}
		if cur.parent != nil && node.pkg.Purl != "" {
			cur.parent.dependsOn = append(cur.parent.dependsOn, node.pkg.Purl)
		}
		// Only a newly admitted artifact is expanded. A second path reaching an
		// artifact already in the tree contributes its edge (above) and nothing
		// else: its subtree is the same subtree, and Maven resolved it at the
		// nearer version.
		// isNew is also true when an existing artifact was just upgraded to a
		// stronger scope: its subtree was pruned under the weaker one and has to
		// be walked again, or the strengthening stops at this node.
		if !isNew || cur.depth >= maxClosureDepth {
			continue
		}
		child := w.pomFor(node)
		if child == nil {
			// The artifact's POM could not be read. Its own dependencies are
			// unknown -- NOT absent -- so the subtree simply ends here.
			continue
		}
		for _, dep := range child.Dependencies {
			next, ok := transitiveScope(node.scope, scopeOrDefault(child.effectiveScope(dep)))
			if !ok {
				continue
			}
			// An optional dependency is one the artifact's own author says its
			// consumers must declare themselves if they want it. It is declared,
			// but it is not dragged in, so it never enters a consumer's tree.
			if child.isOptional(dep) {
				continue
			}
			queue = append(queue, pending{
				dep:        dep,
				owner:      child,
				scope:      next,
				exclusions: append(append([]pomExclusion{}, cur.exclusions...), dep.Exclusions...),
				depth:      cur.depth + 1,
				parent:     node,
			})
		}
	}

	for _, n := range w.order {
		n.pkg.DependsOn = dedupe(n.dependsOn)
	}
	return w.order
}

// admit renders one pending edge as an artifact and records it, or reports that
// it is excluded, unidentifiable, or already in the tree.
//
// isNew is false for an artifact already picked, whose returned node is the one
// that won mediation -- so the caller can still attach the edge to it.
func (w *closureWalker) admit(cur pending) (node *closureNode, isNew bool) {
	groupID := strings.TrimSpace(cur.owner.resolve(cur.dep.GroupId))
	artifactID := strings.TrimSpace(cur.owner.resolve(cur.dep.ArtifactId))
	if groupID == "" || artifactID == "" {
		return nil, false
	}
	if excluded(cur.exclusions, groupID, artifactID) {
		return nil, false
	}
	key := groupID + ":" + artifactID
	if existing, ok := w.picked[key]; ok {
		// Version is settled by the NEAREST path and does not move. Scope is
		// mediated differently: Maven takes the STRONGEST scope any path reaches
		// an artifact under, because an artifact on the compile classpath is on
		// it however many test-scoped paths also happen to reach it.
		//
		// Taking the first scope instead reports an artifact the application
		// ships as a test-only dependency, purely because a test-scoped
		// declaration was enumerated first -- which tells a user to care less
		// about a vulnerability that is live in production. spring-petclinic has
		// eight such artifacts, spring-beans and slf4j-api among them.
		if scopeStronger(cur.scope, existing.scope) {
			existing.scope = cur.scope
			existing.pkg.Scope = packageScope(cur.scope)
			return existing, true // re-expand: the subtree was pruned under a weaker scope
		}
		return existing, false
	}

	version := w.versionFor(cur, groupID, artifactID)
	pkg := &languages.Package{
		Name:         key,
		Version:      version,
		Purl:         java.NewPackageUrl(groupID, artifactID, version),
		Cpes:         java.NewCpes(groupID, artifactID, version),
		EvidenceList: java.NewEvidenceList(w.root.evidence),
		Scope:        packageScope(cur.scope),
		// Only a dependency the PROJECT declares can be optional in the tree: an
		// optional transitive is not dragged in at all (see the walk), so it is
		// never here to be flagged.
		Optional: cur.owner.isOptional(cur.dep),
	}
	node = &closureNode{key: key, pkg: pkg, scope: cur.scope}
	w.picked[key] = node
	w.order = append(w.order, node)
	return node, true
}

// versionFor decides which version of an artifact the tree carries.
//
// The ROOT's dependencyManagement is consulted first and wins outright. That is
// Maven's rule and it is the one that matters most in practice: a project pins a
// transitive artifact by managing it, precisely because it does not declare it
// and cannot state a version any other way. Reading the declaring POM first
// would report the version the project overrode.
func (w *closureWalker) versionFor(cur pending, groupID, artifactID string) string {
	managed := pomDependency{
		GroupId:    groupID,
		ArtifactId: artifactID,
		Type:       cur.dep.Type,
		Classifier: cur.dep.Classifier,
	}
	if v := strings.TrimSpace(w.root.resolve(w.root.managedVersion(managed))); v != "" && !hasUnresolvedProperty(v) {
		return v
	}
	v := strings.TrimSpace(cur.owner.resolve(cur.dep.Version))
	if v == "" {
		v = strings.TrimSpace(cur.owner.resolve(cur.owner.managedVersion(cur.dep)))
	}
	if hasUnresolvedProperty(v) {
		// A version that is still a property reference is not a version. Left
		// as written it would become part of a purl that matches nothing while
		// reading as a stated identity; empty at least says "not known".
		return ""
	}
	return v
}

// pomFor reads and parent-resolves one artifact's POM, caching the result.
//
// The artifact's own parent chain is resolved before its dependencies are read,
// for exactly the reason inherit.go exists: a Spring Boot library declares its
// dependencies without versions and inherits them from a BOM, so reading its
// POM alone yields a subtree of versionless artifacts.
func (w *closureWalker) pomFor(node *closureNode) *pomProject {
	if node.pkg.Version == "" {
		// Without a version there is no artifact to look up. This is a gap, and
		// caching it keeps the walk from asking again for every path that
		// reaches the same unversioned coordinate.
		w.poms[node.key] = nil
		return nil
	}
	full := node.key + ":" + node.pkg.Version
	if cached, ok := w.poms[full]; ok {
		return cached
	}
	if w.read >= maxClosurePoms {
		w.poms[full] = nil
		return nil
	}
	g, a := splitCoord(node.key)
	data, ok := w.resolver.ResolvePom(g, a, node.pkg.Version)
	if !ok || len(data) == 0 {
		w.poms[full] = nil
		return nil
	}
	w.read++
	parsed, err := parsePomXml(bytesReader(data))
	if err != nil {
		w.poms[full] = nil
		return nil
	}
	parsed.inherit(w.resolver)
	w.poms[full] = parsed
	return parsed
}

// transitiveScope applies Maven's transitive-scope table: the scope a dependency
// is reached under, given the scope of the artifact that declares it.
//
//	                      declared scope
//	reached as   compile   provided   runtime   test
//	compile      compile   -          runtime   -
//	provided     provided  -          provided  -
//	runtime      runtime   -          runtime   -
//	test         test      -          test      -
//
// The two empty columns are the load-bearing part. A `provided` dependency is
// supplied by the container and a `test` dependency exists only for the test
// run; NEITHER is passed on to a consumer, so an artifact's test dependencies
// are not in your application and must not be reported as if they were. Walking
// them anyway is how a naive closure reports several times the artifacts a
// project actually ships.
func transitiveScope(parent, child string) (string, bool) {
	switch child {
	case "provided", "test", "system":
		return "", false
	}
	switch parent {
	case "compile":
		if child == "runtime" {
			return "runtime", true
		}
		return "compile", true
	case "runtime":
		return "runtime", true
	case "provided":
		return "provided", true
	case "test":
		return "test", true
	}
	return "", false
}

// scopeStrength orders the scopes by how much of the classpath they put an
// artifact on. compile is everywhere; runtime is the runtime and test
// classpaths; provided is compile and test but not the shipped artifact; test
// is the test classpath alone.
func scopeStrength(scope string) int {
	switch scope {
	case "compile":
		return 4
	case "runtime":
		return 3
	case "provided":
		return 2
	case "test":
		return 1
	}
	return 0
}

func scopeStronger(a, b string) bool { return scopeStrength(a) > scopeStrength(b) }

// packageScope maps a Maven scope to the SBOM's prod/dev distinction.
//
// Only `test` is dev. `provided` is deliberately production: the artifact is
// absent from the built package but present at runtime, supplied by the
// container -- a servlet API a deployed application calls on every request is
// not a development tool, and marking it dev tells a consumer to care less about
// a vulnerability that is live in their deployment.
func packageScope(scope string) string {
	if scope == "test" {
		return languages.PackageScopeDev
	}
	return languages.PackageScopeProd
}

// scopeOrDefault fills in the scope Maven assumes when a dependency states none.
func scopeOrDefault(scope string) string {
	if s := strings.TrimSpace(scope); s != "" {
		return s
	}
	return "compile"
}

// excluded reports whether an artifact is excluded by any exclusion accumulated
// along the path to it.
//
// Maven accepts `*` as a wildcard in either coordinate, which is how a project
// says "nothing from this subtree" or "no artifact from this group".
func excluded(exclusions []pomExclusion, groupID, artifactID string) bool {
	for _, e := range exclusions {
		g := strings.TrimSpace(e.GroupId)
		a := strings.TrimSpace(e.ArtifactId)
		if (g == "*" || g == groupID) && (a == "*" || a == artifactID) {
			return true
		}
	}
	return false
}

// isOptional reports whether a dependency is declared <optional>true</optional>.
//
// Read through the same property resolution as every other field: a POM may
// state it as ${some.flag}, and an unresolved one is not optional.
func (p *pomProject) isOptional(dep pomDependency) bool {
	return strings.EqualFold(strings.TrimSpace(p.resolve(dep.Optional)), "true")
}

func splitCoord(key string) (groupID, artifactID string) {
	if i := strings.LastIndex(key, ":"); i >= 0 {
		return key[:i], key[i+1:]
	}
	return key, ""
}

func dedupe(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
