package resources

import (
	"testing"
	"time"

	"github.com/tj/assert"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/lumi"
	motor "go.mondoo.io/mondoo/motor/motoros"
	mock "go.mondoo.io/mondoo/motor/motoros/mock/toml"
	"go.mondoo.io/mondoo/motor/motoros/types"
	"go.mondoo.io/mondoo/policy/executor"
)

func initExecutor() *executor.Executor {
	registry := lumi.NewRegistry()
	Init(registry)

	transport, err := mock.New(&types.Endpoint{Backend: "mock", Path: "./testdata/arch.toml"})
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
	defer executor.Remove(bundle.Code.Id)

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
		executor.Remove(bundle.Code.Id)
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

func TestStableCore(t *testing.T) {
	res := stableResults(t, "mondoo.version")
	for _, v := range res {
		assert.Equal(t, "unstable", v.Data.Value)
	}
}

func TestString_Methods(t *testing.T) {
	tests := []struct {
		code         string
		expectations interface{}
	}{
		{
			"'hello'.contains('ll')",
			true,
		},
		{
			"'hello'.contains('lloo')",
			false,
		},
		{
			"'hello'.contains(['lo', 'la'])",
			true,
		},
		{
			"'hello'.contains(['lu', 'la'])",
			false,
		},
	}

	for i := range tests {
		cur := tests[i]
		t.Run(cur.code, func(t *testing.T) {
			res := testQuery(t, cur.code)
			assert.NotEmpty(t, res)
			assert.Equal(t, cur.expectations, res[0].Data.Value)
		})
	}
}

func TestArray_Block(t *testing.T) {
	tests := []struct {
		code         string
		expectations interface{}
	}{
		{
			"[1,2,3].where()",
			[]interface{}{int64(1), int64(2), int64(3)},
		},
		{
			"[1,2,3].where(_ > 2)",
			[]interface{}{int64(3)},
		},
		{
			"[1,2,3].where(_ >= 2)",
			[]interface{}{int64(2), int64(3)},
		},
		{
			"[1,2,3].contains(_ >= 2)",
			true,
		},
		{
			"[1,2,3].one(_ == 2)",
			true,
		},
		{
			"[1,2,3].all(_ < 9)",
			true,
		},
	}

	for i := range tests {
		cur := tests[i]
		t.Run(cur.code, func(t *testing.T) {
			res := testQuery(t, cur.code)
			assert.NotEmpty(t, res)
			assert.Equal(t, cur.expectations, res[0].Data.Value)
		})
	}
}

func TestWhere(t *testing.T) {
	t.Run("users.where", func(t *testing.T) {
		res := testQuery(t, "users.where(username == 'root').length")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, int64(1), res[0].Data.Value)
	})

	t.Run("users.where.list + block on empty", func(t *testing.T) {
		res := testQuery(t, "users.where(username == 'rooot').list { uid }")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, []interface{}{}, res[0].Data.Value)
	})
}

func TestContains(t *testing.T) {
	t.Run("users.contains", func(t *testing.T) {
		res := testQuery(t, "users.contains(username == 'root')")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, true, res[0].Data.Value)
	})

	t.Run("users.where.contains", func(t *testing.T) {
		res := testQuery(t, "users.where(uid < 100).contains(username == 'root')")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, true, res[0].Data.Value)
	})
}

func TestAll(t *testing.T) {
	t.Run("users.all", func(t *testing.T) {
		res := testQuery(t, "users.all(uid >= 0)")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, true, res[0].Data.Value)
	})

	t.Run("users.where.all", func(t *testing.T) {
		res := testQuery(t, "users.where(uid < 100).all(uid >= 0)")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, true, res[0].Data.Value)
	})
}

func TestOne(t *testing.T) {
	t.Run("users.one", func(t *testing.T) {
		res := testQuery(t, "users.one(uid == 0)")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, true, res[0].Data.Value)
	})

	t.Run("users.where.one", func(t *testing.T) {
		res := testQuery(t, "users.where(uid < 100).one(uid == 0)")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, true, res[0].Data.Value)
	})
}
