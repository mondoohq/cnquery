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
		"{}":     block,
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
