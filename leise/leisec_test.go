package leise

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/llx/registry"
	"go.mondoo.io/mondoo/logger"
	"go.mondoo.io/mondoo/types"
)

var schema = registry.Default.Schema()

func init() {
	logger.InitTestEnv()
}

func compile(t *testing.T, s string, f func(res *llx.CodeBundle)) {
	res, err := Compile(s, schema)
	assert.Nil(t, err)
	assert.NotNil(t, res)
	if res != nil && res.Code != nil {
		assert.Nil(t, res.Suggestions)
		assert.NotEmpty(t, res.Code.Code)
		if len(res.Code.Code) > 0 {
			f(res)
		}
	}
}

func TestCompiler_Simple(t *testing.T) {
	data := []struct {
		code string
		res  *llx.Primitive
	}{
		{"false", llx.BoolPrimitive(false)},
		{"true", llx.BoolPrimitive(true)},
		{"123", llx.IntPrimitive(123)},
		{"12.3", llx.FloatPrimitive(12.3)},
		{"\"hi\"", llx.StringPrimitive("hi")},
		{"/hi/", llx.RegexPrimitive("hi")},
		{"[true, false]", &llx.Primitive{
			Type: string(types.Array(types.Bool)),
			Array: []*llx.Primitive{
				llx.BoolPrimitive(true),
				llx.BoolPrimitive(false),
			},
		}},
		{"[1, 2]", &llx.Primitive{
			Type: string(types.Array(types.Int)),
			Array: []*llx.Primitive{
				llx.IntPrimitive(1),
				llx.IntPrimitive(2),
			},
		}},
		{"[1.2,3.4]", &llx.Primitive{
			Type: string(types.Array(types.Float)),
			Array: []*llx.Primitive{
				llx.FloatPrimitive(1.2),
				llx.FloatPrimitive(3.4),
			},
		}},
		{"[\"a\",\"b\"]", &llx.Primitive{
			Type: string(types.Array(types.String)),
			Array: []*llx.Primitive{
				llx.StringPrimitive("a"),
				llx.StringPrimitive("b"),
			},
		}},
		{"[1.2,1]", &llx.Primitive{
			Type: string(types.Array(types.Any)),
			Array: []*llx.Primitive{
				llx.FloatPrimitive(1.2),
				llx.IntPrimitive(1),
			},
		}},
	}
	for _, v := range data {
		t.Run(v.code, func(t *testing.T) {
			compile(t, v.code, func(res *llx.CodeBundle) {
				o := res.Code.Code[0]
				assert.Equal(t, llx.Chunk_PRIMITIVE, o.Call)
				assert.Equal(t, v.res, o.Primitive)
			})
		})
	}
}

// FIXME: this is weirdly failing
// func TestCompiler_SimpleArrayResource(t *testing.T) {
// 	res := compile(t, "[mochi, mochi]").Code.Code[2]
// 	assert.Equal(t, llx.Chunk_PRIMITIVE, res.Call)
// 	assert.Equal(t, []types.Type{types.Type_ARRAY, types.Type_ANY}, res.Primitive.Type)
// 	assert.Equal(t, []*llx.Primitive{
// 		llx.RefPrimitive(1),
// 		llx.RefPrimitive(2),
// 	}, res.Primitive.Array)
// 	assert.Nil(t, res.Primitive.Value)
// }

func TestCompiler_Comparisons(t *testing.T) {
	ops := []string{"==", "!=", ">", "<", ">=", "<="}
	vals := map[string]*llx.Primitive{
		"1":       llx.IntPrimitive(1),
		"1.2":     llx.FloatPrimitive(1.2),
		"true":    llx.BoolPrimitive(true),
		"\"str\"": llx.StringPrimitive("str"),
		"/str/":   llx.RegexPrimitive("str"),
	}
	for _, op := range ops {
		for val, valres := range vals {
			if types.Type(valres.Type) != types.Int && types.Type(valres.Type) != types.Float && types.Type(valres.Type) != types.String {
				continue
			}
			code := val + " " + op + " " + val
			t.Run(code, func(t *testing.T) {
				compile(t, code, func(res *llx.CodeBundle) {
					o := res.Code.Code[0]
					assert.Equal(t, valres, o.Primitive)
					o = res.Code.Code[1]
					assert.Equal(t, llx.Chunk_FUNCTION, o.Call)
					assert.Equal(t, op+valres.Type, o.Id)
					assert.Equal(t, int32(1), o.Function.Binding)
					assert.Equal(t, string(types.Bool), o.Function.Type)
					assert.Equal(t, valres, o.Function.Args[0])
				})
			})
		}
	}
}

func TestCompiler_OperatorPrecedence(t *testing.T) {
	data := []struct {
		code   string
		first  string
		second string
	}{
		// {"1 && 2 || 3", "&&", "||"},
	}

	for _, d := range data {
		t.Run(d.code, func(t *testing.T) {
			compile(t, d.code, func(res *llx.CodeBundle) {
				fmt.Printf("compiled: %#v\n", res)
				o := res.Code.Code[0]
				assert.Equal(t, d.first, o.Id)
			})
		})
	}
}

func TestSuggestions(t *testing.T) {
	t.Run("no suggestions", func(t *testing.T) {
		res, err := Compile("notthere", schema)
		assert.Nil(t, res.Code.Entrypoints)
		assert.Empty(t, res.Suggestions)
		assert.Equal(t, errors.New("Cannot find resource for identifier 'notthere'"), err)
	})

	t.Run("resource suggestions", func(t *testing.T) {
		res, err := Compile("ssh", schema)
		assert.Nil(t, res.Code.Entrypoints)
		assert.Equal(t, []string{"sshd", "sshd.config"}, res.Suggestions)
		assert.Equal(t, errors.New("Cannot find resource for identifier 'ssh'"), err)
	})

	t.Run("field suggestions", func(t *testing.T) {
		res, err := Compile("sshd.config.p", schema)
		assert.Nil(t, res.Code.Entrypoints)
		assert.Equal(t, []string{"params"}, res.Suggestions)
		assert.Equal(t, errors.New("Cannot find field 'p' in resource sshd.config"), err)
	})

	t.Run("field in block suggestions", func(t *testing.T) {
		res, err := Compile("sshd.config { p }", schema)
		assert.Nil(t, res.Code.Entrypoints)
		assert.Equal(t, []string{"params"}, res.Suggestions)
		assert.Equal(t, errors.New("Cannot find field or resource 'p' in block for type 'sshd.config'"), err)
	})

	t.Run("field suggestions on partial map", func(t *testing.T) {
		res, err := Compile("sshd.config.params.l", schema)
		assert.Nil(t, res.Code.Entrypoints)
		assert.Equal(t, []string{"length"}, res.Suggestions)
		assert.Equal(t, errors.New("Cannot find field 'l' in resource map[string]string"), err)
	})
}

func assertFunction(t *testing.T, id string, f *llx.Function, chunk *llx.Chunk) {
	assert.Equal(t, llx.Chunk_FUNCTION, chunk.Call)
	assert.Equal(t, id, chunk.Id, "chunk.Id")
	assert.Nil(t, chunk.Primitive, "it is not a primitive")
	assert.Equal(t, f, chunk.Function, "chunk.Function")
}

func assertPrimitive(t *testing.T, p *llx.Primitive, chunk *llx.Chunk) {
	assert.Equal(t, llx.Chunk_PRIMITIVE, chunk.Call)
	assert.Nil(t, chunk.Function)
	assert.Equal(t, p, chunk.Primitive)
}

func TestCompiler_Resource(t *testing.T) {
	compile(t, "sshd", func(res *llx.CodeBundle) {
		assertFunction(t, "sshd", nil, res.Code.Code[0])
	})
}

func TestCompiler_ResourceWithCall(t *testing.T) {
	compile(t, "sshd()", func(res *llx.CodeBundle) {
		assertFunction(t, "sshd", nil, res.Code.Code[0])
	})
}

func TestCompiler_LongResource(t *testing.T) {
	compile(t, "sshd.config", func(res *llx.CodeBundle) {
		assertFunction(t, "sshd.config", nil, res.Code.Code[0])
	})
}

func TestCompiler_ResourceMap(t *testing.T) {
	compile(t, "sshd.config.params", func(res *llx.CodeBundle) {
		assertFunction(t, "sshd.config", nil, res.Code.Code[0])
		assertFunction(t, "params", &llx.Function{
			Type:    string(types.Map(types.String, types.String)),
			Binding: 1,
		}, res.Code.Code[1])
	})
}

func TestCompiler_ResourceMapLength(t *testing.T) {
	compile(t, "sshd.config.params.length", func(res *llx.CodeBundle) {
		assertFunction(t, "length", &llx.Function{
			Type:    string(types.Int),
			Binding: 2,
		}, res.Code.Code[2])
	})
}

func TestCompiler_ResourceArrayAccessor(t *testing.T) {
	compile(t, "packages.list[123]", func(res *llx.CodeBundle) {
		assertFunction(t, "[]", &llx.Function{
			Binding: 2,
			Args:    []*llx.Primitive{llx.IntPrimitive(123)},
			Type:    string(types.Resource("package")),
		}, res.Code.Code[2])
	})
}

func TestCompiler_ResourceArrayLength(t *testing.T) {
	compile(t, "packages.list.length", func(res *llx.CodeBundle) {
		assertFunction(t, "length", &llx.Function{
			Binding: 2,
			Type:    string(types.Int),
		}, res.Code.Code[2])
	})
}

func TestCompiler_ResourceFieldArrayAccessor(t *testing.T) {
	compile(t, "sshd.config.params[\"Protocol\"]", func(res *llx.CodeBundle) {
		assertFunction(t, "[]", &llx.Function{
			Type:    string(types.String),
			Binding: 2,
			Args: []*llx.Primitive{
				llx.StringPrimitive("Protocol"),
			},
		}, res.Code.Code[2])
	})
}

func TestCompiler_ResourceWithUnnamedArgs(t *testing.T) {
	compile(t, "file(\"/path\")", func(res *llx.CodeBundle) {
		assertFunction(t, "file", &llx.Function{
			Type:    string(types.Resource("file")),
			Binding: 0,
			Args: []*llx.Primitive{
				llx.StringPrimitive("path"),
				llx.StringPrimitive("/path"),
			},
		}, res.Code.Code[0])
	})
}

func TestCompiler_ResourceWithNamedArgs(t *testing.T) {
	compile(t, "file(path: \"/path\")", func(res *llx.CodeBundle) {
		assertFunction(t, "file", &llx.Function{
			Type:    string(types.Resource("file")),
			Binding: 0,
			Args: []*llx.Primitive{
				llx.StringPrimitive("path"),
				llx.StringPrimitive("/path"),
			},
		}, res.Code.Code[0])
	})
}

func TestCompiler_LongResourceWithUnnamedArgs(t *testing.T) {
	compile(t, "sshd.config(\"/path\")", func(res *llx.CodeBundle) {
		assertFunction(t, "sshd.config", &llx.Function{
			Type:    string(types.Resource("sshd.config")),
			Binding: 0,
			Args: []*llx.Primitive{
				llx.StringPrimitive("path"),
				llx.StringPrimitive("/path"),
			},
		}, res.Code.Code[0])
	})
}

func TestCompiler_ExpectSimplest(t *testing.T) {
	compile(t, "expect(true)", func(res *llx.CodeBundle) {
		f := res.Code.Code[0]
		assert.Equal(t, llx.Chunk_FUNCTION, f.Call)
		assert.Equal(t, "expect", f.Id)
		assert.Equal(t, []int32{1}, res.Code.Entrypoints)
		assert.Equal(t, &llx.Function{
			Type:    string(types.Bool),
			Binding: 0,
			Args:    []*llx.Primitive{llx.BoolPrimitive(true)},
		}, f.Function)
	})
}

func TestCompiler_ExpectEq(t *testing.T) {
	compile(t, "expect(1 == \"1\")", func(res *llx.CodeBundle) {
		cmp := res.Code.Code[1]
		assert.Equal(t, llx.Chunk_FUNCTION, cmp.Call)
		assert.Equal(t, []int32{3}, res.Code.Entrypoints)
		assert.Equal(t, "=="+string(types.String), cmp.Id)
		assert.Equal(t, &llx.Function{
			Type:    string(types.Bool),
			Binding: 1,
			Args: []*llx.Primitive{
				llx.StringPrimitive("1"),
			},
		}, cmp.Function)

		f := res.Code.Code[2]
		assert.Equal(t, llx.Chunk_FUNCTION, f.Call)
		assert.Equal(t, "expect", f.Id)
		assert.Equal(t, &llx.Function{
			Type:    string(types.Bool),
			Binding: 0,
			Args:    []*llx.Primitive{llx.RefPrimitive(2)},
		}, f.Function)
	})
}

func TestCompiler_EmptyBlock(t *testing.T) {
	compile(t, "mondoo { }", func(res *llx.CodeBundle) {
		assertFunction(t, "mondoo", nil, res.Code.Code[0])
		assert.Equal(t, 1, len(res.Code.Code))
		assert.Nil(t, res.Code.Functions)
	})
}

func TestCompiler_Block(t *testing.T) {
	compile(t, "mondoo { version build }", func(res *llx.CodeBundle) {
		assertFunction(t, "mondoo", nil, res.Code.Code[0])
		assertFunction(t, "{}", &llx.Function{
			Type:    string(types.Any),
			Binding: 1,
			Args:    []*llx.Primitive{llx.FunctionPrimitive(1)},
		}, res.Code.Code[1])
		assert.Equal(t, []int32{2}, res.Code.Entrypoints)

		assertPrimitive(t, &llx.Primitive{
			Type: string(types.Resource("mondoo")),
		}, res.Code.Functions[0].Code[0])
		assertFunction(t, "version", &llx.Function{
			Type:    string(types.String),
			Binding: 1,
		}, res.Code.Functions[0].Code[1])
		assertFunction(t, "build", &llx.Function{
			Type:    string(types.String),
			Binding: 1,
		}, res.Code.Functions[0].Code[2])
		assert.Equal(t, []int32{2, 3}, res.Code.Functions[0].Entrypoints)
	})
}

func TestCompiler_List(t *testing.T) {
	compile(t, "packages.list { name }", func(res *llx.CodeBundle) {
		assertFunction(t, "packages", nil, res.Code.Code[0])
		assertFunction(t, "list", &llx.Function{
			Type:    string(types.Array(types.Resource("package"))),
			Binding: 1,
		}, res.Code.Code[1])
		assertFunction(t, "{}", &llx.Function{
			Type:    string(types.Array(types.Any)),
			Binding: 2,
			Args:    []*llx.Primitive{llx.FunctionPrimitive(1)},
		}, res.Code.Code[2])
		assert.Equal(t, 3, len(res.Code.Code))

		assertPrimitive(t, &llx.Primitive{
			Type: string(types.Resource("package")),
		}, res.Code.Functions[0].Code[0])
		assertFunction(t, "name", &llx.Function{
			Type:    string(types.String),
			Binding: 1,
		}, res.Code.Functions[0].Code[1])
		assert.Equal(t, []int32{2}, res.Code.Functions[0].Entrypoints)
	})
}

func TestCompiler_EmptyWhere(t *testing.T) {
	compile(t, "packages.where()", func(res *llx.CodeBundle) {
		assertFunction(t, "packages", nil, res.Code.Code[0])
		assert.Equal(t, 1, len(res.Code.Code))
	})
}

func TestCompiler_Where(t *testing.T) {
	compile(t, "packages.where(outdated)", func(res *llx.CodeBundle) {
		assertFunction(t, "packages", nil, res.Code.Code[0])
		assertFunction(t, "list", &llx.Function{
			Type:    string(types.Array(types.Resource("package"))),
			Binding: 1,
		}, res.Code.Code[1])
		assertFunction(t, "where", &llx.Function{
			Type:    string(types.Resource("packages")),
			Binding: 1,
			Args: []*llx.Primitive{
				llx.RefPrimitive(2),
				llx.FunctionPrimitive(1),
			},
		}, res.Code.Code[2])

		assertPrimitive(t, &llx.Primitive{
			Type: string(types.Resource("package")),
		}, res.Code.Functions[0].Code[0])
		assertFunction(t, "outdated", &llx.Function{
			Type:    string(types.Bool),
			Binding: 1,
		}, res.Code.Functions[0].Code[1])
		assert.Equal(t, []int32{2}, res.Code.Functions[0].Entrypoints)
	})
}
