package leise

import (
	"errors"

	"go.mondoo.io/mondoo/leise/parser"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/types"
)

func compileStringContains(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
	if call == nil || len(call.Function) != 1 {
		return types.Nil, errors.New("function " + id + " needs one argument")
	}

	f := call.Function[0]
	if f.Value == nil || f.Value.Operand == nil {
		return types.Nil, errors.New("function " + id + " needs one argument")
	}

	valRaw := f.Value.Operand.Value
	val, err := c.compileValue(valRaw)
	if err != nil {
		return types.Nil, err
	}

	switch types.Type(val.Type) {
	case types.String:
		c.Result.Code.AddChunk(&llx.Chunk{
			Call: llx.Chunk_FUNCTION,
			Id:   "contains" + string(types.String),
			Function: &llx.Function{
				Type:    string(types.Bool),
				Binding: ref,
				Args:    []*llx.Primitive{val},
			},
		})
		return types.Bool, nil

	case types.Array(types.String):
		c.Result.Code.AddChunk(&llx.Chunk{
			Call: llx.Chunk_FUNCTION,
			Id:   "contains" + string(types.Array(types.String)),
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
