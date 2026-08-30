// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go/ast"
	"go/parser"
	gotoken "go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Account-level singletons: exactly one instance exists per connection, so the
// empty cache key is both correct and desirable -- it is what makes the
// account-wide fetch behind them happen once.
var cacheKeySingletons = map[string]bool{}

// TestEveryConstructionSiteHasACacheKey fails when a resource is built without
// anything to key it on.
//
// createXxx sets __id from an explicit "__id" arg and, only when that leaves it
// empty, falls back to the resource's id() method. A resource with neither ends
// up on the key "<name>\x00", which every instance of that resource then shares
// -- so the first one built is handed back for all of them, carrying the first
// one's data. Nothing errors; the wrong answer just looks like a right one.
//
// Sibling providers have shipped this repeatedly: a resource passing the
// parent-qualified value as a public `id` FIELD -- an ordinary declared field,
// which does NOT feed the cache key -- so every instance resolved to the first
// one built. It is what made every GKE cluster report the first cluster's
// network policy (#10506), and every AWS flow log report the first one
// (#10497).
//
// Only inline argument literals are checked. A site whose args come from a
// variable or a helper is skipped rather than guessed at, so this test reports
// no false failures -- it is a floor, not a full audit.
func TestEveryConstructionSiteHasACacheKey(t *testing.T) {
	fset := gotoken.NewFileSet()

	gen, err := os.ReadFile("oci.lr.go")
	if err != nil {
		t.Fatalf("read generated schema: %v", err)
	}
	genSrc := string(gen)

	// Resource* constants, so a site that names the resource by constant is
	// resolved the same as one that spells out the string.
	constResource := map[string]string{}
	reConst := regexp.MustCompile(`(?m)^\t(Resource[A-Za-z0-9_]+)\s+string = "([a-zA-Z0-9_.]+)"`)
	for _, m := range reConst.FindAllStringSubmatch(genSrc, -1) {
		constResource[m[1]] = m[2]
	}

	// A registered Init resolves the resource before createXxx is reached, and
	// the resource it returns carries its own key -- so an init-backed resource
	// is out of scope here. (An init that falls through on a lookup miss is a
	// separate defect, class 4, and not what this test is for.)
	registered := map[string]bool{}
	hasInit := map[string]bool{}
	reEntry := regexp.MustCompile(`(?m)^\t\t"([a-zA-Z0-9_.]+)": \{\n((?:\t\t\t.*\n)+?)\t\t\},`)
	for _, m := range reEntry.FindAllStringSubmatch(genSrc, -1) {
		registered[m[1]] = true
		hasInit[m[1]] = strings.Contains(m[2], "Init:")
	}

	// A resource has an id() method exactly when its generated constructor
	// falls back to it.
	genFile, err := parser.ParseFile(fset, "oci.lr.go", nil, 0)
	if err != nil {
		t.Fatalf("parse generated schema: %v", err)
	}
	hasIDMethod := map[string]bool{}
	for _, decl := range genFile.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || !strings.HasPrefix(fn.Name.Name, "create") {
			continue
		}
		var resource string
		callsID := false
		ast.Inspect(fn, func(n ast.Node) bool {
			if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == gotoken.STRING && resource == "" {
				if v, err := strconv.Unquote(lit.Value); err == nil && registered[v] {
					resource = v
				}
			}
			if sel, ok := n.(*ast.SelectorExpr); ok && sel.Sel.Name == "id" {
				callsID = true
			}
			return true
		})
		if resource != "" {
			hasIDMethod[resource] = callsID
		}
	}

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("list package files: %v", err)
	}

	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, ".lr.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			fn, ok := call.Fun.(*ast.Ident)
			if !ok || (fn.Name != "CreateResource" && fn.Name != "NewResource") || len(call.Args) < 3 {
				return true
			}

			var resource string
			switch arg := call.Args[1].(type) {
			case *ast.BasicLit:
				if arg.Kind != gotoken.STRING {
					return true
				}
				resource, _ = strconv.Unquote(arg.Value)
			case *ast.Ident:
				resource = constResource[arg.Name]
			}
			if resource == "" || !registered[resource] {
				return true
			}
			if hasIDMethod[resource] || hasInit[resource] || cacheKeySingletons[resource] {
				return true
			}

			// Only inline literals can be judged; anything else is skipped.
			lit, ok := call.Args[2].(*ast.CompositeLit)
			if !ok {
				return true
			}
			for _, elt := range lit.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := kv.Key.(*ast.BasicLit)
				if !ok {
					continue
				}
				if name, _ := strconv.Unquote(key.Value); name == "__id" {
					return true
				}
			}

			pos := fset.Position(call.Pos())
			t.Errorf("%s:%d: %s(%q) passes no \"__id\" and %s has no id() method, "+
				"so every instance shares the cache key %q and resolves to the first one built. "+
				"Pass an explicit \"__id\", or give the resource an id() method.",
				path, pos.Line, fn.Name, resource, resource, resource+"\x00")
			return true
		})
	}
}

// TestInitBackedResourcesCanSupplyACacheKey covers the case the test above
// deliberately skips.
//
// A registered Init is normally what supplies a resource's identity: it either
// puts "__id" into args, or it returns an already-built resource that carries
// its own key. An init that does neither, on a resource with no id() method,
// leaves __id empty just as surely as having no init at all -- and it does so
// invisibly, because the presence of an Init reads as "identity handled".
//
// The azure provider shipped exactly that shape: an init that only filled in
// subscriptionId, on a resource with no id(), so every subscription aliased to
// the first. This provider is clean today; the test keeps it that way.
func TestInitBackedResourcesCanSupplyACacheKey(t *testing.T) {
	fset := gotoken.NewFileSet()

	gen, err := os.ReadFile("oci.lr.go")
	if err != nil {
		t.Fatalf("read generated schema: %v", err)
	}
	genSrc := string(gen)

	reEntry := regexp.MustCompile(`(?m)^\t\t"([a-zA-Z0-9_.]+)": \{\n((?:\t\t\t.*\n)+?)\t\t\},`)
	reInitName := regexp.MustCompile(`Init:\s+(init[A-Za-z0-9_]+)`)
	initFuncs := map[string]bool{}
	resourceOfInit := map[string]string{}
	for _, m := range reEntry.FindAllStringSubmatch(genSrc, -1) {
		if n := reInitName.FindStringSubmatch(m[2]); n != nil {
			initFuncs[n[1]] = true
			resourceOfInit[n[1]] = m[1]
		}
	}

	genFile, err := parser.ParseFile(fset, "oci.lr.go", nil, 0)
	if err != nil {
		t.Fatalf("parse generated schema: %v", err)
	}
	hasIDMethod := map[string]bool{}
	for _, decl := range genFile.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || !strings.HasPrefix(fn.Name.Name, "create") {
			continue
		}
		var resource string
		callsID := false
		ast.Inspect(fn, func(n ast.Node) bool {
			if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == gotoken.STRING && resource == "" {
				if v, err := strconv.Unquote(lit.Value); err == nil && strings.Contains(v, ".") {
					resource = v
				}
			}
			if sel, ok := n.(*ast.SelectorExpr); ok && sel.Sel.Name == "id" {
				callsID = true
			}
			return true
		})
		if resource != "" {
			hasIDMethod[resource] = callsID
		}
	}

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("list package files: %v", err)
	}

	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, ".lr.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !initFuncs[fn.Name.Name] {
				continue
			}
			resource := resourceOfInit[fn.Name.Name]
			if resource == "" || hasIDMethod[resource] || cacheKeySingletons[resource] {
				continue
			}

			// An init that just delegates (`return initFromServiceList(...)`)
			// is out of scope: the key is supplied by the helper, which this
			// body-only walk cannot see. Skipping them keeps the test free of
			// false failures at the cost of not covering the delegated shape.
			if len(fn.Body.List) == 1 {
				if ret, ok := fn.Body.List[0].(*ast.ReturnStmt); ok && len(ret.Results) == 1 {
					if _, ok := ret.Results[0].(*ast.CallExpr); ok {
						continue
					}
				}
			}

			suppliesKey := false
			ast.Inspect(fn, func(n ast.Node) bool {
				if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == gotoken.STRING {
					if v, err := strconv.Unquote(lit.Value); err == nil && v == "__id" {
						suppliesKey = true
					}
				}
				// An init that returns an already-built resource hands back one
				// that already carries its own key.
				if ret, ok := n.(*ast.ReturnStmt); ok && len(ret.Results) == 3 {
					if id, ok := ret.Results[1].(*ast.Ident); !ok || id.Name != "nil" {
						suppliesKey = true
					}
				}
				return true
			})
			if suppliesKey {
				continue
			}

			pos := fset.Position(fn.Pos())
			t.Errorf("%s:%d: %s never supplies a cache key -- it sets no \"__id\" and returns no resource -- "+
				"and %s has no id() method, so every instance shares the key %q and resolves to the first one built. "+
				"Give the resource an id() method, or have the init set \"__id\".",
				path, pos.Line, fn.Name.Name, resource, resource+"\x00")
		}
	}
}
