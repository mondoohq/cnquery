// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyReference(t *testing.T) {
	tests := []struct {
		parts      []string
		wantNil    bool
		wantKind   string
		wantType   string
		wantName   string
		wantTarget string
	}{
		{parts: []string{"var", "region"}, wantKind: "var", wantName: "region", wantTarget: "var.region"},
		{parts: []string{"local", "subnet"}, wantKind: "local", wantName: "subnet", wantTarget: "local.subnet"},
		{parts: []string{"module", "network", "vpc_id"}, wantKind: "module", wantName: "network", wantTarget: "module.network"},
		{parts: []string{"data", "aws_ami", "ubuntu", "id"}, wantKind: "data", wantType: "aws_ami", wantName: "ubuntu", wantTarget: "data.aws_ami.ubuntu"},
		{parts: []string{"aws_vpc", "main", "id"}, wantKind: "resource", wantType: "aws_vpc", wantName: "main", wantTarget: "aws_vpc.main"},
		{parts: []string{"path", "module"}, wantKind: "path", wantTarget: "path.module"},
		{parts: []string{"each", "value"}, wantKind: "each", wantTarget: "each.value"},
		{parts: []string{"count", "index"}, wantKind: "count", wantTarget: "count.index"},
		{parts: []string{"terraform", "workspace"}, wantKind: "terraform", wantTarget: "terraform.workspace"},
		// not referable
		{parts: []string{}, wantNil: true},
		{parts: []string{"var"}, wantNil: true},
		{parts: []string{"data", "aws_ami"}, wantNil: true},
		{parts: []string{"lonely"}, wantNil: true},
	}

	for _, tc := range tests {
		got := classifyReference(tc.parts)
		if tc.wantNil {
			assert.Nil(t, got, "parts %v should not be referable", tc.parts)
			continue
		}
		require.NotNil(t, got, "parts %v should be referable", tc.parts)
		assert.Equal(t, tc.wantKind, got.kind, "kind for %v", tc.parts)
		assert.Equal(t, tc.wantType, got.typ, "type for %v", tc.parts)
		assert.Equal(t, tc.wantName, got.name, "name for %v", tc.parts)
		assert.Equal(t, tc.wantTarget, got.target, "target for %v", tc.parts)
	}
}

// classifyAttr parses a single HCL attribute and returns the classified
// references its expression makes, exercising the full
// Variables -> traversalParts -> classifyReference path.
func classifyAttr(t *testing.T, src string) []*tfRef {
	t.Helper()
	f, diags := hclsyntax.ParseConfig([]byte(src), "test.tf", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors(), diags.Error())
	attrs, diags := f.Body.JustAttributes()
	require.False(t, diags.HasErrors(), diags.Error())

	var out []*tfRef
	for _, attr := range attrs {
		for _, tr := range attr.Expr.Variables() {
			if ref := classifyReference(traversalParts(tr)); ref != nil {
				out = append(out, ref)
			}
		}
	}
	return out
}

func targets(refs []*tfRef) map[string]string {
	m := map[string]string{}
	for _, r := range refs {
		m[r.target] = r.kind
	}
	return m
}

func TestReferenceClassificationFromHCL(t *testing.T) {
	// A conditional pulling from a var and a data source.
	refs := classifyAttr(t, `ami = var.dr_mode ? var.dr_ami : data.aws_ami.shared.id`)
	got := targets(refs)
	assert.Equal(t, "var", got["var.dr_mode"])
	assert.Equal(t, "var", got["var.dr_ami"])
	assert.Equal(t, "data", got["data.aws_ami.shared"])

	// A splat over a managed resource.
	refs = classifyAttr(t, `ids = aws_instance.web[*].id`)
	got = targets(refs)
	assert.Equal(t, "resource", got["aws_instance.web"])

	// depends_on style list of addresses.
	refs = classifyAttr(t, `depends_on = [aws_iam_role.exec, module.network, data.aws_vpc.main]`)
	got = targets(refs)
	assert.Equal(t, "resource", got["aws_iam_role.exec"])
	assert.Equal(t, "module", got["module.network"])
	assert.Equal(t, "data", got["data.aws_vpc.main"])
}
