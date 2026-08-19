// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePolicyPaths(t *testing.T) {
	t.Run("parses paths and capabilities", func(t *testing.T) {
		rules := `
path "secret/data/app/*" {
  capabilities = ["read", "list"]
}

path "sys/mounts" {
  capabilities = ["read"]
}
`
		parsed := parsePolicyPaths(rules)
		require.Len(t, parsed, 2)

		// sorted by path
		assert.Equal(t, "secret/data/app/*", parsed[0].Path)
		assert.Equal(t, []string{"list", "read"}, parsed[0].Capabilities)
		assert.Equal(t, "sys/mounts", parsed[1].Path)
		assert.Equal(t, []string{"read"}, parsed[1].Capabilities)
	})

	t.Run("normalizes capability casing and spacing", func(t *testing.T) {
		parsed := parsePolicyPaths(`path "a/b" { capabilities = [" READ ", "Sudo"] }`)
		require.Len(t, parsed, 1)
		assert.Equal(t, []string{"read", "sudo"}, parsed[0].Capabilities)
	})

	t.Run("empty policy yields no rules", func(t *testing.T) {
		assert.Empty(t, parsePolicyPaths(""))
		assert.Empty(t, parsePolicyPaths("   \n\t "))
	})

	t.Run("unparseable policy yields no rules rather than panicking", func(t *testing.T) {
		assert.Empty(t, parsePolicyPaths(`path "unterminated {{{`))
	})

	t.Run("path with no capabilities is kept", func(t *testing.T) {
		parsed := parsePolicyPaths(`path "a/b" {}`)
		require.Len(t, parsed, 1)
		assert.Equal(t, "a/b", parsed[0].Path)
		assert.Empty(t, parsed[0].Capabilities)
	})
}

func TestGrantsSudo(t *testing.T) {
	tests := []struct {
		name  string
		rules string
		want  bool
	}{
		{"sudo granted", `path "sys/*" { capabilities = ["sudo"] }`, true},
		{"sudo among others", `path "a" { capabilities = ["read", "sudo"] }`, true},
		{"no sudo", `path "a" { capabilities = ["read", "list"] }`, false},
		{"empty policy", ``, false},
		// "sudo" must match the capability, not appear anywhere in the text
		{"sudo only in path name", `path "pseudo/x" { capabilities = ["read"] }`, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, grantsCapability(parsePolicyPaths(tc.rules), "sudo"))
		})
	}
}

func TestGrantsRootPath(t *testing.T) {
	tests := []struct {
		name  string
		rules string
		want  bool
	}{
		{"bare star", `path "*" { capabilities = ["read"] }`, true},
		{"leading slash star", `path "/*" { capabilities = ["read"] }`, true},
		{"scoped wildcard is not root", `path "secret/*" { capabilities = ["read"] }`, false},
		{"literal path", `path "secret/data" { capabilities = ["read"] }`, false},
		// a deny on everything restricts rather than grants
		{"deny only does not count", `path "*" { capabilities = ["deny"] }`, false},
		{"deny alongside read counts", `path "*" { capabilities = ["deny", "read"] }`, true},
		{"empty policy", ``, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, grantsRootPath(parsePolicyPaths(tc.rules)))
		})
	}
}

func TestGrantsWildcardPath(t *testing.T) {
	tests := []struct {
		name  string
		rules string
		want  bool
	}{
		{"trailing star", `path "secret/*" { capabilities = ["read"] }`, true},
		{"segment matcher", `path "secret/+/config" { capabilities = ["read"] }`, true},
		{"root star", `path "*" { capabilities = ["read"] }`, true},
		{"literal path", `path "secret/data/app" { capabilities = ["read"] }`, false},
		// a plus inside a segment is a literal character, not a matcher
		{"plus inside segment is literal", `path "secret/a+b" { capabilities = ["read"] }`, false},
		{"deny only does not count", `path "secret/*" { capabilities = ["deny"] }`, false},
		{"empty policy", ``, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, grantsWildcardPath(parsePolicyPaths(tc.rules)))
		})
	}
}

func TestIsBuiltinRootPolicy(t *testing.T) {
	// Vault's root policy carries an empty document, so reading its grants off
	// the text would report the most privileged policy on the server as
	// granting nothing
	assert.True(t, isBuiltinRootPolicy("root"))
	assert.False(t, isBuiltinRootPolicy("lab-admin"))
	assert.False(t, isBuiltinRootPolicy("root-ish"))
	assert.False(t, isBuiltinRootPolicy(""))

	// the empty root document itself parses to no rules, which is why the
	// name has to carry the answer
	assert.Empty(t, parsePolicyPaths(""))
	assert.False(t, grantsCapability(parsePolicyPaths(""), "sudo"))
}

func TestIsDenyOnly(t *testing.T) {
	// a rule with no capabilities grants nothing, but it is not a deny either;
	// treating it as deny would suppress a wildcard path that should be flagged
	assert.False(t, isDenyOnly(policyRule{Path: "a", Capabilities: nil}))
	assert.True(t, isDenyOnly(policyRule{Path: "a", Capabilities: []string{"deny"}}))
	assert.False(t, isDenyOnly(policyRule{Path: "a", Capabilities: []string{"deny", "read"}}))
}
