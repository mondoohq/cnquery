package resources

import (
	"testing"
	"time"

	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/lumi"
	motor "go.mondoo.io/mondoo/motor/motoros"
	"go.mondoo.io/mondoo/motor/motoros/local"
	"go.mondoo.io/mondoo/policy/executor"
)

func initExecutor() *executor.Executor {
	registry := lumi.NewRegistry()
	Init(registry)

	transport, err := local.New()
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

	executor.AddCode(query)
	if executor.WaitForResults(2*time.Second) == false {
		t.Error("ran into timeout on testing query " + query)
	}

	return results
}
