// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoBareServiceEnabledGate pins the service-enabled contract.
//
// `serviceEnabled` is populated only by the gcp.project.<service>() accessor.
// The same resource is reachable without it -- addressed by its own type name
// (gcp.project.cloudRunService.services), through an alias (gcp.storage), or
// built by a resource init with CreateResource. On those paths the Go zero value
// is `false`, so a bare
//
//	if !g.serviceEnabled { return nil, nil }
//
// reports an authoritative "there is nothing here" for a service that is in fact
// enabled and populated. An empty list is the most dangerous wrong answer for a
// posture check: it passes vacuously instead of failing or erroring.
//
// Verified against a live project: gcp.project.cloudRunService.services returned
// [] while gcp.project.cloudRun.services returned the real service.
//
// Every gate must therefore resolve the flag first, via the isEnabled() helper
// (bigquery.go, storage.go, ...) or the serviceChecked companion flag (dns.go).
func TestNoBareServiceEnabledGate(t *testing.T) {
	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob package sources: %v", err)
	}

	fset := token.NewFileSet()
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}

		ast.Inspect(file, func(n ast.Node) bool {
			ifs, ok := n.(*ast.IfStmt)
			if !ok {
				return true
			}
			// match exactly `!<recv>.serviceEnabled`, with no other operand
			unary, ok := ifs.Cond.(*ast.UnaryExpr)
			if !ok || unary.Op != token.NOT {
				return true
			}
			sel, ok := unary.X.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "serviceEnabled" {
				return true
			}
			t.Errorf("%s: gates on serviceEnabled without resolving it first; "+
				"an unresolved flag is false and reports an empty collection for an "+
				"enabled service. Call isEnabled() (see bigquery.go) instead",
				fset.Position(ifs.Pos()))
			return true
		})
	}
}

// TestEveryServiceEnabledFlagHasAResolver checks the other direction: a resource
// that carries the flag must also carry the machinery to resolve it, so a new
// service cannot reintroduce the zero-value bug by omission.
func TestEveryServiceEnabledFlagHasAResolver(t *testing.T) {
	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob package sources: %v", err)
	}

	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		src := string(raw)
		if !strings.Contains(src, "serviceEnabled bool") {
			continue
		}
		// either the lazy resolver or the explicit "was it checked" companion
		if strings.Contains(src, "serviceOnce") || strings.Contains(src, "serviceChecked") {
			continue
		}
		t.Errorf("%s: declares serviceEnabled but has neither serviceOnce "+
			"(the isEnabled() resolver) nor serviceChecked; the flag defaults to "+
			"false and would report empty collections for an enabled service", path)
	}
}
