package leise

import (
	"errors"

	"go.mondoo.io/mondoo/leise/parser"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/types"
)

func compileArrayWhere(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
	if call == nil {
		return types.Nil, errors.New("missing filter argument for calling '" + id + "'")
	}
	if len(call.Function) > 1 {
		return types.Nil, errors.New("too many arguments when calling '" + id + "', only 1 is supported")
	}

	// if the where function is called without arguments, we don't have to do anything
	// so we just return the caller type as no additional step in the compiler is necessary
	if len(call.Function) == 0 {
		return typ, nil
	}

	arg := call.Function[0]
	if arg.Name != "" {
		return types.Nil, errors.New("called '" + id + "' function with a named parameter, which is not supported")
	}

	functionRef, err := c.blockExpressions([]*parser.Expression{arg.Value}, typ)
	if err != nil {
		return types.Nil, err
	}
	if functionRef == 0 {
		return types.Nil, errors.New("called '" + id + "' clause without a function block")
	}

	c.Result.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   id,
		Function: &llx.Function{
			Type:    string(typ),
			Binding: ref,
			Args: []*llx.Primitive{
				llx.RefPrimitive(ref),
				llx.FunctionPrimitive(functionRef),
			},
		},
	})
	return typ, nil
}
