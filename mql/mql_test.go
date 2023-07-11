package mql_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/mql"
	"go.mondoo.com/cnquery/providers"
	"go.mondoo.com/cnquery/providers/mock"
)

var features cnquery.Features

func init() {
	features = getEnvFeatures()
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

func initRuntime() llx.Runtime {
	m, err := mock.NewFromTomlFile("./testdata/arch.toml")
	if err != nil {
		panic(err.Error())
	}
	if err = m.LoadSchemas(providers.Coordinator.LoadSchema); err != nil {
		panic(err.Error())
	}
	return m
}

func TestMqlSimple(t *testing.T) {
	tests := []struct {
		query     string
		assertion interface{}
	}{
		{"asset.platform", "arch"},
		{"asset { platform version }", map[string]interface{}{
			"platform": "arch",
			"version":  "",
		}},
		{"users { name uid }", []interface{}{
			map[string]interface{}{"name": "root", "uid": int64(0)},
			map[string]interface{}{"name": "chris", "uid": int64(1000)},
			map[string]interface{}{"name": "christopher", "uid": int64(1000)},
			map[string]interface{}{"name": "chris", "uid": int64(1002)},
			map[string]interface{}{"name": "bin", "uid": int64(1)},
		}},
	}

	for i := range tests {
		one := tests[i]
		t.Run(one.query, func(t *testing.T) {
			runtime := initRuntime()
			res, err := mql.Exec(one.query, runtime, features, nil)
			assert.NoError(t, err)
			assert.NoError(t, res.Error)
			assert.Equal(t, one.assertion, res.Value)
		})
	}
}

func TestCustomData(t *testing.T) {
	query := "{ \"a\": \"valuea\", \"b\": \"valueb\"}"

	runtime := initRuntime()
	value, err := mql.Exec(query, runtime, features, nil)
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"a": "valuea", "b": "valueb"}, value.Value)
}

func TestMqlProps(t *testing.T) {
	query := "props.a + props.b"
	props := map[string]*llx.Primitive{
		"a": llx.IntPrimitive(2),
		"b": llx.IntPrimitive(2),
	}

	runtime := initRuntime()
	value, err := mql.Exec(query, runtime, features, props)
	require.NoError(t, err)
	assert.Equal(t, int64(4), value.Value)
}

func TestMqlIfElseProps(t *testing.T) {
	me := mql.New(initRuntime(), cnquery.DefaultFeatures)
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
	me := mql.New(initRuntime(), cnquery.DefaultFeatures)
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
