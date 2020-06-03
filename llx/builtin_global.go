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
		"{}":     block,
	}
}

func ifCall(c *LeiseExecutor, f *Function, ref int32) (*RawData, int32, error) {
	if len(f.Args) != 2 {
		return nil, 0, errors.New("Called if with " + strconv.Itoa(len(f.Args)) + " arguments, expected 2")
	}

	res, dref, err := c.resolveValue(f.Args[0], ref)
	if err != nil || dref != 0 || res == nil {
		return res, dref, err
	}

	if truthy, _ := res.IsTruthy(); truthy {
		res, dref, err = c.runBlock(nil, f.Args[1], ref)
		return res, dref, err
	} else {
		return NilData, 0, nil
	}
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
