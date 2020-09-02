package leise

import (
	"errors"
	"strconv"

	"go.mondoo.io/mondoo/leise/parser"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/types"
)

func compileResourceDefault(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
	name := typ.Name()
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
		addFieldSuggestions(fieldsInfo(resource), id, c.Result)
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
func (f *FunctionSignature) Validate(args []*llx.Primitive) error {
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

		// TODO: find out the real type from these REF types
		if argT == req || req == types.Any {
			continue
		}

		return errors.New("incorrect argument " + strconv.Itoa(i) + ": expected " + req.Label() + " got " + argT.Label())
	}
	return nil
}

func listResource(c *compiler, typ types.Type) (*lumi.ResourceInfo, error) {
	name := typ.Name()
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

	functionRef, err := c.blockExpressions([]*parser.Expression{arg.Value}, types.Array(types.Type(resource.ListType)))
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
	c.Result.Labels.Labels[checksum] = typ.Name() + ".contains()"

	return types.Bool, nil
}

func compileResourceOne(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
	res, err := compileResourceContains(c, typ, ref, id, call)
	if err != nil {
		return res, err
	}

	// all we need to do is change the last operation on contains, as it tests for
	// where( .. ).list.length > 0
	// and we want
	// where( .. ).list.length == 1
	lastOp := c.Result.Code.LastChunk()
	lastOp.Id = string("==" + types.Int)
	lastOp.Function.Args[0] = llx.IntPrimitive(1)

	checksum := c.Result.Code.Checksums[c.Result.Code.ChunkIndex()]
	c.Result.Labels.Labels[checksum] = typ.Name() + ".one()"

	return res, err
}

func compileResourceAll(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
	// resource.where
	_, err := compileResourceWhere(c, typ, ref, "where", call)
	if err != nil {
		return types.Nil, err
	}
	resourceRef := c.Result.Code.ChunkIndex()

	// .length ==> allLen
	c.Result.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "length",
		Function: &llx.Function{
			Type:    string(types.Int),
			Binding: resourceRef - 1, // since the resource calls list right before processing the block
		},
	})
	allLengthRef := c.Result.Code.ChunkIndex()

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

	// == allLen
	c.Result.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   string("==" + types.Int),
		Function: &llx.Function{
			Type:    string(types.Bool),
			Binding: c.Result.Code.ChunkIndex(),
			Args: []*llx.Primitive{
				llx.RefPrimitive(allLengthRef),
			},
		},
	})

	checksum := c.Result.Code.Checksums[c.Result.Code.ChunkIndex()]
	c.Result.Labels.Labels[checksum] = typ.Name() + ".all()"

	return types.Bool, nil
}

func compileResourceAny(c *compiler, typ types.Type, ref int32, id string, call *parser.Call) (types.Type, error) {
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

	// == allLen
	c.Result.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   string("!=" + types.Int),
		Function: &llx.Function{
			Type:    string(types.Bool),
			Binding: c.Result.Code.ChunkIndex(),
			Args: []*llx.Primitive{
				llx.IntPrimitive(0),
			},
		},
	})

	checksum := c.Result.Code.Checksums[c.Result.Code.ChunkIndex()]
	c.Result.Labels.Labels[checksum] = typ.Name() + ".all()"

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
