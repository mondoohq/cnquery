package llx

import (
	"errors"
	"strconv"

	"go.mondoo.io/mondoo/types"
)

// handleGlobal takes a global function and returns a handler if found.
// this is not exported as it is only used internally. it exposes everything
// below this function
func handleGlobalV1(op string) (handleFunctionV1, bool) {
	f, ok := globalFunctionsV1[op]
	if !ok {
		return nil, false
	}
	return f, true
}

// DEFINITIONS

type handleFunctionV1 func(*MQLExecutorV1, *Function, int32) (*RawData, int32, error)

var globalFunctionsV1 map[string]handleFunctionV1

func init() {
	globalFunctionsV1 = map[string]handleFunctionV1{
		"expect": expectV1,
		"if":     ifCallV1,
		"switch": switchCallV1,
		"score":  scoreCallV1,
		"typeof": typeofCallV1,
		"{}":     blockV1,
		"return": returnCallV1,
	}
}

func ifCallV1(c *MQLExecutorV1, f *Function, ref int32) (*RawData, int32, error) {
	if len(f.Args) < 2 {
		return nil, 0, errors.New("Called if with " + strconv.Itoa(len(f.Args)) + " arguments, expected at least 2")
	}

	var idx int
	max := len(f.Args)
	for idx+1 < max {
		res, dref, err := c.resolveValue(f.Args[idx], ref)
		if err != nil || dref != 0 || res == nil {
			return res, dref, err
		}

		if truthy, _ := res.IsTruthy(); truthy {
			res, dref, err = c.runBlock(nil, f.Args[idx+1], nil, ref)
			return res, dref, err
		}

		idx += 2
	}

	if idx < max {
		res, dref, err := c.runBlock(nil, f.Args[idx], nil, ref)
		return res, dref, err
	}

	return NilData, 0, nil
}

func switchCallV1(c *MQLExecutorV1, f *Function, ref int32) (*RawData, int32, error) {
	// very similar to the if-call above; minor differences:
	// - we have an optional reference value (which is in the function call)
	// - default is translated to `true` in its condition; everything else is a function

	if len(f.Args) < 2 {
		return nil, 0, errors.New("Called switch with no arguments, expected at least one case statement")
	}

	var bind *RawData
	if types.Type(f.Args[0].Type) != types.Unset {
		var dref int32
		var err error
		bind, dref, err = c.resolveValue(f.Args[0], ref)
		if err != nil || dref != 0 || bind == nil {
			return bind, dref, err
		}
	}

	// ignore the first argument, it's just the reference value
	idx := 1
	max := len(f.Args)
	defaultCaseIdx := -1
	for idx+1 < max {
		if types.Type(f.Args[idx].Type) == types.Bool {
			defaultCaseIdx = idx
			idx += 2
			continue
		}

		res, dref, err := c.resolveValue(f.Args[idx], ref)
		if err != nil || dref != 0 || res == nil {
			return res, dref, err
		}

		if truthy, _ := res.IsTruthy(); truthy {
			res, dref, err = c.runBlock(bind, f.Args[idx+1], nil, ref)
			return res, dref, err
		}

		idx += 2
	}

	if defaultCaseIdx != -1 {
		res, dref, err := c.runBlock(nil, f.Args[defaultCaseIdx+1], nil, ref)
		return res, dref, err
	}

	return NilData, 0, nil
}

func scoreCallV1(c *MQLExecutorV1, f *Function, ref int32) (*RawData, int32, error) {
	if len(f.Args) != 1 {
		return nil, 0, errors.New("Called `score` with " + strconv.Itoa(len(f.Args)) + " arguments, expected one")
	}

	res, dref, err := c.resolveValue(f.Args[0], ref)
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

func typeofCallV1(c *MQLExecutorV1, f *Function, ref int32) (*RawData, int32, error) {
	if len(f.Args) != 1 {
		return nil, 0, errors.New("Called `typeof` with " + strconv.Itoa(len(f.Args)) + " arguments, expected one")
	}

	res, dref, err := c.resolveValue(f.Args[0], ref)
	if err != nil || dref != 0 || res == nil {
		return res, dref, err
	}

	return StringData(res.Type.Label()), 0, nil
}

func expectV1(c *MQLExecutorV1, f *Function, ref int32) (*RawData, int32, error) {
	if len(f.Args) != 1 {
		return nil, 0, errors.New("Called expect with " + strconv.Itoa(len(f.Args)) + " arguments, expected 1")
	}
	res, dref, err := c.resolveValue(f.Args[0], ref)
	if res != nil && res.Type != types.Bool {
		return nil, 0, errors.New("Called expect body with wrong type, it should be a boolean (type mismatch)")
	}
	return res, dref, err
}

func blockV1(c *MQLExecutorV1, f *Function, ref int32) (*RawData, int32, error) {
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

func returnCallV1(c *MQLExecutorV1, f *Function, ref int32) (*RawData, int32, error) {
	arg := f.Args[0]
	return c.resolveValue(arg, ref)
}
