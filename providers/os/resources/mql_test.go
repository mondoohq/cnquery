// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/testutils"
)

// Core Language constructs
// ------------------------
// These tests are more generic MQL and resource tests. We have migrated them
// from their previous core package into the OS package, because it requires
// more resources (like file). Long-term we'd like to move them to a standalone
// (and dedicated) mock provider for testing. Other tests are found in the
// core provider counterpart to this test file.

func testChain(t *testing.T, codes ...string) {
	tr := testutils.InitTester(testutils.LinuxMock())
	for i := range codes {
		code := codes[i]
		t.Run(code, func(t *testing.T) {
			tr.TestQuery(t, code)
		})
	}
}

func TestErroneousLlxChains(t *testing.T) {
	testChain(t,
		`file("/etc/crontab") {
			permissions.group_readable == false
			permissions.group_writeable == false
			permissions.group_executable == false
		}`,
	)

	testChain(t,
		`file("/etc/profile").content.contains("umask 027") || file("/etc/bashrc").content.contains("umask 027")`,
		`file("/etc/profile").content.contains("umask 027") || file("/etc/bashrc").content.contains("umask 027")`,
	)

	testChain(t,
		`users.map(name) { _.contains("a") _.contains("b") }`,
	)

	testChain(t,
		`user(name: 'i_definitely_dont_exist').authorizedkeys`,
	)
}

func TestResource_InitWithResource(t *testing.T) {
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        "file(asset.platform).exists",
			Expectation: false,
		},
		{
			Code:        "'linux'.contains(asset.family)",
			Expectation: true,
		},
	})
}

func TestOS_Vars(t *testing.T) {
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        "p = file('/dummy.json'); parse.json(file: p).params.length",
			Expectation: int64(15),
		},
	})
}

func TestMap(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock())
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        "{a: 123}",
			Expectation: map[string]any{"a": int64(123)},
		},
		{
			Code:        "return {a: 123}",
			Expectation: map[string]any{"a": int64(123)},
		},
		{
			Code:        "{a: 1, b: 2, c: 3}.where(key == 'c')",
			Expectation: map[string]any{"c": int64(3)},
		},
		{
			Code:        "{a: 1, b: 2, c: 3}.where(value < 3)",
			Expectation: map[string]any{"a": int64(1), "b": int64(2)},
		},
		{
			Code:        "parse.xml('/dummy.xml').params.length",
			Expectation: int64(1),
		},
		{
			Code:        "parse.xml('/dummy.xml').params.root.box.length",
			Expectation: int64(3),
		},
		{
			Code:        "parse.json('/dummy.json').params.length",
			Expectation: int64(15),
		},
		{
			Code:        "parse.json('/dummy.json').params.keys.length",
			Expectation: int64(15),
		},
		{
			Code:        "parse.json('/dummy.json').params.values.length",
			Expectation: int64(15),
		},
		{
			Code: "parse.json('/dummy.json').params { _['Protocol'] != 1 }",
			Expectation: map[string]any{
				"__t": llx.BoolTrue,
				"__s": llx.BoolTrue,
				"CQ28lTwZsvVdJM4dCyeTdbQhExY8oiUIcMoPyPjXAJNgtjMLnHK6qgEVywRY1Hbw9QqInuL06EWIOaEMj2e9NA==": llx.BoolTrue,
			},
		},
	})
}

func TestListResource(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock())

	t.Run("list resource by default returns the list", func(t *testing.T) {
		res := x.TestQuery(t, "users")
		assert.NotEmpty(t, res)
		assert.Len(t, res[0].Data.Value, 4)
	})

	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        "users.where(name == 'root').length",
			Expectation: int64(1),
		},
		{
			Code:        "users.list.where(name == 'root').length",
			Expectation: int64(1),
		},
		{
			Code:        "users.where(name == 'rooot').list { uid }",
			Expectation: []any{},
		},
		{
			Code:        "users.where(uid > 0).where(uid < 0).list",
			Expectation: []any{},
		},
		{
			Code: `users.where(name == 'root').list {
				uid == 0
				gid == 0
			}`,
			Expectation: []any{
				map[string]any{
					"__t": llx.BoolTrue,
					"__s": llx.BoolTrue,
					"BamDDGp87sNG0hVjpmEAPEjF6fZmdA6j3nDinlgr/y5xK3KaLgulyscoeEEaEASm2RkRXifnWj3ZbF0OZBF6XA==": llx.BoolTrue,
					"ytOUfV4UyOjY0C6HKzQ8GcA/hshrh2ahRySNG41RbFt3TNNf+6gBuHvs2hGTNDPUZR/oN8WH0QFIYYm/Vj3pGQ==": llx.BoolTrue,
				},
			},
		},
		{
			Code:        "users.map(name)",
			Expectation: []any([]any{"root", "bin", "chris", "christopher"}),
		},
		{
			// outside variables cause the block to be standalone
			Code:        "n=false; users.contains(n)",
			ResultIndex: 1,
			Expectation: false,
		},
		{
			// variables do not override local fields in blocks
			Code:        "name=false; users.contains(name)",
			ResultIndex: 1,
			Expectation: true,
		},
	})
}

func TestListResource_Assertions(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock())
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        "users.contains(name == 'root')",
			ResultIndex: 1,
			Expectation: true,
		},
		{
			Code:        "users.where(uid < 100).contains(name == 'root')",
			ResultIndex: 1,
			Expectation: true,
		},
		{
			Code:        "users.all(uid >= 0)",
			Expectation: true,
		},
		{
			Code:        "users.where(uid < 100).all(uid >= 0)",
			Expectation: true,
		},
		{
			Code:        "users.any(uid < 100)",
			Expectation: true,
		},
		{
			Code:        "users.where(uid < 100).any(uid < 50)",
			Expectation: true,
		},
		{
			Code:        "users.one(uid == 0)",
			Expectation: true,
		},
		{
			Code:        "users.where(uid < 100).one(uid == 0)",
			Expectation: true,
		},
		{
			Code:        "users.none(uid == 99999)",
			Expectation: true,
		},
		{
			Code:        "users.where(uid < 100).none(uid == 1000)",
			Expectation: true,
		},
	})
}

func TestResource_duplicateFields(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock())
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code: "users.list.duplicates(gid) { gid }",
			Expectation: []any{
				map[string]any{
					"__t": llx.BoolTrue,
					"__s": llx.NilData,
					"Cuv5ImO3PMlg/BnsKFcT/K88cResNOFnEZnbYwBT44aycwbRuvhhMqjq0E96i+POSgNSxO1QPi6U2VNNRuSPtQ==": &llx.RawData{
						Type:  "\x05",
						Value: int64(1000),
						Error: nil,
					},
				},
				map[string]any{
					"__t": llx.BoolTrue,
					"__s": llx.NilData,
					"Cuv5ImO3PMlg/BnsKFcT/K88cResNOFnEZnbYwBT44aycwbRuvhhMqjq0E96i+POSgNSxO1QPi6U2VNNRuSPtQ==": &llx.RawData{
						Type:  "\x05",
						Value: int64(1000),
						Error: nil,
					},
				},
			},
		},
	})
}

func TestDict_Methods_In(t *testing.T) {
	p := "parse.json('/dummy.json')."

	x := testutils.InitTester(testutils.LinuxMock())
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        p + "params['hello'].in(['1','2','hello'])",
			ResultIndex: 1,
			Expectation: true,
		},
		{
			Code:        p + "params['hello'].in(['1','2'])",
			ResultIndex: 1,
			Expectation: false,
		},
		{
			// embedded value doesn't exist
			Code:        p + "params.e.hi.in(['hello','world'])",
			ResultIndex: 1,
			Expectation: true,
		},
		{
			// embedded value doesn't exist
			Code:        p + "params.e.hi.in(['world'])",
			ResultIndex: 1,
			Expectation: false,
		},
	})
}

func TestDict_Methods_NotIn(t *testing.T) {
	p := "parse.json('/dummy.json')."

	x := testutils.InitTester(testutils.LinuxMock())
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        p + "params['hello'].notIn(['1','2','hello'])",
			ResultIndex: 1,
			Expectation: false,
		},
		{
			Code:        p + "params['hello'].notIn(['1','2'])",
			ResultIndex: 1,
			Expectation: true,
		},
	})
}

func TestDict_Methods_InRange(t *testing.T) {
	p := "parse.json('/dummy.json')."

	x := testutils.InitTester(testutils.LinuxMock())
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        p + "params['1'].inRange(1,3)",
			ResultIndex: 1,
			Expectation: true,
		},
		{
			Code:        p + "params['1'].inRange(3,4)",
			ResultIndex: 1,
			Expectation: false,
		},
		{
			// value doesn't exist
			Code:        p + "params['123'].inRange(0,999)",
			ResultIndex: 1,
			Expectation: false,
		},
	})
}

func TestDict_Methods_Contains(t *testing.T) {
	p := "parse.json('/dummy.json')."

	x := testutils.InitTester(testutils.LinuxMock())
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        p + "params['hello'].contains('ll')",
			ResultIndex: 2,
			Expectation: true,
		},
		{
			Code:        p + "params['hello'].contains('lloo')",
			ResultIndex: 2,
			Expectation: false,
		},
		{
			Code:        p + "params['hello'].contains(['xx','he'])",
			ResultIndex: 2,
			Expectation: true,
		},
		{
			Code:        p + "params['hello'].contains(['xx'])",
			ResultIndex: 2,
			Expectation: false,
		},
		{
			Code:        p + "params['string-array'].contains('a')",
			ResultIndex: 2,
			Expectation: true,
		},
		{
			Code:        p + "params['string-array'].containsOnly(['c', 'a', 'b'])",
			ResultIndex: 2,
			Expectation: true,
		},
		{
			Code:        p + "params['string-array'].containsOnly(['a', 'b'])",
			ResultIndex: 2,
			Expectation: false,
		},
		// {
		// 	p + "params['string-array'].containsOnly('a')",
		// 	1, false,
		// },
		{
			Code:        p + "params['string-array'].containsNone(['d','e'])",
			ResultIndex: 2,
			Expectation: true,
		},
		{
			Code:        p + "params['string-array'].containsNone(['a', 'e'])",
			ResultIndex: 2,
			Expectation: false,
		},
		{
			Code:        p + "params['string-array'].containsNone([/z/, /ひ/])",
			ResultIndex: 2,
			Expectation: true,
		},
		{
			Code:        p + "params['string-array'].containsNone([/a/, /z/])",
			ResultIndex: 2,
			Expectation: false,
		},
		{
			Code:        p + "params['string-array'].none('a')",
			ResultIndex: 2,
			Expectation: false,
		},
		{
			Code:        p + "params['string-array'].contains(_ == 'a')",
			ResultIndex: 2,
			Expectation: true,
		},
		{
			Code:        p + "params['string-array'].none(_ == /a/)",
			ResultIndex: 2,
			Expectation: false,
		},
		{
			Code:        p + "params['string-array'].contains(value == 'a')",
			ResultIndex: 2,
			Expectation: true,
		},
		{
			Code:        p + "params['string-array'].none(value == 'a')",
			ResultIndex: 2,
			Expectation: false,
		},
	})
}

func TestDict_Methods_Map(t *testing.T) {
	p := "parse.json('/dummy.json')."

	expectedTime, err := time.Parse(time.RFC3339, "2016-01-28T23:02:24Z")
	if err != nil {
		panic(err.Error())
	}

	x := testutils.InitTester(testutils.LinuxMock())
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        p + "params.nonexistent.contains('sth')",
			Expectation: false,
			ResultIndex: 1,
		},
		{
			Code:        p + "params['string-array'].where(_ == 'a')",
			Expectation: []any{"a"},
		},
		{
			Code:        p + "params.users.recurse(name != empty).map(name)",
			Expectation: []any{"yor", "loid", "anya"},
		},
		{
			Code:        p + "params['string-array'].in(['a', 'b', 'c'])",
			Expectation: true,
		},
		{
			Code:        p + "params['string-array'].in(['z', 'b'])",
			Expectation: false,
		},
		{
			Code:        p + "params['string-array'].one(_ == 'a')",
			ResultIndex: 1,
			Expectation: true,
		},
		{
			Code:        p + "params['string-array'].all(_ != 'z')",
			ResultIndex: 1,
			Expectation: true,
		},
		{
			Code:        p + "params['string-array'].any(_ != 'a')",
			ResultIndex: 1,
			Expectation: true,
		},
		{
			Code:        p + "params['does_not_exist'].any(_ != 'a')",
			ResultIndex: 1,
			Expectation: nil,
		},
		{
			Code:        p + "params['f'].map(_['ff'])",
			Expectation: []any{float64(3)},
		},
		// {
		// 	p + "params { _['1'] == _['1.0'] }",
		// 	0, true,
		// },
		{
			Code:        p + "params['1'] - 2",
			Expectation: float64(-1),
		},
		{
			Code:        p + "params['int-array']",
			Expectation: []any{float64(1), float64(2), float64(3)},
		},
		{
			Code:        p + "params['hello'] + ' world'",
			Expectation: "hello world",
		},
		{
			Code:        p + "params['hello'].trim('ho')",
			Expectation: "ell",
		},
		{
			Code:        p + "params['dict'].length",
			Expectation: int64(3),
		},
		{
			Code:        p + "params['dict'].keys.length",
			Expectation: int64(3),
		},
		{
			Code:        p + "params['dict'].values.length",
			Expectation: int64(3),
		},
		{
			Code:        "parse.date(" + p + "params['date'])",
			Expectation: &expectedTime,
		},
		{
			Code:        p + "params.first",
			Expectation: float64(1),
		},
		{
			Code:        p + "params.last",
			Expectation: "🌒",
		},
		{
			Code:        p + "params['aoa'].flat",
			Expectation: []any{float64(1), float64(2), float64(3)},
		},
	})

	x.TestSimpleErrors(t, []testutils.SimpleTest{
		{
			Code:        p + "params['does not exist'].values",
			Expectation: "Failed to get values of `null`",
		},
		{
			Code:        p + "params['yo'] > 3",
			ResultIndex: 1,
			Expectation: "left side of operation is null",
		},
	})
}

func TestDict_Methods_Array(t *testing.T) {
	p := "parse.json('/dummy.array.json')."

	x := testutils.InitTester(testutils.LinuxMock())
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        p + "params[0]",
			Expectation: float64(1),
		},
		{
			Code:        p + "params[1]",
			Expectation: "hi",
		},
		{
			Code:        p + "params[2]",
			Expectation: map[string]any{"ll": float64(0)},
		},
		{
			Code:        p + "params.first",
			Expectation: float64(1),
		},
		{
			Code:        p + "params.last",
			Expectation: "z",
		},
		{
			Code:        p + "params.where(-1).first",
			Expectation: nil,
		},
		{
			Code:        p + "params.where(-1).last",
			Expectation: nil,
		},
	})
}

func TestDict_Methods_OtherJson(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock())
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        "parse.json('/dummy.number.json').params",
			Expectation: float64(1.23),
		},
		{
			Code:        "parse.json('/dummy.string.json').params",
			Expectation: "hi",
		},
		{
			Code:        "parse.json('/dummy.true.json').params",
			Expectation: true,
		},
		{
			Code:        "parse.json('/dummy.false.json').params",
			Expectation: false,
		},
		{
			Code:        "parse.json('/dummy.null.json').params",
			Expectation: nil,
		},
	})
}

func TestDict_KeyNotFound(t *testing.T) {
	p := "parse.json('/dummy.json')."
	keyNotFoundErr := func(key, suggestion string) string {
		return fmt.Sprintf("key '%s' not found, did you mean '%s'? (keys are case-sensitive)", key, suggestion)
	}

	x := testutils.InitTester(testutils.LinuxMock())
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        p + "params.nonExistentKey",
			Expectation: nil,
		},
		{
			Code:        p + "params['nonExistentKey']",
			Expectation: nil,
		},
		{
			Code:  p + "params.Hello",
			Error: keyNotFoundErr("Hello", "hello"),
		},
		{
			Code:  p + "params['HELLO']",
			Error: keyNotFoundErr("HELLO", "hello"),
		},
		{
			Code:  p + "params.dict.EE",
			Error: keyNotFoundErr("EE", "ee"),
		},
		{
			Code:        p + "params.hello",
			Expectation: "hello",
		},
		{
			Code:        p + "params.dict.ee",
			Expectation: float64(3),
		},
	})
}

func TestDict_KeyNotFound_ChainedFunctions(t *testing.T) {
	p := "parse.json('/dummy.json')."
	keyNotFoundErr := func(key, suggestion string) string {
		return fmt.Sprintf("key '%s' not found, did you mean '%s'? (keys are case-sensitive)", key, suggestion)
	}

	x := testutils.InitTester(testutils.LinuxMock())
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:  p + "params.Hello.length",
			Error: keyNotFoundErr("Hello", "hello"),
		},
		{
			Code:  p + "params.Hello == 'hello'",
			Error: keyNotFoundErr("Hello", "hello"),
		},
		{
			Code:  p + "params.Hello != empty",
			Error: keyNotFoundErr("Hello", "hello"),
		},
		{
			Code:  p + "params.dict.EE == 3",
			Error: keyNotFoundErr("EE", "ee"),
		},
		{
			Code:        p + "params.nonExistentKey.length",
			Expectation: nil,
		},
		{
			Code:        p + "params.nonExistentKey == empty",
			Expectation: nil,
		},
		{
			Code:        p + "params.f.map(_['ff'])",
			Expectation: []any{float64(3)},
		},
	})
}

func TestDict_KeyNotFound_BlockOperations(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock())
	keyNotFoundErr := func(key, suggestion string) string {
		return fmt.Sprintf("key '%s' not found, did you mean '%s'?", key, suggestion)
	}

	tests := []struct {
		name       string
		code       string
		key        string
		suggestion string
	}{
		{"where", "parse.json('/dummy.json').params.f.where(_['FF'] == 3)", "FF", "ff"},
		{"all", "parse.json('/dummy.json').params.f.all(_['FF'] == 3)", "FF", "ff"},
		{"any", "parse.json('/dummy.json').params.f.any(_['FF'] == 3)", "FF", "ff"},
		{"one", "parse.json('/dummy.json').params.f.one(_['FF'] == 3)", "FF", "ff"},
		{"none", "parse.json('/dummy.json').params.f.none(_['FF'] == 3)", "FF", "ff"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := x.TestQuery(t, tc.code)
			require.NotEmpty(t, res)
			lastResult := res[len(res)-1]
			require.NotNil(t, lastResult)
			require.Error(t, lastResult.Data.Error)
			assert.Contains(t, lastResult.Data.Error.Error(), keyNotFoundErr(tc.key, tc.suggestion))
		})
	}
}

func TestDict_KeyNotFound_FuzzyMatching(t *testing.T) {
	// Tests fuzzy matching suggestions for typos (not just case differences).
	// The threshold is 40% of key length (min 2 edits), so:
	// - "hello" (5 chars): allows up to 2 edits
	// - "string-array" (12 chars): allows up to 4 edits
	// - "zzzlast" (7 chars): allows up to 2 edits
	p := "parse.json('/dummy.json')."
	keyNotFoundErr := func(key, suggestion string) string {
		return fmt.Sprintf("key '%s' not found, did you mean '%s'?", key, suggestion)
	}

	x := testutils.InitTester(testutils.LinuxMock())

	// Test cases where fuzzy matching SHOULD suggest a key
	testsWithSuggestion := []struct {
		name       string
		code       string
		key        string
		suggestion string
	}{
		// 1 edit distance - should always suggest
		{"1 char typo in hello", p + "params.hallo", "hallo", "hello"},
		{"1 char missing in hello", p + "params.hell", "hell", "hello"},
		{"1 char added to hello", p + "params.helloo", "helloo", "hello"},

		// 2 edit distance - still within threshold for 5+ char keys
		{"2 char typo in hello", p + "params.hillo", "hillo", "hello"},
		{"swap chars in hello", p + "params.hlelo", "hlelo", "hello"},

		// longer keys allow more edits (string-array = 12 chars, allows 4 edits)
		{"typo in string-array", p + "params['string-aray']", "string-aray", "string-array"},
		{"2 typos in string-array", p + "params['strng-aray']", "strng-aray", "string-array"},
		{"3 typos in string-array", p + "params['strng-arry']", "strng-arry", "string-array"},

		// nested dict fuzzy matching
		{"typo in nested key", p + "params.dict.ea", "ea", "ee"},
	}

	for _, tc := range testsWithSuggestion {
		t.Run(tc.name, func(t *testing.T) {
			res := x.TestQuery(t, tc.code)
			require.NotEmpty(t, res)
			lastResult := res[len(res)-1]
			require.NotNil(t, lastResult)
			require.Error(t, lastResult.Data.Error)
			assert.Contains(t, lastResult.Data.Error.Error(), keyNotFoundErr(tc.key, tc.suggestion))
		})
	}

	// Test cases where fuzzy matching should NOT suggest (too different)
	// These should return nil (backward compatible)
	x.TestSimple(t, []testutils.SimpleTest{
		{
			// completely different key - no suggestion, returns nil
			Code:        p + "params.xyz",
			Expectation: nil,
		},
		{
			// too many edits for short key (ee = 2 chars, "ab" is 2 edits away)
			Code:        p + "params.dict.ab",
			Expectation: nil,
		},
		{
			// way too different from any key
			Code:        p + "params.completelyWrong",
			Expectation: nil,
		},
		{
			// empty-ish keys don't match
			Code:        p + "params.x",
			Expectation: nil,
		},
	})
}

func TestArrayBlockError(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock())
	res := x.TestQuery(t, "users.list { file(_.name + 'doesnotexist').content }")
	assert.NotEmpty(t, res)
	queryResult := res[len(res)-1]
	require.NotNil(t, queryResult)
	require.Error(t, queryResult.Data.Error)
}

func TestBrokenQueryExecutionGH674(t *testing.T) {
	// See https://github.com/mondoohq/cnquery/issues/674
	x := testutils.InitTester(testutils.LinuxMock())
	bundle, err := x.Compile(`
a = file("/tmp/ref1").content.trim
file(a).path == "/tmp/ref2"
file(a).content.trim == "asdf"
	`)
	require.NoError(t, err)

	results := x.TestMqlc(t, bundle, nil)
	require.Len(t, results, 5)
}
