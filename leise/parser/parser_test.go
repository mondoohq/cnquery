package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/logger"
)

func init() {
	logger.InitTestEnv()
}

func TestParser_Lex(t *testing.T) {
	tests := []struct {
		typ rune
		str string
	}{
		{Ident, "name"},
		{Float, "1.23"},
		{Int, "123"},
		{String, "'hi'"},
		{String, "\"hi\""},
		{Regex, "/regex/"},
		{Op, "+"},
	}
	for i := range tests {
		res, err := Lex(tests[i].str)
		assert.Nil(t, err)
		assert.Equal(t, tests[i].typ, res[0].Type)
	}
}

func sPtr(s string) *string {
	return &s
}

func vBool(b bool) *Value {
	return &Value{Bool: &b}
}

func vIdent(v string) *Value {
	return &Value{Ident: &v}
}

func vFloat(v float64) *Value {
	return &Value{Float: &v}
}

func vInt(v int64) *Value {
	return &Value{Int: &v}
}

func vString(v string) *Value {
	return &Value{String: &v}
}

func vRegex(v string) *Value {
	return &Value{Regex: &v}
}

func vMap(v map[string]*Expression) *Value {
	return &Value{Map: v}
}

func callIdent(ident string) *Call {
	return &Call{Ident: &ident}
}

type parserTest struct {
	code string
	res  *Expression
}

func runParserTests(t *testing.T, tests []parserTest) {
	for i := range tests {
		test := tests[i]

		t.Run(test.code, func(t *testing.T) {
			res, err := Parse(test.code)
			require.NoError(t, err, "parsing should not generate an error")
			require.NotNil(t, res, "parsing must generate a result value")
			require.NotEmpty(t, res.Expressions, "parsing must generate one expression")

			assert.Equal(t, test.res, res.Expressions[0])
		})
	}
}

type multiTest struct {
	code string
	res  []*Expression
}

func runMultiTest(t *testing.T, tests []multiTest) {
	for i := range tests {
		test := tests[i]

		t.Run(test.code, func(t *testing.T) {
			res, err := Parse(test.code)
			require.NoError(t, err, "parsing should not generate an error")
			require.NotNil(t, res, "parsing must generate a result value")
			assert.Equal(t, test.res, res.Expressions, "resulting expressions must match")
		})
	}
}

func TestParser_ParseValues(t *testing.T) {
	runParserTests(t, []parserTest{
		{"null", &Expression{Operand: &Operand{Value: &nilValue}}},
		{"NaN", &Expression{Operand: &Operand{Value: &nanValue}}},
		{"Infinity", &Expression{Operand: &Operand{Value: &infinityValue}}},
		{"Never", &Expression{Operand: &Operand{Value: &neverValue}}},
		{"true", &Expression{Operand: &Operand{Value: vBool(true)}}},
		{"false", &Expression{Operand: &Operand{Value: vBool(false)}}},
		{"name", &Expression{Operand: &Operand{Value: vIdent("name")}}},
		{"1.23", &Expression{Operand: &Operand{Value: vFloat(1.23)}}},
		{"123", &Expression{Operand: &Operand{Value: vInt(123)}}},
		{"'hi'", &Expression{Operand: &Operand{Value: vString("hi")}}},
		{"'h\\ni'", &Expression{Operand: &Operand{Value: vString("h\\ni")}}},
		{"'h\\i'", &Expression{Operand: &Operand{Value: vString("h\\i")}}},
		{"\"hi\"", &Expression{Operand: &Operand{Value: vString("hi")}}},
		{"\"h\\ni\"", &Expression{Operand: &Operand{Value: vString("h\ni")}}},
		{"\"h\\i\"", &Expression{Operand: &Operand{Value: vString("hi")}}},
		{"/hi/", &Expression{Operand: &Operand{Value: vRegex("hi")}}},
		{"[]", &Expression{Operand: &Operand{Value: &Value{Array: []*Expression{}}}}},
		{"[1]", &Expression{Operand: &Operand{Value: &Value{Array: []*Expression{
			{Operand: &Operand{Value: vInt(1)}},
		}}}}},
		{"[1,2.3]", &Expression{Operand: &Operand{Value: &Value{Array: []*Expression{
			{Operand: &Operand{Value: vInt(1)}},
			{Operand: &Operand{Value: vFloat(2.3)}},
		}}}}},
		{"[1,2,]", &Expression{Operand: &Operand{Value: &Value{Array: []*Expression{
			{Operand: &Operand{Value: vInt(1)}},
			{Operand: &Operand{Value: vInt(2)}},
		}}}}},
		{"{}", &Expression{Operand: &Operand{Value: vMap(map[string]*Expression{})}}},
		{"{'a': 'word'}", &Expression{Operand: &Operand{Value: vMap(map[string]*Expression{
			"a": {Operand: &Operand{Value: vString("word")}},
		})}}},
		{"{\"b\": \"there\"}", &Expression{Operand: &Operand{Value: vMap(map[string]*Expression{
			"b": {Operand: &Operand{Value: vString("there")}},
		})}}},
		{"{c: 123}", &Expression{Operand: &Operand{Value: vMap(map[string]*Expression{
			"c": {Operand: &Operand{Value: vInt(123)}},
		})}}},
		{"{a: 1, b: 2,}", &Expression{Operand: &Operand{Value: vMap(map[string]*Expression{
			"a": {Operand: &Operand{Value: vInt(1)}},
			"b": {Operand: &Operand{Value: vInt(2)}},
		})}}},
		{"name.last", &Expression{Operand: &Operand{
			Value: vIdent("name"),
			Calls: []*Call{callIdent("last")},
		}}},
		{"name[1]", &Expression{Operand: &Operand{
			Value: vIdent("name"),
			Calls: []*Call{{Accessor: &Expression{Operand: &Operand{Value: vInt(1)}}}},
		}}},
		{"name()", &Expression{Operand: &Operand{
			Value: vIdent("name"),
			Calls: []*Call{{Function: []*Arg{}}},
		}}},
		{"name(1)", &Expression{Operand: &Operand{
			Value: vIdent("name"),
			Calls: []*Call{{Function: []*Arg{
				{Value: &Expression{Operand: &Operand{Value: vInt(1)}}},
			}}},
		}}},
		{"name(arg)", &Expression{Operand: &Operand{
			Value: vIdent("name"),
			Calls: []*Call{{Function: []*Arg{
				{Value: &Expression{Operand: &Operand{Value: vIdent("arg")}}},
			}}},
		}}},
		{"name(uid: 1)", &Expression{Operand: &Operand{
			Value: vIdent("name"),
			Calls: []*Call{{Function: []*Arg{
				{Name: "uid", Value: &Expression{Operand: &Operand{Value: vInt(1)}}},
			}}},
		}}},
		{"a(b(c,d))", &Expression{Operand: &Operand{
			Value: vIdent("a"),
			Calls: []*Call{{Function: []*Arg{
				{Value: &Expression{Operand: &Operand{
					Value: vIdent("b"),
					Calls: []*Call{{Function: []*Arg{
						{Value: &Expression{Operand: &Operand{Value: vIdent("c")}}},
						{Value: &Expression{Operand: &Operand{Value: vIdent("d")}}},
					}}},
				}}},
			}}},
		}}},
		{"a(\nb(\nc,\nd\n)\n)", &Expression{Operand: &Operand{
			Value: vIdent("a"),
			Calls: []*Call{{Function: []*Arg{
				{Value: &Expression{Operand: &Operand{
					Value: vIdent("b"),
					Calls: []*Call{{Function: []*Arg{
						{Value: &Expression{Operand: &Operand{Value: vIdent("c")}}},
						{Value: &Expression{Operand: &Operand{Value: vIdent("d")}}},
					}}},
				}}},
			}}},
		}}},
		{"user { name uid }", &Expression{Operand: &Operand{
			Value: vIdent("user"),
			Block: []*Expression{
				{Operand: &Operand{Value: vIdent("name")}},
				{Operand: &Operand{Value: vIdent("uid")}},
			},
		}}},
		{"user {\n  name\n  uid\n}", &Expression{Operand: &Operand{
			Value: vIdent("user"),
			Block: []*Expression{
				{Operand: &Operand{Value: vIdent("name")}},
				{Operand: &Operand{Value: vIdent("uid")}},
			},
		}}},
		{"users.list { uid }", &Expression{Operand: &Operand{
			Value: vIdent("users"),
			Calls: []*Call{callIdent("list")},
			Block: []*Expression{
				{Operand: &Operand{Value: vIdent("uid")}},
			},
		}}},
		{"users.where()", &Expression{Operand: &Operand{
			Value: vIdent("users"),
			Calls: []*Call{
				callIdent("where"),
				{Function: []*Arg{}},
			},
		}}},
		{"users.where(uid > 2).list { uid }", &Expression{Operand: &Operand{
			Value: vIdent("users"),
			Calls: []*Call{
				callIdent("where"),
				{Function: []*Arg{{Value: &Expression{
					Operand: &Operand{Value: vIdent("uid")},
					Operations: []*Operation{{
						Operator: OpGreater,
						Operand:  &Operand{Value: vInt(2)},
					}},
				}}}},
				callIdent("list"),
			},
			Block: []*Expression{
				{Operand: &Operand{Value: vIdent("uid")}},
			},
		}}},
		{"1 + 2 == 3", &Expression{
			Operand: &Operand{Value: vInt(1)},
			Operations: []*Operation{
				{Operator: OpAdd, Operand: &Operand{Value: vInt(2)}},
				{Operator: OpEqual, Operand: &Operand{Value: vInt(3)}},
			},
		}},
		{"1 && 2 || 3", &Expression{
			Operand: &Operand{Value: vInt(1)},
			Operations: []*Operation{
				{Operator: OpAnd, Operand: &Operand{Value: vInt(2)}},
				{Operator: OpOr, Operand: &Operand{Value: vInt(3)}},
			},
		}},
		{"true + 'some'.length()", &Expression{
			Operand: &Operand{Value: vBool(true)},
			Operations: []*Operation{
				{Operator: OpAdd, Operand: &Operand{
					Value: vString("some"),
					Calls: []*Call{callIdent("length"), {Function: []*Arg{}}},
				}},
			},
		}},
		{"// this // is a comment\n'hi'", &Expression{Operand: &Operand{
			Value:    vString("hi"),
			Comments: "this // is a comment\n",
		}}},
		{"# this # is a comment\n'hi'", &Expression{Operand: &Operand{
			Value:    vString("hi"),
			Comments: "this # is a comment\n",
		}}},
	})
}

func TestParser_Comments(t *testing.T) {
	runMultiTest(t, []multiTest{
		// call chain with many newlines
		{"// 1\nsshd\n// 2\n\t.\n// 3\nconfig // 4", []*Expression{
			{Operand: &Operand{
				Comments: "1\n",
				Value:    vIdent("sshd"),
				Calls: []*Call{
					{
						Comments: "2\n3\n",
						Ident:    sPtr("config"),
					},
				},
			}},
			{Operand: &Operand{Comments: "4"}},
		}},
		// blocks and newlines
		{"file\n// 1\n{\n// 2\npath\n// 3\n==\n// 4\n'abc'\n// 5\n}\n// 6", []*Expression{
			{Operand: &Operand{
				Value: vIdent("file"),
				Block: []*Expression{
					{
						Operand: &Operand{
							Comments: "1\n2\n",
							Value:    vIdent("path"),
						},
						Operations: []*Operation{{
							Operator: 103,
							Operand: &Operand{
								Comments: "3\n4\n",
								Value:    vString("abc"),
							},
						}},
					},
					{
						Operand: &Operand{Comments: "5\n"},
					},
				},
			}},
			{Operand: &Operand{Comments: "6"}},
		}},
	})
}

func TestParser_Multiline(t *testing.T) {
	runMultiTest(t, []multiTest{
		{"true\n1\n2\n", []*Expression{
			{Operand: &Operand{Value: vBool(true)}},
			{Operand: &Operand{Value: vInt(1)}},
			{Operand: &Operand{Value: vInt(2)}},
		}},
	})
}
