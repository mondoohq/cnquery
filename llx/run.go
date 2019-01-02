package llx

import (
	"sync"

	"go.mondoo.io/mondoo/lumi"
)

// Run a piece of compiled code against a runtime. Just a friendly helper method
func Run(code *Code, runtime *lumi.Runtime, callback ResultCallback) (*LeiseExecutor, error) {
	x, err := NewExecutor(code, runtime, callback)
	if err != nil {
		return nil, err
	}
	x.Run()
	return x, nil
}

// RunOnce the code that was provided and call the callback
func RunOnce(code *Code, runtime *lumi.Runtime, callback ResultCallback) error {
	cnt := 0
	var executor *LeiseExecutor
	var err error

	// Note: We cannot copy the code from the Run method above as it may
	// lead to a race condition where the callback is run BEFORE the
	// executor is created. The way we do it here guarantees everything
	// including the closure-based executor is in place before the callback
	// runs.
	executor, err = NewExecutor(code, runtime, func(one *RawResult) {
		cnt++
		if cnt >= len(code.Entrypoints) {
			executor.Unregister()
		}
		callback(one)
	})
	if err != nil {
		return err
	}

	executor.Run()
	return nil
}

// RunOnceSync will run the code only once and report on the results it gets
func RunOnceSync(code *Code, runtime *lumi.Runtime) ([]*RawResult, error) {
	res := []*RawResult{}
	var done sync.WaitGroup
	// FIXME: shouldnt this be:
	done.Add(len(code.Entrypoints))

	err := RunOnce(code, runtime, func(one *RawResult) {
		res = append(res, one)
		done.Done()
	})
	if err != nil {
		return nil, err
	}

	done.Wait()

	return res, nil
}
