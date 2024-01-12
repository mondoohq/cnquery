// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/testutils"
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
			Expectation: int64(11),
		},
	})
}

func TestMap(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock())
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        "{a: 123}",
			Expectation: map[string]interface{}{"a": int64(123)},
		},
		{
			Code:        "return {a: 123}",
			Expectation: map[string]interface{}{"a": int64(123)},
		},
		{
			Code:        "{a: 1, b: 2, c: 3}.where(key == 'c')",
			Expectation: map[string]interface{}{"c": int64(3)},
		},
		{
			Code:        "{a: 1, b: 2, c: 3}.where(value < 3)",
			Expectation: map[string]interface{}{"a": int64(1), "b": int64(2)},
		},
		{
			Code:        "parse.json('/dummy.json').params.length",
			Expectation: int64(11),
		},
		{
			Code:        "parse.json('/dummy.json').params.keys.length",
			Expectation: int64(11),
		},
		{
			Code:        "parse.json('/dummy.json').params.values.length",
			Expectation: int64(11),
		},
		{
			Code: "parse.json('/dummy.json').params { _['Protocol'] != 1 }",
			Expectation: map[string]interface{}{
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
			Expectation: []interface{}{},
		},
		{
			Code:        "users.where(uid > 0).where(uid < 0).list",
			Expectation: []interface{}{},
		},
		{
			Code: `users.where(name == 'root').list {
				uid == 0
				gid == 0
			}`,
			Expectation: []interface{}{
				map[string]interface{}{
					"__t": llx.BoolTrue,
					"__s": llx.BoolTrue,
					"BamDDGp87sNG0hVjpmEAPEjF6fZmdA6j3nDinlgr/y5xK3KaLgulyscoeEEaEASm2RkRXifnWj3ZbF0OZBF6XA==": llx.BoolTrue,
					"ytOUfV4UyOjY0C6HKzQ8GcA/hshrh2ahRySNG41RbFt3TNNf+6gBuHvs2hGTNDPUZR/oN8WH0QFIYYm/Vj3pGQ==": llx.BoolTrue,
				},
			},
		},
		{
			Code:        "users.map(name)",
			Expectation: []interface{}([]interface{}{"root", "bin", "chris", "christopher"}),
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
			Expectation: []interface{}{
				map[string]interface{}{
					"__t": llx.BoolTrue,
					"__s": llx.NilData,
					"Cuv5ImO3PMlg/BnsKFcT/K88cResNOFnEZnbYwBT44aycwbRuvhhMqjq0E96i+POSgNSxO1QPi6U2VNNRuSPtQ==": &llx.RawData{
						Type:  "\x05",
						Value: int64(1000),
						Error: nil,
					},
				},
				map[string]interface{}{
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
			Expectation: []interface{}{"a"},
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
			Expectation: []interface{}{float64(3)},
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
			Expectation: []interface{}{float64(1), float64(2), float64(3)},
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
			Expectation: true,
		},
		{
			Code:        p + "params['aoa'].flat",
			Expectation: []interface{}{float64(1), float64(2), float64(3)},
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
			Expectation: map[string]interface{}{"ll": float64(0)},
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
