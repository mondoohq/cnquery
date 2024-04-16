// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mqlc

import (
	"errors"
	"strconv"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/mqlc/parser"
	"go.mondoo.com/cnquery/v11/types"
)

func callArgTypeIs(c *compiler, call *parser.Call, id string, argName string, idx int, types ...types.Type) (*llx.Primitive, error) {
	if len(call.Function) <= idx {
		return nil, errors.New("function " + id + " is missing a " + argName + " (arg #" + strconv.Itoa(idx+1) + ")")
	}

	arg := call.Function[idx]
	if arg.Value == nil || arg.Value.Operand == nil {
		return nil, errors.New("function " + id + " is missing a " + argName + " (arg #" + strconv.Itoa(idx+1) + " is null)")
	}

	val, err := c.compileOperand(arg.Value.Operand)
	if err != nil {
		return nil, err
	}

	valType, err := c.dereferenceType(val)
	if err != nil {
		return nil, err
	}

	for _, t := range types {
		if t == valType {
			return val, nil
		}
	}

	var typesStr string
	for _, t := range types {
		typesStr += t.Label() + "/"
	}
	return nil, errors.New("function " + id + " type mismatch for " + argName + " (expected: " + typesStr[0:len(typesStr)-1] + ", got: " + valType.Label() + ")")
}

func compileStringContains(c *compiler, typ types.Type, ref uint64, id string, call *parser.Call) (types.Type, error) {
	if call == nil || len(call.Function) != 1 {
		return types.Nil, errors.New("function " + id + " needs one argument (function missing)")
	}

	f := call.Function[0]
	if f.Value == nil || f.Value.Operand == nil {
		return types.Nil, errors.New("function " + id + " needs one argument")
	}

	val, err := c.compileOperand(f.Value.Operand)
	if err != nil {
		return types.Nil, err
	}

	valType, err := c.dereferenceType(val)
	if err != nil {
		return types.Nil, err
	}

	switch valType {
	case types.String:
		c.addChunk(&llx.Chunk{
			Call: llx.Chunk_FUNCTION,
			Id:   "contains" + string(types.String),
			Function: &llx.Function{
				Type:    string(types.Bool),
				Binding: ref,
				Args:    []*llx.Primitive{val},
			},
		})
		return types.Bool, nil
	case types.Int:
		c.addChunk(&llx.Chunk{
			Call: llx.Chunk_FUNCTION,
			Id:   "contains" + string(types.Int),
			Function: &llx.Function{
				Type:    string(types.Bool),
				Binding: ref,
				Args:    []*llx.Primitive{val},
			},
		})
		return types.Bool, nil
	case types.Regex:
		c.addChunk(&llx.Chunk{
			Call: llx.Chunk_FUNCTION,
			Id:   "contains" + string(types.Regex),
			Function: &llx.Function{
				Type:    string(types.Bool),
				Binding: ref,
				Args:    []*llx.Primitive{val},
			},
		})
		return types.Bool, nil
	case types.Array(types.String):
		c.addChunk(&llx.Chunk{
			Call: llx.Chunk_FUNCTION,
			Id:   "contains" + string(types.Array(types.String)),
			Function: &llx.Function{
				Type:    string(types.Bool),
				Binding: ref,
				Args:    []*llx.Primitive{val},
			},
		})
		return types.Bool, nil
	case types.Array(types.Int):
		c.addChunk(&llx.Chunk{
			Call: llx.Chunk_FUNCTION,
			Id:   "contains" + string(types.Array(types.Int)),
			Function: &llx.Function{
				Type:    string(types.Bool),
				Binding: ref,
				Args:    []*llx.Primitive{val},
			},
		})
		return types.Bool, nil
	case types.Array(types.Regex):
		c.addChunk(&llx.Chunk{
			Call: llx.Chunk_FUNCTION,
			Id:   "contains" + string(types.Array(types.Regex)),
			Function: &llx.Function{
				Type:    string(types.Bool),
				Binding: ref,
				Args:    []*llx.Primitive{val},
			},
		})
		return types.Bool, nil
	default:
		return types.Nil, errors.New("cannot find #string.contains with this type " + types.Type(val.Type).Label())
	}
}

func compileStringIn(c *compiler, typ types.Type, ref uint64, id string, call *parser.Call) (types.Type, error) {
	if call == nil || len(call.Function) != 1 {
		return types.Nil, errors.New("function " + id + " needs one argument")
	}

	arr, err := callArgTypeIs(c, call, id, "list", 0, types.Array(types.String), types.Array(types.Unset))
	if err != nil {
		return types.Nil, err
	}

	c.addChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "in",
		Function: &llx.Function{
			Type:    string(types.Bool),
			Binding: ref,
			Args:    []*llx.Primitive{arr},
		},
	})
	return types.Bool, nil
}

func compileNumberInRange(c *compiler, typ types.Type, ref uint64, id string, call *parser.Call) (types.Type, error) {
	if call == nil || len(call.Function) != 2 {
		return types.Nil, errors.New("function " + id + " needs two arguments")
	}

	min, err := callArgTypeIs(c, call, id, "min", 0, types.Int, types.Float, types.Dict)
	if err != nil {
		return types.Nil, err
	}
	max, err := callArgTypeIs(c, call, id, "max", 1, types.Int, types.Float, types.Dict)
	if err != nil {
		return types.Nil, err
	}

	c.addChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "inRange",
		Function: &llx.Function{
			Type:    string(types.Bool),
			Binding: ref,
			Args:    []*llx.Primitive{min, max},
		},
	})
	return types.Bool, nil
}

func compileTimeInRange(c *compiler, typ types.Type, ref uint64, id string, call *parser.Call) (types.Type, error) {
	if call == nil || len(call.Function) != 2 {
		return types.Nil, errors.New("function " + id + " needs two arguments")
	}

	min, err := callArgTypeIs(c, call, id, "min", 0, types.Time)
	if err != nil {
		return types.Nil, err
	}
	max, err := callArgTypeIs(c, call, id, "max", 1, types.Time)
	if err != nil {
		return types.Nil, err
	}

	c.addChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "inRange",
		Function: &llx.Function{
			Type:    string(types.Bool),
			Binding: ref,
			Args:    []*llx.Primitive{min, max},
		},
	})
	return types.Bool, nil
}
