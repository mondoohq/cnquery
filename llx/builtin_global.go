package llx

import (
	"errors"
	"strconv"

	"go.mondoo.io/mondoo/types"
)

// handleGlobal takes a global function and returns a handler if found.
// this is not exported as it is only used internally. it exposes everything
// below this function
func handleGlobal(op string) (handleFunction, bool) {
	f, ok := globalFunctions[op]
	if !ok {
		return nil, false
	}
	return f, true
}

// DEFINITIONS

type handleFunction func(*LeiseExecutor, *Function, int32) (*RawData, int32, error)

var globalFunctions map[string]handleFunction

func init() {
	globalFunctions = map[string]handleFunction{
		"expect": expect,
		"if":     ifCall,
		"score":  scoreCall,
		"typeof": typeofCall,
		"{}":     block,
		"return": returnCall,
	}
}

func ifCall(c *LeiseExecutor, f *Function, ref int32) (*RawData, int32, error) {
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
			res, dref, err = c.runBlock(nil, f.Args[idx+1], ref)
			return res, dref, err
		}

		idx += 2
	}

	if idx < max {
		res, dref, err := c.runBlock(nil, f.Args[idx], ref)
		return res, dref, err
	}

	return NilData, 0, nil
}

func scoreCall(c *LeiseExecutor, f *Function, ref int32) (*RawData, int32, error) {
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

func typeofCall(c *LeiseExecutor, f *Function, ref int32) (*RawData, int32, error) {
	if len(f.Args) != 1 {
		return nil, 0, errors.New("Called `typeof` with " + strconv.Itoa(len(f.Args)) + " arguments, expected one")
	}

	res, dref, err := c.resolveValue(f.Args[0], ref)
	if err != nil || dref != 0 || res == nil {
		return res, dref, err
	}

	return StringData(res.Type.Label()), 0, nil
}

func expect(c *LeiseExecutor, f *Function, ref int32) (*RawData, int32, error) {
	if len(f.Args) != 1 {
		return nil, 0, errors.New("Called expect with " + strconv.Itoa(len(f.Args)) + " arguments, expected 1")
	}
	res, dref, err := c.resolveValue(f.Args[0], ref)
	if res != nil && res.Type != types.Bool {
		return nil, 0, errors.New("Called expect body with wrong type, it should be a boolean (type mismatch)")
	}
	return res, dref, err
}

func block(c *LeiseExecutor, f *Function, ref int32) (*RawData, int32, error) {
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

func returnCall(c *LeiseExecutor, f *Function, ref int32) (*RawData, int32, error) {
	arg := f.Args[0]
	return c.resolveValue(arg, ref)
}
