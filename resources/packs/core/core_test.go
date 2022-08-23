package core_test

import (
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/resources/packs/core"
	"go.mondoo.io/mondoo/resources/packs/testutils"
)

var x = testutils.InitTester(testutils.LinuxMock(), core.Registry)

func testErrornous(t *testing.T, codes ...string) {
	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)

	for i := range codes {
		code := codes[i]
		t.Run(code, func(t *testing.T) {
			x.TestQuery(t, code)
		})
	}
}

func TestErroneousLlxChains(t *testing.T) {
	testErrornous(t, `file("/etc/crontab") {
		permissions.group_readable == false
		permissions.group_writeable == false
		permissions.group_executable == false
	}`)

	testErrornous(t,
		`file("/etc/profile").content.contains("umask 027") || file("/etc/bashrc").content.contains("umask 027")`,
		`file("/etc/profile").content.contains("umask 027") || file("/etc/bashrc").content.contains("umask 027")`,
	)

	testErrornous(t,
		`users.map(name) { _.contains("a") _.contains("b") }`,
	)

	testErrornous(t,
		`user(name: 'i_definitely_dont_exist').authorizedkeys`,
	)
}

func TestResource_InitWithResource(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)

	x.TestSimple(t, []testutils.SimpleTest{
		{
			"file(platform.name).exists",
			0, false,
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

	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)

	for i := range tests {
		cur := tests[i]
		t.Run(cur.code, func(t *testing.T) {
			res := x.TestQueryP(t, cur.code, cur.props)
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
	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	x.TestSimple(t, []testutils.SimpleTest{
		{
			"if ( mondoo.version == null ) { 123 }",
			1, nil,
		},
		{
			"if (true) { return 123 } return 456",
			0, int64(123),
		},
		{
			"if (true) { return [1] } return [2,3]",
			0,
			[]interface{}{int64(1)},
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
			// This test comes out from an issue we had where return was not
			// generating a single entrypoint, causing the first reported
			// value to be used as the return value.
			`
				if (true) {
					// file has content so should return true
					a = file('/etc/ssh/sshd_config').content != ''
					b = false
					return a || b
				}
			`, 0, true,
		},
		{
			"if ( mondoo.version != null ) { 123 }",
			1,
			map[string]interface{}{
				"__t": llx.BoolData(true),
				"__s": llx.NilData,
				"NmGComMxT/GJkwpf/IcA+qceUmwZCEzHKGt+8GEh+f8Y0579FxuDO+4FJf0/q2vWRE4dN2STPMZ+3xG3Mdm1fA==": llx.IntData(123),
			},
		},
		{
			"if ( mondoo.version != null ) { 123 } else { 456 }",
			1,
			map[string]interface{}{
				"__t": llx.BoolData(true),
				"__s": llx.NilData,
				"NmGComMxT/GJkwpf/IcA+qceUmwZCEzHKGt+8GEh+f8Y0579FxuDO+4FJf0/q2vWRE4dN2STPMZ+3xG3Mdm1fA==": llx.IntData(123),
			},
		},
		{
			"if ( mondoo.version == null ) { 123 } else { 456 }",
			1,
			map[string]interface{}{
				"__t": llx.BoolData(true),
				"__s": llx.NilData,
				"3ZDJLpfu1OBftQi3eANcQSCltQum8mPyR9+fI7XAY9ZUMRpyERirCqag9CFMforO/u0zJolHNyg+2gE9hSTyGQ==": llx.IntData(456),
			},
		},
		{
			"if (false) { 123 } else if (true) { 456 } else { 789 }",
			0,
			map[string]interface{}{
				"__t": llx.BoolData(true),
				"__s": llx.NilData,
				"3ZDJLpfu1OBftQi3eANcQSCltQum8mPyR9+fI7XAY9ZUMRpyERirCqag9CFMforO/u0zJolHNyg+2gE9hSTyGQ==": llx.IntData(456),
			},
		},
		{
			"if (false) { 123 } else if (false) { 456 } else { 789 }",
			0,
			map[string]interface{}{
				"__t": llx.BoolData(true),
				"__s": llx.NilData,
				"Oy5SF8NbUtxaBwvZPpsnd0K21CY+fvC44FSd2QpgvIL689658Na52udy7qF2+hHjczk35TAstDtFZq7JIHNCmg==": llx.IntData(789),
			},
		},
	})

	x.TestSimpleErrors(t, []testutils.SimpleTest{
		// if-conditions need to be called with a bloc
		{
			"if(platform.family.contains('arch'))",
			1, "Called if with 1 arguments, expected at least 3",
		},
	})
}

func TestCore_Switch(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	x.TestSimple(t, []testutils.SimpleTest{
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
	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	x.TestSimple(t, []testutils.SimpleTest{
		{
			"p = file('/some.yaml'); parse.yaml(file: p)",
			1, true,
		},
		{
			"a = [1,2,3]; return a",
			0,
			[]interface{}{int64(1), int64(2), int64(3)},
		},
		{
			"a = 1; b = [a]; return b",
			0,
			[]interface{}{int64(1)},
		},
		{
			"a = 1; b = a + 2; return b",
			0, int64(3),
		},
		{
			"a = 1; b = [a + 2]; return b",
			0,
			[]interface{}{int64(3)},
		},
	})
}

//
// Base types and operations
// -------------------------

func TestBooleans(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	x.TestSimple(t, []testutils.SimpleTest{
		{
			"true || false || false",
			1, true,
		},
		{
			"false || true || false",
			1, true,
		},
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

	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	x.TestSimple(t, simpleTests)
}

func TestNumber_Methods(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	x.TestSimple(t, []testutils.SimpleTest{
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

var emojiTestString = []rune("☀⛺➿🌀🎂👍🔒😀🙈🚵🛼🤌🤣🥳🧡🧿🩰🫖")

func TestRegex_Methods(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	x.TestSimple(t, []testutils.SimpleTest{
		{
			"'hello bob'.find(/he\\w*\\s?[bo]+/)",
			0,
			[]interface{}{"hello bob"},
		},
		{
			"'HellO'.find(/hello/i)",
			0,
			[]interface{}{"HellO"},
		},
		{
			"'hello\nworld'.find(/hello.world/s)",
			0,
			[]interface{}{"hello\nworld"},
		},
		{
			"'yo! hello\nto the world'.find(/\\w+$/m)",
			0,
			[]interface{}{"hello", "world"},
		},
		{
			"'IPv4: 0.0.0.0, 255.255.255.255, 1.50.120.230, 256.0.0.0 '.find(regex.ipv4)",
			0,
			[]interface{}{"0.0.0.0", "255.255.255.255", "1.50.120.230"},
		},
		{
			"'IPv6: 2001:0db8:85a3:0000:0000:8a2e:0370:7334'.find(regex.ipv6)",
			0,
			[]interface{}{"2001:0db8:85a3:0000:0000:8a2e:0370:7334"},
		},
		{
			"'Sarah Summers <sarah@summe.rs>'.find( regex.email )",
			0,
			[]interface{}{"sarah@summe.rs"},
		},
		{
			"'one+1@sum.me.rs:'.find( regex.email )",
			0,
			[]interface{}{"one+1@sum.me.rs"},
		},
		{
			"'Urls: http://mondoo.com/welcome'.find( regex.url )",
			0,
			[]interface{}{"http://mondoo.com/welcome"},
		},
		{
			"'mac 01:23:45:67:89:ab attack'.find(regex.mac)",
			0,
			[]interface{}{"01:23:45:67:89:ab"},
		},
		{
			"'uuid: b7f99555-5bca-48f4-b86f-a953a4883383.'.find(regex.uuid)",
			0,
			[]interface{}{"b7f99555-5bca-48f4-b86f-a953a4883383"},
		},
		{
			"'some ⮆" + string(emojiTestString) + " ⮄ emojis'.find(regex.emoji).length",
			0, int64(len(emojiTestString)),
		},
		{
			"'semvers: 1, 1.2, 1.2.3, 1.2.3-4'.find(regex.semver)",
			0,
			[]interface{}{"1.2.3", "1.2.3-4"},
		},
	})
}

func TestString_Methods(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	x.TestSimple(t, []testutils.SimpleTest{
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
			"'hello'.contains(23)",
			0, false,
		},
		{
			"'hello123'.contains(23)",
			0, true,
		},
		{
			"'hello123'.contains([5,6,7])",
			0, false,
		},
		{
			"'hello123'.contains([5,1,7])",
			0, true,
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
			0,
			[]interface{}{"hello", "world"},
		},
		{
			"'he\nll\no'.lines",
			0,
			[]interface{}{"he", "ll", "o"},
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
	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	x.TestSimple(t, []testutils.SimpleTest{
		{
			"score(100)",
			0,
			[]byte{0x00, byte(100)},
		},
		{
			"score(\"CVSS:3.1/AV:P/AC:H/PR:L/UI:N/S:U/C:H/I:L/A:H\")",
			0,
			[]byte{0x01, 0x03, 0x01, 0x04, 0x00, 0x00, 0x00, 0x01, 0x00},
		},
	})
}

func TestTypeof_Methods(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	x.TestSimple(t, []testutils.SimpleTest{
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

func TestFuzzyTime(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	code := "time.now.unix"
	t.Run(code, func(t *testing.T) {
		res := x.TestQuery(t, code)
		now := time.Now().Unix()
		assert.NotEmpty(t, res)

		assert.NotNil(t, res[0].Result().Error)
		val := res[0].Data.Value
		valInt, ok := val.(int64)
		assert.Equal(t, true, ok)
		min := now - 1
		max := now + 1
		between := min <= valInt && valInt <= max
		assert.Equal(t, true, between)
	})
}

func TestTime_Methods(t *testing.T) {
	now := time.Now()
	today, _ := time.ParseInLocation("2006-01-02", now.Format("2006-01-02"), now.Location())
	tomorrow := today.Add(24 * time.Hour)

	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	x.TestSimple(t, []testutils.SimpleTest{
		{
			"time.now",
			1, true,
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
			0, int64(1*60*60 + 0o2*60 + 0o3),
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
			3, true,
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
	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	x.TestSimpleErrors(t, []testutils.SimpleTest{
		{
			"[0,1,2][100000]",
			0, "array index out of bound (trying to access element 100000, max: 2)",
		},
	})

	x.TestSimple(t, []testutils.SimpleTest{
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
	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	x.TestSimple(t, []testutils.SimpleTest{
		{
			"[1,2,3]",
			0,
			[]interface{}{int64(1), int64(2), int64(3)},
		},
		{
			"return [1,2,3]",
			0,
			[]interface{}{int64(1), int64(2), int64(3)},
		},
		{
			"[1,2,3] { _ == 2 }",
			0,
			[]interface{}{
				map[string]interface{}{"__t": llx.BoolFalse, "__s": llx.BoolFalse, "OPhfwvbw0iVuMErS9tKL5qNj1lqTg3PEE1LITWEwW7a70nH8z8eZLi4x/aZqZQlyrQK13GAlUMY1w8g131EPog==": llx.BoolFalse},
				map[string]interface{}{"__t": llx.BoolTrue, "__s": llx.BoolTrue, "OPhfwvbw0iVuMErS9tKL5qNj1lqTg3PEE1LITWEwW7a70nH8z8eZLi4x/aZqZQlyrQK13GAlUMY1w8g131EPog==": llx.BoolTrue},
				map[string]interface{}{"__t": llx.BoolFalse, "__s": llx.BoolFalse, "OPhfwvbw0iVuMErS9tKL5qNj1lqTg3PEE1LITWEwW7a70nH8z8eZLi4x/aZqZQlyrQK13GAlUMY1w8g131EPog==": llx.BoolFalse},
			},
		},
		{
			"[1,2,3] { a = _ }",
			0,
			[]interface{}{
				map[string]interface{}{"__t": llx.BoolTrue, "__s": llx.NilData},
				map[string]interface{}{"__t": llx.BoolTrue, "__s": llx.NilData},
				map[string]interface{}{"__t": llx.BoolTrue, "__s": llx.NilData},
			},
		},
		{
			"[1,2,3].where()",
			0,
			[]interface{}{int64(1), int64(2), int64(3)},
		},
		{
			"[true, true, false].where(true)",
			0,
			[]interface{}{true, true},
		},
		{
			"[false, true, false].where(false)",
			0,
			[]interface{}{false, false},
		},
		{
			"[1,2,3].where(2)",
			0,
			[]interface{}{int64(2)},
		},
		{
			"[1,2,3].where(_ > 2)",
			0,
			[]interface{}{int64(3)},
		},
		{
			"[1,2,3].where(_ >= 2)",
			0,
			[]interface{}{int64(2), int64(3)},
		},
		{
			"['yo','ho','ho'].where( /y.$/ )",
			0,
			[]interface{}{"yo"},
		},
		{
			"[1,2,3].contains(_ >= 2)",
			2, true,
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
			"[1,2,3].one(_ == 2)",
			2, true,
		},
		{
			"[1,2,3].none(_ == 4)",
			2, true,
		},
		{
			"[[0,1],[1,2]].map(_[1])",
			0,
			[]interface{}{int64(1), int64(2)},
		},
		{
			"[0].where(_ > 0).where(_ > 0)",
			0,
			[]interface{}{},
		},
		{
			"[1,2,2,2,3].unique()",
			0,
			[]interface{}{int64(1), int64(2), int64(3)},
		},
		{
			"[1,1,2,2,2,3].duplicates()",
			0,
			[]interface{}{int64(1), int64(2)},
		},
		{
			"[2,1,2,2].containsOnly([2])",
			0,
			[]interface{}{int64(1)},
		},
		{
			"[2,1,2,1].containsOnly([1,2])",
			0, []interface{}(nil),
		},
		{
			"a = [1]; [2,1,2,1].containsOnly(a)",
			0,
			[]interface{}{int64(2), int64(2)},
		},
		{
			"[2,1,2,2].containsNone([1])",
			0,
			[]interface{}{int64(1)},
		},
		{
			"[2,1,2,1].containsNone([3,4])",
			0, []interface{}(nil),
		},
		{
			"a = [1]; [2,1,2,1].containsNone(a)",
			0,
			[]interface{}{int64(1), int64(1)},
		},
		{
			"['a','b'] != /c/",
			0, true,
		},
	})
}

func TestMap(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	x.TestSimple(t, []testutils.SimpleTest{
		{
			"{a: 123}",
			0,
			map[string]interface{}{"a": int64(123)},
		},
		{
			"return {a: 123}",
			0,
			map[string]interface{}{"a": int64(123)},
		},
		{
			"{a: 1, b: 2, c: 3}.where(key == 'c')",
			0,
			map[string]interface{}{"c": int64(3)},
		},
		{
			"{a: 1, b: 2, c: 3}.where(value < 3)",
			0,
			map[string]interface{}{"a": int64(1), "b": int64(2)},
		},
		{
			"parse.json('/dummy.json').params.length",
			0, int64(10),
		},
		{
			"parse.json('/dummy.json').params.keys.length",
			0, int64(10),
		},
		{
			"parse.json('/dummy.json').params.values.length",
			0, int64(10),
		},
		{
			"parse.json('/dummy.json').params { _['Protocol'] != 1 }",
			0,
			map[string]interface{}{
				"__t": llx.BoolTrue,
				"__s": llx.BoolTrue,
				"CQ28lTwZsvVdJM4dCyeTdbQhExY8oiUIcMoPyPjXAJNgtjMLnHK6qgEVywRY1Hbw9QqInuL06EWIOaEMj2e9NA==": llx.BoolTrue,
			},
		},
	})
}

func TestResource_Filters(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	x.TestSimple(t, []testutils.SimpleTest{
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
			0,
			[]interface{}{},
		},
		{
			"users.where(uid > 0).where(uid < 0).list",
			0,
			[]interface{}{},
		},
	})
}

func TestResource_Filters_v1(t *testing.T) {
	testutils.OnlyV1(t)
	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	x.TestSimple(t, []testutils.SimpleTest{
		{
			`users.where(name == 'root').list {
				uid == 0
				gid == 0
			}`,
			0,
			[]interface{}{
				map[string]interface{}{
					"__t": llx.BoolTrue,
					"__s": llx.BoolTrue,
					"BamDDGp87sNG0hVjpmEAPEjF6fZmdA6j3nDinlgr/y5xK3KaLgulyscoeEEaEASm2RkRXifnWj3ZbF0OZBF6XA==": llx.BoolTrue,
					"ytOUfV4UyOjY0C6HKzQ8GcA/hshrh2ahRySNG41RbFt3TNNf+6gBuHvs2hGTNDPUZR/oN8WH0QFIYYm/Vj3pGQ==": llx.BoolTrue,
				},
			},
		},
	})
}

func TestResource_Filters_piper(t *testing.T) {
	testutils.OnlyPiper(t)
	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	x.TestSimple(t, []testutils.SimpleTest{
		{
			`users.where(name == 'root').list {
				uid == 0
				gid == 0
			}`,
			0,
			[]interface{}{
				map[string]interface{}{
					"__t": llx.BoolTrue,
					"__s": llx.BoolTrue,
					"BamDDGp87sNG0hVjpmEAPEjF6fZmdA6j3nDinlgr/y5xK3KaLgulyscoeEEaEASm2RkRXifnWj3ZbF0OZBF6XA==": llx.BoolTrue,
					"ytOUfV4UyOjY0C6HKzQ8GcA/hshrh2ahRySNG41RbFt3TNNf+6gBuHvs2hGTNDPUZR/oN8WH0QFIYYm/Vj3pGQ==": llx.BoolTrue,
				},
			},
		},
	})
}

func TestResource_Contains(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	x.TestSimple(t, []testutils.SimpleTest{
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
	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	x.TestSimple(t, []testutils.SimpleTest{
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
	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	x.TestSimple(t, []testutils.SimpleTest{
		{
			"users.any(uid < 100)",
			2, true,
		},
		{
			"users.where(uid < 100).any(uid < 50)",
			2, true,
		},
	})
}

func TestResource_One(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	x.TestSimple(t, []testutils.SimpleTest{
		{
			"users.one(uid == 0)",
			2, true,
		},
		{
			"users.where(uid < 100).one(uid == 0)",
			2, true,
		},
	})
}

func TestResource_None(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	x.TestSimple(t, []testutils.SimpleTest{
		{
			"users.none(uid == 99999)",
			2, true,
		},
		{
			"users.where(uid < 100).none(uid == 1000)",
			2, true,
		},
	})
}

func TestResource_Map(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	x.TestSimple(t, []testutils.SimpleTest{
		{
			"users.map(name)",
			0, []interface{}([]interface{}{"root", "chris", "christopher", "chris", "bin"}),
		},
	})
}

func TestResource_duplicateFields_v1(t *testing.T) {
	testutils.OnlyV1(t)

	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	x.TestSimple(t, []testutils.SimpleTest{
		{
			"users.list.duplicates(uid) { uid }",
			0,
			[]interface{}{
				map[string]interface{}{
					"__t": llx.BoolTrue,
					"__s": llx.NilData,
					"sYZO9ps0Y4tx2p0TkrAn73WTQx83QIQu70uPtNukYNnVAzaer3Pf6xe7vAplB+cAgPbteXzizlUioUMnNJr5sg==": &llx.RawData{
						Type:  "\x05",
						Value: int64(1000),
						Error: nil,
					},
				},
				map[string]interface{}{
					"__t": llx.BoolTrue,
					"__s": llx.NilData,
					"sYZO9ps0Y4tx2p0TkrAn73WTQx83QIQu70uPtNukYNnVAzaer3Pf6xe7vAplB+cAgPbteXzizlUioUMnNJr5sg==": &llx.RawData{
						Type:  "\x05",
						Value: int64(1000),
						Error: nil,
					},
				},
			},
		},
	})
}

func TestResource_duplicateFields_piper(t *testing.T) {
	testutils.OnlyPiper(t)

	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	x.TestSimple(t, []testutils.SimpleTest{
		{
			"users.list.duplicates(uid) { uid }",
			0,
			[]interface{}{
				map[string]interface{}{
					"__t": llx.BoolTrue,
					"__s": llx.NilData,
					"sYZO9ps0Y4tx2p0TkrAn73WTQx83QIQu70uPtNukYNnVAzaer3Pf6xe7vAplB+cAgPbteXzizlUioUMnNJr5sg==": &llx.RawData{
						Type:  "\x05",
						Value: int64(1000),
						Error: nil,
					},
				},
				map[string]interface{}{
					"__t": llx.BoolTrue,
					"__s": llx.NilData,
					"sYZO9ps0Y4tx2p0TkrAn73WTQx83QIQu70uPtNukYNnVAzaer3Pf6xe7vAplB+cAgPbteXzizlUioUMnNJr5sg==": &llx.RawData{
						Type:  "\x05",
						Value: int64(1000),
						Error: nil,
					},
				},
			},
		},
	})
}

func TestDict_Methods_Map(t *testing.T) {
	p := "parse.json('/dummy.json')."

	expectedTime, err := time.Parse(time.RFC3339, "2016-01-28T23:02:24Z")
	if err != nil {
		panic(err.Error())
	}

	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	x.TestSimple(t, []testutils.SimpleTest{
		{
			p + "params['string-array'].where(_ == 'a')",
			0,
			[]interface{}{"a"},
		},
		{
			p + "params['string-array'].one(_ == 'a')",
			2, true,
		},
		{
			p + "params['string-array'].all(_ != 'z')",
			2, true,
		},
		{
			p + "params['string-array'].any(_ != 'a')",
			2, true,
		},
		{
			p + "params['does_not_exist'].any(_ != 'a')",
			2, nil,
		},
		{
			p + "params['f'].map(_['ff'])",
			0,
			[]interface{}{float64(3)},
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

	x.TestSimpleErrors(t, []testutils.SimpleTest{
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

func TestDict_Methods_Array(t *testing.T) {
	p := "parse.json('/dummy.array.json')."

	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	x.TestSimple(t, []testutils.SimpleTest{
		{
			p + "params[0]",
			0, float64(1),
		},
		{
			p + "params[1]",
			0, "hi",
		},
		{
			p + "params[2]",
			0,
			map[string]interface{}{"ll": float64(0)},
		},
	})
}

func TestDict_Methods_OtherJson(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	x.TestSimple(t, []testutils.SimpleTest{
		{
			"parse.json('/dummy.number.json').params",
			0, float64(1.23),
		},
		{
			"parse.json('/dummy.string.json').params",
			0, "hi",
		},
		{
			"parse.json('/dummy.true.json').params",
			0, true,
		},
		{
			"parse.json('/dummy.false.json').params",
			0, false,
		},
		{
			"parse.json('/dummy.null.json').params",
			0, nil,
		},
	})
}

func TestArrayBlockError(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	res := x.TestQuery(t, "users.list { file(_.name + 'doesnotexist').content }")
	assert.NotEmpty(t, res)
	queryResult := res[len(res)-1]
	require.NotNil(t, queryResult)
	require.Error(t, queryResult.Data.Error)
}

func TestBrokenQueryExecution(t *testing.T) {
	x := testutils.InitTester(testutils.LinuxMock(), core.Registry)
	bundle, err := x.Compile("'asdf'.contains('asdf') == true")
	require.NoError(t, err)
	if testutils.Features.IsActive(mondoo.PiperCode) {
		bundle.CodeV2.Blocks[0].Chunks[1].Id = "fakecontains"
	} else {
		bundle.DeprecatedV5Code.Code[1].Id = "fakecontains"
	}
	results := x.TestMqlc(t, bundle, nil)
	require.Len(t, results, 3)
	require.Error(t, results[0].Data.Error)
	require.Error(t, results[1].Data.Error)
	require.Error(t, results[2].Data.Error)
}
