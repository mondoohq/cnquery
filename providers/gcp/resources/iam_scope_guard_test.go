// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// TestIamServiceUsesFullCloudPlatformScope pins the OAuth scope every
// iam.googleapis.com call in this package is built with.
//
// Most Google APIs accept https://www.googleapis.com/auth/cloud-platform.read-only,
// and this provider reaches for it by default because it is a read-only tool.
// The IAM API does not: its reference lists cloud-platform as the ONLY accepted
// scope for the workload-identity and workforce pool methods, and a read-only
// token is rejected with
//
//	Error 403: Request had insufficient authentication scopes.
//	reason = ACCESS_TOKEN_SCOPE_INSUFFICIENT
//
// The rejection happens in the token check, before any IAM permission is
// consulted, so granting roles does not fix it and the failure looks like a
// permission gap. Verified against a live project: the same list call returns
// the pools under cloud-platform and 403s under cloud-platform.read-only.
//
// gcp.project.iamService.workloadIdentityPools (and its providers) and
// gcp.organization.workforcePools (and its providers) all failed this way on
// every scan authenticated with a service-account key -- the entire federated
// trust surface, which is exactly the data a "who can assume an identity in
// this project from outside it" audit reads.
func TestIamServiceUsesFullCloudPlatformScope(t *testing.T) {
	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob package sources: %v", err)
	}

	fset := token.NewFileSet()
	checked := 0

	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, ".lr.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}

		ast.Inspect(file, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				return true
			}
			body := render(fset, fn.Body)
			if !strings.Contains(body, "iam.NewService(") {
				return true
			}
			checked++
			for _, line := range strings.Split(body, "\n") {
				if !strings.Contains(line, "conn.Client(") {
					continue
				}
				if strings.Contains(line, "CloudPlatformScope") {
					continue
				}
				t.Errorf("%s: %s builds an iam.googleapis.com service from %s\n"+
					"the IAM API accepts only cloud-platform; a read-only token is rejected with "+
					"ACCESS_TOKEN_SCOPE_INSUFFICIENT before permissions are checked",
					fset.Position(fn.Pos()), fn.Name.Name, strings.TrimSpace(line))
			}
			return true
		})
	}

	if checked == 0 {
		t.Fatal("no iam.NewService call sites found; this guard has stopped guarding anything")
	}
}

func render(fset *token.FileSet, n ast.Node) string {
	var b bytes.Buffer
	if err := printer.Fprint(&b, fset, n); err != nil {
		return ""
	}
	return b.String()
}
