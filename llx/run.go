package llx

import (
	"go.mondoo.io/mondoo/lumi"
)

// Run a piece of compiled code against a runtime. Just a friendly helper method
func RunV2(code *CodeV2, runtime *lumi.Runtime, props map[string]*Primitive, callback ResultCallback) (*LeiseExecutorV2, error) {
	x, err := NewExecutorV2(code, runtime, props, callback)
	if err != nil {
		return nil, err
	}
	x.Run()
	return x, nil
}

func NoRunV2(code *CodeV2, runerr error, runtime *lumi.Runtime, props map[string]*Primitive, callback ResultCallback) (*LeiseExecutorV2, error) {
	x, err := NewExecutorV2(code, runtime, props, callback)
	if err != nil {
		return nil, err
	}
	x.NoRun(runerr)
	return x, nil
}
