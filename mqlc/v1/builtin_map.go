package v1

import (
	"errors"

	"go.mondoo.io/mondoo/mqlc/parser"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/types"
)

func compileMapWhere(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
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

	keyType := typ.Key()
	valueType := typ.Child()
	resCode := c.Result.DeprecatedV5Code
	bindingChecksum := resCode.Checksums[resCode.ChunkIndex()]

	blockCompiler := c.newBlockCompiler(&llx.CodeV1{
		Id:         "binding",
		Parameters: 2,
		Checksums: map[int32]string{
			// we must provide the first chunk, which is a reference to the caller
			// and which will always be number 1
			// Additionally we are setting the second checksum here as well as a place-
			// holder for the second value.
			1: bindingChecksum,
			2: bindingChecksum,
		},
		Code: []*llx.Chunk{
			{
				Call:      llx.Chunk_PRIMITIVE,
				Primitive: &llx.Primitive{Type: string(keyType)},
			},
			{
				Call:      llx.Chunk_PRIMITIVE,
				Primitive: &llx.Primitive{Type: string(valueType)},
			},
		},
	}, &binding{Type: types.Type(typ), Ref: 1})

	blockCompiler.vars["key"] = variable{ref: 1, typ: keyType}
	blockCompiler.vars["value"] = variable{ref: 2, typ: valueType}

	err := blockCompiler.compileExpressions([]*parser.Expression{arg.Value})
	c.Result.Suggestions = append(c.Result.Suggestions, blockCompiler.Result.Suggestions...)
	if err != nil {
		return typ, err
	}

	code := blockCompiler.Result.DeprecatedV5Code
	code.UpdateID()
	resCode.Functions = append(resCode.Functions, code)
	functionRef := resCode.FunctionsIndex()
	argExpectation := llx.FunctionPrimitiveV1(functionRef)

	resCode.AddChunk(&llx.Chunk{
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
