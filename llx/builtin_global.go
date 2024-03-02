// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package llx

import (
	"errors"
	"strconv"

	"go.mondoo.com/cnquery/v10/types"
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
		// type-conversions
		"string": stringCall,
		"$regex": regexCall, // TODO: support both the regex resource and the internal typemap!
		"float":  floatCall,
		"int":    intCall,
		"bool":   boolCall,
		"dict":   dictCall,
		"semver": semverCall,
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
		if dref != 0 || (err != nil && res == nil) {
			return res, dref, err
		}
		if res.Error != nil {
			return NilData, 0, res.Error
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

func semverCall(e *blockExecutor, f *Function, ref uint64) (*RawData, uint64, error) {
	if len(f.Args) != 1 {
		return nil, 0, errors.New("Called `semver` with " + strconv.Itoa(len(f.Args)) + " arguments, expected one")
	}

	res, dref, err := e.resolveValue(f.Args[0], ref)
	if err != nil || dref != 0 || res == nil {
		return res, dref, err
	}

	return &RawData{Type: types.Semver, Value: res.Value}, 0, nil
}

func stringCall(e *blockExecutor, f *Function, ref uint64) (*RawData, uint64, error) {
	if len(f.Args) != 1 {
		return nil, 0, errors.New("Called `string` with " + strconv.Itoa(len(f.Args)) + " arguments, expected one")
	}

	res, dref, err := e.resolveValue(f.Args[0], ref)
	if err != nil || dref != 0 || res == nil {
		return res, dref, err
	}

	switch v := res.Value.(type) {
	case string:
		return StringData(v), 0, nil
	case int64:
		i := strconv.FormatInt(v, 10)
		return StringData(i), 0, nil
	case float64:
		f := strconv.FormatFloat(v, 'f', 2, 64)
		return StringData(f), 0, nil
	case bool:
		if v {
			return StringData("true"), 0, nil
		}
		return StringData("false"), 0, nil
	default:
		return NilData, 0, nil
	}
}

func regexCall(e *blockExecutor, f *Function, ref uint64) (*RawData, uint64, error) {
	if len(f.Args) != 1 {
		return nil, 0, errors.New("Called `regex` with " + strconv.Itoa(len(f.Args)) + " arguments, expected one")
	}

	res, dref, err := e.resolveValue(f.Args[0], ref)
	if err != nil || dref != 0 || res == nil {
		return res, dref, err
	}

	switch v := res.Value.(type) {
	case string:
		return RegexData(v), 0, nil
	case int64:
		i := strconv.FormatInt(v, 10)
		return RegexData(i), 0, nil
	case float64:
		f := strconv.FormatFloat(v, 'f', 2, 64)
		return RegexData(f), 0, nil
	case bool:
		if v {
			return RegexData("true"), 0, nil
		}
		return RegexData("false"), 0, nil
	default:
		return NilData, 0, nil
	}
}

func intCall(e *blockExecutor, f *Function, ref uint64) (*RawData, uint64, error) {
	if len(f.Args) != 1 {
		return nil, 0, errors.New("Called `int` with " + strconv.Itoa(len(f.Args)) + " arguments, expected one")
	}

	res, dref, err := e.resolveValue(f.Args[0], ref)
	if err != nil || dref != 0 || res == nil {
		return res, dref, err
	}

	switch v := res.Value.(type) {
	case string:
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, 0, err
		}
		return IntData(i), 0, nil
	case int64:
		return IntData(v), 0, nil
	case float64:
		return IntData(int64(v)), 0, nil
	case bool:
		if v {
			return IntData(1), 0, nil
		}
		return IntData(0), 0, nil
	default:
		return NilData, 0, nil
	}
}

func floatCall(e *blockExecutor, f *Function, ref uint64) (*RawData, uint64, error) {
	if len(f.Args) != 1 {
		return nil, 0, errors.New("Called `float` with " + strconv.Itoa(len(f.Args)) + " arguments, expected one")
	}

	res, dref, err := e.resolveValue(f.Args[0], ref)
	if err != nil || dref != 0 || res == nil {
		return res, dref, err
	}

	switch v := res.Value.(type) {
	case string:
		i, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, 0, err
		}
		return FloatData(i), 0, nil
	case int64:
		return FloatData(float64(v)), 0, nil
	case float64:
		return FloatData(v), 0, nil
	case bool:
		if v {
			return FloatData(1), 0, nil
		}
		return FloatData(0), 0, nil
	default:
		return NilData, 0, nil
	}
}

func boolCall(e *blockExecutor, f *Function, ref uint64) (*RawData, uint64, error) {
	if len(f.Args) != 1 {
		return nil, 0, errors.New("Called `bool` with " + strconv.Itoa(len(f.Args)) + " arguments, expected one")
	}

	res, dref, err := e.resolveValue(f.Args[0], ref)
	if err != nil || dref != 0 || res == nil {
		return res, dref, err
	}

	switch v := res.Value.(type) {
	case string:
		return BoolData(v == "true"), 0, nil
	case int64:
		return BoolData(v != 0), 0, nil
	case float64:
		return BoolData(v != 0), 0, nil
	case bool:
		return BoolData(v), 0, nil
	default:
		return NilData, 0, nil
	}
}

func dictCall(e *blockExecutor, f *Function, ref uint64) (*RawData, uint64, error) {
	if len(f.Args) != 1 {
		return nil, 0, errors.New("Called `dict` with " + strconv.Itoa(len(f.Args)) + " arguments, expected one")
	}

	res, dref, err := e.resolveValue(f.Args[0], ref)
	if err != nil || dref != 0 || res == nil {
		return res, dref, err
	}

	switch v := res.Value.(type) {
	case string, int64, float64, bool, []any, map[string]any:
		return DictData(v), 0, nil
	default:
		return NilData, 0, nil
	}
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
