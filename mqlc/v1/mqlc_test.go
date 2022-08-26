package v1

import (
	"errors"
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/logger"
	resource_info "go.mondoo.com/cnquery/resources/packs/os/info"
	"go.mondoo.com/cnquery/types"
)

var schema = resource_info.Registry.Schema()

func init() {
	logger.InitTestEnv()
}

func compileProps(t *testing.T, s string, props map[string]*llx.Primitive, f func(res *llx.CodeBundle)) {
	res, err := Compile(s, schema, props)
	assert.Nil(t, err)
	assert.NotNil(t, res)
	assert.NoError(t, Invariants.Check(res))
	if res != nil && res.DeprecatedV5Code != nil {
		assert.Nil(t, res.Suggestions)
		assert.NotEmpty(t, res.DeprecatedV5Code.Code)
		if len(res.DeprecatedV5Code.Code) > 0 {
			f(res)
		}
	}
}

func compile(t *testing.T, s string, f func(res *llx.CodeBundle)) {
	compileProps(t, s, nil, f)
}

func compileEmpty(t *testing.T, s string, f func(res *llx.CodeBundle)) {
	res, err := Compile(s, schema, nil)
	assert.Nil(t, err)
	assert.NotNil(t, res)

	if res != nil && res.DeprecatedV5Code != nil {
		assert.Nil(t, res.Suggestions)
		f(res)
	}
}

func compileErroneous(t *testing.T, s string, expectedError error, f func(res *llx.CodeBundle)) {
	res, err := Compile(s, schema, nil)

	if err != nil && expectedError != nil {
		assert.Equal(t, expectedError.Error(), err.Error())
	} else {
		assert.Equal(t, expectedError, err)
	}

	if f != nil {
		f(res)
	}
}

func assertPrimitive(t *testing.T, p *llx.Primitive, chunk *llx.Chunk) {
	assert.Equal(t, llx.Chunk_PRIMITIVE, chunk.Call)
	assert.Nil(t, chunk.Function)
	assert.Equal(t, p, chunk.Primitive)
}

func assertFunction(t *testing.T, id string, f *llx.Function, chunk *llx.Chunk) {
	assert.Equal(t, llx.Chunk_FUNCTION, chunk.Call)
	assert.Equal(t, id, chunk.Id, "chunk.Id")
	assert.Nil(t, chunk.Primitive, "it is not a primitive")
	assert.Equal(t, f, chunk.Function, "chunk.Function")
}

func assertProperty(t *testing.T, name string, typ types.Type, chunk *llx.Chunk) {
	assert.Equal(t, llx.Chunk_PROPERTY, chunk.Call)
	assert.Equal(t, name, chunk.Id, "property name is set")
	assert.Equal(t, &llx.Primitive{Type: string(typ)}, chunk.Primitive, "property type is set")
}

//    ===========================
//   👋   VALUES + OPERATIONS   🍹
//    ===========================

func TestCompiler_Basics(t *testing.T) {
	data := []struct {
		code string
		res  []*llx.Chunk
	}{
		{"", nil},
		{"// some comment", nil},
		{"// some comment\n", nil},
	}
	for _, v := range data {
		t.Run(v.code, func(t *testing.T) {
			compileEmpty(t, v.code, func(res *llx.CodeBundle) {
				assert.Equal(t, v.res, res.DeprecatedV5Code.Code)
			})
		})
	}
}

func TestCompiler_Buggy(t *testing.T) {
	data := []struct {
		code string
		res  []*llx.Chunk
		err  error
	}{
		{`mondoo mondoo`, []*llx.Chunk{
			{Id: "mondoo", Call: llx.Chunk_FUNCTION},
			{Id: "mondoo", Call: llx.Chunk_FUNCTION},
		}, nil},
		{`mondoo # mondoo`, []*llx.Chunk{
			{Id: "mondoo", Call: llx.Chunk_FUNCTION},
		}, nil},
		{`mondoo }`, []*llx.Chunk{
			{Id: "mondoo", Call: llx.Chunk_FUNCTION},
		}, errors.New("mismatched symbol '}' at the end of expression")},
		{`mondoo ]`, []*llx.Chunk{
			{Id: "mondoo", Call: llx.Chunk_FUNCTION},
		}, errors.New("mismatched symbol ']' at the end of expression")},
		{`mondoo )`, []*llx.Chunk{
			{Id: "mondoo", Call: llx.Chunk_FUNCTION},
		}, errors.New("mismatched symbol ')' at the end of expression")},
		{`mondoo { version }`, []*llx.Chunk{
			{Id: "mondoo", Call: llx.Chunk_FUNCTION},
			{Id: "{}", Call: llx.Chunk_FUNCTION, Function: &llx.Function{
				Type:                string(types.Block),
				DeprecatedV5Binding: 1,
				Args:                []*llx.Primitive{llx.FunctionPrimitiveV1(1)},
			}},
		}, nil},
		{"# ..\nmondoo { \n# ..\nversion\n# ..\n}\n# ..", []*llx.Chunk{
			{Call: llx.Chunk_FUNCTION, Id: "mondoo"},
			{Call: llx.Chunk_FUNCTION, Id: "{}", Function: &llx.Function{
				Type:                string(types.Block),
				DeprecatedV5Binding: 1,
				Args:                []*llx.Primitive{llx.FunctionPrimitiveV1(1)},
			}},
		}, nil},
		{`users.list[]`, nil, errors.New("missing value inside of `[]` at <source>:1:12")},
		{`file(not-there)`, nil, errors.New("addResourceCall error: cannot find resource for identifier 'not'")},
		{`if(true) {`, []*llx.Chunk{
			{Call: llx.Chunk_FUNCTION, Id: "if", Function: &llx.Function{
				Type: string(types.Unset),
				Args: []*llx.Primitive{
					llx.BoolPrimitive(true),
					llx.FunctionPrimitiveV1(1),
					llx.FunctionPrimitiveV1(2),
				},
			}},
		}, errors.New("incomplete query, missing closing '}' at <source>:1:11")},
		{`parse.date`, []*llx.Chunk{
			{Id: "parse", Call: llx.Chunk_FUNCTION},
		}, errors.New("missing arguments to parse date")},
		{`parse.date()`, []*llx.Chunk{
			{Id: "parse", Call: llx.Chunk_FUNCTION},
		}, errors.New("missing arguments to parse date")},
	}

	for _, v := range data {
		t.Run(v.code, func(t *testing.T) {
			compileErroneous(t, v.code, v.err, func(res *llx.CodeBundle) {
				if res.DeprecatedV5Code != nil {
					assert.Equal(t, v.res, res.DeprecatedV5Code.Code)
				} else {
					assert.Nil(t, v.res)
				}
			})
		})
	}
}

func TestCompiler_Simple(t *testing.T) {
	data := []struct {
		code string
		res  *llx.Primitive
	}{
		{"null", llx.NilPrimitive},
		{"false", llx.BoolPrimitive(false)},
		{"true", llx.BoolPrimitive(true)},
		{"123", llx.IntPrimitive(123)},
		{"010", llx.IntPrimitive(8)},
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
		{"[\n  1.2,\n  1\n]", &llx.Primitive{
			Type: string(types.Array(types.Any)),
			Array: []*llx.Primitive{
				llx.FloatPrimitive(1.2),
				llx.IntPrimitive(1),
			},
		}},
		{"{a: 123}", &llx.Primitive{
			Type: string(types.Map(types.String, types.Int)),
			Map: map[string]*llx.Primitive{
				"a": llx.IntPrimitive(123),
			},
		}},
	}
	for _, v := range data {
		t.Run(v.code, func(t *testing.T) {
			compile(t, v.code, func(res *llx.CodeBundle) {
				o := res.DeprecatedV5Code.Code[0]
				assert.Equal(t, llx.Chunk_PRIMITIVE, o.Call)
				assert.Equal(t, v.res, o.Primitive)
			})
		})
	}
}

// // FIXME: this is weirdly failing
// // func TestCompiler_SimpleArrayResource(t *testing.T) {
// // 	res := compile(t, "[mochi, mochi]").Code.Code[2]
// // 	assert.Equal(t, llx.Chunk_PRIMITIVE, res.Call)
// // 	assert.Equal(t, []types.Type{types.Type_ARRAY, types.Type_ANY}, res.Primitive.Type)
// // 	assert.Equal(t, []*llx.Primitive{
// // 		llx.RefPrimitiveV1(1),
// // 		llx.RefPrimitiveV1(2),
// // 	}, res.Primitive.Array)
// // 	assert.Nil(t, res.Primitive.Value)
// // }

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
					o := res.DeprecatedV5Code.Code[0]
					assert.Equal(t, valres, o.Primitive)
					o = res.DeprecatedV5Code.Code[1]
					assert.Equal(t, llx.Chunk_FUNCTION, o.Call)
					assert.Equal(t, op+string(valres.Type), o.Id)
					assert.Equal(t, int32(1), o.Function.DeprecatedV5Binding)
					assert.Equal(t, types.Bool, types.Type(o.Function.Type))
					assert.Equal(t, valres, o.Function.Args[0])
				})
			})
		}
	}
}

func TestCompiler_LogicalOps(t *testing.T) {
	ops := []string{"&&", "||"}
	vals := map[string]*llx.Primitive{
		"1":       llx.IntPrimitive(1),
		"1.2":     llx.FloatPrimitive(1.2),
		"true":    llx.BoolPrimitive(true),
		"\"str\"": llx.StringPrimitive("str"),
		"/str/":   llx.RegexPrimitive("str"),
		"[]":      llx.ArrayPrimitive([]*llx.Primitive{}, types.Unset),
		"{}":      llx.MapPrimitive(map[string]*llx.Primitive{}, types.Unset),
	}
	for _, op := range ops {
		for val1, valres1 := range vals {
			for val2, valres2 := range vals {
				code := val1 + " " + op + " " + val2
				t.Run(code, func(t *testing.T) {
					compile(t, code, func(res *llx.CodeBundle) {
						l := res.DeprecatedV5Code.Code[0]
						assert.Equal(t, valres1, l.Primitive)

						r := res.DeprecatedV5Code.Code[1]
						assert.Equal(t, llx.Chunk_FUNCTION, r.Call)
						assert.Equal(t, int32(1), r.Function.DeprecatedV5Binding)
						assert.Equal(t, types.Bool, types.Type(r.Function.Type))
						assert.Equal(t, valres2, r.Function.Args[0])

						f, err := llx.BuiltinFunctionV1(l.Type(), r.Id)
						assert.NoError(t, err, "was able to find builtin function for llx execution")
						assert.NotNil(t, f, "was able to get non-nil builtin function")
					})
				})
			}
		}
	}
}

func TestCompiler_OperatorPrecedence(t *testing.T) {
	data := []struct {
		code   string
		idx    int
		first  string
		second string
	}{
		{"1 || 2 && 3", 2, string("&&" + types.Int), string("||" + types.Bool)},
		{"1 && 2 || 3", 1, string("&&" + types.Int), string("||" + types.Int)},
	}

	for _, d := range data {
		t.Run(d.code, func(t *testing.T) {
			compile(t, d.code, func(res *llx.CodeBundle) {
				fmt.Printf("compiled: %#v\n", res)

				o := res.DeprecatedV5Code.Code[d.idx]
				assert.Equal(t, d.first, o.Id)

				o = res.DeprecatedV5Code.Code[d.idx+1]
				assert.Equal(t, d.second, o.Id)
			})
		})
	}
}

func TestCompiler_Assignment(t *testing.T) {
	compile(t, "a = 123", func(res *llx.CodeBundle) {
		assertPrimitive(t, llx.IntPrimitive(123), res.DeprecatedV5Code.Code[0])
		assert.Equal(t, []int32{}, res.DeprecatedV5Code.Entrypoints)
	})
	compile(t, "a = 123\na", func(res *llx.CodeBundle) {
		assertPrimitive(t, llx.RefPrimitiveV1(1), res.DeprecatedV5Code.Code[1])
		assert.Equal(t, []int32{2}, res.DeprecatedV5Code.Entrypoints)
	})
}

func TestCompiler_Props(t *testing.T) {
	compileProps(t, "props.name", map[string]*llx.Primitive{
		"name": {Type: string(types.String)},
	}, func(res *llx.CodeBundle) {
		assertProperty(t, "name", types.String, res.DeprecatedV5Code.Code[0])
		assert.Equal(t, []int32{1}, res.DeprecatedV5Code.Entrypoints)
		assert.Equal(t, map[string]string{"name": string(types.String)}, res.Props)
	})

	// prop <op> value
	compileProps(t, "props.name == 'bob'", map[string]*llx.Primitive{
		"name": {Type: string(types.String)},
	}, func(res *llx.CodeBundle) {
		assertProperty(t, "name", types.String, res.DeprecatedV5Code.Code[0])
		assertFunction(t, "=="+string(types.String), &llx.Function{
			Type:                string(types.Bool),
			DeprecatedV5Binding: 1,
			Args:                []*llx.Primitive{llx.StringPrimitive("bob")},
		}, res.DeprecatedV5Code.Code[1])
		assert.Equal(t, []int32{2}, res.DeprecatedV5Code.Entrypoints)
		assert.Equal(t, map[string]string{"name": string(types.String)}, res.Props)
	})

	// different compile stages yielding the same checksums
	compileProps(t, "props.name == 'bob'", map[string]*llx.Primitive{
		"name": {Type: string(types.String)},
	}, func(res1 *llx.CodeBundle) {
		compileProps(t, "props.name == 'bob'", map[string]*llx.Primitive{
			"name": {Type: string(types.String), Value: []byte("yoman")},
		}, func(res2 *llx.CodeBundle) {
			assert.Equal(t, res2.DeprecatedV5Code.Id, res1.DeprecatedV5Code.Id)
		})
	})

	compileProps(t, "props.name == props.name", map[string]*llx.Primitive{
		"name": {Type: string(types.String)},
	}, func(res *llx.CodeBundle) {
		assertProperty(t, "name", types.String, res.DeprecatedV5Code.Code[0])
		assertProperty(t, "name", types.String, res.DeprecatedV5Code.Code[1])
		assertFunction(t, "=="+string(types.String), &llx.Function{
			Type:                string(types.Bool),
			DeprecatedV5Binding: 1,
			Args:                []*llx.Primitive{llx.RefPrimitiveV1(2)},
		}, res.DeprecatedV5Code.Code[2])
		assert.Equal(t, []int32{3}, res.DeprecatedV5Code.Entrypoints)
		assert.Equal(t, map[string]string{"name": string(types.String)}, res.Props)
	})
}

func TestCompiler_If(t *testing.T) {
	compile(t, "if ( mondoo ) { return 123 } if ( true ) { return 456 } 789", func(res *llx.CodeBundle) {
		assertFunction(t, "mondoo", nil, res.DeprecatedV5Code.Code[0])

		functions := res.DeprecatedV5Code.Functions

		assertFunction(t, "if", &llx.Function{
			Type:                string(types.Int),
			DeprecatedV5Binding: 0,
			Args: []*llx.Primitive{
				llx.RefPrimitiveV1(1),
				llx.FunctionPrimitiveV1(1),
				llx.FunctionPrimitiveV1(2),
			},
		}, res.DeprecatedV5Code.Code[1])
		assert.Equal(t, []int32{2}, res.DeprecatedV5Code.Entrypoints)
		assert.Equal(t, []int32(nil), res.DeprecatedV5Code.Datapoints)

		assertPrimitive(t, llx.IntPrimitive(123), functions[0].Code[0])
		assertFunction(t, "return", &llx.Function{
			Type:                string(types.Int),
			DeprecatedV5Binding: 0,
			Args: []*llx.Primitive{
				llx.RefPrimitiveV1(1),
			},
		}, functions[0].Code[1])
		assert.Equal(t, []int32{2}, functions[0].Entrypoints)

		assertFunction(t, "if", &llx.Function{
			Type:                string(types.Int),
			DeprecatedV5Binding: 0,
			Args: []*llx.Primitive{
				llx.BoolPrimitive(true),
				llx.FunctionPrimitiveV1(1),
				llx.FunctionPrimitiveV1(2),
			},
		}, functions[1].Code[0])
		assert.Equal(t, []int32{1}, functions[1].Entrypoints)

		assertPrimitive(t, llx.IntPrimitive(456), functions[1].Functions[0].Code[0])
		assertFunction(t, "return", &llx.Function{
			Type:                string(types.Int),
			DeprecatedV5Binding: 0,
			Args: []*llx.Primitive{
				llx.RefPrimitiveV1(1),
			},
		}, functions[1].Functions[0].Code[1])
		assert.Equal(t, []int32{2}, functions[1].Functions[0].Entrypoints)

		assertPrimitive(t, llx.IntPrimitive(789), functions[1].Functions[1].Code[0])
		assert.Equal(t, []int32{1}, functions[1].Functions[1].Entrypoints)
	})

	compile(t, "if ( mondoo ) { return 123 } 456", func(res *llx.CodeBundle) {
		assertFunction(t, "mondoo", nil, res.DeprecatedV5Code.Code[0])

		assertFunction(t, "if", &llx.Function{
			Type:                string(types.Int),
			DeprecatedV5Binding: 0,
			Args: []*llx.Primitive{
				llx.RefPrimitiveV1(1),
				llx.FunctionPrimitiveV1(1),
				llx.FunctionPrimitiveV1(2),
			},
		}, res.DeprecatedV5Code.Code[1])
		assert.Equal(t, []int32{2}, res.DeprecatedV5Code.Entrypoints)
		assert.Equal(t, []int32(nil), res.DeprecatedV5Code.Datapoints)

		functions := res.DeprecatedV5Code.Functions
		assertPrimitive(t, llx.IntPrimitive(123), functions[0].Code[0])
		assertFunction(t, "return", &llx.Function{
			Type:                string(types.Int),
			DeprecatedV5Binding: 0,
			Args: []*llx.Primitive{
				llx.RefPrimitiveV1(1),
			},
		}, functions[0].Code[1])
		assert.Equal(t, []int32{2}, functions[0].Entrypoints)

		assertPrimitive(t, llx.IntPrimitive(456), functions[1].Code[0])
		assert.Equal(t, []int32{1}, functions[1].Entrypoints)
	})

	// Test empty array with filled array and type-consolidation in the compiler
	compile(t, "if ( mondoo ) { return [] } return [1,2,3]", func(res *llx.CodeBundle) {
		assertFunction(t, "mondoo", nil, res.DeprecatedV5Code.Code[0])

		assertFunction(t, "if", &llx.Function{
			Type:                string(types.Array(types.Int)),
			DeprecatedV5Binding: 0,
			Args: []*llx.Primitive{
				llx.RefPrimitiveV1(1),
				llx.FunctionPrimitiveV1(1),
				llx.FunctionPrimitiveV1(2),
			},
		}, res.DeprecatedV5Code.Code[1])
		assert.Equal(t, []int32{2}, res.DeprecatedV5Code.Entrypoints)
		assert.Equal(t, []int32(nil), res.DeprecatedV5Code.Datapoints)

		functions := res.DeprecatedV5Code.Functions
		assertPrimitive(t, llx.ArrayPrimitive([]*llx.Primitive{}, types.Unset), functions[0].Code[0])
		assertFunction(t, "return", &llx.Function{
			Type:                string(types.Array(types.Unset)),
			DeprecatedV5Binding: 0,
			Args: []*llx.Primitive{
				llx.RefPrimitiveV1(1),
			},
		}, functions[0].Code[1])
		assert.Equal(t, []int32{2}, functions[0].Entrypoints)
	})

	compile(t, "if ( mondoo.version != null ) { 123 }", func(res *llx.CodeBundle) {
		assertFunction(t, "mondoo", nil, res.DeprecatedV5Code.Code[0])
		assertFunction(t, "version", &llx.Function{
			Type:                string(types.String),
			DeprecatedV5Binding: 1,
		}, res.DeprecatedV5Code.Code[1])
		assertFunction(t, "!=\x02", &llx.Function{
			Type:                string(types.Bool),
			DeprecatedV5Binding: 2,
			Args:                []*llx.Primitive{llx.NilPrimitive},
		}, res.DeprecatedV5Code.Code[2])

		assertFunction(t, "if", &llx.Function{
			Type:                string(types.Block),
			DeprecatedV5Binding: 0,
			Args: []*llx.Primitive{
				llx.RefPrimitiveV1(3),
				llx.FunctionPrimitiveV1(1),
			},
		}, res.DeprecatedV5Code.Code[3])
		assert.Equal(t, []int32{4}, res.DeprecatedV5Code.Entrypoints)
		assert.Equal(t, []int32{2}, res.DeprecatedV5Code.Datapoints)

		functions := res.DeprecatedV5Code.Functions
		assertPrimitive(t, llx.IntPrimitive(123), functions[0].Code[0])
		assert.Equal(t, []int32{1}, functions[0].Entrypoints)
	})

	compile(t, "if ( mondoo ) { 123 } else { 456 }", func(res *llx.CodeBundle) {
		assertFunction(t, "mondoo", nil, res.DeprecatedV5Code.Code[0])

		assertFunction(t, "if", &llx.Function{
			Type:                string(types.Block),
			DeprecatedV5Binding: 0,
			Args: []*llx.Primitive{
				llx.RefPrimitiveV1(1),
				llx.FunctionPrimitiveV1(1),
				llx.FunctionPrimitiveV1(2),
			},
		}, res.DeprecatedV5Code.Code[1])
		assert.Equal(t, []int32{2}, res.DeprecatedV5Code.Entrypoints)
		assert.Equal(t, []int32(nil), res.DeprecatedV5Code.Datapoints)

		functions := res.DeprecatedV5Code.Functions

		assertPrimitive(t, llx.IntPrimitive(123), functions[0].Code[0])
		assert.Equal(t, []int32{1}, functions[0].Entrypoints)

		assertPrimitive(t, llx.IntPrimitive(456), functions[1].Code[0])
		assert.Equal(t, []int32{1}, functions[1].Entrypoints)
	})

	compile(t, "if ( mondoo ) { 123 } else if ( true ) { 456 } else { 789 }", func(res *llx.CodeBundle) {
		assertFunction(t, "mondoo", nil, res.DeprecatedV5Code.Code[0])

		assertFunction(t, "if", &llx.Function{
			Type:                string(types.Block),
			DeprecatedV5Binding: 0,
			Args: []*llx.Primitive{
				llx.RefPrimitiveV1(1),
				llx.FunctionPrimitiveV1(1),
				llx.BoolPrimitive(true),
				llx.FunctionPrimitiveV1(2),
				llx.FunctionPrimitiveV1(3),
			},
		}, res.DeprecatedV5Code.Code[1])
		assert.Equal(t, []int32{2}, res.DeprecatedV5Code.Entrypoints)
		assert.Equal(t, []int32(nil), res.DeprecatedV5Code.Datapoints)

		functions := res.DeprecatedV5Code.Functions

		assertPrimitive(t, llx.IntPrimitive(123), functions[0].Code[0])
		assert.Equal(t, []int32{1}, functions[0].Entrypoints)

		assertPrimitive(t, llx.IntPrimitive(456), functions[1].Code[0])
		assert.Equal(t, []int32{1}, functions[1].Entrypoints)

		assertPrimitive(t, llx.IntPrimitive(789), functions[2].Code[0])
		assert.Equal(t, []int32{1}, functions[2].Entrypoints)
	})
}

func TestCompiler_Switch(t *testing.T) {
	compile(t, "switch ( 1 ) { case _ > 0: true; default: false }", func(res *llx.CodeBundle) {
		assertFunction(t, "switch", &llx.Function{
			Type:                string(types.Unset),
			DeprecatedV5Binding: 0,
			Args: []*llx.Primitive{
				llx.IntPrimitive(1),
				llx.RefPrimitiveV1(2),
				llx.FunctionPrimitiveV1(1),
				llx.BoolPrimitive(true),
				llx.FunctionPrimitiveV1(2),
			},
		}, res.DeprecatedV5Code.Code[2])
		assert.Equal(t, []int32{3}, res.DeprecatedV5Code.Entrypoints)
		assert.Equal(t, []int32(nil), res.DeprecatedV5Code.Datapoints)
	})
}

//    =======================
//   👋   ARRAYS and MAPS   🍹
//    =======================

func TestCompiler_ArrayEmptyWhere(t *testing.T) {
	compile(t, "[1,2,3].where()", func(res *llx.CodeBundle) {
		assertPrimitive(t, &llx.Primitive{
			Type: string(types.Array(types.Int)),
			Array: []*llx.Primitive{
				llx.IntPrimitive(1),
				llx.IntPrimitive(2),
				llx.IntPrimitive(3),
			},
		}, res.DeprecatedV5Code.Code[0])
		assert.Equal(t, 1, len(res.DeprecatedV5Code.Code))
	})
}

func TestCompiler_ArrayWhereStatic(t *testing.T) {
	compileErroneous(t, "[1,2,3].where(sshd)", errors.New("called 'where' with wrong type; either provide a type int value or write it as an expression (e.g. \"_ == 123\")"), func(res *llx.CodeBundle) {
		assertPrimitive(t, &llx.Primitive{
			Type: string(types.Array(types.Int)),
			Array: []*llx.Primitive{
				llx.IntPrimitive(1),
				llx.IntPrimitive(2),
				llx.IntPrimitive(3),
			},
		}, res.DeprecatedV5Code.Code[0])
	})

	compile(t, "[1,2,3].where(2)", func(res *llx.CodeBundle) {
		assertPrimitive(t, &llx.Primitive{
			Type: string(types.Array(types.Int)),
			Array: []*llx.Primitive{
				llx.IntPrimitive(1),
				llx.IntPrimitive(2),
				llx.IntPrimitive(3),
			},
		}, res.DeprecatedV5Code.Code[0])

		assertFunction(t, "where", &llx.Function{
			Type:                string(types.Array(types.Int)),
			DeprecatedV5Binding: 1,
			Args: []*llx.Primitive{
				llx.RefPrimitiveV1(1),
				llx.FunctionPrimitiveV1(1),
			},
		}, res.DeprecatedV5Code.Code[1])

		assert.Equal(t, 2, len(res.DeprecatedV5Code.Code))
	})
}

func TestCompiler_ArrayContains(t *testing.T) {
	compile(t, "[1,2,3].contains(_ == 2)", func(res *llx.CodeBundle) {
		assertPrimitive(t, &llx.Primitive{
			Type: string(types.Array(types.Int)),
			Array: []*llx.Primitive{
				llx.IntPrimitive(1),
				llx.IntPrimitive(2),
				llx.IntPrimitive(3),
			},
		}, res.DeprecatedV5Code.Code[0])

		assertFunction(t, "where", &llx.Function{
			Type:                string(types.Array(types.Int)),
			DeprecatedV5Binding: 1,
			Args: []*llx.Primitive{
				llx.RefPrimitiveV1(1),
				llx.FunctionPrimitiveV1(1),
			},
		}, res.DeprecatedV5Code.Code[1])

		assertFunction(t, "length", &llx.Function{
			Type:                string(types.Int),
			DeprecatedV5Binding: 2,
		}, res.DeprecatedV5Code.Code[2])
		assertFunction(t, string(">"+types.Int), &llx.Function{
			Type:                string(types.Bool),
			DeprecatedV5Binding: 3,
			Args:                []*llx.Primitive{llx.IntPrimitive(0)},
		}, res.DeprecatedV5Code.Code[3])

		assert.Equal(t, 4, len(res.DeprecatedV5Code.Code))
	})
}

func TestCompiler_ArrayOne(t *testing.T) {
	compile(t, "[1,2,3].one(_ == 2)", func(res *llx.CodeBundle) {
		assertPrimitive(t, &llx.Primitive{
			Type: string(types.Array(types.Int)),
			Array: []*llx.Primitive{
				llx.IntPrimitive(1),
				llx.IntPrimitive(2),
				llx.IntPrimitive(3),
			},
		}, res.DeprecatedV5Code.Code[0])

		assertFunction(t, "where", &llx.Function{
			Type:                string(types.Array(types.Int)),
			DeprecatedV5Binding: 1,
			Args: []*llx.Primitive{
				llx.RefPrimitiveV1(1),
				llx.FunctionPrimitiveV1(1),
			},
		}, res.DeprecatedV5Code.Code[1])

		assertFunction(t, "$one", &llx.Function{
			Type:                string(types.Bool),
			DeprecatedV5Binding: 2,
		}, res.DeprecatedV5Code.Code[2])
		assert.Equal(t, 3, len(res.DeprecatedV5Code.Code))
	})
}

func TestCompiler_ArrayAll(t *testing.T) {
	compile(t, "[1,2,3].all(_ < 9)", func(res *llx.CodeBundle) {
		assertPrimitive(t, &llx.Primitive{
			Type: string(types.Array(types.Int)),
			Array: []*llx.Primitive{
				llx.IntPrimitive(1),
				llx.IntPrimitive(2),
				llx.IntPrimitive(3),
			},
		}, res.DeprecatedV5Code.Code[0])

		assertFunction(t, "$whereNot", &llx.Function{
			Type:                string(types.Array(types.Int)),
			DeprecatedV5Binding: 1,
			Args: []*llx.Primitive{
				llx.RefPrimitiveV1(1),
				llx.FunctionPrimitiveV1(1),
			},
		}, res.DeprecatedV5Code.Code[1])

		assertFunction(t, "$all", &llx.Function{
			Type:                string(types.Bool),
			DeprecatedV5Binding: 2,
		}, res.DeprecatedV5Code.Code[2])

		assert.Equal(t, 3, len(res.DeprecatedV5Code.Code))
	})
}

//    =================
//   👋   RESOURCES   🍹
//    =================

func TestCompiler_Resource(t *testing.T) {
	compile(t, "sshd", func(res *llx.CodeBundle) {
		assertFunction(t, "sshd", nil, res.DeprecatedV5Code.Code[0])
	})
}

func TestCompiler_Resource_versioning(t *testing.T) {
	compile(t, "sshd", func(res *llx.CodeBundle) {
		assert.Equal(t, "5.15.0", res.MinMondooVersion)
	})
}

func TestCompiler_Resource_versioning2(t *testing.T) {
	compile(t, "file.empty", func(res *llx.CodeBundle) {
		assert.Equal(t, "5.18.0", res.MinMondooVersion)
	})
}

func TestCompiler_ResourceWithCall(t *testing.T) {
	compile(t, "sshd()", func(res *llx.CodeBundle) {
		assertFunction(t, "sshd", nil, res.DeprecatedV5Code.Code[0])
	})
}

func TestCompiler_LongResource(t *testing.T) {
	compile(t, "sshd.config", func(res *llx.CodeBundle) {
		assertFunction(t, "sshd.config", nil, res.DeprecatedV5Code.Code[0])
	})
}

func TestCompiler_ResourceMap(t *testing.T) {
	compile(t, "sshd.config.params", func(res *llx.CodeBundle) {
		assertFunction(t, "sshd.config", nil, res.DeprecatedV5Code.Code[0])
		assert.Equal(t, "5.15.0", res.MinMondooVersion)
		assertFunction(t, "params", &llx.Function{
			Type:                string(types.Map(types.String, types.String)),
			DeprecatedV5Binding: 1,
		}, res.DeprecatedV5Code.Code[1])
	})
}

func TestCompiler_ResourceMapLength(t *testing.T) {
	compile(t, "sshd.config.params.length", func(res *llx.CodeBundle) {
		assertFunction(t, "length", &llx.Function{
			Type:                string(types.Int),
			DeprecatedV5Binding: 2,
		}, res.DeprecatedV5Code.Code[2])
	})
}

func TestCompiler_ResourceArrayAccessor(t *testing.T) {
	compile(t, "packages.list[123]", func(res *llx.CodeBundle) {
		assertFunction(t, "[]", &llx.Function{
			DeprecatedV5Binding: 2,
			Args:                []*llx.Primitive{llx.IntPrimitive(123)},
			Type:                string(types.Resource("package")),
		}, res.DeprecatedV5Code.Code[2])
	})
}

func TestCompiler_ResourceArrayLength(t *testing.T) {
	compile(t, "packages.list.length", func(res *llx.CodeBundle) {
		assertFunction(t, "length", &llx.Function{
			DeprecatedV5Binding: 2,
			Type:                string(types.Int),
		}, res.DeprecatedV5Code.Code[2])
	})
}

func TestCompiler_ResourceArrayImplicitLength(t *testing.T) {
	compile(t, "packages.length", func(res *llx.CodeBundle) {
		assertFunction(t, "list", &llx.Function{
			DeprecatedV5Binding: 1,
			Type:                string(types.Array(types.Resource("package"))),
		}, res.DeprecatedV5Code.Code[1])
		assertFunction(t, "length", &llx.Function{
			DeprecatedV5Binding: 1,
			Args:                []*llx.Primitive{llx.RefPrimitiveV1(2)},
			Type:                string(types.Int),
		}, res.DeprecatedV5Code.Code[2])
	})
}

func TestCompiler_ResourceFieldGlob(t *testing.T) {
	compile(t, "pam.conf { * }", func(res *llx.CodeBundle) {
		assertFunction(t, "pam.conf", nil, res.DeprecatedV5Code.Code[0])
		assertFunction(t, "{}", &llx.Function{
			Type:                string(types.Block),
			DeprecatedV5Binding: 1,
			Args:                []*llx.Primitive{llx.FunctionPrimitiveV1(1)},
		}, res.DeprecatedV5Code.Code[1])
		assert.Equal(t, []int32{2}, res.DeprecatedV5Code.Entrypoints)

		functions := res.DeprecatedV5Code.Functions

		assertPrimitive(t, &llx.Primitive{
			Type: string(types.Resource("pam.conf")),
		}, functions[0].Code[0])
		assertFunction(t, "content", &llx.Function{
			Type:                string(types.String),
			DeprecatedV5Binding: 1,
		}, functions[0].Code[1])
		assertFunction(t, "entries", &llx.Function{
			Type:                string(types.Map(types.String, types.Array(types.Resource("pam.conf.serviceEntry")))),
			DeprecatedV5Binding: 1,
		}, functions[0].Code[2])
		assertFunction(t, "files", &llx.Function{
			Type:                string(types.Array(types.Resource("file"))),
			DeprecatedV5Binding: 1,
		}, functions[0].Code[3])
		assertFunction(t, "services", &llx.Function{
			Type:                string(types.Map(types.String, types.Array(types.String))),
			DeprecatedV5Binding: 1,
		}, functions[0].Code[4])
		assert.Equal(t, []int32{2, 3, 4, 5}, functions[0].Entrypoints)
	})
}

func TestCompiler_ArrayResourceFieldGlob(t *testing.T) {
	compile(t, "groups.list { * }", func(res *llx.CodeBundle) {
		assertFunction(t, "groups", nil, res.DeprecatedV5Code.Code[0])
		assertFunction(t, "list", &llx.Function{
			Type:                string(types.Array(types.Resource("group"))),
			DeprecatedV5Binding: 1,
		}, res.DeprecatedV5Code.Code[1])
		assertFunction(t, "{}", &llx.Function{
			Type:                string(types.Array(types.Block)),
			DeprecatedV5Binding: 2,
			Args:                []*llx.Primitive{llx.FunctionPrimitiveV1(1)},
		}, res.DeprecatedV5Code.Code[2])
		assert.Equal(t, []int32{3}, res.DeprecatedV5Code.Entrypoints)

		functions := res.DeprecatedV5Code.Functions

		assertPrimitive(t, &llx.Primitive{
			Type: string(types.Resource("group")),
		}, functions[0].Code[0])
		assertFunction(t, "gid", &llx.Function{
			Type:                string(types.Int),
			DeprecatedV5Binding: 1,
		}, functions[0].Code[1])
		assertFunction(t, "members", &llx.Function{
			Type:                string(types.Array(types.Resource("user"))),
			DeprecatedV5Binding: 1,
		}, functions[0].Code[2])
		assertFunction(t, "name", &llx.Function{
			Type:                string(types.String),
			DeprecatedV5Binding: 1,
		}, functions[0].Code[3])
		assertFunction(t, "sid", &llx.Function{
			Type:                string(types.String),
			DeprecatedV5Binding: 1,
		}, functions[0].Code[4])
		assert.Equal(t, []int32{2, 3, 4, 5}, functions[0].Entrypoints)
	})
}

func TestCompiler_ResourceFieldArrayAccessor(t *testing.T) {
	compile(t, "sshd.config.params[\"Protocol\"]", func(res *llx.CodeBundle) {
		assertFunction(t, "[]", &llx.Function{
			Type:                string(types.String),
			DeprecatedV5Binding: 2,
			Args: []*llx.Primitive{
				llx.StringPrimitive("Protocol"),
			},
		}, res.DeprecatedV5Code.Code[2])
	})
}

func TestCompiler_ResourceWithUnnamedArgs(t *testing.T) {
	compile(t, "file(\"/path\")", func(res *llx.CodeBundle) {
		assertFunction(t, "file", &llx.Function{
			Type:                string(types.Resource("file")),
			DeprecatedV5Binding: 0,
			Args: []*llx.Primitive{
				llx.StringPrimitive("path"),
				llx.StringPrimitive("/path"),
			},
		}, res.DeprecatedV5Code.Code[0])
	})
}

func TestCompiler_ResourceWithNamedArgs(t *testing.T) {
	compile(t, "file(path: \"/path\")", func(res *llx.CodeBundle) {
		assertFunction(t, "file", &llx.Function{
			Type:                string(types.Resource("file")),
			DeprecatedV5Binding: 0,
			Args: []*llx.Primitive{
				llx.StringPrimitive("path"),
				llx.StringPrimitive("/path"),
			},
		}, res.DeprecatedV5Code.Code[0])
	})
}

func TestCompiler_LongResourceWithUnnamedArgs(t *testing.T) {
	compile(t, "sshd.config(\"/path\")", func(res *llx.CodeBundle) {
		assertFunction(t, "sshd.config", &llx.Function{
			Type:                string(types.Resource("sshd.config")),
			DeprecatedV5Binding: 0,
			Args: []*llx.Primitive{
				llx.StringPrimitive("path"),
				llx.StringPrimitive("/path"),
			},
		}, res.DeprecatedV5Code.Code[0])
	})
}

func TestCompiler_ExpectSimplest(t *testing.T) {
	compile(t, "expect(true)", func(res *llx.CodeBundle) {
		f := res.DeprecatedV5Code.Code[0]
		assert.Equal(t, llx.Chunk_FUNCTION, f.Call)
		assert.Equal(t, "expect", f.Id)
		assert.Equal(t, []int32{1}, res.DeprecatedV5Code.Entrypoints)
		assert.Equal(t, &llx.Function{
			Type:                string(types.Bool),
			DeprecatedV5Binding: 0,
			Args:                []*llx.Primitive{llx.BoolPrimitive(true)},
		}, f.Function)
	})
}

func TestCompiler_ExpectEq(t *testing.T) {
	compile(t, "expect(1 == \"1\")", func(res *llx.CodeBundle) {
		cmp := res.DeprecatedV5Code.Code[1]
		assert.Equal(t, llx.Chunk_FUNCTION, cmp.Call)
		assert.Equal(t, []int32{3}, res.DeprecatedV5Code.Entrypoints)
		assert.Equal(t, string("=="+types.String), cmp.Id)
		assert.Equal(t, &llx.Function{
			Type:                string(types.Bool),
			DeprecatedV5Binding: 1,
			Args: []*llx.Primitive{
				llx.StringPrimitive("1"),
			},
		}, cmp.Function)

		f := res.DeprecatedV5Code.Code[2]
		assert.Equal(t, llx.Chunk_FUNCTION, f.Call)
		assert.Equal(t, "expect", f.Id)
		assert.Equal(t, &llx.Function{
			Type:                string(types.Bool),
			DeprecatedV5Binding: 0,
			Args:                []*llx.Primitive{llx.RefPrimitiveV1(2)},
		}, f.Function)
	})
}

func TestCompiler_EmptyBlock(t *testing.T) {
	compile(t, "mondoo { }", func(res *llx.CodeBundle) {
		assertFunction(t, "mondoo", nil, res.DeprecatedV5Code.Code[0])
		assert.Equal(t, 1, len(res.DeprecatedV5Code.Code))
		assert.Nil(t, res.DeprecatedV5Code.Functions)
	})
}

func TestCompiler_Block(t *testing.T) {
	compile(t, "mondoo { version build }", func(res *llx.CodeBundle) {
		assertFunction(t, "mondoo", nil, res.DeprecatedV5Code.Code[0])
		assertFunction(t, "{}", &llx.Function{
			Type:                string(types.Block),
			DeprecatedV5Binding: 1,
			Args:                []*llx.Primitive{llx.FunctionPrimitiveV1(1)},
		}, res.DeprecatedV5Code.Code[1])
		assert.Equal(t, []int32{2}, res.DeprecatedV5Code.Entrypoints)

		functions := res.DeprecatedV5Code.Functions

		assertPrimitive(t, &llx.Primitive{
			Type: string(types.Resource("mondoo")),
		}, functions[0].Code[0])
		assertFunction(t, "version", &llx.Function{
			Type:                string(types.String),
			DeprecatedV5Binding: 1,
		}, functions[0].Code[1])
		assertFunction(t, "build", &llx.Function{
			Type:                string(types.String),
			DeprecatedV5Binding: 1,
		}, functions[0].Code[2])
		assert.Equal(t, []int32{2, 3}, functions[0].Entrypoints)
	})
}

func TestCompiler_BlockWithSelf(t *testing.T) {
	compile(t, "mondoo { _.version }", func(res *llx.CodeBundle) {
		assertFunction(t, "mondoo", nil, res.DeprecatedV5Code.Code[0])
		assertFunction(t, "{}", &llx.Function{
			Type:                string(types.Block),
			DeprecatedV5Binding: 1,
			Args:                []*llx.Primitive{llx.FunctionPrimitiveV1(1)},
		}, res.DeprecatedV5Code.Code[1])
		assert.Equal(t, []int32{2}, res.DeprecatedV5Code.Entrypoints)

		functions := res.DeprecatedV5Code.Functions

		assertPrimitive(t, &llx.Primitive{
			Type: string(types.Resource("mondoo")),
		}, functions[0].Code[0])
		assertFunction(t, "version", &llx.Function{
			Type:                string(types.String),
			DeprecatedV5Binding: 1,
		}, functions[0].Code[1])
		assert.Equal(t, []int32{2}, functions[0].Entrypoints)
	})

	compile(t, "sshd.config.params { _['A'] != _['B'] }", func(res *llx.CodeBundle) {
		assertFunction(t, "sshd.config", nil, res.DeprecatedV5Code.Code[0])
		assertFunction(t, "params", &llx.Function{
			Type:                string(types.Map(types.String, types.String)),
			DeprecatedV5Binding: 1,
		}, res.DeprecatedV5Code.Code[1])
		assertFunction(t, "{}", &llx.Function{
			Type:                string(types.Block),
			DeprecatedV5Binding: 2,
			Args:                []*llx.Primitive{llx.FunctionPrimitiveV1(1)},
		}, res.DeprecatedV5Code.Code[2])
		assert.Equal(t, []int32{3}, res.DeprecatedV5Code.Entrypoints)

		functions := res.DeprecatedV5Code.Functions

		assertPrimitive(t, &llx.Primitive{
			Type: string(types.Map(types.String, types.String)),
		}, functions[0].Code[0])
		assertFunction(t, "[]", &llx.Function{
			Type:                string(types.String),
			DeprecatedV5Binding: 1,
			Args:                []*llx.Primitive{llx.StringPrimitive("A")},
		}, functions[0].Code[1])
		assertFunction(t, "[]", &llx.Function{
			Type:                string(types.String),
			DeprecatedV5Binding: 1,
			Args:                []*llx.Primitive{llx.StringPrimitive("B")},
		}, functions[0].Code[2])
		assertFunction(t, string("!="+types.String), &llx.Function{
			Type:                string(types.Bool),
			DeprecatedV5Binding: 2,
			Args:                []*llx.Primitive{llx.RefPrimitiveV1(3)},
		}, functions[0].Code[3])
		assert.Equal(t, []int32{4}, functions[0].Entrypoints)
	})

	compile(t, "\"alice\\nbob\".lines { _ != \"alice\" && _ != \"bob\" }", func(res *llx.CodeBundle) {
		assertPrimitive(t, llx.StringPrimitive("alice\nbob"), res.DeprecatedV5Code.Code[0])
		assertFunction(t, "lines", &llx.Function{
			Type:                string(types.Array(types.String)),
			DeprecatedV5Binding: 1,
		}, res.DeprecatedV5Code.Code[1])
		assertFunction(t, "{}", &llx.Function{
			Type:                string(types.Array(types.Block)),
			DeprecatedV5Binding: 2,
			Args:                []*llx.Primitive{llx.FunctionPrimitiveV1(1)},
		}, res.DeprecatedV5Code.Code[2])
		assert.Equal(t, []int32{3}, res.DeprecatedV5Code.Entrypoints)

		functions := res.DeprecatedV5Code.Functions

		assertPrimitive(t, &llx.Primitive{
			Type: string(types.String),
		}, functions[0].Code[0])
		assertFunction(t, string("!="+types.String), &llx.Function{
			Type:                string(types.Bool),
			DeprecatedV5Binding: 1,
			Args:                []*llx.Primitive{llx.StringPrimitive("alice")},
		}, functions[0].Code[1])
		assertFunction(t, string("!="+types.String), &llx.Function{
			Type:                string(types.Bool),
			DeprecatedV5Binding: 1,
			Args:                []*llx.Primitive{llx.StringPrimitive("bob")},
		}, functions[0].Code[2])
		assertFunction(t, string("&&"+types.Bool), &llx.Function{
			Type:                string(types.Bool),
			DeprecatedV5Binding: 2,
			Args:                []*llx.Primitive{llx.RefPrimitiveV1(3)},
		}, functions[0].Code[3])
		assert.Equal(t, []int32{4}, functions[0].Entrypoints)
	})
}

func TestCompiler_ContainsWithResource(t *testing.T) {
	compile(t, "'hello'.contains(platform.family)", func(res *llx.CodeBundle) {
		assertPrimitive(t, llx.StringPrimitive("hello"), res.DeprecatedV5Code.Code[0])
		assertFunction(t, "platform", nil, res.DeprecatedV5Code.Code[1])
		assertFunction(t, "family", &llx.Function{
			Type:                string(types.Array(types.String)),
			DeprecatedV5Binding: 2,
		}, res.DeprecatedV5Code.Code[2])
		assertFunction(t, "contains"+string(types.Array(types.String)), &llx.Function{
			Type:                string(types.Bool),
			DeprecatedV5Binding: 1,
			Args:                []*llx.Primitive{llx.RefPrimitiveV1(3)},
		}, res.DeprecatedV5Code.Code[3])

		assert.Equal(t, []int32{4}, res.DeprecatedV5Code.Entrypoints)
	})
}

func TestCompiler_StringContainsWithInt(t *testing.T) {
	compile(t, "'hello123'.contains(23)", func(res *llx.CodeBundle) {
		assertPrimitive(t, llx.StringPrimitive("hello123"), res.DeprecatedV5Code.Code[0])
		assertFunction(t, "contains"+string(types.Int), &llx.Function{
			Type:                string(types.Bool),
			DeprecatedV5Binding: 1,
			Args:                []*llx.Primitive{llx.IntPrimitive(23)},
		}, res.DeprecatedV5Code.Code[1])

		assert.Equal(t, []int32{2}, res.DeprecatedV5Code.Entrypoints)
	})
}

func TestCompiler_CallWithResource(t *testing.T) {
	compile(t, "users.list { file(home) }", func(res *llx.CodeBundle) {
		assertFunction(t, "users", nil, res.DeprecatedV5Code.Code[0])
		assertFunction(t, "list", &llx.Function{
			Type:                string(types.Array(types.Resource("user"))),
			DeprecatedV5Binding: 1,
		}, res.DeprecatedV5Code.Code[1])
		assertFunction(t, "{}", &llx.Function{
			Type:                string(types.Array(types.Block)),
			DeprecatedV5Binding: 2,
			Args:                []*llx.Primitive{llx.FunctionPrimitiveV1(1)},
		}, res.DeprecatedV5Code.Code[2])
		assert.Equal(t, 3, len(res.DeprecatedV5Code.Code))

		functions := res.DeprecatedV5Code.Functions

		assertPrimitive(t, &llx.Primitive{
			Type: string(types.Resource("user")),
		}, functions[0].Code[0])
		assertFunction(t, "home", &llx.Function{
			Type:                string(types.String),
			DeprecatedV5Binding: 1,
		}, functions[0].Code[1])
		assertFunction(t, "file", &llx.Function{
			Type:                string(types.Resource("file")),
			DeprecatedV5Binding: 0,
			Args: []*llx.Primitive{
				llx.StringPrimitive("path"),
				llx.RefPrimitiveV1(2),
			},
		}, functions[0].Code[2])
	})
}

func TestCompiler_List(t *testing.T) {
	compile(t, "packages.list { name }", func(res *llx.CodeBundle) {
		assertFunction(t, "packages", nil, res.DeprecatedV5Code.Code[0])
		assertFunction(t, "list", &llx.Function{
			Type:                string(types.Array(types.Resource("package"))),
			DeprecatedV5Binding: 1,
		}, res.DeprecatedV5Code.Code[1])
		assertFunction(t, "{}", &llx.Function{
			Type:                string(types.Array(types.Block)),
			DeprecatedV5Binding: 2,
			Args:                []*llx.Primitive{llx.FunctionPrimitiveV1(1)},
		}, res.DeprecatedV5Code.Code[2])
		assert.Equal(t, 3, len(res.DeprecatedV5Code.Code))

		functions := res.DeprecatedV5Code.Functions

		assertPrimitive(t, &llx.Primitive{
			Type: string(types.Resource("package")),
		}, functions[0].Code[0])
		assertFunction(t, "name", &llx.Function{
			Type:                string(types.String),
			DeprecatedV5Binding: 1,
		}, functions[0].Code[1])
		assert.Equal(t, []int32{2}, functions[0].Entrypoints)
	})
}

func TestCompiler_ResourceEmptyWhere(t *testing.T) {
	compile(t, "packages.where()", func(res *llx.CodeBundle) {
		assertFunction(t, "packages", nil, res.DeprecatedV5Code.Code[0])
		assert.Equal(t, 1, len(res.DeprecatedV5Code.Code))
	})
}

func TestCompiler_ResourceWhere(t *testing.T) {
	compile(t, "packages.where(outdated)", func(res *llx.CodeBundle) {
		assertFunction(t, "packages", nil, res.DeprecatedV5Code.Code[0])
		assertFunction(t, "list", &llx.Function{
			Type:                string(types.Array(types.Resource("package"))),
			DeprecatedV5Binding: 1,
		}, res.DeprecatedV5Code.Code[1])
		assertFunction(t, "where", &llx.Function{
			Type:                string(types.Resource("packages")),
			DeprecatedV5Binding: 1,
			Args: []*llx.Primitive{
				llx.RefPrimitiveV1(2),
				llx.FunctionPrimitiveV1(1),
			},
		}, res.DeprecatedV5Code.Code[2])

		functions := res.DeprecatedV5Code.Functions

		assertPrimitive(t, &llx.Primitive{
			Type: string(types.Resource("package")),
		}, functions[0].Code[0])
		assertFunction(t, "outdated", &llx.Function{
			Type:                string(types.Bool),
			DeprecatedV5Binding: 1,
		}, functions[0].Code[1])
		assert.Equal(t, []int32{2}, functions[0].Entrypoints)
	})
}

func TestCompiler_ResourceContains(t *testing.T) {
	compile(t, "packages.contains(outdated)", func(res *llx.CodeBundle) {
		assertFunction(t, "packages", nil, res.DeprecatedV5Code.Code[0])
		assertFunction(t, "list", &llx.Function{
			Type:                string(types.Array(types.Resource("package"))),
			DeprecatedV5Binding: 1,
		}, res.DeprecatedV5Code.Code[1])
		assertFunction(t, "where", &llx.Function{
			Type:                string(types.Resource("packages")),
			DeprecatedV5Binding: 1,
			Args: []*llx.Primitive{
				llx.RefPrimitiveV1(2),
				llx.FunctionPrimitiveV1(1),
			},
		}, res.DeprecatedV5Code.Code[2])
		assertFunction(t, "list", &llx.Function{
			Type:                string(types.Array(types.Resource("package"))),
			DeprecatedV5Binding: 3,
		}, res.DeprecatedV5Code.Code[3])
		assertFunction(t, "length", &llx.Function{
			Type:                string(types.Int),
			DeprecatedV5Binding: 4,
		}, res.DeprecatedV5Code.Code[4])
		assertFunction(t, string(">"+types.Int), &llx.Function{
			Type:                string(types.Bool),
			DeprecatedV5Binding: 5,
			Args:                []*llx.Primitive{llx.IntPrimitive(0)},
		}, res.DeprecatedV5Code.Code[5])

		functions := res.DeprecatedV5Code.Functions

		assertPrimitive(t, &llx.Primitive{
			Type: string(types.Resource("package")),
		}, functions[0].Code[0])
		assertFunction(t, "outdated", &llx.Function{
			Type:                string(types.Bool),
			DeprecatedV5Binding: 1,
		}, functions[0].Code[1])
		assert.Equal(t, []int32{2}, functions[0].Entrypoints)
	})
}

//    ================
//   👋   INTERNAL   🍹
//    ================

func TestChecksums(t *testing.T) {
	t.Run("no duplicate code IDs", func(t *testing.T) {
		dupes := []struct {
			qa string
			qb string
		}{
			{
				"users.list { uid == 1 }",
				"users.list { uid == 2 }",
			},
			{
				"platform.name\nplatform.release",
				"platform.name",
			},
			{
				"platform.name\nplatform.release",
				"platform.release",
			},
			{
				"if (true) { 2 }",
				"if (true) { 3 }",
			},
		}

		for i := range dupes {
			t.Run(dupes[i].qa+" != "+dupes[i].qb, func(t *testing.T) {
				a, err := Compile(dupes[i].qa, schema, nil)
				assert.NoError(t, err)
				b, err := Compile(dupes[i].qb, schema, nil)
				assert.NoError(t, err)
				assert.NotEqual(t, a.DeprecatedV5Code.Id, b.DeprecatedV5Code.Id)
			})
		}
	})
}

func TestSuggestions(t *testing.T) {
	tests := []struct {
		code        string
		suggestions []string
		err         error
	}{
		{
			"does_not_get_suggestions",
			[]string{},
			errors.New("cannot find resource for identifier 'does_not_get_suggestions'"),
		},
		{
			// resource suggestions
			// TODO: "msgraph.beta.rolemanagement.roledefinition" shows up because it includes tem`plat`eId
			"platfo",
			[]string{"platform", "platform.advisories", "platform.cves", "platform.eol", "platform.virtualization"},
			errors.New("cannot find resource for identifier 'platfo'"),
		},
		{
			// resource with empty field call
			"sshd.",
			[]string{"config"},
			errors.New("incomplete query, missing identifier after '.' at <source>:1:6"),
		},
		{
			// list resource with empty field call
			"users.",
			[]string{"all", "any", "contains", "length", "list", "map", "none", "one", "where"},
			errors.New("incomplete query, missing identifier after '.' at <source>:1:7"),
		},
		{
			// resource with partial field call
			"sshd.config.para",
			[]string{"params"},
			errors.New("cannot find field 'para' in sshd.config"),
		},
		{
			// resource with partial field call in block
			"sshd.config { para }",
			[]string{"params"},
			errors.New("cannot find field or resource 'para' in block for type 'sshd.config'"),
		},
		{
			// native type function call
			"sshd.config.params.leng",
			[]string{"length"},
			errors.New("cannot find field 'leng' in map[string]string"),
		},
		{
			// builtin calls
			"parse.d",
			[]string{"date"},
			errors.New("cannot find field 'd' in parse"),
		},
	}

	for i := range tests {
		cur := tests[i]
		t.Run(cur.code, func(t *testing.T) {
			res, err := Compile(cur.code, schema, nil)
			assert.Nil(t, res.DeprecatedV5Code.Entrypoints)
			assert.Equal(t, cur.err.Error(), err.Error())

			suggestions := make([]string, len(res.Suggestions))
			for i := range res.Suggestions {
				suggestions[i] = res.Suggestions[i].Field
			}
			sort.Strings(suggestions)
			assert.Equal(t, cur.suggestions, suggestions)
		})
	}
}

func TestCompiler_Error(t *testing.T) {
	t.Run("unknown term", func(t *testing.T) {
		_, err := Compile("sshd.config.params == enabled", schema, nil)
		// assert.Nil(t, res)
		assert.EqualError(t, err, "failed to compile: cannot find resource for identifier 'enabled'")
	})
}

func TestCompiler_Multiline(t *testing.T) {
	compile(t, "1 < 2\n2 != 3", func(res *llx.CodeBundle) {
		assert.Equal(t, 4, len(res.DeprecatedV5Code.Code))
	})
}

func TestCompiler_Entrypoints(t *testing.T) {
	tests := []struct {
		code        string
		datapoints  []int32
		entrypoints []int32
	}{
		{
			"1",
			[]int32(nil),
			[]int32{1},
		},
		{
			"mondoo.version == 1",
			[]int32{2},
			[]int32{3},
		},
		{
			"mondoo.version == mondoo.build",
			[]int32{2, 4},
			[]int32{5},
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.code, func(t *testing.T) {
			compile(t, test.code, func(res *llx.CodeBundle) {
				assert.Equal(t, test.entrypoints, res.DeprecatedV5Code.Entrypoints)
				assert.Equal(t, test.datapoints, res.DeprecatedV5Code.Datapoints)
			})
		})
	}
}
