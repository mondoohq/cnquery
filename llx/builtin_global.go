package llx

import (
	"errors"
	"strconv"

	"go.mondoo.com/cnquery/types"
)

// handleGlobal takes a global function and returns a handler if found.
// this is not exported as it is only used internally. it exposes everything
// below this function
func handleGlobalV2(op string) (handleFunctionV2, bool) {
	f, ok := globalFunctionsV2[op]
	if !ok {
		return nil, false
	}
	return f, true
}

// DEFINITIONS

type handleFunctionV2 func(*blockExecutor, *Function, uint64) (*RawData, uint64, error)

var globalFunctionsV2 map[string]handleFunctionV2

func init() {
	globalFunctionsV2 = map[string]handleFunctionV2{
		"expect":         expectV2,
		"if":             ifCallV2,
		"switch":         switchCallV2,
		"score":          scoreCallV2,
		"typeof":         typeofCallV2,
		"{}":             blockV2,
		"return":         returnCallV2,
		"createResource": globalCreateResource,
	}
}

func globalCreateResource(e *blockExecutor, f *Function, ref uint64) (*RawData, uint64, error) {
	if l := len(f.Args); l%2 != 1 || l == 0 {
		return nil, 0, errors.New("Called `createResource` with invalid number of arguments")
	}

	binding, ok := f.Args[0].RefV2()
	if !ok {
		return nil, 0, errors.New("Called `createResource` with invalid arguments: expected ref")
	}

	t := types.Type(f.Type)
	return e.createResource(t.ResourceName(), binding, &Function{
		Type: f.Type,
		Args: f.Args[1:],
	}, ref)
}

func ifCallV2(e *blockExecutor, f *Function, ref uint64) (*RawData, uint64, error) {
	if len(f.Args) < 3 {
		return nil, 0, errors.New("Called if with " + strconv.Itoa(len(f.Args)) + " arguments, expected at least 3")
	}

	var idx int
	max := len(f.Args)
	for idx+2 < max {
		res, dref, err := e.resolveValue(f.Args[idx], ref)
		if err != nil || dref != 0 || res == nil {
			return res, dref, err
		}

		if truthy, _ := res.IsTruthy(); truthy {
			depArgs := f.Args[idx+2]
			res, dref, err = e.runBlock(nil, f.Args[idx+1], depArgs.Array, ref)
			return res, dref, err
		}

		idx += 3
	}

	if idx < max {
		depArgs := f.Args[idx+1]
		res, dref, err := e.runBlock(nil, f.Args[idx], depArgs.Array, ref)
		return res, dref, err
	}

	return NilData, 0, nil
}

func switchCallV2(e *blockExecutor, f *Function, ref uint64) (*RawData, uint64, error) {
	// very similar to the if-call above; minor differences:
	// - we have an optional reference value (which is in the function call)
	// - default is translated to `true` in its condition; everything else is a function

	if len(f.Args) < 2 {
		return nil, 0, errors.New("Called switch with no arguments, expected at least one case statement")
	}

	var bind *RawData
	if types.Type(f.Args[0].Type) != types.Unset {
		var dref uint64
		var err error
		bind, dref, err = e.resolveValue(f.Args[0], ref)
		if err != nil || dref != 0 || bind == nil {
			return bind, dref, err
		}
	}

	// ignore the first argument, it's just the reference value
	idx := 1
	max := len(f.Args)
	defaultCaseIdx := -1
	for idx+2 < max {
		if types.Type(f.Args[idx].Type) == types.Bool {
			defaultCaseIdx = idx
			idx += 3
			continue
		}

		res, dref, err := e.resolveValue(f.Args[idx], ref)
		if err != nil || dref != 0 || res == nil {
			return res, dref, err
		}

		if truthy, _ := res.IsTruthy(); truthy {
			depArgs := f.Args[idx+2]
			res, dref, err = e.runBlock(bind, f.Args[idx+1], depArgs.Array, ref)
			return res, dref, err
		}

		idx += 3
	}

	if defaultCaseIdx != -1 {
		res, dref, err := e.runBlock(nil, f.Args[defaultCaseIdx+1], f.Args[defaultCaseIdx+2].Array, ref)
		return res, dref, err
	}

	return NilData, 0, nil
}

func scoreCallV2(e *blockExecutor, f *Function, ref uint64) (*RawData, uint64, error) {
	if len(f.Args) != 1 {
		return nil, 0, errors.New("Called `score` with " + strconv.Itoa(len(f.Args)) + " arguments, expected one")
	}

	res, dref, err := e.resolveValue(f.Args[0], ref)
	if err != nil || dref != 0 || res == nil {
		return res, dref, err
	}

	var b []byte
	switch res.Type {
	case types.Int:
		b, err = scoreVector(int32(res.Value.(int64)))

	case types.Float:
		b, err = scoreVector(int32(res.Value.(float64)))

	case types.String:
		b, err = scoreString(res.Value.(string))
	}

	if err != nil {
		return nil, 0, err
	}

	return ScoreData(b), 0, nil
}

func typeofCallV2(e *blockExecutor, f *Function, ref uint64) (*RawData, uint64, error) {
	if len(f.Args) != 1 {
		return nil, 0, errors.New("Called `typeof` with " + strconv.Itoa(len(f.Args)) + " arguments, expected one")
	}

	res, dref, err := e.resolveValue(f.Args[0], ref)
	if err != nil || dref != 0 || res == nil {
		return res, dref, err
	}

	return StringData(res.Type.Label()), 0, nil
}

func expectV2(e *blockExecutor, f *Function, ref uint64) (*RawData, uint64, error) {
	if len(f.Args) != 1 {
		return nil, 0, errors.New("Called expect with " + strconv.Itoa(len(f.Args)) + " arguments, expected 1")
	}
	res, dref, err := e.resolveValue(f.Args[0], ref)
	if res != nil && res.Type != types.Bool {
		return nil, 0, errors.New("Called expect body with wrong type, it should be a boolean (type mismatch)")
	}
	return res, dref, err
}

func blockV2(e *blockExecutor, f *Function, ref uint64) (*RawData, uint64, error) {
	if len(f.Args) != 1 {
		return nil, 0, errors.New("Called block with " + strconv.Itoa(len(f.Args)) + " arguments, expected 1")
	}
	panic("NOT YET BLOCK CALL")
	// res, dref, err := c.resolveValue(f.Args[0], ref)
	// if res != nil && res.Type[0] != types.Bool {
	// 	return nil, 0, errors.New("Called expect body with wrong type, it should be a boolean (type mismatch)")
	// }
	// return res, dref, err
}

func returnCallV2(e *blockExecutor, f *Function, ref uint64) (*RawData, uint64, error) {
	arg := f.Args[0]
	return e.resolveValue(arg, ref)
}
