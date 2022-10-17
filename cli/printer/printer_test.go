package printer

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
	results  []string
}

func runSimpleTests(t *testing.T, tests []simpleTest) {
	for i := range tests {
		cur := tests[i]
		t.Run(cur.code, func(t *testing.T) {
			bundle, results := testQuery(t, cur.code)

			s := DefaultPrinter.CodeBundle(bundle)
			if cur.expected != "" {
				assert.Equal(t, cur.expected, s)
			}

			length := len(results)

			assert.Equal(t, length, len(cur.results), "make sure the right number of results are returned")
			keys := make([]string, 0, len(results))
			for k := range results {
				keys = append(keys, k)
			}

			sort.Strings(keys)

			for idx, id := range keys {
				result, _ := results[id]
				s = DefaultPrinter.Result(result, bundle)
				assert.Equal(t, cur.results[idx], s)
			}
		})
	}
}

type assessmentTest struct {
	code   string
	result string
}

func runAssessmentTests(t *testing.T, tests []assessmentTest) {
	for i := range tests {
		cur := tests[i]
		t.Run(cur.code, func(t *testing.T) {
			bundle, resultsMap := testQuery(t, cur.code)

			raw := DefaultPrinter.Results(bundle, resultsMap)
			assert.Equal(t, cur.result, raw)
		})
	}
}

func TestPrinter(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"if ( mondoo.version != null ) { mondoo.build }",
			"", // ignore
			[]string{
				"mondoo.version: \"unstable\"",
				"if: {\n" +
					"  mondoo.build: \"development\"\n" +
					"}",
			},
		},
		{
			"file('zzz') { content }",
			"",
			[]string{
				"error: Query encountered errors:\n" +
					"1 error occurred:\n" +
					"\t* file not found: 'zzz' does not exist\n" +
					"file: {\n  content: error: file not found: 'zzz' does not exist\n}",
			},
		},
		{
			"[]",
			"", // ignore
			[]string{
				"[]",
			},
		},
		{
			"{}",
			"", // ignore
			[]string{
				"{}",
			},
		},
		{
			"['1-2'] { _.split('-') }",
			"", // ignore
			[]string{
				"[\n" +
					"  0: {\n" +
					"    split: [\n" +
					"      0: \"1\"\n" +
					"      1: \"2\"\n" +
					"    ]\n" +
					"  }\n" +
					"]",
			},
		},
		{
			"mondoo { version }",
			"-> block 1\n   entrypoints: [<1,2>]\n   1: mondoo \n   2: {} bind: <1,1> type:block (=> <2,0>)\n-> block 2\n   entrypoints: [<2,2>]\n   1: mondoo id = context\n   2: version bind: <2,1> type:string\n",
			[]string{
				"mondoo: {\n  version: \"unstable\"\n}",
			},
		},
		{
			"mondoo { _.version }",
			"-> block 1\n   entrypoints: [<1,2>]\n   1: mondoo \n   2: {} bind: <1,1> type:block (=> <2,0>)\n-> block 2\n   entrypoints: [<2,2>]\n   1: mondoo id = context\n   2: version bind: <2,1> type:string\n",
			[]string{
				"mondoo: {\n  version: \"unstable\"\n}",
			},
		},
		{
			"[1].where( _ > 0 )",
			"-> block 1\n   entrypoints: [<1,2>]\n   1: [\n     0: 1\n   ]\n   2: where bind: <1,1> type:[]int (ref<1,1>, => <2,0>)\n-> block 2\n   entrypoints: [<2,2>]\n   1: _\n   2: >\005 bind: <2,1> type:bool (0)\n",
			[]string{
				"where: [\n  0: 1\n]",
			},
		},
		{
			"a = 3\n if(true) {\n a == 3 \n}",
			"-> block 1\n   entrypoints: [<1,2>]\n   1: 3\n   2: if bind: <0,0> type:block (true, => <2,0>, [\n     0: ref<1,1>\n   ])\n-> block 2\n   entrypoints: [<2,2>]\n   1: ref<1,1>\n   2: ==\x05 bind: <1,1> type:bool (3)\n",
			[]string{"if: {\n   == 3: true\n}"},
		},
	})
}

func TestPrinter_Assessment(t *testing.T) {
	runAssessmentTests(t, []assessmentTest{
		{
			// mixed use: assertion and erroneous data field
			"mondoo.build == 1; user(name: 'notthere').authorizedkeys.file",
			strings.Join([]string{
				"[failed] mondoo.build == 1; user(name: 'notthere').authorizedkeys.file",
				"  [failed] mondoo.build == 1",
				"    expected: == 1",
				"    actual:   \"development\"",
				"  [failed] user.authorizedkeys.file",
				"    error: failed to create resource 'user': user 'notthere' does not exist",
				"",
			}, "\n"),
		},
		{
			// mixed use: assertion and working data field
			"mondoo.build == 1; user(name: 'root').authorizedkeys.file",
			strings.Join([]string{
				"[failed] mondoo.build == 1; user(name: 'root').authorizedkeys.file",
				"  [failed] mondoo.build == 1",
				"    expected: == 1",
				"    actual:   \"development\"",
				"  [ok] value: file id = /root/.ssh/authorized_keys",
				"",
			}, "\n"),
		},
		{
			"[1,2,3].\n" +
				"# @msg Found ${length} numbers\n" +
				"none( _ > 1 )",
			strings.Join([]string{
				"[failed] Found 2 numbers",
				"",
			}, "\n"),
		},
		{
			"# @msg Expected ${$expected.length} users but got ${length}\n" +
				"users.none( uid == 0 )",
			strings.Join([]string{
				"[failed] Expected 5 users but got 1",
				"",
			}, "\n"),
		},
		{
			"mondoo.build == 1",
			strings.Join([]string{
				"[failed] mondoo.build == 1",
				"  expected: == 1",
				"  actual:   \"development\"",
				"",
			}, "\n"),
		},
		{
			"sshd.config { params['test'] }",
			strings.Join([]string{
				"sshd.config: {",
				"  params[test]: null",
				"}",
			}, "\n"),
		},
		{
			"mondoo.build == 1;mondoo.version =='unstable';",
			strings.Join([]string{
				"[failed] mondoo.build == 1;mondoo.version =='unstable';",
				"  [failed] mondoo.build == 1",
				"    expected: == 1",
				"    actual:   \"development\"",
				"  [ok] value: \"unstable\"",
				"",
			}, "\n"),
		},
		{
			"if(true) {\n" +
				"  # @msg Expected ${$expected.length} users but got ${length}\n" +
				"  users.none( uid == 0 )\n" +
				"}",
			strings.Join([]string{
				"if: {",
				"  [failed] Expected 5 users but got 1",
				"}",
			}, "\n"),
		},
	})
}

func TestPrinter_Buggy(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"mondoo",
			"", // ignore
			[]string{
				"mondoo: mondoo id = mondoo",
			},
		},
	})
}
