// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mqlc

import (
	"errors"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/mqlc/parser"
	"go.mondoo.com/cnquery/types"
)

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
