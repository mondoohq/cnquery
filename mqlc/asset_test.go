// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package mqlc_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/mqlc"
	"go.mondoo.com/mql/providers-sdk/v1/testutils"
	"go.mondoo.com/mql/types"
)

// The mockprovider carries both asset-root cases: `muser.running` is typed
// `asset<mgroup>`, whose root this schema defines, and `muser.runningUnknown` is
// typed `asset<mcp>`, whose root no loaded schema defines.
var assetConf = mqlc.NewConfig(
	core_schema.Add(testutils.MustLoadSchema(testutils.SchemaProvider{Provider: "mockprovider"})),
	mql.Features{byte(mql.ResourceContext)},
)

// chunkShape renders one block's chunks as `id(type)`, which is what the asset
// tests are about: whether the deref chunk is there, how many there are, and
// what the chain is typed as.
func chunkShape(t *testing.T, block *llx.Block) []string {
	t.Helper()
	res := make([]string, len(block.Chunks))
	for i, c := range block.Chunks {
		res[i] = c.Id + "(" + c.Type().Label() + ")"
	}
	return res
}

func TestAssetChain(t *testing.T) {
	t.Run("the reference itself needs no deref", func(t *testing.T) {
		res, err := mqlc.Compile("muser.running", nil, assetConf)
		require.NoError(t, err)
		assert.Equal(t, []string{"muser(muser)", "running(asset<mgroup>)"},
			chunkShape(t, res.CodeV2.Blocks[0]))
	})

	// The nil comparisons are the only thing MQL could do with an asset before
	// this change, and they are registered on the asset type itself. Chaining is
	// tried after them, so a root field named `==` could never shadow one.
	t.Run("nil comparisons stay on the asset", func(t *testing.T) {
		for _, q := range []string{"muser.running == null", "muser.running != null"} {
			res, err := mqlc.Compile(q, nil, assetConf)
			require.NoError(t, err, q)
			shape := chunkShape(t, res.CodeV2.Blocks[0])
			assert.Len(t, shape, 3, q)
			assert.NotContains(t, shape, llx.AssetRootChunkID+"(mgroup)", q)
		}
	})

	t.Run("a field derefs into the root, then reads it as a resource", func(t *testing.T) {
		res, err := mqlc.Compile("muser.running.name", nil, assetConf)
		require.NoError(t, err)
		assert.Equal(t, []string{
			"muser(muser)", "running(asset<mgroup>)",
			llx.AssetRootChunkID + "(mgroup)", "name(string)",
		}, chunkShape(t, res.CodeV2.Blocks[0]))

		// The deref is not a step the user wrote, so it must not show up in the
		// label the result is reported under.
		labels := []string{}
		for _, l := range res.Labels.GetLabels() {
			labels = append(labels, l)
		}
		assert.Contains(t, labels, "muser.running.name")
	})

	// One deref for the whole block, not one per field: each deref resolves the
	// referenced asset, so a block of five fields would otherwise resolve it
	// five times.
	t.Run("a block derefs once", func(t *testing.T) {
		res, err := mqlc.Compile("muser.running { name }", nil, assetConf)
		require.NoError(t, err)

		derefs := 0
		for _, block := range res.CodeV2.Blocks {
			for _, c := range block.Chunks {
				if c.Id == llx.AssetRootChunkID {
					derefs++
					assert.Equal(t, string(types.Resource("mgroup")), c.Function.Type)
				}
			}
		}
		assert.Equal(t, 1, derefs)
	})

	t.Run("unknown field of a known root", func(t *testing.T) {
		_, err := mqlc.Compile("muser.running.nope", nil, assetConf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot find field 'nope'")
	})

	// The fields reachable through a reference are the root's fields, so they
	// are what gets offered - for the error above, and for shell completion,
	// which reads the same list.
	t.Run("suggestions come from the root", func(t *testing.T) {
		res, err := mqlc.Compile("muser.running.nam", nil, assetConf)
		require.Error(t, err)
		suggestions := []string{}
		for _, s := range res.GetSuggestions() {
			suggestions = append(suggestions, s.Field)
		}
		assert.Equal(t, []string{"name"}, suggestions)
	})

	// Nothing can be offered for a root this compile cannot see, and guessing
	// from the asset's own type would offer fields of the wrong resource.
	t.Run("no suggestions for an unloaded root", func(t *testing.T) {
		res, err := mqlc.Compile("muser.runningUnknown.nam", nil, assetConf)
		require.NoError(t, err, "an unchecked member compiles, it is not a suggestion case")
		assert.Empty(t, res.GetSuggestions())
	})
}

// A root whose schema is not loaded here is the compile-here / run-there case:
// the compiler knows the root by name only, so the member cannot be checked and
// its type is left to the executing runtime, which does have the schema. See
// ADR 031 decision 2.
func TestAssetChainDegradesWithoutTheRootSchema(t *testing.T) {
	t.Run("a single field defers its type to runtime", func(t *testing.T) {
		res, err := mqlc.Compile("muser.runningUnknown.tools", nil, assetConf)
		require.NoError(t, err)
		assert.Equal(t, []string{
			"muser(muser)", "runningUnknown(asset<mcp>)",
			llx.AssetRootChunkID + "(mcp)", "tools(any)",
		}, chunkShape(t, res.CodeV2.Blocks[0]))
	})

	// A block needs a real type to compile against, so it cannot degrade the way
	// a single field can. The error has to say why rather than report a missing
	// resource, which reads like a typo.
	t.Run("a block cannot", func(t *testing.T) {
		_, err := mqlc.Compile("muser.runningUnknown { tools }", nil, assetConf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "asset root 'mcp'")
		assert.Contains(t, err.Error(), "not loaded")
	})

	// Arguments would have to be checked against an init signature this compile
	// cannot see, so they are refused rather than passed along unchecked.
	t.Run("arguments are refused", func(t *testing.T) {
		_, err := mqlc.Compile(`muser.runningUnknown.tools("x")`, nil, assetConf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not loaded")
	})
}

// `_` at the top level is the connected asset's root resource (ADR 031
// decision 6). The root comes from the provider serving the connection, so the
// compiler is handed a name rather than discovering one.
func TestTopLevelUnderscore(t *testing.T) {
	rooted := assetConf
	rooted.AssetRoot = "muser"

	t.Run("resolves to the root resource", func(t *testing.T) {
		res, err := mqlc.Compile("_", nil, rooted)
		require.NoError(t, err)
		assert.Equal(t, []string{"muser(muser)"}, chunkShape(t, res.CodeV2.Blocks[0]))
	})

	t.Run("fields and blocks read off it", func(t *testing.T) {
		res, err := mqlc.Compile("_.name", nil, rooted)
		require.NoError(t, err)
		assert.Equal(t, []string{"muser(muser)", "name(string)"}, chunkShape(t, res.CodeV2.Blocks[0]))

		_, err = mqlc.Compile("_ { name }", nil, rooted)
		require.NoError(t, err)
	})

	// Nothing is guessed when the connection declares no root: answering with a
	// resource that only partly covers the asset would be worse than failing,
	// and this is the state every provider starts in.
	t.Run("without a declared root", func(t *testing.T) {
		_, err := mqlc.Compile("_", nil, assetConf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "declares no root resource")
	})

	// A bare `_` inside a block emits no chunk of its own - it is the binding -
	// so the operand's value is the binding's ref. The top-level case had to
	// stop reading that binding unconditionally, which is a nil dereference
	// once `_` compiles without one.
	t.Run("inside a block it is still the binding", func(t *testing.T) {
		for _, q := range []string{"muser { _ }", "muser { _.name }", "muser.groups { _ }"} {
			_, err := mqlc.Compile(q, nil, rooted)
			assert.NoError(t, err, q)
		}
	})
}

// The os provider declares `os.any` as its root: the union of the OS roots,
// with the universal surface arriving through the embedded `os.base` and the
// family-specific surface through aliases. This pins the chain end to end -
// alias or embed -> implicit field on the root -> compiled field read - for
// members from every root, including the ones that take arguments.
func TestOsRootAttachments(t *testing.T) {
	conf := mqlc.NewConfig(core_schema.Add(os_schema), mql.Features{byte(mql.ResourceContext)})
	conf.AssetRoot = "os.any"

	for _, q := range []string{
		"_",
		"_.hostname",
		"_.packages.list.length",
		"_.users.list { name }",
		"_.services.where(running).length",
		"_.kernel.info",
		"_.sshd.config.params",
		// a resource that takes arguments cannot be a plain field, which is why
		// these are aliases rather than accessors on the root
		`_.file("/etc/hostname").exists`,
		`_.command("echo hi").stdout`,
	} {
		t.Run(q, func(t *testing.T) {
			_, err := mqlc.Compile(q, nil, conf)
			assert.NoError(t, err)
		})
	}

	// The global names are untouched: an alias adds a path, it does not move
	// the resource.
	for _, q := range []string{"packages.list.length", "sshd.config.params", `file("/etc/hostname").exists`} {
		t.Run(q, func(t *testing.T) {
			_, err := mqlc.Compile(q, nil, conf)
			assert.NoError(t, err)
		})
	}
}

// A field missing from a platform root is a platform mismatch, not version
// skew, so the diagnostic has to say which root does carry it instead of
// suggesting a provider upgrade that cannot help. See ADR 031.
func TestMissingFieldOnAPlatformRoot(t *testing.T) {
	conf := mqlc.NewConfig(core_schema.Add(os_schema), mql.Features{byte(mql.ResourceContext)})
	conf.AssetRoot = "os.linux"
	conf.DeclaredAssetRoot = "os.any"

	t.Run("names the root that has it", func(t *testing.T) {
		_, err := mqlc.Compile("_.registrykey", nil, conf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rooted at os.linux")
		assert.Contains(t, err.Error(), "available on os.windows")
		// The union carries every platform's members, so offering it as the
		// place to find one answers nothing.
		assert.NotContains(t, err.Error(), "os.any")
		// And this is not a version problem.
		assert.NotContains(t, err.Error(), "may require a newer one")
	})

	// A name no root carries gets no platform hint - there is no platform to
	// point at - and falls through to the version-skew lead, which needs
	// provider versions in the schema to render (it does live, where the
	// coordinator supplies them).
	t.Run("no platform hint when no root has it", func(t *testing.T) {
		_, err := mqlc.Compile("_.nosuchthing", nil, conf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot find field 'nosuchthing' in os.linux")
		assert.NotContains(t, err.Error(), "available on")
	})

	t.Run("the platform's own members still resolve", func(t *testing.T) {
		for _, q := range []string{"_.iptables", "_.selinux.mode", "_.hostname", "_.packages.list.length"} {
			_, err := mqlc.Compile(q, nil, conf)
			assert.NoError(t, err, q)
		}
	})
}

// A query is a block on the asset root, so a bare member of the root resolves:
// `hostname` means `assetRoot.hostname` (ADR 031 point 7). Before this, only
// `_.hostname` worked and `hostname` failed outright.
func TestBareRootMembers(t *testing.T) {
	rooted := mqlc.NewConfig(core_schema.Add(os_schema), mql.Features{byte(mql.ResourceContext)})
	rooted.AssetRoot = "os.linux"
	unrooted := mqlc.NewConfig(core_schema.Add(os_schema), mql.Features{byte(mql.ResourceContext)})

	// Members of the root, reached through its embed chain (os.linux -> os.unix
	// -> os.base) and through aliases attached to it.
	for _, q := range []string{"hostname", "uptime", "machineid", "date.timezone", "env.length"} {
		t.Run(q, func(t *testing.T) {
			_, err := mqlc.Compile(q, nil, rooted)
			assert.NoError(t, err, "resolves as a member of the root")

			_, err = mqlc.Compile(q, nil, unrooted)
			assert.Error(t, err, "without a root there is nothing to resolve it against")
		})
	}

	// A name that is neither a global resource nor a root member fails the way
	// it always did - the fallback must not change what an unknown name says.
	t.Run("unknown name", func(t *testing.T) {
		_, err := mqlc.Compile("nosuchthing", nil, rooted)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot find resource for identifier 'nosuchthing'")
	})
}

// The v14 rule is global-first, root-fallback, and this is what it buys: every
// query that compiles today compiles to the *same code*. Root-first would
// recompile `packages.list.length` through the root, changing its checksum -
// which cnspec keys scoring continuity on, and which would bake one platform
// into a bundle it shares across assets. See ADR 031 point 7.
func TestRootedFallbackDoesNotChangeExistingBundles(t *testing.T) {
	rooted := mqlc.NewConfig(core_schema.Add(os_schema), mql.Features{byte(mql.ResourceContext)})
	rooted.AssetRoot = "os.linux"
	unrooted := mqlc.NewConfig(core_schema.Add(os_schema), mql.Features{byte(mql.ResourceContext)})

	for _, q := range []string{
		"packages.list.length",
		"users.list { name }",
		"sshd.config.params",
		`file("/etc/hostname").exists`,
		"asset.platform",
		"services.where(running).length",
	} {
		t.Run(q, func(t *testing.T) {
			with, err := mqlc.Compile(q, nil, rooted)
			require.NoError(t, err)
			without, err := mqlc.Compile(q, nil, unrooted)
			require.NoError(t, err)

			assert.Equal(t, without.CodeV2.Id, with.CodeV2.Id,
				"a root must not change the compilation of a name the global namespace already answers")
		})
	}
}
