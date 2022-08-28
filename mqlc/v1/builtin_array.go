package v1

import (
	"errors"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/mqlc/parser"
	"go.mondoo.com/cnquery/types"
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
	argExpectation := llx.FunctionPrimitiveV1(functionRef)

	// if we have a standalone body in the where clause, then we need to check if
	// it's a value, in which case we need to compare the array value to it
	if standalone {
		t, err := c.functionBlockType(functionRef)
		if err != nil {
			return types.Nil, err
		}

		childType := typ.Child()
		chunkId := "==" + string(childType)
		if t != childType {
			chunkId = "==" + string(t)
			_, err := llx.BuiltinFunctionV1(t, chunkId)
			if err != nil {
				return types.Nil, errors.New("called '" + id + "' with wrong type; either provide a type " + childType.Label() + " value or write it as an expression (e.g. \"_ == 123\")")
			}
		}

		functionCode := c.Result.DeprecatedV5Code.Functions[functionRef-1]
		valueRef := functionCode.Entrypoints[len(functionCode.Entrypoints)-1]

		functionCode.AddChunk(&llx.Chunk{
			Call: llx.Chunk_FUNCTION,
			Id:   chunkId,
			Function: &llx.Function{
				Type:                string(types.Bool),
				DeprecatedV5Binding: 1,
				Args: []*llx.Primitive{
					llx.RefPrimitiveV1(valueRef),
				},
			},
		})

		functionCode.Entrypoints = []int32{functionCode.ChunkIndex()}
	}

	c.Result.DeprecatedV5Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   id,
		Function: &llx.Function{
			Type:                string(typ),
			DeprecatedV5Binding: ref,
			Args: []*llx.Primitive{
				llx.RefPrimitiveV1(ref),
				argExpectation,
			},
		},
	})
	return typ, nil
}

func compileArrayDuplicates(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
	if call != nil && len(call.Function) > 1 {
		return types.Nil, errors.New("too many arguments when calling '" + id + "'")
	} else if call != nil && len(call.Function) == 1 {
		arg := call.Function[0]

		functionRef, standalone, err := c.blockExpressions([]*parser.Expression{arg.Value}, typ)
		if err != nil {
			return types.Nil, err
		}
		if functionRef == 0 {
			return types.Nil, errors.New("called '" + id + "' without a function block")
		}
		argExpectation := llx.FunctionPrimitiveV1(functionRef)

		if standalone {
			return typ, errors.New("called duplicates with a field name on an invalid type")
		}

		c.Result.DeprecatedV5Code.AddChunk(&llx.Chunk{
			Call: llx.Chunk_FUNCTION,
			Id:   "fieldDuplicates",
			Function: &llx.Function{
				Type:                string(typ),
				DeprecatedV5Binding: ref,
				Args: []*llx.Primitive{
					llx.RefPrimitiveV1(ref),
					argExpectation,
				},
			},
		})
		return typ, nil
	}

	// Duplicates is being called with 0 arguments, which means it should be on an
	// array of basic types
	ct := typ.Child()
	_, ok := types.Equal[ct]
	if !ok {
		return typ, errors.New("cannot extract duplicates from array, must be a basic type. Try using a field argument.")
	}

	c.Result.DeprecatedV5Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   id,
		Function: &llx.Function{
			Type:                string(typ),
			DeprecatedV5Binding: ref,
		},
	})
	return typ, nil
}

func compileArrayUnique(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
	if call != nil && len(call.Function) > 0 {
		return types.Nil, errors.New("too many arguments when calling '" + id + "'")
	}

	ct := typ.Child()
	_, ok := types.Equal[ct]
	if !ok {
		return typ, errors.New("cannot extract uniques from array, don't know how to compare entries")
	}

	c.Result.DeprecatedV5Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   id,
		Function: &llx.Function{
			Type:                string(typ),
			DeprecatedV5Binding: ref,
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
	c.Result.DeprecatedV5Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "length",
		Function: &llx.Function{
			Type:                string(types.Int),
			DeprecatedV5Binding: c.Result.DeprecatedV5Code.ChunkIndex(),
		},
	})

	// > 0
	c.Result.DeprecatedV5Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   string(">" + types.Int),
		Function: &llx.Function{
			Type:                string(types.Bool),
			DeprecatedV5Binding: c.Result.DeprecatedV5Code.ChunkIndex(),
			Args: []*llx.Primitive{
				llx.IntPrimitive(0),
			},
		},
	})

	checksum := c.Result.DeprecatedV5Code.Checksums[c.Result.DeprecatedV5Code.ChunkIndex()]
	c.Result.Labels.Labels[checksum] = "[].contains()"

	return types.Bool, nil
}

func compileArrayContainsOnly(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
	if call == nil || len(call.Function) != 1 {
		return types.Nil, errors.New("function " + id + " needs one argument (array)")
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

	if valType != typ {
		return types.Nil, errors.New("types don't match for calling contains (got: " + valType.Label() + ", expected: " + typ.Label() + ")")
	}

	// .difference
	c.Result.DeprecatedV5Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "difference",
		Function: &llx.Function{
			Type:                string(typ),
			DeprecatedV5Binding: ref,
			Args: []*llx.Primitive{
				val,
			},
		},
	})

	// == []
	c.Result.DeprecatedV5Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   string("=="),
		Function: &llx.Function{
			Type:                string(types.Bool),
			DeprecatedV5Binding: c.Result.DeprecatedV5Code.ChunkIndex(),
			Args: []*llx.Primitive{
				llx.ArrayPrimitive([]*llx.Primitive{}, typ.Child()),
			},
		},
	})

	checksum := c.Result.DeprecatedV5Code.Checksums[c.Result.DeprecatedV5Code.ChunkIndex()]
	c.Result.Labels.Labels[checksum] = "[].containsOnly()"

	return types.Bool, nil
}

func compileArrayContainsNone(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
	if call == nil || len(call.Function) != 1 {
		return types.Nil, errors.New("function " + id + " needs one argument (array)")
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

	if valType != typ {
		return types.Nil, errors.New("types don't match for calling contains (got: " + valType.Label() + ", expected: " + typ.Label() + ")")
	}

	// .containsNone
	c.Result.DeprecatedV5Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "containsNone",
		Function: &llx.Function{
			Type:                string(typ),
			DeprecatedV5Binding: ref,
			Args: []*llx.Primitive{
				val,
			},
		},
	})

	// == []
	c.Result.DeprecatedV5Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   string("=="),
		Function: &llx.Function{
			Type:                string(types.Bool),
			DeprecatedV5Binding: c.Result.DeprecatedV5Code.ChunkIndex(),
			Args: []*llx.Primitive{
				llx.ArrayPrimitive([]*llx.Primitive{}, typ.Child()),
			},
		},
	})

	checksum := c.Result.DeprecatedV5Code.Checksums[c.Result.DeprecatedV5Code.ChunkIndex()]
	c.Result.Labels.Labels[checksum] = "[].containsNone()"

	return types.Bool, nil
}

func compileArrayAll(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
	_, err := compileWhere(c, typ, ref, "$whereNot", call)
	if err != nil {
		return types.Nil, err
	}
	listRef := c.Result.DeprecatedV5Code.ChunkIndex()

	if err := compileListAssertionMsg(c, typ, ref, listRef, listRef); err != nil {
		return types.Nil, err
	}

	c.Result.DeprecatedV5Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "$all",
		Function: &llx.Function{
			Type:                string(types.Bool),
			DeprecatedV5Binding: listRef,
		},
	})

	checksum := c.Result.DeprecatedV5Code.Checksums[c.Result.DeprecatedV5Code.ChunkIndex()]
	c.Result.Labels.Labels[checksum] = "[].all()"

	return types.Bool, nil
}

func compileArrayAny(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
	_, err := compileWhere(c, typ, ref, "where", call)
	if err != nil {
		return types.Nil, err
	}
	listRef := c.Result.DeprecatedV5Code.ChunkIndex()

	if err := compileListAssertionMsg(c, typ, ref, ref, listRef); err != nil {
		return types.Nil, err
	}

	c.Result.DeprecatedV5Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "$any",
		Function: &llx.Function{
			Type:                string(types.Bool),
			DeprecatedV5Binding: listRef,
		},
	})

	checksum := c.Result.DeprecatedV5Code.Checksums[c.Result.DeprecatedV5Code.ChunkIndex()]
	c.Result.Labels.Labels[checksum] = "[].any()"

	return types.Bool, nil
}

func compileArrayOne(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
	_, err := compileWhere(c, typ, ref, "where", call)
	if err != nil {
		return types.Nil, err
	}
	listRef := c.Result.DeprecatedV5Code.ChunkIndex()

	if err := compileListAssertionMsg(c, typ, ref, listRef, listRef); err != nil {
		return types.Nil, err
	}

	c.Result.DeprecatedV5Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "$one",
		Function: &llx.Function{
			Type:                string(types.Bool),
			DeprecatedV5Binding: listRef,
		},
	})

	checksum := c.Result.DeprecatedV5Code.Checksums[c.Result.DeprecatedV5Code.ChunkIndex()]
	c.Result.Labels.Labels[checksum] = "[].one()"

	return types.Bool, nil
}

func compileArrayNone(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
	_, err := compileWhere(c, typ, ref, "where", call)
	if err != nil {
		return types.Nil, err
	}
	listRef := c.Result.DeprecatedV5Code.ChunkIndex()

	if err := compileListAssertionMsg(c, typ, ref, listRef, listRef); err != nil {
		return types.Nil, err
	}

	c.Result.DeprecatedV5Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "$none",
		Function: &llx.Function{
			Type:                string(types.Bool),
			DeprecatedV5Binding: listRef,
		},
	})

	checksum := c.Result.DeprecatedV5Code.Checksums[c.Result.DeprecatedV5Code.ChunkIndex()]
	c.Result.Labels.Labels[checksum] = "[].none()"

	return types.Bool, nil
}

func compileArrayMap(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
	if call == nil {
		return types.Nil, errors.New("missing filter argument for calling '" + id + "'")
	}
	if len(call.Function) > 1 {
		return types.Nil, errors.New("too many arguments when calling '" + id + "', only 1 is supported")
	}

	// if the map function is called without arguments, we don't have to do anything
	// so we just return the caller type as no additional step in the compiler is necessary
	if len(call.Function) == 0 {
		return typ, nil
	}

	arg := call.Function[0]
	if arg.Name != "" {
		return types.Nil, errors.New("called '" + id + "' with a named parameter, which is not supported")
	}

	functionRef, _, err := c.blockExpressions([]*parser.Expression{arg.Value}, typ)
	if err != nil {
		return types.Nil, err
	}
	if functionRef == 0 {
		return types.Nil, errors.New("called '" + id + "' without a function block")
	}
	argExpectation := llx.FunctionPrimitiveV1(functionRef)

	f := c.Result.DeprecatedV5Code.Functions[functionRef-1]
	if len(f.Entrypoints) != 1 {
		return types.Nil, errors.New("called '" + id + "' with a bad function block, you can only return 1 value")
	}
	mappedType := f.Code[f.Entrypoints[0]-1].Type()

	c.Result.DeprecatedV5Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   id,
		Function: &llx.Function{
			Type:                string(types.Array(mappedType)),
			DeprecatedV5Binding: ref,
			Args: []*llx.Primitive{
				llx.RefPrimitiveV1(ref),
				argExpectation,
			},
		},
	})
	return types.Array(mappedType), nil
}
