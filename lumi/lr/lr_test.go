// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann
package lr

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func parse(t *testing.T, cmd string, f func(*LR)) {
	res, err := Parse(cmd)
	assert.Nil(t, err)
	if err == nil {
		f(res)
	}
}

func TestParse(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		parse(t, "", func(res *LR) {
			assert.Equal(t, &LR{}, res)
		})
	})

	t.Run("empty resource", func(t *testing.T) {
		parse(t, "name", func(res *LR) {
			assert.Equal(t, []*Resource{{ID: "name"}}, res.Resources)
		})
	})

	t.Run("empty resources", func(t *testing.T) {
		parse(t, "one tw2 thr33", func(res *LR) {
			assert.Equal(t, []*Resource{
				{ID: "one"},
				{ID: "tw2"},
				{ID: "thr33"},
			}, res.Resources)
		})
	})

	t.Run("resource with a static field", func(t *testing.T) {
		parse(t, `
		// resource-docs
		// with multiline
		name {
			// field docs...
			field type
		}
		`, func(res *LR) {
			assert.Equal(t, "name", res.Resources[0].ID)
			assert.Equal(t, []string{
				"// resource-docs",
				"// with multiline",
			}, res.Resources[0].Comments)

			f := []*Field{
				{
					ID:       "field",
					Args:     nil,
					Type:     Type{SimpleType: &SimpleType{"type"}},
					Comments: []string{"// field docs..."},
				},
			}
			assert.Equal(t, f, res.Resources[0].Body.Fields)
		})
	})

	t.Run("resource with a list type", func(t *testing.T) {
		parse(t, "name {\nfield []type\n}", func(res *LR) {
			f := []*Field{
				{ID: "field", Args: nil, Type: Type{ListType: &ListType{Type{SimpleType: &SimpleType{"type"}}}}},
			}
			assert.Equal(t, "name", res.Resources[0].ID)
			assert.Equal(t, f, res.Resources[0].Body.Fields)
		})
	})

	t.Run("resource with a map type", func(t *testing.T) {
		parse(t, "name {\nfield map[a]b\n}", func(res *LR) {
			f := []*Field{
				{ID: "field", Args: nil, Type: Type{
					MapType: &MapType{SimpleType{"a"}, Type{SimpleType: &SimpleType{"b"}}},
				}},
			}
			assert.Equal(t, "name", res.Resources[0].ID)
			assert.Equal(t, f, res.Resources[0].Body.Fields)
		})
	})

	t.Run("resource with a dependent field, no args", func(t *testing.T) {
		parse(t, "name {\nfield() type\n}", func(res *LR) {
			f := []*Field{
				{ID: "field", Args: &FieldArgs{}, Type: Type{SimpleType: &SimpleType{"type"}}},
			}
			assert.Equal(t, "name", res.Resources[0].ID)
			assert.Equal(t, f, res.Resources[0].Body.Fields)
		})
	})

	t.Run("resource with a dependent field, with args", func(t *testing.T) {
		parse(t, "name {\nfield(one, two.three) type\n}", func(res *LR) {
			f := []*Field{
				{ID: "field", Type: Type{SimpleType: &SimpleType{"type"}}, Args: &FieldArgs{
					List: []SimpleType{{"one"}, {"two.three"}},
				}},
			}
			assert.Equal(t, "name", res.Resources[0].ID)
			assert.Equal(t, f, res.Resources[0].Body.Fields)
		})
	})

	t.Run("resource with init, with args", func(t *testing.T) {
		parse(t, "name {\ninit(one int, two? string)\n}", func(res *LR) {
			f := []*Init{
				{Args: []TypedArg{
					{ID: "one", Type: Type{SimpleType: &SimpleType{"int"}}},
					{ID: "two", Type: Type{SimpleType: &SimpleType{"string"}}, Optional: true},
				}},
			}
			assert.Equal(t, "name", res.Resources[0].ID)
			assert.Equal(t, f, res.Resources[0].Body.Inits)
		})
	})

	t.Run("resource which is a list type", func(t *testing.T) {
		parse(t, "name {\n[]base\n}", func(res *LR) {
			lt := &SimplListType{Type: SimpleType{"base"}}
			assert.Equal(t, "name", res.Resources[0].ID)
			assert.Equal(t, lt, res.Resources[0].ListType)
		})
	})

	t.Run("resource which is a list type, with args", func(t *testing.T) {
		parse(t, "name {\n[]base(content)\ncontent string\n}", func(res *LR) {
			lt := &SimplListType{
				Type: SimpleType{"base"},
				Args: &FieldArgs{
					List: []SimpleType{{Type: "content"}},
				},
			}
			assert.Equal(t, "name", res.Resources[0].ID)
			assert.Equal(t, lt, res.Resources[0].ListType)
		})
	})

	t.Run("resource which is a list type based on resource chain", func(t *testing.T) {
		parse(t, "name {\n[]base.type.name\n}", func(res *LR) {
			lt := &SimplListType{Type: SimpleType{"base.type.name"}}
			assert.Equal(t, "name", res.Resources[0].ID)
			assert.Equal(t, lt, res.Resources[0].ListType)
		})
	})

	t.Run("complex resource", func(t *testing.T) {
		parse(t, `
	private name.no {
		init(i1 string, i2 map[int]int)
		field map[string]int
		call(resource.field) []int
	}`, func(res *LR) {
			i := []*Init{
				{Args: []TypedArg{
					{ID: "i1", Type: Type{SimpleType: &SimpleType{"string"}}},
					{ID: "i2", Type: Type{MapType: &MapType{SimpleType{"int"}, Type{SimpleType: &SimpleType{"int"}}}}},
				}},
			}
			f := []*Field{
				{ID: "field", Type: Type{MapType: &MapType{Key: SimpleType{"string"}, Value: Type{SimpleType: &SimpleType{"int"}}}}},
				{
					ID:   "call",
					Type: Type{ListType: &ListType{Type: Type{SimpleType: &SimpleType{"int"}}}},
					Args: &FieldArgs{
						List: []SimpleType{{"resource.field"}},
					},
				},
			}
			assert.Equal(t, "name.no", res.Resources[0].ID)
			assert.Equal(t, true, res.Resources[0].IsPrivate)
			assert.Equal(t, i, res.Resources[0].Body.Inits)
			assert.Equal(t, f, res.Resources[0].Body.Fields)
		})
	})
}

func TestParseCoreLR(t *testing.T) {
	path := "../resources/core.lr"
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal("failed to load core.lr: " + err.Error())
	}

	res, err := Parse(string(raw))
	if err != nil {
		t.Fatal("failed to compile core.lr: " + err.Error())
	}

	godata, err := Go(res, NewCollector(path))
	if err != nil {
		t.Fatal("failed to go-convert core.lr: " + err.Error())
	}

	assert.NotEmpty(t, godata)
}
