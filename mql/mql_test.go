// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mql_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v9"
	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/mql"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/testutils"
	"go.mondoo.com/cnquery/v9/types"
)

var features cnquery.Features

func init() {
	features = getEnvFeatures()
}

func runtime() llx.Runtime {
	return testutils.LinuxMock()
}

func getEnvFeatures() cnquery.Features {
	env := os.Getenv("FEATURES")
	if env == "" {
		return cnquery.Features{}
	}

	arr := strings.Split(env, ",")
	var fts cnquery.Features
	for i := range arr {
		v, ok := cnquery.FeaturesValue[arr[i]]
		if ok {
			fmt.Println("--> activate feature: " + arr[i])
			fts = append(features, byte(v))
		} else {
			panic("cannot find requested feature: " + arr[i])
		}
	}
	return fts
}

func TestMqlSimple(t *testing.T) {
	tests := []struct {
		query     string
		assertion interface{}
	}{
		{"asset.platform", "arch"},
		{"asset { platform version }", map[string]interface{}{
			"platform": "arch",
			"version":  "rolling",
		}},
		{"users { name uid }", []interface{}{
			map[string]interface{}{"name": "root", "uid": int64(0)},
			map[string]interface{}{"name": "bin", "uid": int64(1)},
			map[string]interface{}{"name": "chris", "uid": int64(1000)},
			map[string]interface{}{"name": "christopher", "uid": int64(1001)},
		}},
	}

	for i := range tests {
		one := tests[i]
		t.Run(one.query, func(t *testing.T) {
			res, err := mql.Exec(one.query, runtime(), features, nil)
			assert.NoError(t, err)
			assert.NoError(t, res.Error)
			assert.Equal(t, one.assertion, res.Value)
		})
	}
}

func TestCustomData(t *testing.T) {
	query := "{ \"a\": \"valuea\", \"b\": \"valueb\"}"

	value, err := mql.Exec(query, runtime(), features, nil)
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"a": "valuea", "b": "valueb"}, value.Value)
}

func TestMqlProps(t *testing.T) {
	query := "props.a + props.b"
	props := map[string]*llx.Primitive{
		"a": llx.IntPrimitive(2),
		"b": llx.IntPrimitive(2),
	}

	value, err := mql.Exec(query, runtime(), features, props)
	require.NoError(t, err)
	assert.Equal(t, int64(4), value.Value)
}

func TestMqlIfElseProps(t *testing.T) {
	me := mql.New(runtime(), cnquery.DefaultFeatures)
	query := "if (props.a > 2) { return {\"a\": \"valuea\"} } return {\"a\": \"valueb\"}"

	props := map[string]*llx.Primitive{
		"a": llx.IntPrimitive(3),
	}
	value, err := me.Exec(query, props)
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"a": "valuea"}, value.Value)

	props = map[string]*llx.Primitive{
		"a": llx.IntPrimitive(2),
	}
	value, err = me.Exec(query, props)
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"a": "valueb"}, value.Value)
}

func TestMqlIfAndProps(t *testing.T) {
	me := mql.New(runtime(), cnquery.DefaultFeatures)
	query := "if (props.a > 2) { return {\"a\": \"valuea\"} }"

	props := map[string]*llx.Primitive{
		"a": llx.IntPrimitive(3),
	}
	value, err := me.Exec(query, props)
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"a": "valuea"}, value.Value)

	props = map[string]*llx.Primitive{
		"a": llx.IntPrimitive(2),
	}
	value, err = me.Exec(query, props)
	require.NoError(t, err)
	assert.Equal(t, nil, value.Value)
}

func TestResourceAliases(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock())
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        "os.unix.sshd.config.file.path",
			ResultIndex: 0,
			Expectation: "/etc/ssh/sshd_config",
		},
		{
			Code:        "os.unix.sshd { config.file.path }",
			ResultIndex: 0,
			Expectation: map[string]interface{}{
				"_":   llx.ResourceData(&llx.MockResource{Name: "sshd"}, "os.unix.sshd"),
				"__s": llx.NilData,
				"__t": llx.BoolData(true),
				"k6rlXoYpV48Qd19gKeNl+/IiPnkI5VNQBiqZBca3gDKsIRiLcpXQUlDv52x9sscIWiqOMpC7+x/aBpY0IUq0ww==": llx.StringData("/etc/ssh/sshd_config"),
			},
		},
	})
}

func TestNullResources(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock())
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        "muser.group",
			ResultIndex: 0,
			Expectation: &llx.MockResource{Name: "mgroup", ID: "group one"},
		},
		{
			Code:        "muser.nullgroup",
			ResultIndex: 0,
			Expectation: nil,
		},
		{
			Code:        "muser.nullgroup.name",
			ResultIndex: 0,
			Expectation: nil,
		},
		{
			Code:        "muser.nullgroup == null",
			ResultIndex: 1,
			Expectation: true,
		},
		{
			Code:        "muser.nullgroup == empty",
			ResultIndex: 2,
			Expectation: true,
		},
		{
			Code:        "muser.groups.where(null) == empty",
			ResultIndex: 2,
			Expectation: false,
		},
		{
			Code:        "muser.groups.where(name == '').map(name) + ['one']",
			ResultIndex: 0,
			Expectation: []interface{}{"one"},
		},
		{
			Code:        "muser.groups.where(name == '') == empty",
			ResultIndex: 2,
			Expectation: true,
		},
		{
			Code:        "muser.groups",
			ResultIndex: 0,
			Expectation: []interface{}{
				&llx.MockResource{Name: "mgroup", ID: "group one"},
				nil,
			},
		},
		{
			Code:        "muser { nullgroup }",
			ResultIndex: 0,
			Expectation: map[string]interface{}{
				"_":   &llx.RawData{Type: types.Resource("muser"), Value: &llx.MockResource{Name: "muser"}},
				"__s": llx.NilData,
				"__t": llx.BoolTrue,
				"A8qiFMpyfjKsr3OzVu+L+43W0BvYXoCPiwM7zu8AFQkBYEBMvZfR73ZsdfIqswmN1n9Qs/Soc1D7qxJipXv/ZA==": llx.ResourceData(nil, "mgroup"),
			},
		},
	})
}

func TestNamedFunctions(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock())
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        "muser.groups.where(group: group != empty).length",
			ResultIndex: 0, Expectation: int64(1),
		},
		{
			Code:        "muser.groups.where(_: _ != empty).length",
			ResultIndex: 0, Expectation: int64(1),
		},
	})
}

func TestNullString(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock())
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        "muser.nullstring.contains('123')",
			ResultIndex: 0,
			Expectation: false,
		},
	})
}

func TestDictMethods(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock())
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        "muser.dict.nonexisting.contains('abc')",
			ResultIndex: 1,
			Expectation: false,
		},
	})
}
