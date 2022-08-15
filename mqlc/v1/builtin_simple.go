package v1

import (
	"errors"

	"go.mondoo.io/mondoo/mqlc/parser"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/types"
)

func compileStringContains(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
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

	code := c.Result.DeprecatedV5Code

	switch valType {
	case types.String:
		code.AddChunk(&llx.Chunk{
			Call: llx.Chunk_FUNCTION,
			Id:   "contains" + string(types.String),
			Function: &llx.Function{
				Type:                string(types.Bool),
				DeprecatedV5Binding: ref,
				Args:                []*llx.Primitive{val},
			},
		})
		return types.Bool, nil
	case types.Int:
		code.AddChunk(&llx.Chunk{
			Call: llx.Chunk_FUNCTION,
			Id:   "contains" + string(types.Int),
			Function: &llx.Function{
				Type:                string(types.Bool),
				DeprecatedV5Binding: ref,
				Args:                []*llx.Primitive{val},
			},
		})
		return types.Bool, nil
	case types.Array(types.String):
		code.AddChunk(&llx.Chunk{
			Call: llx.Chunk_FUNCTION,
			Id:   "contains" + string(types.Array(types.String)),
			Function: &llx.Function{
				Type:                string(types.Bool),
				DeprecatedV5Binding: ref,
				Args:                []*llx.Primitive{val},
			},
		})
		return types.Bool, nil
	case types.Array(types.Int):
		code.AddChunk(&llx.Chunk{
			Call: llx.Chunk_FUNCTION,
			Id:   "contains" + string(types.Array(types.Int)),
			Function: &llx.Function{
				Type:                string(types.Bool),
				DeprecatedV5Binding: ref,
				Args:                []*llx.Primitive{val},
			},
		})
		return types.Bool, nil
	default:
		return types.Nil, errors.New("cannot find #string.contains with this type " + types.Type(val.Type).Label())
	}
}
