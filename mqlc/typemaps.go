// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mqlc

import (
	"errors"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/mqlc/parser"
	"go.mondoo.com/cnquery/v11/types"
)

var typeConversions map[string]fieldCompiler

func init() {
	typeConversions = map[string]fieldCompiler{
		"bool":    compileTypeConversion("bool", types.Bool),
		"int":     compileTypeConversion("int", types.Int),
		"float":   compileTypeConversion("float", types.Float),
		"string":  compileTypeConversion("string", types.String),
		"regex":   compileTypeConversion("$regex", types.Regex),
		"dict":    compileTypeConversion("dict", types.Dict),
		"version": compileTypeConversion("version", types.Version),
		// FIXME: DEPRECATED, remove in v13.0 vv
		"semver": compileTypeConversion("semver", types.Version), // deprecated
		//
	}
}

var errNotConversion = errors.New("not a type-conversion")

func compileTypeConversion(llxID string, typ types.Type) fieldCompiler {
	return func(c *compiler, id string, call *parser.Call) (types.Type, error) {
		if call == nil || len(call.Function) < 1 {
			return types.Nil, errNotConversion
		}

		arg := call.Function[0]
		if arg == nil || arg.Value == nil || arg.Value.Operand == nil || arg.Value.Operand.Value == nil {
			return types.Nil, errors.New("failed to get parameter for '" + id + "'")
		}

		argValue, err := c.compileExpression(arg.Value)
		if err != nil {
			return types.Nil, err
		}

		c.addChunk(&llx.Chunk{
			Call: llx.Chunk_FUNCTION,
			Id:   llxID,
			Function: &llx.Function{
				Type: string(typ),
				Args: []*llx.Primitive{argValue},
			},
		})

		return typ, nil
	}
}
