package leise

import (
	"strconv"

	"github.com/cockroachdb/errors"
	"go.mondoo.io/mondoo/leise/parser"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/types"
)

func compileResourceDefault(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
	name := typ.ResourceName()
	resource := c.Schema.Resources[name]
	if resource == nil {
		return types.Nil, errors.New("cannot find resource '" + name + "' when compiling field '" + id + "'")
	}

	// special case that we can optimize: the previous call was a resource
	// without any call arguments + the combined type is a resource itself
	// in that case save the outer call and go for the resource directly
	prev := c.Result.Code.LastChunk()
	if prev.Call == llx.Chunk_FUNCTION && prev.Function == nil {
		name := prev.Id + "." + id
		resourceinfo, isResource := c.Schema.Resources[name]
		if isResource {
			c.Result.Code.RemoveLastChunk()
			return c.addResource(name, resourceinfo, call)
		}
	}

	fieldinfo := resource.Fields[id]
	if fieldinfo == nil {
		addFieldSuggestions(publicFieldsInfo(resource), id, c.Result)
		return "", errors.New("cannot find field '" + id + "' in resource " + resource.Name)
	}

	c.Result.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   id,
		Function: &llx.Function{
			Type:    fieldinfo.Type,
			Binding: ref,
			// no Args for field calls yet
		},
	})

	return types.Type(fieldinfo.Type), nil
}

// FunctionSignature of any function type
type FunctionSignature struct {
	Required int
	Args     []types.Type
}

func (f *FunctionSignature) expected2string() string {
	if f.Required == len(f.Args) {
		return strconv.Itoa(f.Required)
	}
	return strconv.Itoa(f.Required) + "-" + strconv.Itoa(len(f.Args))
}

// Validate the field call against the signature. Returns nil if valid and
// an error message otherwise
func (f *FunctionSignature) Validate(args []*llx.Primitive, c *compiler) error {
	max := len(f.Args)
	given := len(args)

	if given == 0 {
		if f.Required > 0 {
			return errors.New("no arguments given (expected " + f.expected2string() + ")")
		}
		return nil
	}

	if given < f.Required {
		return errors.New("not enough arguments (expected " + f.expected2string() + ", got " + strconv.Itoa(given) + ")")
	}
	if given > max {
		return errors.New("too many arguments (expected " + f.expected2string() + ", got " + strconv.Itoa(given) + ")")
	}

	for i := range args {
		req := f.Args[i]
		argT := types.Type(args[i].Type)

		var err error
		if argT == types.Ref {
			argT, err = c.dereferenceType(args[i])
			if err != nil {
				return errors.Wrap(err, "failed to dereference argument in validating function signature")
			}
		}

		// TODO: find out the real type from these REF types
		if argT == req || req == types.Any {
			continue
		}

		return errors.New("incorrect argument " + strconv.Itoa(i) + ": expected " + req.Label() + " got " + argT.Label())
	}
	return nil
}

func listResource(c *compiler, typ types.Type) (*lumi.ResourceInfo, error) {
	name := typ.ResourceName()
	resource := c.Schema.Resources[name]
	if resource == nil {
		return nil, errors.New("cannot find resource '" + name + "'")
	}
	if resource.ListType == "" {
		return nil, errors.New("resource '" + name + "' is not a list type")
	}
	return resource, nil
}

func compileResourceWhere(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
	resource, err := listResource(c, typ)
	if err != nil {
		return types.Nil, errors.New("failed to compile " + id + ": " + err.Error())
	}

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

	functionRef, _, err := c.blockExpressions([]*parser.Expression{arg.Value}, types.Array(types.Type(resource.ListType)))
	if err != nil {
		return types.Nil, err
	}
	if functionRef == 0 {
		return types.Nil, errors.New("called '" + id + "' clause without a function block")
	}

	resourceRef := c.Result.Code.ChunkIndex()

	listType, err := compileResourceDefault(c, typ, ref, "list", nil)
	if err != nil {
		return listType, err
	}
	listRef := c.Result.Code.ChunkIndex()

	c.Result.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   id,
		Function: &llx.Function{
			Type:    string(types.Resource(resource.Name)),
			Binding: resourceRef,
			Args: []*llx.Primitive{
				llx.RefPrimitive(listRef),
				llx.FunctionPrimitive(functionRef),
			},
		},
	})
	return typ, nil
}

func compileResourceContains(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
	// resource.where
	_, err := compileResourceWhere(c, typ, ref, "where", call)
	if err != nil {
		return types.Nil, err
	}
	resourceRef := c.Result.Code.ChunkIndex()

	// .list
	t, err := compileResourceDefault(c, typ, resourceRef, "list", nil)
	if err != nil {
		return t, err
	}
	listRef := c.Result.Code.ChunkIndex()

	// .length
	c.Result.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "length",
		Function: &llx.Function{
			Type:    string(types.Int),
			Binding: listRef,
		},
	})

	// > 0
	c.Result.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   string(">" + types.Int),
		Function: &llx.Function{
			Type:    string(types.Bool),
			Binding: c.Result.Code.ChunkIndex(),
			Args: []*llx.Primitive{
				llx.IntPrimitive(0),
			},
		},
	})

	checksum := c.Result.Code.Checksums[c.Result.Code.ChunkIndex()]
	c.Result.Labels.Labels[checksum] = typ.ResourceName() + ".contains()"

	return types.Bool, nil
}

func compileListAssertionMsg(c *compiler, typ types.Type, failedRef int32, assertionRef int32) error {
	// assertions
	msg := extractMsgTag(c.comment)
	if msg == "" {
		return nil
	}

	blockCompiler := c.newBlockCompiler(&llx.Code{
		Id:         "binding",
		Parameters: 1,
		Checksums: map[int32]string{
			// we must provide the first chunk, which is a reference to the caller
			// and which will always be number 1
			1: c.Result.Code.Checksums[c.Result.Code.ChunkIndex()-1],
		},
		Code: []*llx.Chunk{
			{
				Call:      llx.Chunk_PRIMITIVE,
				Primitive: &llx.Primitive{Type: string(typ)},
			},
		},
	}, &binding{Type: types.Type(typ), Ref: 1})

	assertionMsg, err := compileAssertionMsg(msg, &blockCompiler)
	if err != nil {
		return err
	}
	if assertionMsg != nil {
		if c.Result.Code.Assertions == nil {
			c.Result.Code.Assertions = map[int32]*llx.AssertionMessage{}
		}
		c.Result.Code.Assertions[assertionRef+2] = assertionMsg

		code := blockCompiler.Result.Code
		code.UpdateID()
		c.Result.Code.Functions = append(c.Result.Code.Functions, code)
		//return c.Result.Code.FunctionsIndex(), blockCompiler.standalone, nil

		fref := c.Result.Code.FunctionsIndex()
		c.Result.Code.AddChunk(&llx.Chunk{
			Call: llx.Chunk_FUNCTION,
			Id:   "${}",
			Function: &llx.Function{
				Type:    string(types.Block),
				Binding: failedRef,
				Args:    []*llx.Primitive{llx.FunctionPrimitive(fref)},
			},
		})

		// since it operators on top of a block, we have to add its
		// checksum as the first entry in the list. Once the block is received,
		// all of its child entries are processed for the final result
		blockRef := c.Result.Code.ChunkIndex()
		checksum := c.Result.Code.Checksums[blockRef]
		assertionMsg.Checksums = make([]string, len(assertionMsg.Datapoint)+1)
		assertionMsg.Checksums[0] = checksum
		c.Result.Code.Datapoints = append(c.Result.Code.Datapoints, blockRef)

		blocksums := blockCompiler.Result.Code.Checksums
		for i := range assertionMsg.Datapoint {
			sum, ok := blocksums[assertionMsg.Datapoint[i]]
			if !ok {
				return errors.New("cannot find checksum for datapoint in @msg tag")
			}

			assertionMsg.Checksums[i+1] = sum
		}
		assertionMsg.Datapoint = nil
		assertionMsg.DecodeBlock = true
	}

	return nil
}

func compileResourceAll(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
	// resource.$whereNot
	_, err := compileResourceWhere(c, typ, ref, "$whereNot", call)
	if err != nil {
		return types.Nil, err
	}

	listType, err := compileResourceDefault(c, typ, c.Result.Code.ChunkIndex(), "list", nil)
	if err != nil {
		return listType, err
	}
	listRef := c.Result.Code.ChunkIndex()

	if err := compileListAssertionMsg(c, listType, listRef, listRef); err != nil {
		return types.Nil, err
	}

	c.Result.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "$all",
		Function: &llx.Function{
			Type:    string(types.Bool),
			Binding: listRef,
		},
	})

	checksum := c.Result.Code.Checksums[c.Result.Code.ChunkIndex()]
	c.Result.Labels.Labels[checksum] = typ.ResourceName() + ".all()"

	return types.Bool, nil
}

func compileResourceAny(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
	// resource.where
	_, err := compileResourceWhere(c, typ, ref, "where", call)
	if err != nil {
		return types.Nil, err
	}
	whereRef := c.Result.Code.ChunkIndex()

	listType, err := compileResourceDefault(c, typ, whereRef, "list", nil)
	if err != nil {
		return listType, err
	}
	listRef := c.Result.Code.ChunkIndex()

	if err := compileListAssertionMsg(c, listType, whereRef-1, listRef); err != nil {
		return types.Nil, err
	}

	c.Result.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "$any",
		Function: &llx.Function{
			Type:    string(types.Bool),
			Binding: listRef,
		},
	})

	checksum := c.Result.Code.Checksums[c.Result.Code.ChunkIndex()]
	c.Result.Labels.Labels[checksum] = typ.ResourceName() + ".any()"

	return types.Bool, nil
}

func compileResourceOne(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
	// resource.where
	_, err := compileResourceWhere(c, typ, ref, "where", call)
	if err != nil {
		return types.Nil, err
	}

	listType, err := compileResourceDefault(c, typ, c.Result.Code.ChunkIndex(), "list", nil)
	if err != nil {
		return listType, err
	}
	listRef := c.Result.Code.ChunkIndex()

	if err := compileListAssertionMsg(c, listType, listRef, listRef); err != nil {
		return types.Nil, err
	}

	c.Result.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "$one",
		Function: &llx.Function{
			Type:    string(types.Bool),
			Binding: listRef,
		},
	})

	checksum := c.Result.Code.Checksums[c.Result.Code.ChunkIndex()]
	c.Result.Labels.Labels[checksum] = typ.ResourceName() + ".one()"

	return types.Bool, nil
}

func compileResourceNone(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
	// resource.where
	_, err := compileResourceWhere(c, typ, ref, "where", call)
	if err != nil {
		return types.Nil, err
	}

	listType, err := compileResourceDefault(c, typ, c.Result.Code.ChunkIndex(), "list", nil)
	if err != nil {
		return listType, err
	}
	listRef := c.Result.Code.ChunkIndex()

	if err := compileListAssertionMsg(c, listType, listRef, listRef); err != nil {
		return types.Nil, err
	}

	c.Result.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "$none",
		Function: &llx.Function{
			Type:    string(types.Bool),
			Binding: listRef,
		},
	})

	checksum := c.Result.Code.Checksums[c.Result.Code.ChunkIndex()]
	c.Result.Labels.Labels[checksum] = typ.ResourceName() + ".none()"

	return types.Bool, nil
}

func compileResourceLength(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
	if call != nil && len(call.Function) > 0 {
		return types.Nil, errors.New("function " + id + " does not take arguments")
	}

	_, err := listResource(c, typ)
	if err != nil {
		return types.Nil, errors.New("failed to compile " + id + ": " + err.Error())
	}

	resourceRef := c.Result.Code.ChunkIndex()

	t, err := compileResourceDefault(c, typ, ref, "list", nil)
	if err != nil {
		return t, err
	}
	listRef := c.Result.Code.ChunkIndex()

	c.Result.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   id,
		Function: &llx.Function{
			Type:    string(types.Int),
			Binding: resourceRef,
			Args: []*llx.Primitive{
				llx.RefPrimitive(listRef),
			},
		},
	})
	return typ, nil
}

func compileResourceParseDate(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
	if call == nil {
		return types.Nil, errors.New("missing arguments to parse date")
	}

	functionID := string(typ) + "." + id

	init := &lumi.Init{
		Args: []*lumi.TypedArg{
			{Name: "value", Type: string(types.String)},
			{Name: "format", Type: string(types.String)},
		},
	}
	args, err := c.unnamedArgs("parse."+id, init, call.Function)
	if err != nil {
		return types.Nil, err
	}

	rawArgs := make([]*llx.Primitive, len(call.Function))
	for i := range call.Function {
		rawArgs[i] = args[i*2+1]
	}

	if len(rawArgs) == 0 {
		return types.Nil, errors.New("missing arguments to parse date")
	}

	c.Result.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   functionID,
		Function: &llx.Function{
			Type:    string(types.Time),
			Binding: ref,
			Args:    rawArgs,
		},
	})
	return types.Time, nil
}
