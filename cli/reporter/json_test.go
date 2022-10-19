package reporter

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/logger"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/providers/mock"
	"go.mondoo.com/cnquery/mql"
	"go.mondoo.com/cnquery/mqlc"
	"go.mondoo.com/cnquery/resources"
	resource_pack "go.mondoo.com/cnquery/resources/packs/os"
	"go.mondoo.com/cnquery/shared"
	"gotest.tools/assert"
)

var features cnquery.Features

func init() {
	logger.InitTestEnv()
	features = getEnvFeatures()
}

func getEnvFeatures() cnquery.Features {
	env := os.Getenv("FEATURES")
	if env == "" {
		return cnquery.Features{byte(cnquery.PiperCode)}
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

func executionContext() (*resources.Schema, *resources.Runtime) {
	transport, err := mock.NewFromTomlFile("../../mql/testdata/arch.toml")
	if err != nil {
		panic(err.Error())
	}

	motor, err := motor.New(transport)
	if err != nil {
		panic(err.Error())
	}

	registry := resource_pack.Registry
	runtime := resources.NewRuntime(registry, motor)
	return registry.Schema(), runtime
}

func testQuery(t *testing.T, query string) (*llx.CodeBundle, map[string]*llx.RawResult) {
	schema, runtime := executionContext()
	codeBundle, err := mqlc.Compile(query, nil, mqlc.NewConfig(schema, features))
	require.NoError(t, err)

	results, err := mql.ExecuteCode(schema, runtime, codeBundle, nil, features)
	require.NoError(t, err)

	return codeBundle, results
}

type simpleTest struct {
	code     string
	expected string
}

func runSimpleTests(t *testing.T, tests []simpleTest) {
	var out strings.Builder
	w := shared.IOWriter{Writer: &out}

	for i := range tests {
		cur := tests[i]
		t.Run(cur.code, func(t *testing.T) {
			bundle, results := testQuery(t, cur.code)
			err := BundleResultsToJSON(bundle, results, &w)
			require.NoError(t, err)
			assert.Equal(t, cur.expected, out.String())
		})
	}
}

func TestJsonReporter(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"users.where(uid==0)",
			`{"users.where.list":[{"gid":0,"name":"root","uid":0}]}`,
		},
	})
}
