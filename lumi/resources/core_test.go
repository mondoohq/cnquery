package resources_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/tj/assert"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/mock"
	"go.mondoo.io/mondoo/policy/executor"
)

func initExecutor() *executor.Executor {
	registry := lumi.NewRegistry()
	resources.Init(registry)

	transport, err := mock.NewFromToml(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_MOCK, Path: "./testdata/arch.toml"})
	if err != nil {
		panic(err.Error())
	}

	motor, err := motor.New(transport)
	if err != nil {
		panic(err.Error())
	}
	runtime := lumi.NewRuntime(registry, motor)

	executor := executor.New(registry.Schema(), runtime)

	return executor
}

func testQuery(t *testing.T, query string) []*llx.RawResult {
	executor := initExecutor()

	var results []*llx.RawResult
	executor.AddWatcher("test", func(res *llx.RawResult) {
		results = append(results, res)
	})
	defer executor.RemoveWatcher("test")

	bundle, err := executor.AddCode(query)
	if err != nil {
		t.Fatal("failed to add code to executor: " + err.Error())
	}
	defer executor.RemoveCode(bundle.Code.Id)

	if executor.WaitForResults(2*time.Second) == false {
		t.Fatal("ran into timeout on testing query " + query)
	}

	return results
}

func testResultsErrors(t *testing.T, r []*llx.RawResult) bool {
	var found bool
	for i := range r {
		err := r[i].Data.Error
		if err != nil {
			t.Error("result has error: " + err.Error())
			found = true
		}
	}
	return found
}

// StableTestRepetitions specifies the repetitions used in testing
// to see if queries are deterministic
var StableTestRepetitions = 5

func stableResults(t *testing.T, query string) map[string]*llx.RawResult {
	executor := initExecutor()
	results := make([]map[string]*llx.RawResult, StableTestRepetitions)

	for i := 0; i < StableTestRepetitions; i++ {
		results[i] = map[string]*llx.RawResult{}
		watcherID := "test"

		executor.AddWatcher(watcherID, func(res *llx.RawResult) {
			results[i][res.CodeID] = res
		})

		bundle, err := executor.AddCode(query)
		if err != nil {
			t.Fatal("failed to add code to executor: " + err.Error())
			return nil
		}
		if executor.WaitForResults(2*time.Second) == false {
			t.Fatal("ran into timeout on testing query " + query)
			return nil
		}

		executor.RemoveWatcher(watcherID)
		executor.RemoveCode(bundle.Code.Id)
	}

	first := results[0]
	for i := 1; i < StableTestRepetitions; i++ {
		next := results[i]
		for id, firstRes := range first {
			nextRes := next[id]

			if firstRes == nil {
				t.Fatalf("received nil as the result for query '%s' codeID '%s'", query, id)
				return nil
			}

			if nextRes == nil {
				t.Fatalf("received nil as the result for query '%s' codeID '%s'", query, id)
				return nil
			}

			firstData := firstRes.Data
			nextData := nextRes.Data
			if firstData.Value == nextData.Value && firstData.Error == nextData.Error {
				continue
			}

			if firstData.Value != nextData.Value {
				t.Errorf("unstable result for '%s'\n  first = %v\n  next = %v\n", query, firstData.Value, nextData.Value)
			}
			if firstData.Error != nextData.Error {
				t.Errorf("unstable result error for '%s'\n  error1 = %v\n  error2 = %v\n", query, firstData.Error, nextData.Error)
			}
			break
		}
	}

	return results[0]
}

type simpleTest struct {
	code        string
	resultIndex int
	expectation interface{}
}

func runSimpleTests(t *testing.T, tests []simpleTest) {
	for i := range tests {
		cur := tests[i]
		t.Run(cur.code, func(t *testing.T) {
			res := testQuery(t, cur.code)
			assert.NotEmpty(t, res)

			if len(res) <= cur.resultIndex {
				t.Error("insufficient results, looking for result idx " + strconv.Itoa(cur.resultIndex))
				return
			}

			assert.NotNil(t, res[cur.resultIndex].Result().Error)
			assert.Equal(t, cur.expectation, res[cur.resultIndex].Data.Value)
		})
	}
}

func runSimpleErrorTests(t *testing.T, tests []simpleTest) {
	for i := range tests {
		cur := tests[i]
		t.Run(cur.code, func(t *testing.T) {
			res := testQuery(t, cur.code)
			assert.NotEmpty(t, res)
			assert.Equal(t, cur.expectation, res[0].Result().Error)
			assert.Nil(t, res[0].Data.Value)
		})
	}
}

// func TestStableCore(t *testing.T) {
// 	res := stableResults(t, "mondoo.version")
// 	for _, v := range res {
// 		assert.Equal(t, "unstable", v.Data.Value)
// 	}
// }

func testTimeout(t *testing.T, codes ...string) {
	executor := initExecutor()
	for i := range codes {
		code := codes[i]
		t.Run(code, func(t *testing.T) {
			code, err := executor.AddCode(code)
			if err != nil {
				t.Error("failed to compile: " + err.Error())
				return
			}
			defer executor.RemoveCode(code.Code.Id)

			var timeoutTime = 5
			if !executor.WaitForResults(time.Duration(timeoutTime) * time.Second) {
				t.Error("ran into timeout after ", timeoutTime, " seconds")
				return
			}
		})
	}
}

func TestErroneousLlxChains(t *testing.T) {
	testTimeout(t, `file("/etc/crontab") {
		permissions.group_readable == false
		permissions.group_writeable == false
		permissions.group_executable == false
	}`)

	testTimeout(t,
		`file("/etc/profile").content.contains("umask 027") || file("/etc/bashrc").content.contains("umask 027")`,
		`file("/etc/profile").content.contains("umask 027") || file("/etc/bashrc").content.contains("umask 027")`,
	)

	testTimeout(t,
		`ntp.conf { settings.contains("a") settings.contains("b") }`,
	)
}

func TestResource_InitWithResource(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"command(platform.name).stdout",
			0, "",
		},
		{
			"'linux'.contains(platform.family)",
			0, true,
		},
	})
}

func TestCore_If(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"if ( mondoo.version != null ) { 123 }",
			1, map[string]interface{}{
				"NmGComMxT/GJkwpf/IcA+qceUmwZCEzHKGt+8GEh+f8Y0579FxuDO+4FJf0/q2vWRE4dN2STPMZ+3xG3Mdm1fA==": llx.IntData(123),
			},
		},
		{
			"if ( mondoo.version == null ) { 123 }",
			1, nil,
		},
	})
}

func TestBooleans(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"true || false || false",
			1, true,
		},
	})
	runSimpleTests(t, []simpleTest{
		{
			"false || true || false",
			1, true,
		},
	})
	runSimpleTests(t, []simpleTest{
		{
			"false || false || true",
			1, true,
		},
	})
}

func TestString_Methods(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"'hello'.contains('ll')",
			0, true,
		},
		{
			"'hello'.contains('lloo')",
			0, false,
		},
		{
			"'hello'.contains(['lo', 'la'])",
			0, true,
		},
		{
			"'hello'.contains(['lu', 'la'])",
			0, false,
		},
		{
			"'hello bob'.find(/he\\w*\\s?[bo]+/)",
			0, []interface{}{"hello bob"},
		},
		{
			"'HeLlO'.downcase",
			0, "hello",
		},
		{
			"'hello'.length",
			0, int64(5),
		},
		{
			"'hello world'.split(' ')",
			0, []interface{}{"hello", "world"},
		},
		{
			"'he\nll\no'.lines",
			0, []interface{}{"he", "ll", "o"},
		},
	})
}

func duration(i int64) *time.Time {
	res := llx.DurationToTime(i)
	return &res
}

func TestTime_Methods(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"time.now",
			1, true,
		},
		{
			"time.now.unix",
			0, time.Now().Unix(),
		},
		{
			"parse.date('0000-01-01T02:03:04Z').seconds",
			0, int64(4 + 3*60 + 2*60*60),
		},
		{
			"parse.date('0000-01-01T02:03:04Z').minutes",
			0, int64(3 + 2*60),
		},
		{
			"parse.date('0000-01-01T02:03:04Z').hours",
			0, int64(2),
		},
		{
			"parse.date('0000-01-11T02:03:04Z').days",
			0, int64(10),
		},
		{
			"parse.date('1970-01-01T01:02:03Z').unix",
			0, int64(1*60*60 + 02*60 + 03),
		},
		{
			"parse.date('1970-01-01T01:02:04Z') - parse.date('1970-01-01T01:02:03Z')",
			0, duration(1),
		},
		{
			"parse.date('0000-01-01T00:00:03Z') * 3",
			0, duration(9),
		},
		{
			"3 * time.second",
			0, duration(3),
		},
		{
			"3 * time.minute",
			0, duration(3 * 60),
		},
		{
			"3 * time.hour",
			0, duration(3 * 60 * 60),
		},
		{
			"3 * time.day",
			0, duration(3 * 60 * 60 * 24),
		},
		{
			"1 * time.day > 3 * time.hour",
			2, true,
		},
	})
}

func TestArray_Access(t *testing.T) {
	runSimpleErrorTests(t, []simpleTest{
		{
			"[0,1,2][100000]",
			0, "array index out of bound (trying to access element 100000, max: 2)",
		},
		{
			"sshd.config('1').params['2'] == '3'",
			0, "file '1' does not exist",
		},
	})
}

func TestArray_Block(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"[1,2,3] { _ == 2 }",
			0, []interface{}{
				map[string]interface{}{"H1/Sy2Mih0/ZbyAPVrYJgUuJH09rTBHw1CnafKZFa3wIrZzZsHEwKqr+bgBy6ymTjc1JW94vshmwLLW8kb4CtQ==": llx.BoolFalse},
				map[string]interface{}{"H1/Sy2Mih0/ZbyAPVrYJgUuJH09rTBHw1CnafKZFa3wIrZzZsHEwKqr+bgBy6ymTjc1JW94vshmwLLW8kb4CtQ==": llx.BoolTrue},
				map[string]interface{}{"H1/Sy2Mih0/ZbyAPVrYJgUuJH09rTBHw1CnafKZFa3wIrZzZsHEwKqr+bgBy6ymTjc1JW94vshmwLLW8kb4CtQ==": llx.BoolFalse},
			},
		},
		{
			"[1,2,3].where()",
			0, []interface{}{int64(1), int64(2), int64(3)},
		},
		{
			"[1,2,3].where(_ > 2)",
			0, []interface{}{int64(3)},
		},
		{
			"[1,2,3].where(_ >= 2)",
			0, []interface{}{int64(2), int64(3)},
		},
		{
			"[1,2,3].contains(_ >= 2)",
			1, true,
		},
		{
			"[1,2,3].one(_ == 2)",
			1, true,
		},
		{
			"[1,2,3].all(_ < 9)",
			2, true,
		},
		{
			"[1,2,3].any(_ > 1)",
			2, true,
		},
		{
			"[0].where(_ > 0).where(_ > 0)",
			0, []interface{}{},
		},
	})
}

func TestMap_Block(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"sshd.config.params { _['Protocol'] != 1 }",
			0, map[string]interface{}{
				"wY2itjYLEbmP9L3U2Z24a7jlTpJxpHoit+s8zoaBkbHW4itI+GhHF1lazZSPjH42eqY106gEXgr/IHV2Q5vB8g==": llx.BoolTrue,
			},
		},
	})
}

func TestResource_Where(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			`users.where(name == 'root').list {
				uid == 0
				gid == 0
			}`,
			0, []interface{}{
				map[string]interface{}{
					"IBJ7+s+IAJiwObGQnaqzH/11QzFL1t1OvBVk84sjZ658GMB1SM1n/TLJF8Y2hws3/qh0kj/JKM04PPQeam0HRA==": llx.BoolTrue,
					"hvIlu70nu2ZxrcctGtHb9WOI1uVTlQKM8YiQX9AC026dO8shkWue9yaruWPqhin9M2cZibXkTqSaVQavfB2yAQ==": llx.BoolTrue,
				},
			},
		},
		{
			"users.where(name == 'root').length",
			0, int64(1),
		},
		{
			"users.where(name == 'rooot').list { uid }",
			0, []interface{}{},
		},
		{
			"users.where(uid > 0).where(uid < 0).list",
			0, []interface{}{},
		},
		{
			"os.rootcertificates.where(  subject.commonname == '' ).length",
			0, int64(0),
		},
	})
}

func TestResource_Contains(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"users.contains(name == 'root')",
			1, true,
		},
		{
			"users.where(uid < 100).contains(name == 'root')",
			1, true,
		},
	})
}

func TestResource_All(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"users.all(uid >= 0)",
			2, true,
		},
		{
			"users.where(uid < 100).all(uid >= 0)",
			2, true,
		},
	})
}

func TestResource_One(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"users.one(uid == 0)",
			1, true,
		},
		{
			"users.where(uid < 100).one(uid == 0)",
			1, true,
		},
	})
}

func TestResource_Any(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"users.any(uid < 100)",
			2, true,
		},
		{
			"users.where(uid < 100).any(uid < 50)",
			1, true,
		},
	})
}

func TestDict_Methods(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"parse.json('/dummy.json').params['g'].where(_ == 'a')",
			0, []interface{}{"a"},
		},
		{
			"parse.json('/dummy.json').params['g'].one(_ == 'a')",
			1, true,
		},
		{
			"parse.json('/dummy.json').params['g'].all(_ != 'z')",
			2, true,
		},
		{
			"parse.json('/dummy.json').params['g'].any(_ != 'a')",
			1, true,
		},
		{
			"parse.json('/dummy.json').params { _['b'] == _['c'] }",
			1, true,
		},
		{
			"parse.json('/dummy.json').params['d'] { _ }",
			1, true,
		},
		{
			"parse.json('/dummy.json').params['h'] { _.contains('llo') }",
			1, true,
		},
	})
}
