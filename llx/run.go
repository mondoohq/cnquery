package llx

import (
	"go.mondoo.io/mondoo/resources"
)

// Run a piece of compiled code against a runtime. Just a friendly helper method
func RunV2(code *CodeV2, runtime *resources.Runtime, props map[string]*Primitive, callback ResultCallback) (*MQLExecutorV2, error) {
	x, err := NewExecutorV2(code, runtime, props, callback)
	if err != nil {
		return nil, err
	}
	x.Run()
	return x, nil
}

func NoRunV2(code *CodeV2, runerr error, runtime *resources.Runtime, props map[string]*Primitive, callback ResultCallback) (*MQLExecutorV2, error) {
	x, err := NewExecutorV2(code, runtime, props, callback)
	if err != nil {
		return nil, err
	}
	x.NoRun(runerr)
	return x, nil
}
