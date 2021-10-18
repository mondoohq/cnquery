package resources_test

import (
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/logger"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/local"
	"go.mondoo.io/mondoo/motor/transports/mock"
	"go.mondoo.io/mondoo/policy/executor"
)

func init() {
	logger.InitTestEnv()
}

func mockTransport(path string) (*motor.Motor, error) {
	transport, err := mock.NewFromToml(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_MOCK, Path: path})
	if err != nil {
		panic(err.Error())
	}

	return motor.New(transport)
}

func initExecutor(motor *motor.Motor) *executor.Executor {
	registry := lumi.NewRegistry()
	resources.Init(registry)

	runtime := lumi.NewRuntime(registry, motor)

	executor := executor.New(registry.Schema(), runtime)

	return executor
}

func testQueryWithExecutor(t *testing.T, executor *executor.Executor, query string, props map[string]*llx.Primitive) []*llx.RawResult {
	var results []*llx.RawResult
	executor.AddWatcher("test", func(res *llx.RawResult) {
		results = append(results, res)
	})
	defer executor.RemoveWatcher("test")

	bundle, err := executor.AddCode(query, props)
	if err != nil {
		t.Fatal("failed to add code to executor: " + err.Error())
	}
	defer executor.RemoveCode(bundle.Code.Id, query)

	if executor.WaitForResults(2*time.Second) == false {
		t.Fatal("ran into timeout on testing query " + query)
	}

	return results
}

func localExecutor() *executor.Executor {
	transport, err := local.New()
	if err != nil {
		panic(err.Error())
	}

	m, err := motor.New(transport)
	if err != nil {
		panic(err.Error())
	}

	executor := initExecutor(m)
	return executor
}

func mockExecutor(path string) *executor.Executor {
	m, err := mockTransport(path)
	if err != nil {
		panic(err.Error())
	}

	executor := initExecutor(m)
	return executor
}

func linuxMockExecutor() *executor.Executor {
	const linuxMockFile = "./testdata/arch.toml"
	return mockExecutor(linuxMockFile)
}

func testQuery(t *testing.T, query string) []*llx.RawResult {
	return testQueryWithExecutor(t, linuxMockExecutor(), query, nil)
}

func testWindowsQuery(t *testing.T, query string) []*llx.RawResult {
	return testQueryWithExecutor(t, mockExecutor("./testdata/windows.toml"), query, nil)
}

func testQueryLocal(t *testing.T, query string) []*llx.RawResult {
	return testQueryWithExecutor(t, localExecutor(), query, nil)
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
	executor := linuxMockExecutor()

	results := make([]map[string]*llx.RawResult, StableTestRepetitions)

	for i := 0; i < StableTestRepetitions; i++ {
		results[i] = map[string]*llx.RawResult{}
		watcherID := "test"

		executor.AddWatcher(watcherID, func(res *llx.RawResult) {
			results[i][res.CodeID] = res
		})

		bundle, err := executor.AddCode(query, nil)
		if err != nil {
			t.Fatal("failed to add code to executor: " + err.Error())
			return nil
		}
		if executor.WaitForResults(2*time.Second) == false {
			t.Fatal("ran into timeout on testing query " + query)
			return nil
		}

		executor.RemoveWatcher(watcherID)
		executor.RemoveCode(bundle.Code.Id, query)
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
			assert.Equal(t, cur.expectation, res[cur.resultIndex].Result().Error)
			assert.Nil(t, res[cur.resultIndex].Data.Value)
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
	executor := linuxMockExecutor()

	for i := range codes {
		code := codes[i]
		t.Run(code, func(t *testing.T) {
			res, err := executor.AddCode(code, nil)
			if err != nil {
				t.Error("failed to compile: " + err.Error())
				return
			}
			defer executor.RemoveCode(res.Code.Id, code)

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

	testTimeout(t,
		`user(name: 'i_definitely_dont_exist').authorizedkeys`,
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

//
// Core Language constructs
// ------------------------

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

	for i := range tests {
		cur := tests[i]
		t.Run(cur.code, func(t *testing.T) {
			res := testQueryWithExecutor(t, linuxMockExecutor(), cur.code, cur.props)
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
		{
			"if ( mondoo.version != null ) { 123 } else { 456 }",
			1, map[string]interface{}{
				"NmGComMxT/GJkwpf/IcA+qceUmwZCEzHKGt+8GEh+f8Y0579FxuDO+4FJf0/q2vWRE4dN2STPMZ+3xG3Mdm1fA==": llx.IntData(123),
			},
		},
		{
			"if ( mondoo.version == null ) { 123 } else { 456 }",
			1, map[string]interface{}{
				"3ZDJLpfu1OBftQi3eANcQSCltQum8mPyR9+fI7XAY9ZUMRpyERirCqag9CFMforO/u0zJolHNyg+2gE9hSTyGQ==": llx.IntData(456),
			},
		},
		{
			"if (false) { 123 } else if (true) { 456 } else { 789 }",
			0, map[string]interface{}{
				"3ZDJLpfu1OBftQi3eANcQSCltQum8mPyR9+fI7XAY9ZUMRpyERirCqag9CFMforO/u0zJolHNyg+2gE9hSTyGQ==": llx.IntData(456),
			},
		},
		{
			"if (false) { 123 } else if (false) { 456 } else { 789 }",
			0, map[string]interface{}{
				"Oy5SF8NbUtxaBwvZPpsnd0K21CY+fvC44FSd2QpgvIL689658Na52udy7qF2+hHjczk35TAstDtFZq7JIHNCmg==": llx.IntData(789),
			},
		},
		{
			"if (true) { return 123 } return 456",
			0, int64(123),
		},
		{
			"if (true) { return [1] } return [2,3]",
			0, []interface{}{int64(1)},
		},
		{
			"if (false) { return 123 } return 456",
			0, int64(456),
		},
		{
			"if (false) { return 123 } if (true) { return 456 } return 789",
			0, int64(456),
		},
		{
			"if (false) { return 123 } if (false) { return 456 } return 789",
			0, int64(789),
		},
		{
			"if(platform.family.contains('arch'))",
			0, nil,
		},
	})
}

func TestCore_Switch(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"switch { case 3 > 2: 123; default: 321 }",
			0, int64(123),
		},
		{
			"switch { case 1 > 2: 123; default: 321 }",
			0, int64(321),
		},
		{
			"switch { case 3 > 2: return 123; default: return 321 }",
			0, int64(123),
		},
		{
			"switch { case 1 > 2: return 123; default: return 321 }",
			0, int64(321),
		},
		{
			"switch ( 3 ) { case _ > 2: return 123; default: return 321 }",
			0, int64(123),
		},
		{
			"switch ( 1 ) { case _ > 2: true; default: false }",
			0, false,
		},
	})
}

func TestCore_Vars(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"p = file('/etc/ssh/sshd_config'); sshd.config(file: p)",
			1, true,
		},
		{
			"a = [1,2,3]; return a",
			0, []interface{}{int64(1), int64(2), int64(3)},
		},
	})
}

//
// Base types and operations
// -------------------------

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

// tests operations + vars
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

	simpleTests := []simpleTest{}

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

			simpleTests = append(simpleTests, []simpleTest{
				{a + " == " + b, 0, res},
				{a + " != " + b, 0, !res},
				{"a = " + a + "  a == " + b, 0, res},
				{"a = " + a + "  a != " + b, 0, !res},
				{"b = " + b + "; " + a + " == b", 1, res},
				{"b = " + b + "; " + a + " != b", 1, !res},
				{"a = " + a + "; b = " + b + "; a == b", 1, res},
				{"a = " + a + "; b = " + b + "; a != b", 1, !res},
			}...)
		}
	}

	runSimpleTests(t, simpleTests)
}

func TestNumber_Methods(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"1 + 2", 0, int64(3),
		},
		{
			"1 - 2", 0, int64(-1),
		},
		{
			"1 * 2", 0, int64(2),
		},
		{
			"4 / 2", 0, int64(2),
		},
		{
			"1.0 + 2.0", 0, float64(3),
		},
		{
			"1 - 2.0", 0, float64(-1),
		},
		{
			"1.0 * 2", 0, float64(2),
		},
		{
			"4.0 / 2.0", 0, float64(2),
		},
		{
			"1 < Infinity", 0, true,
		},
		{
			"1 == NaN", 0, false,
		},
	})
}

func TestRegex_Methods(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"'hello bob'.find(/he\\w*\\s?[bo]+/)",
			0, []interface{}{"hello bob"},
		},
		{
			"'HellO'.find(/hello/i)",
			0, []interface{}{"HellO"},
		},
		{
			"'hello\nworld'.find(/hello.world/s)",
			0, []interface{}{"hello\nworld"},
		},
		{
			"'yo! hello\nto the world'.find(/\\w+$/m)",
			0, []interface{}{"hello", "world"},
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
			"'oh-hello-world!'.camelcase",
			0, "ohHelloWorld!",
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
		{
			"' \n\t yo \t \n   '.trim",
			0, "yo",
		},
		{
			"'  \tyo  \n   '.trim(' \n')",
			0, "\tyo",
		},
		{
			"'hello ' + 'world'",
			0, "hello world",
		},
	})
}

func TestScore_Methods(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"score(100)",
			0, []byte{0x00, byte(100)},
		},
		{
			"score(\"CVSS:3.1/AV:P/AC:H/PR:L/UI:N/S:U/C:H/I:L/A:H\")",
			0, []byte{0x01, 0x03, 0x01, 0x04, 0x00, 0x00, 0x00, 0x01, 0x00},
		},
	})
}

func TestTypeof_Methods(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"typeof(null)",
			0, "null",
		},
		{
			"typeof(123)",
			0, "int",
		},
		{
			"typeof([1,2,3])",
			0, "[]int",
		},
		{
			"a = 123; typeof(a)",
			0, "int",
		},
	})
}

func duration(i int64) *time.Time {
	res := llx.DurationToTime(i)
	return &res
}

func TestTime_Methods(t *testing.T) {
	now := time.Now()
	today, _ := time.ParseInLocation("2006-01-02", now.Format("2006-01-02"), now.Location())
	tomorrow := today.Add(24 * time.Hour)

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
			"time.today",
			0, &today,
		},
		{
			"time.tomorrow",
			0, &tomorrow,
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
		{
			"time.now != Never",
			2, true,
		},
		{
			"time.now - Never",
			0, &llx.NeverPastTime,
		},
		{
			"Never - time.now",
			0, &llx.NeverFutureTime,
		},
		{
			"Never - Never",
			0, &llx.NeverPastTime,
		},
		{
			"Never * 3",
			0, &llx.NeverFutureTime,
		},
		{
			"a = Never - time.now; a.days",
			0, int64(math.MaxInt64),
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
			0, "file not found: '1' does not exist",
		},
	})

	runSimpleTests(t, []simpleTest{
		{
			"[1,2,3][-1]",
			0, int64(3),
		},
		{
			"[1,2,3][-3]",
			0, int64(1),
		},
		{
			"[1,2,3].first",
			0, int64(1),
		},
		{
			"[1,2,3].last",
			0, int64(3),
		},
	})
}

func TestArray(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"[1,2,3]",
			0, []interface{}{int64(1), int64(2), int64(3)},
		},
		{
			"return [1,2,3]",
			0, []interface{}{int64(1), int64(2), int64(3)},
		},
		{
			"[1,2,3] { _ == 2 }",
			0, []interface{}{
				map[string]interface{}{"OPhfwvbw0iVuMErS9tKL5qNj1lqTg3PEE1LITWEwW7a70nH8z8eZLi4x/aZqZQlyrQK13GAlUMY1w8g131EPog==": llx.BoolFalse},
				map[string]interface{}{"OPhfwvbw0iVuMErS9tKL5qNj1lqTg3PEE1LITWEwW7a70nH8z8eZLi4x/aZqZQlyrQK13GAlUMY1w8g131EPog==": llx.BoolTrue},
				map[string]interface{}{"OPhfwvbw0iVuMErS9tKL5qNj1lqTg3PEE1LITWEwW7a70nH8z8eZLi4x/aZqZQlyrQK13GAlUMY1w8g131EPog==": llx.BoolFalse},
			},
		},
		{
			"[1,2,3].where()",
			0, []interface{}{int64(1), int64(2), int64(3)},
		},
		{
			"[true, true, false].where(true)",
			0, []interface{}{true, true},
		},
		{
			"[false, true, false].where(false)",
			0, []interface{}{false, false},
		},
		{
			"[1,2,3].where(2)",
			0, []interface{}{int64(2)},
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
			"[1,2,3].all(_ < 9)",
			2, true,
		},
		{
			"[1,2,3].any(_ > 1)",
			1, true,
		},
		{
			"[1,2,3].one(_ == 2)",
			1, true,
		},
		{
			"[1,2,3].none(_ == 4)",
			1, true,
		},
		{
			"[0].where(_ > 0).where(_ > 0)",
			0, []interface{}{},
		},
		{
			"[1,2,2,2,3].unique()",
			0, []interface{}{int64(1), int64(2), int64(3)},
		},
		{
			"[1,1,2,2,2,3].duplicates()",
			0, []interface{}{int64(1), int64(2)},
		},
		{
			"[2,1,2,2].containsOnly([2])",
			0, []interface{}{int64(1)},
		},
		{
			"[2,1,2,1].containsOnly([1,2])",
			0, []interface{}(nil),
		},
		{
			"a = [1]; [2,1,2,1].containsOnly(a)",
			0, []interface{}{int64(2), int64(2)},
		},
		{
			"[2,1,2,2].containsNone([1])",
			0, []interface{}{int64(1)},
		},
		{
			"[2,1,2,1].containsNone([3,4])",
			0, []interface{}(nil),
		},
		{
			"a = [1]; [2,1,2,1].containsNone(a)",
			0, []interface{}{int64(1), int64(1)},
		},
	})
}

func TestMap(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"{a: 123}",
			0, map[string]interface{}{"a": int64(123)},
		},
		{
			"return {a: 123}",
			0, map[string]interface{}{"a": int64(123)},
		},
		{
			"sshd.config.params { _['Protocol'] != 1 }",
			0, map[string]interface{}{
				"TZsaWUkFbzR9WTfufqRaHuWJa/W4MQsYsrTli6w8DGQnSLYumOg7kduA17NEX/4y5xBfYQMvPIVBRThyB3LsJg==": llx.BoolTrue,
			},
		},
		{
			"sshd.config.params.length",
			0, int64(46),
		},
		{
			"sshd.config.params.keys.length",
			0, int64(46),
		},
		{
			"sshd.config.params.values.length",
			0, int64(46),
		},
	})
}

func TestResource_Filters(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			`users.where(name == 'root').list {
				uid == 0
				gid == 0
			}`,
			0, []interface{}{
				map[string]interface{}{
					"BamDDGp87sNG0hVjpmEAPEjF6fZmdA6j3nDinlgr/y5xK3KaLgulyscoeEEaEASm2RkRXifnWj3ZbF0OZBF6XA==": llx.BoolTrue,
					"ytOUfV4UyOjY0C6HKzQ8GcA/hshrh2ahRySNG41RbFt3TNNf+6gBuHvs2hGTNDPUZR/oN8WH0QFIYYm/Vj3pGQ==": llx.BoolTrue,
				},
			},
		},
		{
			"users.where(name == 'root').length",
			0, int64(1),
		},
		{
			"users.list.where(name == 'root').length",
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

func TestResource_None(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"users.none(uid == 99999)",
			1, true,
		},
		{
			"users.where(uid < 100).none(uid == 1000)",
			1, true,
		},
	})
}

func TestResource_duplicateFields(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"users.list.duplicates(uid) { uid }",
			2, []interface{}{
				map[string]interface{}{"sYZO9ps0Y4tx2p0TkrAn73WTQx83QIQu70uPtNukYNnVAzaer3Pf6xe7vAplB+cAgPbteXzizlUioUMnNJr5sg==": &llx.RawData{
					Type:  "\x05",
					Value: int64(1000),
					Error: nil,
				}},
				map[string]interface{}{"sYZO9ps0Y4tx2p0TkrAn73WTQx83QIQu70uPtNukYNnVAzaer3Pf6xe7vAplB+cAgPbteXzizlUioUMnNJr5sg==": &llx.RawData{
					Type:  "\x05",
					Value: int64(1000),
					Error: nil,
				}},
			},
		},
	})
}

func TestDict_Methods(t *testing.T) {
	p := "parse.json('/dummy.json')."

	expectedTime, err := time.Parse(time.RFC3339, "2016-01-28T23:02:24Z")
	if err != nil {
		panic(err.Error())
	}

	runSimpleTests(t, []simpleTest{
		{
			p + "params['string-array'].where(_ == 'a')",
			0, []interface{}{"a"},
		},
		{
			p + "params['string-array'].one(_ == 'a')",
			1, true,
		},
		{
			p + "params['string-array'].all(_ != 'z')",
			2, true,
		},
		{
			p + "params['string-array'].any(_ != 'a')",
			1, true,
		},
		{
			p + "params['does_not_exist'].any(_ != 'a')",
			1, false,
		},
		{
			p + "params { _['1'] == _['1.0'] }",
			1, true,
		},
		{
			p + "params { _['1'] - 2 }",
			1, true,
		},
		{
			p + "params['int-array'] { _ }",
			1, true,
		},
		{
			p + "params['hello'] + ' world'",
			0, "hello world",
		},
		{
			p + "params['hello'].trim('ho')",
			0, "ell",
		},
		{
			p + "params['hello'] { _.contains('llo') }",
			1, true,
		},
		{
			p + "params['dict'].length",
			0, int64(3),
		},
		{
			p + "params['dict'].keys.length",
			0, int64(3),
		},
		{
			p + "params['dict'].values.length",
			0, int64(3),
		},
		{
			"parse.date(" + p + "params['date'])",
			0, &expectedTime,
		},
	})

	runSimpleErrorTests(t, []simpleTest{
		{
			p + "params['does not exist'].values",
			0, "Failed to get values of `null`",
		},
		{
			p + "params['yo'] > 3",
			2, "left side of operation is null",
		},
	})
}
