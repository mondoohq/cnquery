// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// TestInitGuardsProjectIdBeforeParentConstruction guards a whole class of
// scan-crashing panic.
//
// Most init functions resolve their resource by first building the parent
// service from args["projectId"]:
//
//	CreateResource(runtime, "gcp.project.pubsubService", map[string]*llx.RawData{
//	    "projectId": args["projectId"],
//	})
//
// A map lookup for an absent key yields a nil *llx.RawData, and the generated
// setter dereferences it unconditionally. So a partial query such as
// `gcp.project.pubsubService.topic(name: "x")` panics inside the provider
// instead of returning an error -- and because the executor runs blocks in
// goroutines, that panic is unrecoverable and takes down the whole scan.
//
// A nil-guard placed *after* the parent construction is dead code. This test
// walks the package's own source and fails when any init function reaches a
// CreateResource/NewResource carrying args["projectId"] without having checked
// it first.
func TestInitGuardsProjectIdBeforeParentConstruction(t *testing.T) {
	fset := token.NewFileSet()
	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob package sources: %v", err)
	}

	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		{
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Body == nil || !strings.HasPrefix(fn.Name.Name, "init") {
					continue
				}

				guardPos := token.NoPos
				violation := token.NoPos

				ast.Inspect(fn.Body, func(n ast.Node) bool {
					switch node := n.(type) {
					case *ast.BinaryExpr:
						// args["projectId"] == nil / != nil, either operand order.
						// Matching only the left-hand form would let a reversed
						// `nil == args["projectId"]` silently bypass the detector.
						guarded := (isArgsIndex(node.X, "projectId") && isNilIdent(node.Y)) ||
							(isArgsIndex(node.Y, "projectId") && isNilIdent(node.X))
						if guarded && !guardPos.IsValid() {
							guardPos = node.Pos()
						}
					case *ast.CallExpr:
						name := calleeName(node.Fun)
						if name != "CreateResource" && name != "NewResource" {
							return true
						}
						if !callCarriesProjectIdArg(node) {
							return true
						}
						if !guardPos.IsValid() || guardPos > node.Pos() {
							if !violation.IsValid() {
								violation = node.Pos()
							}
						}
					}
					return true
				})

				if violation.IsValid() {
					t.Errorf("%s: %s passes args[\"projectId\"] into %s before any nil-guard; "+
						"a partial query would panic the provider. Move the guard above the call.",
						fset.Position(violation), fn.Name.Name, "CreateResource/NewResource")
				}
			}
		}
	}
}

// isArgsIndex reports whether e is the expression args["<key>"].
func isArgsIndex(e ast.Expr, key string) bool {
	idx, ok := e.(*ast.IndexExpr)
	if !ok {
		return false
	}
	ident, ok := idx.X.(*ast.Ident)
	if !ok || ident.Name != "args" {
		return false
	}
	lit, ok := idx.Index.(*ast.BasicLit)
	return ok && lit.Kind == token.STRING && strings.Trim(lit.Value, `"`) == key
}

func isNilIdent(e ast.Expr) bool {
	ident, ok := e.(*ast.Ident)
	return ok && ident.Name == "nil"
}

func calleeName(e ast.Expr) string {
	switch f := e.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		return f.Sel.Name
	}
	return ""
}

// callCarriesProjectIdArg reports whether any argument of the call references
// args["projectId"], directly or inside a composite literal.
func callCarriesProjectIdArg(call *ast.CallExpr) bool {
	found := false
	for _, arg := range call.Args {
		ast.Inspect(arg, func(n ast.Node) bool {
			if isArgsIndex(exprOf(n), "projectId") {
				found = true
				return false
			}
			return true
		})
	}
	return found
}

func exprOf(n ast.Node) ast.Expr {
	e, _ := n.(ast.Expr)
	return e
}
