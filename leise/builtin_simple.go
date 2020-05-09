package leise

import (
	"errors"

	"go.mondoo.io/mondoo/leise/parser"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/types"
)

func compileStringContains(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
	if call != nil && len(call.Function) != 1 {
		return types.Nil, errors.New("function " + id + " needs one argument")
	}

	val := call.Function[0].Value.Operand.Value
	if val.String != nil {
		c.Result.Code.AddChunk(&llx.Chunk{
			Call: llx.Chunk_FUNCTION,
			Id:   "containsString",
			Function: &llx.Function{
				Type:    string(types.Bool),
				Binding: ref,
				Args: []*llx.Primitive{
					llx.StringPrimitive(*val.String),
				},
			},
		})
		return types.Bool, nil
	}

	return types.Nil, errors.New("cannot find #string.contains for this call")
}
