// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann
package lr

import (
	"os"
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
		// TODO: needs to be fixed
		// assert.Equal(t, "resource-docs", res.Resources[0].title)
		// assert.Equal(t, "with multiline", res.Resources[0].desc)

		f := []*Field{
			{
				BasicField: &BasicField{
					ID:   "field",
					Args: nil,
					Type: Type{SimpleType: &SimpleType{"type"}},
				},
				Comments: []string{"// field docs..."},
			},
		}
		assert.Equal(t, f, res.Resources[0].Body.Fields)
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
}

func TestParseLR(t *testing.T) {
	files := []string{
		"core/resources/core.lr",
		"os/resources/os.lr",
	}

	for i := range files {
		lrPath := files[i]
		absPath := "../../../providers/" + lrPath

		t.Run(lrPath, func(t *testing.T) {
			res, err := Resolve(absPath, func(path string) ([]byte, error) {
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal("failed to load " + path + ":" + err.Error())
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
			assert.Equal(t, godata[:68], "// Copyright (c) Mondoo, Inc.\n// SPDX-License-Identifier: BUSL-1.1\n\n")

			schema, err := Schema(res)
			if err != nil {
				t.Fatal("failed to generate schema for " + lrPath + ":" + err.Error())
			}
			assert.NotEmpty(t, schema)
		})
	}
}
