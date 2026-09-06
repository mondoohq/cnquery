// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/providers-sdk/v1/mqlr/lrcore"
)

// Every provider you can connect to has to declare an asset root.
//
// A root is the type that bounds a query on a connected asset (ADR 031): it is
// what `_` resolves to, what carries `asset`, and what lets a bundle derive
// which assets it applies to instead of being told by hand. A provider that
// declares connectors hands out assets, so it owes a root. A provider that
// declares none - core, which is always builtin and only supplies globals - has
// nothing to be the root of, so it is exempt by construction rather than by
// being named here.
//
// rootBacklog is the providers that predate the rule. It is a ratchet in both
// directions: a connectable provider that is neither rooted nor listed fails,
// and a *listed* provider that has since been rooted also fails. The second
// half is the point - an allowlist nobody has to maintain stops describing
// reality, and then it is worse than no test at all.
var rootBacklog = map[string]struct{}{}

func init() {
	for _, p := range []string{
		"activedirectory", "alicloud", "ansible", "artifactory",
		"atlassian", "aws", "azure", "bicep", "bitwarden", "cassandra",
		"claude", "clickhousecloud", "clickhousedb", "cloudformation",
		"datadog", "depsdev", "elasticsearch", "gcp", "google-workspace", "hcp", "helm",
		"hetzner", "huggingface", "ipinfo", "ipmi", "iru", "kustomize", "mikrotik", "mistral", "mondoo", "ms365", "network", "nutanix", "oci", "ollama", "opcua",
		"opensearch", "openstack", "proxmox", "redfish", "redisdb", "stackit", "terraform", "vsphere",
	} {
		rootBacklog[p] = struct{}{}
	}
}

var (
	reConfigRoot   = regexp.MustCompile(`\bRoot:\s*"([^"]*)"`)
	reHasConnector = regexp.MustCompile(`Connectors:\s*\[\]plugin\.Connector\{\s*\{`)
)

// providerRootState is what a provider says about its roots, read the way the
// build reads it: the schema through the real parser, the plugin declaration
// out of its config.
type providerRootState struct {
	connectable bool
	roots       []string
	declared    string // `option root` in the schema
	configRoot  string // `Root:` in config.go
}

func readProviderRootState(t *testing.T, dir string) (providerRootState, bool) {
	t.Helper()
	var st providerRootState

	cfg, err := os.ReadFile(filepath.Join(dir, "config", "config.go"))
	if err != nil {
		return st, false // not a provider
	}
	st.connectable = reHasConnector.Match(cfg)
	if m := reConfigRoot.FindSubmatch(cfg); m != nil {
		st.configRoot = string(m[1])
	}

	lrs, _ := filepath.Glob(filepath.Join(dir, "resources", "*.lr"))
	for _, lr := range lrs {
		raw, err := os.ReadFile(lr)
		require.NoError(t, err)
		// Parse, not Resolve: this only needs what the file itself declares,
		// and Parse does not have to chase imports across the repo to say it.
		ast, err := lrcore.Parse(string(raw))
		require.NoErrorf(t, err, "parsing %s", lr)
		if root, ok := ast.Options["root"]; ok && root != "" {
			st.declared = root
		}
		for _, r := range ast.Resources {
			if r.IsRoot {
				st.roots = append(st.roots, r.ID)
			}
		}
	}
	sort.Strings(st.roots)
	return st, true
}

func TestEveryConnectableProviderDeclaresARoot(t *testing.T) {
	dirs, err := filepath.Glob(filepath.Join("..", "providers", "*"))
	require.NoError(t, err)

	var missing, rootedButBacklogged []string
	seen := 0

	for _, dir := range dirs {
		name := filepath.Base(dir)
		st, ok := readProviderRootState(t, dir)
		if !ok {
			continue
		}
		seen++

		// A provider with no connectors is not something you connect to, so
		// there is no asset for it to root.
		if !st.connectable {
			assert.Emptyf(t, st.roots, "%s declares no connectors but declares roots %v", name, st.roots)
			continue
		}

		_, backlogged := rootBacklog[name]
		switch {
		case len(st.roots) == 0 && !backlogged:
			missing = append(missing, name)
		case len(st.roots) > 0 && backlogged:
			rootedButBacklogged = append(rootedButBacklogged, name)
		}
	}

	require.Greater(t, seen, 50, "the provider scan found almost nothing, so it is measuring the wrong directory")

	assert.Emptyf(t, missing, "these providers accept connections but declare no asset root (ADR 031). "+
		"Mark the resource a connection binds to with `@root`, declare `option root`, and set `Root:` in config.go: %v", missing)

	assert.Emptyf(t, rootedButBacklogged, "these providers are rooted now and must come off rootBacklog in this file: %v",
		rootedButBacklogged)
}

// The two places a provider names its root have to agree, and the name has to
// be one of the roots it actually declares. They are written in different files
// - the schema and the plugin config - so nothing but a check keeps them from
// drifting into naming different resources, which surfaces as `_` binding to
// something the connection never exposed.
func TestDeclaredRootsAgree(t *testing.T) {
	dirs, err := filepath.Glob(filepath.Join("..", "providers", "*"))
	require.NoError(t, err)

	for _, dir := range dirs {
		name := filepath.Base(dir)
		st, ok := readProviderRootState(t, dir)
		if !ok || len(st.roots) == 0 {
			continue
		}

		t.Run(name, func(t *testing.T) {
			require.NotEmpty(t, st.declared, "declares roots %v but no `option root`", st.roots)
			assert.Equalf(t, st.declared, st.configRoot,
				"`option root` in the schema and `Root:` in config.go disagree")
			assert.Containsf(t, st.roots, st.declared,
				"`option root` names %q, which is not one of the roots this provider declares", st.declared)
		})
	}
}
