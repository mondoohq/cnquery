// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/testutils"
)

// Core Language constructs
// ------------------------
// These tests are more generic MQL and resource tests. They have no dependency
// on any other resources and test important MQL constructs.

func TestCore_Props(t *testing.T) {
	tests := []struct {
		code        string
		props       map[string]*llx.Primitive
		resultIndex int
		expectation interface{}
		err         error
	}{
		{
			`props.name`,
			map[string]*llx.Primitive{"name": llx.StringPrimitive("bob")},
			0, "bob", nil,
		},
		{
			`props.name == 'bob'`,
			map[string]*llx.Primitive{"name": llx.StringPrimitive("bob")},
			1, true, nil,
		},
	}

	x := testutils.InitTester(testutils.LinuxMock())

	for i := range tests {
		cur := tests[i]
		t.Run(cur.code, func(t *testing.T) {
			res := x.TestQueryP(t, cur.code, cur.props)
			require.NotEmpty(t, res)

			if len(res) <= cur.resultIndex {
				t.Error("insufficient results, looking for result idx " + strconv.Itoa(cur.resultIndex))
				return
			}

			assert.NotNil(t, res[cur.resultIndex].Result().Error)
			assert.Equal(t, cur.expectation, res[cur.resultIndex].Data.Value)
		})
	}
}

func TestCore_If(t *testing.T) {
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        "if ( mondoo.version == null ) { 123 }",
			ResultIndex: 1,
			Expectation: nil,
		},
		{
			Code:        "if (true) { return 123 } return 456",
			Expectation: int64(123),
		},
		{
			Code:        "if (true) { return [1] } return [2,3]",
			Expectation: []interface{}{int64(1)},
		},
		{
			Code:        "if (false) { return 123 } return 456",
			Expectation: int64(456),
		},
		{
			Code:        "if (false) { return 123 } if (true) { return 456 } return 789",
			Expectation: int64(456),
		},
		{
			Code:        "if (false) { return 123 } if (false) { return 456 } return 789",
			Expectation: int64(789),
		},
		{
			// This test comes out from an issue we had where return was not
			// generating a single entrypoint, causing the first reported
			// value to be used as the return value.
			Code: `
				if (true) {
					a = asset.platform != ''
					b = false
					return a || b
				}
			`, Expectation: true,
		},
		{
			Code:        "if ( mondoo.version != null ) { 123 }",
			ResultIndex: 1,
			Expectation: map[string]interface{}{
				"__t": llx.BoolData(true),
				"__s": llx.NilData,
				"NmGComMxT/GJkwpf/IcA+qceUmwZCEzHKGt+8GEh+f8Y0579FxuDO+4FJf0/q2vWRE4dN2STPMZ+3xG3Mdm1fA==": llx.IntData(123),
			},
		},
		{
			Code:        "if ( mondoo.version != null ) { 123 } else { 456 }",
			ResultIndex: 1,
			Expectation: map[string]interface{}{
				"__t": llx.BoolData(true),
				"__s": llx.NilData,
				"NmGComMxT/GJkwpf/IcA+qceUmwZCEzHKGt+8GEh+f8Y0579FxuDO+4FJf0/q2vWRE4dN2STPMZ+3xG3Mdm1fA==": llx.IntData(123),
			},
		},
		{
			Code:        "if ( mondoo.version == null ) { 123 } else { 456 }",
			ResultIndex: 1,
			Expectation: map[string]interface{}{
				"__t": llx.BoolData(true),
				"__s": llx.NilData,
				"3ZDJLpfu1OBftQi3eANcQSCltQum8mPyR9+fI7XAY9ZUMRpyERirCqag9CFMforO/u0zJolHNyg+2gE9hSTyGQ==": llx.IntData(456),
			},
		},
		{
			Code: "if (false) { 123 } else if (true) { 456 } else { 789 }",
			Expectation: map[string]interface{}{
				"__t": llx.BoolData(true),
				"__s": llx.NilData,
				"3ZDJLpfu1OBftQi3eANcQSCltQum8mPyR9+fI7XAY9ZUMRpyERirCqag9CFMforO/u0zJolHNyg+2gE9hSTyGQ==": llx.IntData(456),
			},
		},
		{
			Code: "if (false) { 123 } else if (false) { 456 } else { 789 }",
			Expectation: map[string]interface{}{
				"__t": llx.BoolData(true),
				"__s": llx.NilData,
				"Oy5SF8NbUtxaBwvZPpsnd0K21CY+fvC44FSd2QpgvIL689658Na52udy7qF2+hHjczk35TAstDtFZq7JIHNCmg==": llx.IntData(789),
			},
		},
	})

	x.TestSimpleErrors(t, []testutils.SimpleTest{
		// if-conditions need to be called with a bloc
		{
			Code:        "if(asset.family.contains('arch'))",
			ResultIndex: 1, Expectation: "Called if with 1 arguments, expected at least 3",
		},
	})
}

func TestCore_Switch(t *testing.T) {
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        "switch { case 3 > 2: 123; default: 321 }",
			Expectation: int64(123),
		},
		{
			Code:        "switch { case 1 > 2: 123; default: 321 }",
			Expectation: int64(321),
		},
		{
			Code:        "switch { case 3 > 2: return 123; default: return 321 }",
			Expectation: int64(123),
		},
		{
			Code:        "switch { case 1 > 2: return 123; default: return 321 }",
			Expectation: int64(321),
		},
		{
			Code:        "switch ( 3 ) { case _ > 2: return 123; default: return 321 }",
			Expectation: int64(123),
		},
		{
			Code:        "switch ( 1 ) { case _ > 2: true; default: false }",
			Expectation: false,
		},
	})
}

func TestCore_Vars(t *testing.T) {
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        "a = [1,2,3]; return a",
			Expectation: []interface{}{int64(1), int64(2), int64(3)},
		},
		{
			Code:        "a = 1; b = [a]; return b",
			Expectation: []interface{}{int64(1)},
		},
		{
			Code:        "a = 1; b = a + 2; return b",
			Expectation: int64(3),
		},
		{
			Code:        "a = 1; b = [a + 2]; return b",
			Expectation: []interface{}{int64(3)},
		},
	})
}

// Base types and operations
// -------------------------

func TestBooleans(t *testing.T) {
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        "true || false || false",
			ResultIndex: 1,
			Expectation: true,
		},
		{
			Code:        "false || true || false",
			ResultIndex: 1,
			Expectation: true,
		},
		{
			Code:        "false || false || true",
			ResultIndex: 1,
			Expectation: true,
		},
	})
}

func TestOperations_Equality(t *testing.T) {
	vals := []string{
		"null",
		"true", "false",
		"0", "1",
		"1.0", "1.5",
		"'1'", "'1.0'", "'a'",
		"/1/", "/a/", "/nope/",
		"[1]", "[null]",
	}

	extraEquality := map[string]map[string]struct{}{
		"1": {
			"1.0":   struct{}{},
			"'1'":   struct{}{},
			"/1/":   struct{}{},
			"[1]":   struct{}{},
			"[1.0]": struct{}{},
		},
		"1.0": {
			"[1]": struct{}{},
		},
		"'a'": {
			"/a/": struct{}{},
		},
		"'1'": {
			"1.0": struct{}{},
			"[1]": struct{}{},
		},
		"/1/": {
			"1.0":   struct{}{},
			"'1'":   struct{}{},
			"'1.0'": struct{}{},
			"[1]":   struct{}{},
			"1.5":   struct{}{},
		},
	}

	simpleTests := []testutils.SimpleTest{}

	for i := 0; i < len(vals); i++ {
		for j := i; j < len(vals); j++ {
			a := vals[i]
			b := vals[j]
			res := a == b

			if sub, ok := extraEquality[a]; ok {
				if _, ok := sub[b]; ok {
					res = true
				}
			}
			if sub, ok := extraEquality[b]; ok {
				if _, ok := sub[a]; ok {
					res = true
				}
			}

			simpleTests = append(simpleTests, []testutils.SimpleTest{
				{Code: a + " == " + b, Expectation: res},
				{Code: a + " != " + b, Expectation: !res},
				{Code: "a = " + a + "  a == " + b, ResultIndex: 1, Expectation: res},
				{Code: "a = " + a + "  a != " + b, ResultIndex: 1, Expectation: !res},
				{Code: "b = " + b + "; " + a + " == b", ResultIndex: 1, Expectation: res},
				{Code: "b = " + b + "; " + a + " != b", ResultIndex: 1, Expectation: !res},
				{Code: "a = " + a + "; b = " + b + "; a == b", ResultIndex: 2, Expectation: res},
				{Code: "a = " + a + "; b = " + b + "; a != b", ResultIndex: 2, Expectation: !res},
			}...)
		}
	}

	x.TestSimple(t, simpleTests)
}

func TestEmpty(t *testing.T) {
	empty := []string{
		"null",
		"''",
		"[]",
		"{}",
	}
	nonEmpty := []string{
		"true", "false",
		"0", "1.0",
		"'a'",
		"/a/",
		"[null]", "[1]",
		"{a: 1}",
	}

	tests := []testutils.SimpleTest{}
	for i := range empty {
		tests = append(tests, testutils.SimpleTest{
			Code:        empty[i] + " == empty",
			ResultIndex: 1,
			Expectation: true,
		})
	}

	for i := range nonEmpty {
		tests = append(tests, testutils.SimpleTest{
			Code:        nonEmpty[i] + " == empty",
			ResultIndex: 1,
			Expectation: false,
		})
	}

	x.TestSimple(t, tests)
}

func TestNumber_Methods(t *testing.T) {
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code: "1 + 2", Expectation: int64(3),
		},
		{
			Code: "1 - 2", Expectation: int64(-1),
		},
		{
			Code: "1 * 2", Expectation: int64(2),
		},
		{
			Code: "4 / 2", Expectation: int64(2),
		},
		{
			Code: "1.0 + 2.0", Expectation: float64(3),
		},
		{
			Code: "1 - 2.0", Expectation: float64(-1),
		},
		{
			Code: "1.0 * 2", Expectation: float64(2),
		},
		{
			Code: "4.0 / 2.0", Expectation: float64(2),
		},
		{
			Code: "1 < Infinity", Expectation: true,
		},
		{
			Code: "1 == NaN", Expectation: false,
		},
		{
			Code: "2.inRange(1,2.0)", Expectation: true,
		},
		{
			Code: "3.0.inRange(1.0,2)", Expectation: false,
		},
	})
}

func TestString_Methods(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock())
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        "'hello'.contains('ll')",
			Expectation: true,
		},
		{
			Code:        "'hello'.contains('lloo')",
			Expectation: false,
		},
		{
			Code:        "'hello'.contains(['lo', 'la'])",
			Expectation: true,
		},
		{
			Code:        "'hello'.contains(['lu', 'la'])",
			Expectation: false,
		},
		{
			Code:        "'hello'.contains(23)",
			Expectation: false,
		},
		{
			Code:        "'hello123'.contains(23)",
			Expectation: true,
		},
		{
			Code:        "'hello123'.contains([5,6,7])",
			Expectation: false,
		},
		{
			Code:        "'hello123'.contains([5,1,7])",
			Expectation: true,
		},
		{
			Code:        "'hello'.contains(/l+/)",
			Expectation: true,
		},
		{
			Code:        "'hello'.contains(/l$/)",
			Expectation: false,
		},
		{
			Code:        "'hello'.contains([/^l/, /l$/])",
			Expectation: false,
		},
		{
			Code:        "'hello'.contains([/z/, /ll/])",
			Expectation: true,
		},
		{
			Code:        "'hi'.in(['one','hi','five'])",
			Expectation: true,
		},
		{
			Code:        "'hiya'.in(['one','hi','five'])",
			Expectation: false,
		},
		{
			Code:        "'hiya'.in([])",
			Expectation: false,
		},
		{
			Code:        "'oh-hello-world!'.camelcase",
			Expectation: "ohHelloWorld!",
		},
		{
			Code:        "'HeLlO'.downcase",
			Expectation: "hello",
		},
		{
			Code:        "'hello'.length",
			Expectation: int64(5),
		},
		{
			Code:        "'hello world'.split(' ')",
			Expectation: []interface{}{"hello", "world"},
		},
		{
			Code:        "'he\nll\no'.lines",
			Expectation: []interface{}{"he", "ll", "o"},
		},
		{
			Code:        "' \n\t yo \t \n   '.trim",
			Expectation: "yo",
		},
		{
			Code:        "'  \tyo  \n   '.trim(' \n')",
			Expectation: "\tyo",
		},
		{
			Code:        "'hello ' + 'world'",
			Expectation: "hello world",
		},
	})
}

func TestScore_Methods(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock())
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        "score(100)",
			Expectation: []byte{0x00, byte(100)},
		},
		{
			Code:        "score(\"CVSS:3.1/AV:P/AC:H/PR:L/UI:N/S:U/C:H/I:L/A:H\")",
			Expectation: []byte{0x01, 0x03, 0x01, 0x04, 0x00, 0x00, 0x00, 0x01, 0x00},
		},
	})
}

func TestTypeof_Methods(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock())
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        "typeof(null)",
			Expectation: "null",
		},
		{
			Code:        "typeof(123)",
			Expectation: "int",
		},
		{
			Code:        "typeof([1,2,3])",
			Expectation: "[]int",
		},
		{
			Code:        "a = 123; typeof(a)",
			Expectation: "int",
		},
	})
}

func TestArray_Access(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock())
	x.TestSimpleErrors(t, []testutils.SimpleTest{
		{
			Code:        "[0,1,2][100000]",
			Expectation: "array index out of bound (trying to access element 100000, max: 2)",
		},
	})

	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        "[1,2,3][-1]",
			Expectation: int64(3),
		},
		{
			Code:        "[1,2,3][-3]",
			Expectation: int64(1),
		},
		{
			Code:        "[1,2,3].first",
			Expectation: int64(1),
		},
		{
			Code:        "[1,2,3].last",
			Expectation: int64(3),
		},
		{
			Code:        "[].first",
			Expectation: nil,
		},
		{
			Code:        "[].last",
			Expectation: nil,
		},
	})
}

func TestArray(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock())
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        "[1,2,3]",
			Expectation: []interface{}{int64(1), int64(2), int64(3)},
		},
		{
			Code:        "return [1,2,3]",
			Expectation: []interface{}{int64(1), int64(2), int64(3)},
		},
		{
			Code: "[1,2,3] { _ == 2 }",
			Expectation: []interface{}{
				map[string]interface{}{"__t": llx.BoolFalse, "__s": llx.BoolFalse, "OPhfwvbw0iVuMErS9tKL5qNj1lqTg3PEE1LITWEwW7a70nH8z8eZLi4x/aZqZQlyrQK13GAlUMY1w8g131EPog==": llx.BoolFalse},
				map[string]interface{}{"__t": llx.BoolTrue, "__s": llx.BoolTrue, "OPhfwvbw0iVuMErS9tKL5qNj1lqTg3PEE1LITWEwW7a70nH8z8eZLi4x/aZqZQlyrQK13GAlUMY1w8g131EPog==": llx.BoolTrue},
				map[string]interface{}{"__t": llx.BoolFalse, "__s": llx.BoolFalse, "OPhfwvbw0iVuMErS9tKL5qNj1lqTg3PEE1LITWEwW7a70nH8z8eZLi4x/aZqZQlyrQK13GAlUMY1w8g131EPog==": llx.BoolFalse},
			},
		},
		{
			Code: "[1,2,3] { a = _ }",
			Expectation: []interface{}{
				map[string]interface{}{"__t": llx.BoolTrue, "__s": llx.NilData},
				map[string]interface{}{"__t": llx.BoolTrue, "__s": llx.NilData},
				map[string]interface{}{"__t": llx.BoolTrue, "__s": llx.NilData},
			},
		},
		{
			Code:        "[1,2,3].where()",
			Expectation: []interface{}{int64(1), int64(2), int64(3)},
		},
		{
			Code:        "[true, true, false].where(true)",
			Expectation: []interface{}{true, true},
		},
		{
			Code:        "[false, true, false].where(false)",
			Expectation: []interface{}{false, false},
		},
		{
			Code:        "[1,2,3].where(2)",
			Expectation: []interface{}{int64(2)},
		},
		{
			Code:        "[1,2,3].where(_ > 2)",
			Expectation: []interface{}{int64(3)},
		},
		{
			Code:        "[1,2,3].where(_ >= 2)",
			Expectation: []interface{}{int64(2), int64(3)},
		},
		{
			Code:        "['yo','ho','ho'].where( /y.$/ )",
			Expectation: []interface{}{"yo"},
		},
		{
			Code:        "x = ['a','b']; y = 'c'; x.contains(y)",
			ResultIndex: 1,
			Expectation: false,
		},
		{
			Code:        "[1,2,3].contains(_ >= 2)",
			ResultIndex: 1,
			Expectation: true,
		},
		{
			Code:        "['hi'].in(['one','hi','five'])",
			Expectation: true,
		},
		{
			Code:        "['hi', 'bob'].in(['one','hi','five'])",
			Expectation: false,
		},
		{
			Code:        "[1,2,3].all(_ < 9)",
			ResultIndex: 1,
			Expectation: true,
		},
		{
			Code:        "[1,2,3].any(_ > 1)",
			ResultIndex: 1,
			Expectation: true,
		},
		{
			Code:        "[1,2,3].one(_ == 2)",
			ResultIndex: 1,
			Expectation: true,
		},
		{
			Code:        "[1,2,3].none(_ == 4)",
			ResultIndex: 1,
			Expectation: true,
		},
		{
			Code:        "[[0,1],[1,2]].map(_[1])",
			Expectation: []interface{}{int64(1), int64(2)},
		},
		{
			Code:        "[[0],[[1, 2]], 3].flat",
			Expectation: []interface{}{int64(0), int64(1), int64(2), int64(3)},
		},
		{
			Code:        "[0].where(_ > 0).where(_ > 0)",
			Expectation: []interface{}{},
		},
		{
			Code:        "[1,2,2,2,3].unique()",
			Expectation: []interface{}{int64(1), int64(2), int64(3)},
		},
		{
			Code:        "[1,1,2,2,2,3].duplicates()",
			Expectation: []interface{}{int64(1), int64(2)},
		},
		{
			Code:        "[2,1,2,2].containsOnly([2])",
			Expectation: []interface{}{int64(1)},
		},
		{
			Code:        "[2,1,2,1].containsOnly([1,2])",
			ResultIndex: 0, Expectation: []interface{}(nil),
		},
		{
			Code:        "a = [1]; [2,1,2,1].containsOnly(a)",
			Expectation: []interface{}{int64(2), int64(2)},
		},
		{
			Code:        "[3,3,2,2].containsAll([1,2])",
			Expectation: []interface{}{int64(1)},
		},
		{
			Code:        "[2,1,2,1].containsAll([1,2])",
			ResultIndex: 0, Expectation: []interface{}(nil),
		},
		{
			Code:        "a = [1,3]; [2,1,2,1].containsAll(a)",
			Expectation: []interface{}{int64(3)},
		},
		{
			Code:        "[2,1,2,2].containsNone([1])",
			Expectation: []interface{}{int64(1)},
		},
		{
			Code:        "[2,1,2,1].containsNone([3,4])",
			ResultIndex: 0, Expectation: []interface{}(nil),
		},
		{
			Code:        "a = [1]; [2,1,2,1].containsNone(a)",
			Expectation: []interface{}{int64(1), int64(1)},
		},
		{
			Code:        "['a','b'] != /c/",
			ResultIndex: 0, Expectation: true,
		},
		{
			Code:        "[1,2] + [3]",
			Expectation: []interface{}{int64(1), int64(2), int64(3)},
		},
		{
			Code:        "[3,1,3,4,2] - [3,4,5]",
			Expectation: []interface{}{int64(1), int64(2)},
		},
	})
}

func testSample(t *testing.T, mqlData string, sampleLen int, isMap bool) {
	const samplesCnt = 20

	t.Run(mqlData, func(t *testing.T) {
		x := testutils.InitTester(testutils.LinuxMock())

		// check that the data is different; given enough samples with a good
		// data length, this should very very rarely fail naturally
		allDupes := true
		if samplesCnt < 2 {
			allDupes = false
		}
		samples := make([][samplesCnt]any, samplesCnt)
		ref := [samplesCnt]any{}

		for i := 0; i < samplesCnt; i++ {
			mql := mqlData + ".sample(" + strconv.Itoa(sampleLen) + ")"
			if isMap {
				mql += ".keys"
			}
			res := x.TestQuery(t, mql)
			require.Len(t, res, 2)

			list, ok := res[0].Data.Value.([]any)
			require.True(t, ok, "return a list of values")
			require.Len(t, list, sampleLen)
			curSamples := [samplesCnt]any{}
			copy(curSamples[:], list)

			if i == 0 {
				ref = curSamples
			} else if ref != curSamples {
				allDupes = false
			}
			samples[i] = curSamples
		}

		assert.False(t, allDupes)
	})
}

func TestSample(t *testing.T) {
	t.Run("simple array", func(t *testing.T) {
		testSample(t, "[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20]", 3, false)
	})

	t.Run("simple map", func(t *testing.T) {
		testSample(t, "{\"a\": 1, \"b\": 1, \"c\": 2, \"d\": 4, \"e\": 5, \"f\": 6, \"g\": 7, \"h\": 8, \"i\": 9, \"j\": 10, \"k\": 11, \"l\": 12, \"m\": 13, \"n\": 14, \"o\": 15, \"p\": 16, \"q\": 17, \"r\": 18, \"s\": 19, \"t\": 20}", 3, true)
	})

	t.Run("simple dict array", func(t *testing.T) {
		testSample(t, "parse.json(content: '[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20]').params", 3, false)
	})

	t.Run("simple map", func(t *testing.T) {
		testSample(t, "parse.json(content: '{\"a\": 1, \"b\": 1, \"c\": 2, \"d\": 4, \"e\": 5, \"f\": 6, \"g\": 7, \"h\": 8, \"i\": 9, \"j\": 10, \"k\": 11, \"l\": 12, \"m\": 13, \"n\": 14, \"o\": 15, \"p\": 16, \"q\": 17, \"r\": 18, \"s\": 19, \"t\": 20}').params", 3, true)
	})
}

func TestMap(t *testing.T) {
	m := "{'a': 1, 'b': 1, 'c': 2}"
	x := testutils.InitTester(testutils.LinuxMock())
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        m + "['c']",
			ResultIndex: 0, Expectation: int64(2),
		},
		{
			Code:        m + ".c",
			ResultIndex: 0, Expectation: int64(2),
		},
		// contains
		{
			Code:        m + ".contains(key == 'a')",
			ResultIndex: 1, Expectation: true,
		},
		{
			Code:        m + ".contains(key == 'z')",
			ResultIndex: 1, Expectation: false,
		},
		{
			Code:        m + ".contains(value == 1)",
			ResultIndex: 1, Expectation: true,
		},
		{
			Code:        m + ".contains(value == 0)",
			ResultIndex: 1, Expectation: false,
		},
		// all
		{
			Code:        m + ".all(key == /[abc]/)",
			ResultIndex: 1, Expectation: true,
		},
		{
			Code:        m + ".all(key == 'a')",
			ResultIndex: 1, Expectation: false,
		},
		{
			Code:        m + ".all(value > 0)",
			ResultIndex: 1, Expectation: true,
		},
		{
			Code:        m + ".all(value == 0)",
			ResultIndex: 1, Expectation: false,
		},
		// none
		{
			Code:        m + ".none(key == /[m-z]/)",
			ResultIndex: 1, Expectation: true,
		},
		{
			Code:        m + ".none(key == /[b-z]/)",
			ResultIndex: 1, Expectation: false,
		},
		{
			Code:        m + ".none(value < 1)",
			ResultIndex: 1, Expectation: true,
		},
		{
			Code:        m + ".none(value <= 2)",
			ResultIndex: 1, Expectation: false,
		},
		// one
		{
			Code:        m + ".one(key == 'a')",
			ResultIndex: 1, Expectation: true,
		},
		{
			Code:        m + ".one(key == /[a-b]/)",
			ResultIndex: 1, Expectation: false,
		},
		{
			Code:        m + ".one(value == 2)",
			ResultIndex: 1, Expectation: true,
		},
		{
			Code:        m + ".one(value == 1)",
			ResultIndex: 1, Expectation: false,
		},
	})
}

func TestTime(t *testing.T) {
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        "time.now.inRange(time.now, time.tomorrow)",
			ResultIndex: 1, Expectation: true,
		},
		{
			Code:        "time.now.inRange(time.tomorrow, time.tomorrow)",
			ResultIndex: 1, Expectation: false,
		},
	})
}

func TestVersion(t *testing.T) {
	t.Run("regular version", func(t *testing.T) {
		x.TestSimple(t, []testutils.SimpleTest{
			{
				Code:        "version('1.2.3') == version('1.2.3')",
				ResultIndex: 2, Expectation: true,
			},
			{
				Code:        "version('1.2.3') == version('1.2')",
				ResultIndex: 2, Expectation: false,
			},
			{
				Code:        "version('1.2') < version('1.10.2')",
				ResultIndex: 2, Expectation: true,
			},
			{
				Code:        "version('1.10') >= version('1.2.3')",
				ResultIndex: 2, Expectation: true,
			},
			{
				Code:        "version('1.10') >= '1.2'",
				ResultIndex: 2, Expectation: true,
			},
		})
	})

	t.Run("one-sided epoch", func(t *testing.T) {
		x.TestSimple(t, []testutils.SimpleTest{
			{
				Code:        "version('1.2.3') == version('1:1.2.3')",
				ResultIndex: 2, Expectation: false,
			},
			{
				Code:        "version('2:1.2.3') == version('1.2.3')",
				ResultIndex: 2, Expectation: false,
			},
			{
				Code:        "version('3:1.2') < version('1.10.2')",
				ResultIndex: 2, Expectation: false,
			},
			{
				Code:        "version('1.10') >= version('4:1.2.3')",
				ResultIndex: 2, Expectation: false,
			},
			{
				Code:        "version('1.2') <= version('3:1.10.2')",
				ResultIndex: 2, Expectation: true,
			},
			{
				Code:        "version('4:1.10') > version('1.2.3')",
				ResultIndex: 2, Expectation: true,
			},
		})
	})

	t.Run("deb/rpm epochs", func(t *testing.T) {
		x.TestSimple(t, []testutils.SimpleTest{
			{
				Code:        "version('1.2.3').epoch",
				ResultIndex: 0, Expectation: int64(0),
			},
			{
				Code:        "version('7:1.2.3').epoch",
				ResultIndex: 0, Expectation: int64(7),
			},
		})
	})

	t.Run("python epochs", func(t *testing.T) {
		x.TestSimple(t, []testutils.SimpleTest{
			{
				Code:        "version('1.2.3').epoch",
				ResultIndex: 0, Expectation: int64(0),
			},
			{
				Code:        "version('5!1.2.3').epoch",
				ResultIndex: 0, Expectation: int64(5),
			},
		})
	})

	t.Run("different epochs", func(t *testing.T) {
		x.TestSimple(t, []testutils.SimpleTest{
			{
				Code:        "version('2:1.2.3') == version('1:1.2.3')",
				ResultIndex: 2, Expectation: false,
			},
			{
				Code:        "version('2:1.2.3') == version('3:1.2.3')",
				ResultIndex: 2, Expectation: false,
			},
			{
				Code:        "version('3:1.2') < version('1:1.10.2')",
				ResultIndex: 2, Expectation: false,
			},
			{
				Code:        "version('2:1.10') >= version('4:1.2.3')",
				ResultIndex: 2, Expectation: false,
			},
			{
				Code:        "version('2:1.2') <= version('3:1.0.2')",
				ResultIndex: 2, Expectation: true,
			},
			{
				Code:        "version('4:1.1') > version('1:1.2.3')",
				ResultIndex: 2, Expectation: true,
			},
		})
	})

	t.Run("version with equal epochs", func(t *testing.T) {
		x.TestSimple(t, []testutils.SimpleTest{
			{
				Code:        "version('1:1.2.3') == version('1:1.2.3')",
				ResultIndex: 2, Expectation: true,
			},
			{
				Code:        "version('2:1.2.3') == version('2:1.2')",
				ResultIndex: 2, Expectation: false,
			},
			{
				Code:        "version('3:1.2') < version('3:1.10.2')",
				ResultIndex: 2, Expectation: true,
			},
			{
				Code:        "version('4:1.10') >= version('4:1.2.3')",
				ResultIndex: 2, Expectation: true,
			},
			{
				Code:        "version('5:1.10') >= '5:1.2'",
				ResultIndex: 2, Expectation: true,
			},
		})
	})

	t.Run("version type set to semver", func(t *testing.T) {
		x.TestSimple(t, []testutils.SimpleTest{
			{
				Code:        "version('1.2.3', type: 'semver')",
				ResultIndex: 0, Expectation: "1.2.3",
			},
		})
		x.TestSimpleErrors(t, []testutils.SimpleTest{
			{
				Code:        "version('1:1.2.3', type: 'semver')",
				ResultIndex: 1, Expectation: "version '1:1.2.3' is not a semantic version",
			},
		})
	})
}

func TestResource_Default(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock())
	res := x.TestQuery(t, "mondoo")
	require.NotEmpty(t, res)
	vals := res[0].Data.Value.(map[string]interface{})
	require.NotNil(t, vals)
	require.Equal(t, llx.StringData("unstable"), vals["J4anmJ+mXJX380Qslh563U7Bs5d6fiD2ghVxV9knAU0iy/P+IVNZsDhBbCmbpJch3Tm0NliAMiaY47lmw887Jw=="])
}

func TestBrokenQueryExecution(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock())
	bundle, err := x.Compile("'asdf'.contains('asdf') == true")
	require.NoError(t, err)
	bundle.CodeV2.Blocks[0].Chunks[1].Id = "fakecontains"

	results := x.TestMqlc(t, bundle, nil)
	require.Len(t, results, 3)
	require.Error(t, results[0].Data.Error)
	require.Error(t, results[1].Data.Error)
	require.Error(t, results[2].Data.Error)
}
