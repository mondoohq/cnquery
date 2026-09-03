// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann
package lrcore

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parse(t *testing.T, cmd string) *LR {
	res, err := Parse(cmd)
	require.Nil(t, err)
	return res
}

func TestParse(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		res := parse(t, "")
		assert.Equal(t, &LR{}, res)
	})

	t.Run("empty resource", func(t *testing.T) {
		res := parse(t, "name")
		assert.Equal(t, []*Resource{{ID: "name"}}, res.Resources)
	})

	t.Run("empty resources", func(t *testing.T) {
		res := parse(t, "one tw2 thr33")
		assert.Equal(t, []*Resource{
			{ID: "one"},
			{ID: "tw2"},
			{ID: "thr33"},
		}, res.Resources)
	})

	t.Run("defaults", func(t *testing.T) {
		res := parse(t, "name @defaults(\"id group=group.name\")")
		assert.Equal(t, []*Resource{
			{
				ID:       "name",
				Defaults: "id group=group.name",
			},
		}, res.Resources)
	})

	t.Run("context", func(t *testing.T) {
		res := parse(t, "name @context(\"file.context\")")
		assert.Equal(t, []*Resource{
			{
				ID:      "name",
				Context: "file.context",
			},
		}, res.Resources)
	})

	t.Run("maturity on resource", func(t *testing.T) {
		res := parse(t, `name @maturity("experimental")`)
		assert.Equal(t, []*Resource{
			{
				ID:       "name",
				Maturity: "experimental",
			},
		}, res.Resources)
	})

	t.Run("maturity on resource with other annotations", func(t *testing.T) {
		res := parse(t, `name @defaults("id") @context("file.context") @maturity("preview")`)
		assert.Equal(t, []*Resource{
			{
				ID:       "name",
				Defaults: "id",
				Context:  "file.context",
				Maturity: "preview",
			},
		}, res.Resources)
	})

	t.Run("maturity on static field", func(t *testing.T) {
		res := parse(t, "name {\nfield @maturity(\"deprecated\") string\n}")
		f := []*Field{
			{
				BasicField: &BasicField{
					ID:       "field",
					Maturity: "deprecated",
					Type:     Type{SimpleType: &SimpleType{"string"}},
				},
			},
		}
		assert.Equal(t, "name", res.Resources[0].ID)
		assert.Equal(t, f, res.Resources[0].Body.Fields)
	})

	t.Run("maturity on computed field", func(t *testing.T) {
		res := parse(t, "name {\nfield(dep) @maturity(\"experimental\") string\n}")
		assert.Equal(t, "name", res.Resources[0].ID)
		bf := res.Resources[0].Body.Fields[0].BasicField
		assert.Equal(t, "field", bf.ID)
		assert.Equal(t, "experimental", bf.Maturity)
		assert.NotNil(t, bf.Args)
	})

	t.Run("maturity on resource and field", func(t *testing.T) {
		res := parse(t, `name @maturity("preview") {
			field @maturity("deprecated") string
		}`)
		assert.Equal(t, "preview", res.Resources[0].Maturity)
		assert.Equal(t, "deprecated", res.Resources[0].Body.Fields[0].BasicField.Maturity)
	})

	t.Run("resource with a static field", func(t *testing.T) {
		res := parse(t, `
		// resource-docs
		// with multiline
		name {
			// field docs...
			field type
		}
		`)
		assert.Equal(t, "name", res.Resources[0].ID)
		// Note: resource title/desc are empty because LR.Comments greedily
		// captures all leading comments when there's no option/import before
		// the first resource. In real .lr files, "option provider = ..." breaks
		// the sequence so resource comments are attributed correctly.

		require.Len(t, res.Resources[0].Body.Fields, 1)
		f := res.Resources[0].Body.Fields[0]
		assert.Equal(t, &BasicField{
			ID:   "field",
			Args: nil,
			Type: Type{SimpleType: &SimpleType{"type"}},
		}, f.BasicField)
		require.Len(t, f.Comments, 1)
		assert.Equal(t, "// field docs...", f.Comments[0].Text)
	})

	t.Run("section separator comments don't bleed into resource title", func(t *testing.T) {
		res := parse(t, `
option provider = "test"

// ============================================================
// Section Header
// ============================================================

// Actual resource description
name {
	field type
}
`)
		require.Len(t, res.Resources, 1)
		assert.Equal(t, "name", res.Resources[0].ID)
		assert.Equal(t, "Actual resource description", res.Resources[0].title)
		assert.Equal(t, "", res.Resources[0].desc)
	})

	t.Run("multiple resources with section separators", func(t *testing.T) {
		res := parse(t, `
option provider = "test"

// ============================================================
// Section A
// ============================================================

// First resource
first {
	val type
}

// ============================================================
// Section B
// ============================================================

// Second resource
//
// with extra detail
second {
	val type
}
`)
		require.Len(t, res.Resources, 2)
		assert.Equal(t, "First resource", res.Resources[0].title)
		assert.Equal(t, "", res.Resources[0].desc)
		assert.Equal(t, "Second resource", res.Resources[1].title)
		assert.Equal(t, "with extra detail", res.Resources[1].desc)
	})

	t.Run("rejects multi-line resource comment without blank separator", func(t *testing.T) {
		_, err := Parse(`
option provider = "test"

// Title that the author intended to wrap
// onto a second source line for readability
name {
	field type
}
`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "resource name")
		assert.Contains(t, err.Error(), "missing the required blank `//` separator")
	})

	t.Run("rejects multi-line field comment without blank separator", func(t *testing.T) {
		ast, err := Parse(`
option provider = "test"

// Resource title
name {
	// Budget type: COST, USAGE, RI_UTILIZATION,
	// SAVINGS_PLANS_UTILIZATION, or SAVINGS_PLANS_COVERAGE
	budgetType string
}
`)
		require.NoError(t, err)
		_, err = Schema(ast)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "field name.budgetType")
		assert.Contains(t, err.Error(), "missing the required blank `//` separator")
	})

	t.Run("accepts single-line field comment", func(t *testing.T) {
		ast := parse(t, `
option provider = "test"

// Resource title
name {
	// Just a title, no description
	field type
}
`)
		schema, err := Schema(ast)
		require.NoError(t, err)
		require.NotNil(t, schema.Resources["name"])
		f := schema.Resources["name"].Fields["field"]
		require.NotNil(t, f)
		assert.Equal(t, "Just a title, no description", f.Title)
		assert.Equal(t, "", f.Desc)
	})

	t.Run("rejects resource title over 150 characters", func(t *testing.T) {
		longTitle := strings.Repeat("x", 151)
		_, err := Parse(`
option provider = "test"

// ` + longTitle + `
name {
	field type
}
`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "resource name")
		assert.Contains(t, err.Error(), "title is 151 characters")
		assert.Contains(t, err.Error(), "max is 150")
	})

	t.Run("accepts resource title at exactly 150 characters", func(t *testing.T) {
		title := strings.Repeat("x", 150)
		_, err := Parse(`
option provider = "test"

// ` + title + `
name {
	field type
}
`)
		require.NoError(t, err)
	})

	t.Run("rejects field title over 150 characters", func(t *testing.T) {
		longTitle := strings.Repeat("y", 200)
		ast, err := Parse(`
option provider = "test"

// Title
name {
	// ` + longTitle + `
	field type
}
`)
		require.NoError(t, err)
		_, err = Schema(ast)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "field name.field")
		assert.Contains(t, err.Error(), "title is 200 characters")
	})

	t.Run("title length counts runes, not bytes", func(t *testing.T) {
		// 150 multi-byte runes (each "é" is 2 bytes) — under the rune limit,
		// over the byte limit. Must pass.
		title := strings.Repeat("é", 150)
		_, err := Parse(`
option provider = "test"

// ` + title + `
name {
	field type
}
`)
		require.NoError(t, err)
	})

	t.Run("rejects resource title starting with DEPRECATED", func(t *testing.T) {
		_, err := Parse(`
option provider = "test"

// DEPRECATED: legacy thing, use replacement instead
//
// Long-form description here.
name {
	field type
}
`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "resource name")
		assert.Contains(t, err.Error(), "starts with \"deprecated\"")
		assert.Contains(t, err.Error(), "@maturity")
	})

	t.Run("rejects resource title starting with lowercase deprecated", func(t *testing.T) {
		_, err := Parse(`
option provider = "test"

// deprecated zone plan
//
// Long-form description.
name {
	field type
}
`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "starts with \"deprecated\"")
	})

	t.Run("rejects field title starting with deprecated", func(t *testing.T) {
		ast, err := Parse(`
option provider = "test"

// Resource title
name {
	// Deprecated: use newField instead
	field type
}
`)
		require.NoError(t, err)
		_, err = Schema(ast)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "field name.field")
		assert.Contains(t, err.Error(), "starts with \"deprecated\"")
	})

	t.Run("accepts titles where deprecated is part of a larger word", func(t *testing.T) {
		_, err := Parse(`
option provider = "test"

// Deprecation policy summary
//
// Long-form description.
name {
	field type
}
`)
		require.NoError(t, err)
	})

	t.Run("rejects description starting with Deprecated.", func(t *testing.T) {
		_, err := Parse(`
option provider = "test"

// Legacy thing
//
// Deprecated. Use replacement instead.
name {
	field type
}
`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "resource name")
		assert.Contains(t, err.Error(), "description starts with \"deprecated\"")
		assert.Contains(t, err.Error(), "in favor of")
	})

	t.Run("rejects description starting with Deprecated:", func(t *testing.T) {
		_, err := Parse(`
option provider = "test"

// Legacy thing
//
// Deprecated: use replacement.
name {
	field type
}
`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "description starts with \"deprecated\"")
	})

	t.Run("rejects field description starting with Deprecated.", func(t *testing.T) {
		ast, err := Parse(`
option provider = "test"

// Resource title
name {
	// Field summary
	//
	// Deprecated. Use otherField instead.
	field type
}
`)
		require.NoError(t, err)
		_, err = Schema(ast)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "field name.field")
		assert.Contains(t, err.Error(), "description starts with \"deprecated\"")
	})

	t.Run("accepts description starting with Deprecated in favor of", func(t *testing.T) {
		_, err := Parse(`
option provider = "test"

// Legacy thing
//
// Deprecated in favor of replacementResource.
name {
	field type
}
`)
		require.NoError(t, err)
	})

	t.Run("accepts description starting with Deprecated, please use", func(t *testing.T) {
		_, err := Parse(`
option provider = "test"

// Legacy thing
//
// Deprecated, please use replacementResource.
name {
	field type
}
`)
		require.NoError(t, err)
	})

	t.Run("accepts description that mentions deprecated mid-sentence", func(t *testing.T) {
		_, err := Parse(`
option provider = "test"

// Legacy thing
//
// Examines the legacy thing; this resource is deprecated and will be
// removed soon.
name {
	field type
}
`)
		require.NoError(t, err)
	})

	t.Run("accepts description where deprecation is part of a larger word", func(t *testing.T) {
		_, err := Parse(`
option provider = "test"

// Policy summary
//
// Deprecation policy of the parent organization.
name {
	field type
}
`)
		require.NoError(t, err)
	})

	t.Run("accepts two-part field comment with blank separator", func(t *testing.T) {
		ast := parse(t, `
option provider = "test"

// Resource title
name {
	// Budget type
	//
	// One of COST, USAGE, RI_UTILIZATION, RI_COVERAGE.
	field type
}
`)
		schema, err := Schema(ast)
		require.NoError(t, err)
		f := schema.Resources["name"].Fields["field"]
		require.NotNil(t, f)
		assert.Equal(t, "Budget type", f.Title)
		assert.Equal(t, "One of COST, USAGE, RI_UTILIZATION, RI_COVERAGE.", f.Desc)
	})

	t.Run("resource with a list type", func(t *testing.T) {
		res := parse(t, "name {\nfield []type\n}")
		f := []*Field{
			{
				BasicField: &BasicField{
					ID:   "field",
					Args: nil,
					Type: Type{ListType: &ListType{Type{SimpleType: &SimpleType{"type"}}}},
				},
			},
		}
		assert.Equal(t, "name", res.Resources[0].ID)
		assert.Equal(t, f, res.Resources[0].Body.Fields)
	})

	t.Run("resource with a map type", func(t *testing.T) {
		res := parse(t, "name {\nfield map[a]b\n}")
		f := []*Field{
			{
				BasicField: &BasicField{ID: "field", Args: nil, Type: Type{
					MapType: &MapType{SimpleType{"a"}, Type{SimpleType: &SimpleType{"b"}}},
				}},
			},
		}
		assert.Equal(t, "name", res.Resources[0].ID)
		assert.Equal(t, f, res.Resources[0].Body.Fields)
	})

	t.Run("resource with a dependent field, no args", func(t *testing.T) {
		res := parse(t, "name {\nfield() type\n}")
		f := []*Field{
			{BasicField: &BasicField{ID: "field", Args: &FieldArgs{}, Type: Type{SimpleType: &SimpleType{"type"}}}},
		}
		assert.Equal(t, "name", res.Resources[0].ID)
		assert.Equal(t, f, res.Resources[0].Body.Fields)
	})

	t.Run("resource with a dependent field, with args", func(t *testing.T) {
		res := parse(t, "name {\nfield(one, two.three) type\n}")
		f := []*Field{
			{BasicField: &BasicField{ID: "field", Type: Type{SimpleType: &SimpleType{"type"}}, Args: &FieldArgs{
				List: []SimpleType{{"one"}, {"two.three"}},
			}}},
		}
		assert.Equal(t, "name", res.Resources[0].ID)
		assert.Equal(t, f, res.Resources[0].Body.Fields)
	})

	t.Run("resource with init, with args", func(t *testing.T) {
		res := parse(t, "name {\ninit(one int, two? string)\n}")
		f := []*Field{
			{Init: &Init{
				Args: []TypedArg{
					{ID: "one", Type: Type{SimpleType: &SimpleType{"int"}}},
					{ID: "two", Type: Type{SimpleType: &SimpleType{"string"}}, Optional: true},
				},
			}},
		}
		assert.Equal(t, "name", res.Resources[0].ID)
		assert.Equal(t, f, res.Resources[0].Body.Fields)
	})

	t.Run("resource which is a list type", func(t *testing.T) {
		res := parse(t, "name {\n[]base\n}")
		lt := &SimplListType{Type: SimpleType{"base"}}
		assert.Equal(t, "name", res.Resources[0].ID)
		assert.Equal(t, lt, res.Resources[0].ListType)
	})

	t.Run("resource which is a list type, with args", func(t *testing.T) {
		res := parse(t, "name {\n[]base(content)\ncontent string\n}")
		lt := &SimplListType{
			Type: SimpleType{"base"},
			Args: &FieldArgs{
				List: []SimpleType{{Type: "content"}},
			},
		}
		assert.Equal(t, "name", res.Resources[0].ID)
		assert.Equal(t, lt, res.Resources[0].ListType)
	})

	t.Run("resource which is a list type based on resource chain", func(t *testing.T) {
		res := parse(t, "name {\n[]base.type.name\n}")
		lt := &SimplListType{Type: SimpleType{"base.type.name"}}
		assert.Equal(t, "name", res.Resources[0].ID)
		assert.Equal(t, lt, res.Resources[0].ListType)
	})

	t.Run("embedded resource", func(t *testing.T) {
		res := parse(t, `
	private name.no {
		embed os.any
	}`)
		fields := []*Field{
			{BasicField: &BasicField{
				isEmbedded: true,
				ID:         "os",
				Type:       Type{SimpleType: &SimpleType{Type: "os.any"}},
				Args:       &FieldArgs{},
			}},
		}

		assert.Equal(t, "name.no", res.Resources[0].ID)
		assert.Equal(t, true, res.Resources[0].IsPrivate)
		assert.Equal(t, fields, res.Resources[0].Body.Fields)
	})

	t.Run("embedded resource with an alias", func(t *testing.T) {
		res := parse(t, `
	private name.no {
		embed os.any as testx
	}`)
		fields := []*Field{
			{BasicField: &BasicField{
				isEmbedded: true,
				ID:         "testx",
				Type:       Type{SimpleType: &SimpleType{Type: "os.any"}},
				Args:       &FieldArgs{},
			}},
		}

		assert.Equal(t, "name.no", res.Resources[0].ID)
		assert.Equal(t, true, res.Resources[0].IsPrivate)
		assert.Equal(t, fields, res.Resources[0].Body.Fields)
	})

	t.Run("complex resource", func(t *testing.T) {
		res := parse(t, `
	private name.no {
		init(i1 string, i2 map[int]int)
		field map[string]int
		call(resource.field) []int
		embed os.any
	}`)
		fields := []*Field{
			{Init: &Init{Args: []TypedArg{
				{ID: "i1", Type: Type{SimpleType: &SimpleType{"string"}}},
				{ID: "i2", Type: Type{MapType: &MapType{SimpleType{"int"}, Type{SimpleType: &SimpleType{"int"}}}}},
			}}},
			{BasicField: &BasicField{ID: "field", Type: Type{MapType: &MapType{Key: SimpleType{"string"}, Value: Type{SimpleType: &SimpleType{"int"}}}}}},
			{
				BasicField: &BasicField{
					ID:   "call",
					Type: Type{ListType: &ListType{Type: Type{SimpleType: &SimpleType{"int"}}}},
					Args: &FieldArgs{
						List: []SimpleType{{"resource.field"}},
					},
				},
			},
			{BasicField: &BasicField{isEmbedded: true, ID: "os", Type: Type{SimpleType: &SimpleType{Type: "os.any"}}, Args: &FieldArgs{}}},
		}

		assert.Equal(t, "name.no", res.Resources[0].ID)
		assert.Equal(t, true, res.Resources[0].IsPrivate)
		assert.Equal(t, fields, res.Resources[0].Body.Fields)
	})

	t.Run("file context", func(t *testing.T) {
		res := parse(t, `
	sth @context("file.context") {
		field map[string]int
	}`)

		require.NotEmpty(t, res.Resources)
		assert.Equal(t, "sth", res.Resources[0].ID)
		assert.Equal(t, "file.context", res.Resources[0].Context)

		require.Len(t, res.Resources[0].Body.Fields, 2)

		f0 := res.Resources[0].Body.Fields[0]
		assert.Equal(t, &BasicField{
			ID:   "field",
			Type: Type{MapType: &MapType{Key: SimpleType{"string"}, Value: Type{SimpleType: &SimpleType{"int"}}}},
		}, f0.BasicField)

		f1 := res.Resources[0].Body.Fields[1]
		assert.Equal(t, &BasicField{
			ID:   "context",
			Type: Type{SimpleType: &SimpleType{"file.context"}},
			Args: &FieldArgs{},
		}, f1.BasicField)
		require.Len(t, f1.Comments, 1)
		assert.Equal(t, "# Contextual info, where this resource is located and defined", f1.Comments[0].Text)
	})
}

func TestParseLR(t *testing.T) {
	files := []string{
		"core/resources/core.lr",
		"os/resources/os.lr",
	}

	for i := range files {
		lrPath := files[i]
		absPath := "../../../../providers/" + lrPath

		t.Run(lrPath, func(t *testing.T) {
			hasImports := false
			res, err := Resolve(absPath, func(path string) ([]byte, error) {
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal("failed to load " + path + ":" + err.Error())
				}
				// matches both import forms: `import network` and the legacy
				// `import "../../network/resources/network.lr"`. Keying on the
				// quote alone stopped matching when the imports migrated to
				// names, which silently skipped the assertion below.
				if bytes.Contains(raw, []byte("\nimport ")) {
					hasImports = true
				}
				return raw, err
			})
			if err != nil {
				t.Fatal("failed to compile " + lrPath + ":" + err.Error())
			}

			collector := NewCollector(absPath)
			godata, err := Go("resources", res, collector, nil)
			if err != nil {
				t.Fatal("failed to go-convert " + lrPath + ":" + err.Error())
			}
			assert.NotEmpty(t, godata)
			assert.Equal(t, "// Copyright Mondoo, Inc. 2024, 2026\n// SPDX-License-Identifier: BUSL-1.1\n\n", godata[:75])

			schema, err := Schema(res)
			if err != nil {
				t.Fatal("failed to generate schema for " + lrPath + ":" + err.Error())
			}
			assert.NotEmpty(t, schema)
			assert.NotEmpty(t, schema.Resources)

			if hasImports {
				assert.NotEmpty(t, schema.Dependencies)
			}
		})
	}
}

func TestGetFieldPaths(t *testing.T) {
	res := &Resource{
		ID: "name",
		Body: &ResourceDef{
			Fields: []*Field{
				{BasicField: &BasicField{ID: "field1", Type: Type{SimpleType: &SimpleType{"string"}}}},
				{BasicField: &BasicField{ID: "field2", Type: Type{SimpleType: &SimpleType{"int"}}}},
				{Embeddable: &Embeddable{Type: "os.any"}},
				{Init: &Init{Args: []TypedArg{{ID: "arg1", Type: Type{SimpleType: &SimpleType{"string"}}}}}},
			},
		},
	}
	paths := res.GetFieldPaths()
	assert.Equal(t, []string{"name.field1", "name.field2"}, paths)
}

func TestGetDuplicates(t *testing.T) {
	res1 := &Resource{
		ID: "res1",
		Body: &ResourceDef{
			Fields: []*Field{
				{BasicField: &BasicField{ID: "res2", Type: Type{SimpleType: &SimpleType{"resource"}}}},
				{BasicField: &BasicField{ID: "field2", Type: Type{SimpleType: &SimpleType{"int"}}}},
				{Embeddable: &Embeddable{Type: "os.any"}},
				{Init: &Init{Args: []TypedArg{{ID: "arg1", Type: Type{SimpleType: &SimpleType{"string"}}}}}},
			},
		},
	}
	res2 := &Resource{
		ID: "res1.res2",
		Body: &ResourceDef{
			Fields: []*Field{
				{BasicField: &BasicField{ID: "value", Type: Type{SimpleType: &SimpleType{"string"}}}},
			},
		},
	}
	lr := &LR{
		Resources: []*Resource{res1, res2},
	}

	dups := lr.GetDuplicates()
	assert.Equal(t, []string{"res1.res2"}, dups)
}

// coreFixture stands in for providers/core/resources/core.lr. Only a schema
// that actually names a core type loads it, so most fixtures deliberately do
// NOT supply it: their readFile fails the test on an unexpected read, which is
// what proves core is left alone when nothing references it.
// coreFixture stands in for providers/core/resources/core.lr. Its resources are
// named `asset` and `cpe`, NOT `core.asset` / `core.cpe`: the `core.` in a
// reference is the pack qualifier, and getting that wrong in the fixture is
// what would make these tests pass without resolving anything.
const coreFixture = `
option provider = "go.mondoo.com/mql/providers/core"
option go_package = "go.mondoo.com/mql/providers/core/resources"

asset {
  name string
}

cpe {
  uri string
}
`

func TestImportForms(t *testing.T) {
	const peer = `
option provider = "go.mondoo.com/mql/providers/network"
option go_package = "go.mondoo.com/mql/providers/network/resources"

network.certificate {
  pem string
}
`
	files := map[string]string{
		"providers/network/resources/network.lr": peer,
	}

	for _, tc := range []struct {
		name     string
		imports  string
		packName string
	}{
		{"by name", "import network", "network"},
		{"by path", `import "../../network/resources/network.lr"`, "network"},
		{"aliased", `import nw1 from "go.mondoo.com/mql/providers/network"`, "nw1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := tc.imports + `
option provider = "go.mondoo.com/mql/providers/os"
option go_package = "go.mondoo.com/mql/providers/os/resources"

os.host {
  addr string
}
`
			local := map[string]string{"providers/os/resources/os.lr": src}
			for k, v := range files {
				local[k] = v
			}

			res, err := Resolve("providers/os/resources/os.lr", func(path string) ([]byte, error) {
				raw, ok := local[path]
				require.Truef(t, ok, "unexpected read of %q", path)
				return []byte(raw), nil
			})
			require.NoError(t, err)

			schema, err := Schema(res)
			require.NoError(t, err)

			// whichever form was used, the peer lands under the pack name and
			// carries the peer's own provider ID
			dep, ok := schema.Dependencies[tc.packName]
			require.Truef(t, ok, "expected dependency %q, got %v", tc.packName, schema.Dependencies)
			assert.Equal(t, "go.mondoo.com/mql/providers/network", dep.Id)
		})
	}
}

func TestImportAliasIDMustMatchPeer(t *testing.T) {
	// The alias form names the peer's provider ID. If the peer disagrees, the
	// build has to say so: a declaration nobody checks is how the old
	// CrossProviderTypes list came to name providers that did not exist.
	files := map[string]string{
		"providers/os/resources/os.lr": `
import nw1 from "go.mondoo.com/mql/providers/not-network"

option provider = "go.mondoo.com/mql/providers/os"
option go_package = "go.mondoo.com/mql/providers/os/resources"

os.host {
  addr string
}
`,
		"providers/not-network/resources/not-network.lr": `
option provider = "go.mondoo.com/mql/providers/network"
option go_package = "go.mondoo.com/mql/providers/network/resources"

network {
  ip string
}
`,
	}

	_, err := Resolve("providers/os/resources/os.lr", func(path string) ([]byte, error) {
		raw, ok := files[path]
		require.Truef(t, ok, "unexpected read of %q", path)
		return []byte(raw), nil
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "declares provider")
}

func TestCoreIsNeverLoadedWithoutAnImport(t *testing.T) {
	// core has no privileged status: without `import core` it is not read at
	// all, exactly like any other peer. The readFile fails the test on an
	// unexpected read, so this asserts the absence of the old magic.
	files := map[string]string{
		"providers/demo/resources/demo.lr": `
option provider = "go.mondoo.com/mql/providers/demo"
option go_package = "go.mondoo.com/mql/providers/demo/resources"

demo.thing {
  name string
  parts []demo.part
}

demo.part {
  id string
}
`,
	}

	_, err := Resolve("providers/demo/resources/demo.lr", func(path string) ([]byte, error) {
		raw, ok := files[path]
		require.Truef(t, ok, "unexpected read of %q", path)
		return []byte(raw), nil
	})
	require.NoError(t, err)
}

func TestCoreTypeNeedsItsImport(t *testing.T) {
	// The cost of dropping the implicit load: a `core.cpe` with no import used
	// to be silently fixed up. It must now be reported, because the fallback is
	// to treat the whole dotted string as a resource name and emit
	// types.Resource("core.cpe") for a resource actually called `cpe`.
	files := map[string]string{
		"providers/demo/resources/demo.lr": `
option provider = "go.mondoo.com/mql/providers/demo"
option go_package = "go.mondoo.com/mql/providers/demo/resources"

demo.thing {
  cpes []core.cpe
}
`,
	}

	res, err := Resolve("providers/demo/resources/demo.lr", func(path string) ([]byte, error) {
		raw, ok := files[path]
		require.Truef(t, ok, "unexpected read of %q", path)
		return []byte(raw), nil
	})
	require.NoError(t, err)

	// NewCollector scans the .lr's directory for Go files, so it needs a real
	// one; an empty temp dir contributes nothing and keeps the test hermetic.
	_, err = Go("resources", res, NewCollector(filepath.Join(t.TempDir(), "demo.lr")), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot find resource core.cpe")
	assert.Contains(t, err.Error(), "import core")
}

func TestCoreImportedResolves(t *testing.T) {
	// and with the import present it resolves like any other peer
	files := map[string]string{
		"providers/demo/resources/demo.lr": `
import core

option provider = "go.mondoo.com/mql/providers/demo"
option go_package = "go.mondoo.com/mql/providers/demo/resources"

demo.thing {
  cpes []core.cpe
}
`,
		"providers/core/resources/core.lr": coreFixture,
	}

	res, err := Resolve("providers/demo/resources/demo.lr", func(path string) ([]byte, error) {
		raw, ok := files[path]
		require.Truef(t, ok, "unexpected read of %q", path)
		return []byte(raw), nil
	})
	require.NoError(t, err)

	// through Go(), which is where the pack qualifier is resolved and where the
	// missing-import case is reported
	out, err := Go("resources", res, NewCollector(filepath.Join(t.TempDir(), "demo.lr")), nil)
	require.NoError(t, err)
	assert.Contains(t, out, `types.Resource("cpe")`, "core.cpe must resolve to the bare resource name")
	assert.NotContains(t, out, `types.Resource("core.cpe")`)
}

func TestSchemaDependencyID(t *testing.T) {
	// The peer's go_package and its provider ID are deliberately different
	// here. They are identical for every provider in the tree, so a fixture
	// where they agree cannot tell a correct implementation from one that
	// derives the dependency ID from go_package.
	const peer = `
option provider = "go.mondoo.com/mql/providers/network"
option go_package = "go.example.com/some/other/path/resources"

network {
  ip string
}
`
	const consumer = `
import "../../network/resources/network.lr"

option provider = "go.mondoo.com/mql/providers/os"
option go_package = "go.mondoo.com/mql/providers/os/resources"

os.host {
  addr network
}
`

	files := map[string]string{
		"providers/os/resources/os.lr":           consumer,
		"providers/network/resources/network.lr": peer,
	}

	res, err := Resolve("providers/os/resources/os.lr", func(path string) ([]byte, error) {
		raw, ok := files[path]
		require.Truef(t, ok, "unexpected read of %q", path)
		return []byte(raw), nil
	})
	require.NoError(t, err)

	schema, err := Schema(res)
	require.NoError(t, err)

	dep, ok := schema.Dependencies["network"]
	require.True(t, ok, "expected a dependency on the imported provider")
	assert.Equal(t, "network", dep.Name)
	// Must be the peer's `option provider`, NOT its go_package. Recording the
	// go_package here yields a string no provider answers to, which the
	// runtime's name-based fallback then silently papers over.
	assert.Equal(t, "go.mondoo.com/mql/providers/network", dep.Id)
}

func TestSchemaDependencyRequiresProviderOption(t *testing.T) {
	// An import with no `option provider` cannot be identified at runtime, so
	// resolution must fail loudly rather than record an empty ID.
	files := map[string]string{
		"providers/os/resources/os.lr": `
import "../../network/resources/network.lr"

option provider = "go.mondoo.com/mql/providers/os"
option go_package = "go.mondoo.com/mql/providers/os/resources"

os.host {
  addr network
}
`,
		"providers/network/resources/network.lr": `
option go_package = "go.mondoo.com/mql/providers/network/resources"

network {
  ip string
}
`,
	}

	_, err := Resolve("providers/os/resources/os.lr", func(path string) ([]byte, error) {
		raw, ok := files[path]
		require.Truef(t, ok, "unexpected read of %q", path)
		return []byte(raw), nil
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider")
}

func resolveFixture(t *testing.T, files map[string]string, entry string) (*LR, error) {
	t.Helper()
	return Resolve(entry, func(path string) ([]byte, error) {
		raw, ok := files[path]
		require.Truef(t, ok, "unexpected read of %q", path)
		return []byte(raw), nil
	})
}

func TestExtendQualified(t *testing.T) {
	res, err := resolveFixture(t, map[string]string{
		"providers/demo/resources/demo.lr": `
import core

option provider = "go.mondoo.com/mql/providers/demo"
option go_package = "go.mondoo.com/mql/providers/demo/resources"

extend core.asset {
  extra string
}
`,
		"providers/core/resources/core.lr": coreFixture,
	}, "providers/demo/resources/demo.lr")
	require.NoError(t, err)
	require.Len(t, res.Resources, 1)
	assert.True(t, res.Resources[0].IsExtension)
	assert.Equal(t, "asset", res.Resources[0].ID, "pack qualifier must be stripped")
}

func TestExtendBareStillWorks(t *testing.T) {
	res, err := resolveFixture(t, map[string]string{
		"providers/demo/resources/demo.lr": `
option provider = "go.mondoo.com/mql/providers/demo"
option go_package = "go.mondoo.com/mql/providers/demo/resources"

extend asset {
  extra string
}
`,
	}, "providers/demo/resources/demo.lr")
	require.NoError(t, err)
	assert.Equal(t, "asset", res.Resources[0].ID)
}

func TestExtendQualifiedUnknownResource(t *testing.T) {
	_, err := resolveFixture(t, map[string]string{
		"providers/demo/resources/demo.lr": `
import core

option provider = "go.mondoo.com/mql/providers/demo"
option go_package = "go.mondoo.com/mql/providers/demo/resources"

extend core.doesNotExist {
  extra string
}
`,
		"providers/core/resources/core.lr": coreFixture,
	}, "providers/demo/resources/demo.lr")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `has no resource "doesNotExist"`)
}
