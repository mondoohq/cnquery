// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mqlc

import (
	"errors"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/mqlc/parser"
	"go.mondoo.com/cnquery/v10/types"
)

var typeConversions map[string]fieldCompiler

func init() {
	typeConversions = map[string]fieldCompiler{
		"semver": compileTypeConversion(types.Semver),
		"bool":   compileTypeConversion(types.Bool),
		"int":    compileTypeConversion(types.Int),
		"float":  compileTypeConversion(types.Float),
		"string": compileTypeConversion(types.String),
		"regex":  compileTypeConversion(types.Regex),
		"dict":   compileTypeConversion(types.Dict),
	}
}

func compileTypeConversion(typ types.Type) fieldCompiler {
	return func(c *compiler, id string, call *parser.Call) (types.Type, error) {
		if call == nil || len(call.Function) < 1 {
			return types.Nil, errors.New("missing parameter for '" + id + "', it requires 1")
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
			Id:   id,
			Function: &llx.Function{
				Type: string(types.String),
				Args: []*llx.Primitive{argValue},
			},
		})

		return types.String, nil
	}
}
