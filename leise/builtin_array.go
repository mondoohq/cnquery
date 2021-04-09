package leise

import (
	"errors"

	"go.mondoo.io/mondoo/leise/parser"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/types"
)

func compileWhere(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
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
		return types.Nil, errors.New("called '" + id + "' with a named parameter, which is not supported")
	}

	functionRef, standalone, err := c.blockExpressions([]*parser.Expression{arg.Value}, typ)
	if err != nil {
		return types.Nil, err
	}
	if functionRef == 0 {
		return types.Nil, errors.New("called '" + id + "' without a function block")
	}
	argExpectation := llx.FunctionPrimitive(functionRef)

	// if we have a standalone body in the where clause, then we need to check if
	// it's a value, in which case we need to compare the array value to it
	if standalone {
		t, err := c.functionBlockType(functionRef)
		if err != nil {
			return types.Nil, err
		}

		childType := typ.Child()
		if t != childType {
			return types.Nil, errors.New("called '" + id + "' with wrong type; either provide a type " + childType.Label() + " value or write it as an expression (e.g. \"_ == 123\")")
		}

		functionCode := c.Result.Code.Functions[functionRef-1]
		valueRef := functionCode.Entrypoints[len(functionCode.Entrypoints)-1]

		functionCode.AddChunk(&llx.Chunk{
			Call: llx.Chunk_FUNCTION,
			Id:   "==" + string(childType),
			Function: &llx.Function{
				Type:    types.Bool,
				Binding: 1,
				Args: []*llx.Primitive{
					llx.RefPrimitive(valueRef),
				},
			},
		})

		functionCode.Entrypoints = []int32{functionCode.ChunkIndex()}
	}

	c.Result.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   id,
		Function: &llx.Function{
			Type:    typ,
			Binding: ref,
			Args: []*llx.Primitive{
				llx.RefPrimitive(ref),
				argExpectation,
			},
		},
	})
	return typ, nil
}

func compileArrayContains(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
	_, err := compileWhere(c, typ, ref, "where", call)
	if err != nil {
		return types.Nil, err
	}

	// .length
	c.Result.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "length",
		Function: &llx.Function{
			Type:    types.Int,
			Binding: c.Result.GetCode().ChunkIndex(),
		},
	})

	// > 0
	c.Result.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   string(">" + types.Int),
		Function: &llx.Function{
			Type:    types.Bool,
			Binding: c.Result.Code.ChunkIndex(),
			Args: []*llx.Primitive{
				llx.IntPrimitive(0),
			},
		},
	})

	checksum := c.Result.Code.Checksums[c.Result.Code.ChunkIndex()]
	c.Result.Labels.Labels[checksum] = "[].contains()"

	return types.Bool, nil
}

func compileArrayAll(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
	_, err := compileWhere(c, typ, ref, "where", call)
	if err != nil {
		return types.Nil, err
	}
	listRef := c.Result.GetCode().ChunkIndex()

	// .length ==> allLen
	c.Result.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "length",
		Function: &llx.Function{
			Type:    types.Int,
			Binding: ref,
		},
	})
	allLengthRef := c.Result.Code.ChunkIndex()

	// .length ==> after where clause
	c.Result.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "length",
		Function: &llx.Function{
			Type:    types.Int,
			Binding: listRef,
		},
	})

	// == allLen
	c.Result.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   string("==" + types.Int),
		Function: &llx.Function{
			Type:    types.Bool,
			Binding: c.Result.Code.ChunkIndex(),
			Args: []*llx.Primitive{
				llx.RefPrimitive(allLengthRef),
			},
		},
	})

	checksum := c.Result.Code.Checksums[c.Result.Code.ChunkIndex()]
	c.Result.Labels.Labels[checksum] = "[].all()"

	return types.Bool, nil
}

func compileArrayAny(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
	_, err := compileWhere(c, typ, ref, "where", call)
	if err != nil {
		return types.Nil, err
	}
	listRef := c.Result.GetCode().ChunkIndex()

	// .length ==> after where clause
	c.Result.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "length",
		Function: &llx.Function{
			Type:    types.Int,
			Binding: listRef,
		},
	})

	// == allLen
	c.Result.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   string("!=" + types.Int),
		Function: &llx.Function{
			Type:    types.Bool,
			Binding: c.Result.Code.ChunkIndex(),
			Args: []*llx.Primitive{
				llx.IntPrimitive(0),
			},
		},
	})

	checksum := c.Result.Code.Checksums[c.Result.Code.ChunkIndex()]
	c.Result.Labels.Labels[checksum] = "[].any()"

	return types.Bool, nil
}

func compileArrayOne(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
	_, err := compileWhere(c, typ, ref, "where", call)
	if err != nil {
		return types.Nil, err
	}

	// .length
	c.Result.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "length",
		Function: &llx.Function{
			Type:    types.Int,
			Binding: c.Result.GetCode().ChunkIndex(),
		},
	})

	// == 1
	c.Result.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   string("==" + types.Int),
		Function: &llx.Function{
			Type:    types.Bool,
			Binding: c.Result.Code.ChunkIndex(),
			Args: []*llx.Primitive{
				llx.IntPrimitive(1),
			},
		},
	})

	checksum := c.Result.Code.Checksums[c.Result.Code.ChunkIndex()]
	c.Result.Labels.Labels[checksum] = "[].one()"

	return types.Bool, nil
}

func compileArrayNone(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
	_, err := compileWhere(c, typ, ref, "where", call)
	if err != nil {
		return types.Nil, err
	}

	// .length
	c.Result.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "length",
		Function: &llx.Function{
			Type:    types.Int,
			Binding: c.Result.GetCode().ChunkIndex(),
		},
	})

	// == 0
	c.Result.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   string("==" + types.Int),
		Function: &llx.Function{
			Type:    types.Bool,
			Binding: c.Result.Code.ChunkIndex(),
			Args: []*llx.Primitive{
				llx.IntPrimitive(0),
			},
		},
	})

	checksum := c.Result.Code.Checksums[c.Result.Code.ChunkIndex()]
	c.Result.Labels.Labels[checksum] = "[].none()"

	return types.Bool, nil
}
