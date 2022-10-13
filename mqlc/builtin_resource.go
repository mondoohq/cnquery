package mqlc

import (
	"strconv"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/mqlc/parser"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/types"
)

func compileResourceDefault(c *compiler, typ types.Type, ref uint64, id string, call *parser.Call) (types.Type, error) {
	name := typ.ResourceName()
	resource := c.Schema.Resources[name]
	if resource == nil {
		return types.Nil, errors.New("cannot find resource '" + name + "' when compiling field '" + id + "'")
	}

	// special case that we can optimize: the previous call was a resource
	// without any call arguments + the combined type is a resource itself
	// in that case save the outer call and go for the resource directly
	prevRef := c.tailRef()
	prev := c.Result.CodeV2.Chunk(prevRef)
	if prev.Call == llx.Chunk_FUNCTION && prev.Function == nil {
		name := prev.Id + "." + id
		resourceinfo, isResource := c.Schema.Resources[name]
		if isResource {
			c.popChunk()
			return c.addResource(name, resourceinfo, call)
		}
	}

	fieldinfo := resource.Fields[id]
	if fieldinfo == nil {
		addFieldSuggestions(publicFieldsInfo(c, resource), id, c.Result)
		return "", errors.New("cannot find field '" + id + "' in resource " + resource.Name)
	}

	c.addChunk(&llx.Chunk{
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

func listResource(c *compiler, typ types.Type) (*resources.ResourceInfo, error) {
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

func compileResourceWhere(c *compiler, typ types.Type, ref uint64, id string, call *parser.Call) (types.Type, error) {
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

	blockRef, blockDeps, _, err := c.blockExpressions([]*parser.Expression{arg.Value}, types.Array(types.Type(resource.ListType)))
	if err != nil {
		return types.Nil, err
	}
	if blockRef == 0 {
		return types.Nil, errors.New("called '" + id + "' clause without a function block")
	}

	resourceRef := c.tailRef()

	listType, err := compileResourceDefault(c, typ, ref, "list", nil)
	if err != nil {
		return listType, err
	}
	listRef := c.tailRef()

	args := []*llx.Primitive{
		llx.RefPrimitiveV2(listRef),
		llx.FunctionPrimitive(blockRef),
	}
	for _, v := range blockDeps {
		if c.isInMyBlock(v) {
			args = append(args, llx.RefPrimitiveV2(v))
		}
	}
	c.blockDeps = append(c.blockDeps, blockDeps...)

	c.addChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   id,
		Function: &llx.Function{
			Type:    string(types.Resource(resource.Name)),
			Binding: resourceRef,
			Args:    args,
		},
	})
	return typ, nil
}

func compileResourceMap(c *compiler, typ types.Type, ref uint64, id string, call *parser.Call) (types.Type, error) {
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

	blockRef, blockDeps, _, err := c.blockExpressions([]*parser.Expression{arg.Value}, types.Array(types.Type(resource.ListType)))
	if err != nil {
		return types.Nil, err
	}
	if blockRef == 0 {
		return types.Nil, errors.New("called '" + id + "' clause without a function block")
	}

	mappedType, err := c.blockType(blockRef)
	if err != nil {
		return types.Nil, errors.New("called '" + id + "' with a bad function block, types don't match: " + err.Error())
	}

	resourceRef := c.tailRef()

	listType, err := compileResourceDefault(c, typ, ref, "list", nil)
	if err != nil {
		return listType, err
	}
	listRef := c.tailRef()

	args := []*llx.Primitive{
		llx.RefPrimitiveV2(listRef),
		llx.FunctionPrimitive(blockRef),
	}
	for _, v := range blockDeps {
		if c.isInMyBlock(v) {
			args = append(args, llx.RefPrimitiveV2(v))
		}
	}
	c.blockDeps = append(c.blockDeps, blockDeps...)

	c.addChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   id,
		Function: &llx.Function{
			Type:    string(types.Array(mappedType)),
			Binding: resourceRef,
			Args:    args,
		},
	})

	return types.Array(mappedType), nil
}

func compileResourceContains(c *compiler, typ types.Type, ref uint64, id string, call *parser.Call) (types.Type, error) {
	// resource.where
	_, err := compileResourceWhere(c, typ, ref, "where", call)
	if err != nil {
		return types.Nil, err
	}
	resourceRef := c.tailRef()

	// .list
	t, err := compileResourceDefault(c, typ, resourceRef, "list", nil)
	if err != nil {
		return t, err
	}
	listRef := c.tailRef()

	// .length
	c.addChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "length",
		Function: &llx.Function{
			Type:    string(types.Int),
			Binding: listRef,
		},
	})

	// > 0
	c.addChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   string(">" + types.Int),
		Function: &llx.Function{
			Type:    string(types.Bool),
			Binding: c.tailRef(),
			Args: []*llx.Primitive{
				llx.IntPrimitive(0),
			},
		},
	})

	checksum := c.Result.CodeV2.Checksums[c.tailRef()]
	c.Result.Labels.Labels[checksum] = typ.ResourceName() + ".contains()"

	return types.Bool, nil
}

func compileResourceAll(c *compiler, typ types.Type, ref uint64, id string, call *parser.Call) (types.Type, error) {
	// resource.$whereNot
	_, err := compileResourceWhere(c, typ, ref, "$whereNot", call)
	if err != nil {
		return types.Nil, err
	}
	whereRef := c.tailRef()

	listType, err := compileResourceDefault(c, typ, c.tailRef(), "list", nil)
	if err != nil {
		return listType, err
	}
	listRef := c.tailRef()

	if err := compileListAssertionMsg(c, listType, whereRef-1, listRef, listRef); err != nil {
		return types.Nil, err
	}

	c.addChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "$all",
		Function: &llx.Function{
			Type:    string(types.Bool),
			Binding: listRef,
		},
	})

	checksum := c.Result.CodeV2.Checksums[c.tailRef()]
	c.Result.Labels.Labels[checksum] = typ.ResourceName() + ".all()"

	return types.Bool, nil
}

func compileResourceAny(c *compiler, typ types.Type, ref uint64, id string, call *parser.Call) (types.Type, error) {
	// resource.where
	_, err := compileResourceWhere(c, typ, ref, "where", call)
	if err != nil {
		return types.Nil, err
	}
	whereRef := c.tailRef()

	listType, err := compileResourceDefault(c, typ, whereRef, "list", nil)
	if err != nil {
		return listType, err
	}
	listRef := c.tailRef()

	if err := compileListAssertionMsg(c, listType, whereRef-1, whereRef-1, listRef); err != nil {
		return types.Nil, err
	}

	c.addChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "$any",
		Function: &llx.Function{
			Type:    string(types.Bool),
			Binding: listRef,
		},
	})

	checksum := c.Result.CodeV2.Checksums[c.tailRef()]
	c.Result.Labels.Labels[checksum] = typ.ResourceName() + ".any()"

	return types.Bool, nil
}

func compileResourceOne(c *compiler, typ types.Type, ref uint64, id string, call *parser.Call) (types.Type, error) {
	// resource.where
	_, err := compileResourceWhere(c, typ, ref, "where", call)
	if err != nil {
		return types.Nil, err
	}
	whereRef := c.tailRef()

	listType, err := compileResourceDefault(c, typ, c.tailRef(), "list", nil)
	if err != nil {
		return listType, err
	}
	listRef := c.tailRef()

	if err := compileListAssertionMsg(c, listType, whereRef-1, listRef, listRef); err != nil {
		return types.Nil, err
	}

	c.addChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "$one",
		Function: &llx.Function{
			Type:    string(types.Bool),
			Binding: listRef,
		},
	})

	checksum := c.Result.CodeV2.Checksums[c.tailRef()]
	c.Result.Labels.Labels[checksum] = typ.ResourceName() + ".one()"

	return types.Bool, nil
}

func compileResourceNone(c *compiler, typ types.Type, ref uint64, id string, call *parser.Call) (types.Type, error) {
	// resource.where
	_, err := compileResourceWhere(c, typ, ref, "where", call)
	if err != nil {
		return types.Nil, err
	}
	whereRef := c.tailRef()

	listType, err := compileResourceDefault(c, typ, c.tailRef(), "list", nil)
	if err != nil {
		return listType, err
	}
	listRef := c.tailRef()

	if err := compileListAssertionMsg(c, listType, whereRef-1, listRef, listRef); err != nil {
		return types.Nil, err
	}

	c.addChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "$none",
		Function: &llx.Function{
			Type:    string(types.Bool),
			Binding: listRef,
		},
	})

	checksum := c.Result.CodeV2.Checksums[c.tailRef()]
	c.Result.Labels.Labels[checksum] = typ.ResourceName() + ".none()"

	return types.Bool, nil
}

func compileResourceLength(c *compiler, typ types.Type, ref uint64, id string, call *parser.Call) (types.Type, error) {
	if call != nil && len(call.Function) > 0 {
		return types.Nil, errors.New("function " + id + " does not take arguments")
	}

	_, err := listResource(c, typ)
	if err != nil {
		return types.Nil, errors.New("failed to compile " + id + ": " + err.Error())
	}

	resourceRef := c.tailRef()

	t, err := compileResourceDefault(c, typ, ref, "list", nil)
	if err != nil {
		return t, err
	}
	listRef := c.tailRef()

	c.addChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   id,
		Function: &llx.Function{
			Type:    string(types.Int),
			Binding: resourceRef,
			Args: []*llx.Primitive{
				llx.RefPrimitiveV2(listRef),
			},
		},
	})
	return typ, nil
}

func compileResourceParseDate(c *compiler, typ types.Type, ref uint64, id string, call *parser.Call) (types.Type, error) {
	if call == nil {
		return types.Nil, errors.New("missing arguments to parse date")
	}

	functionID := string(typ) + "." + id

	init := &resources.Init{
		Args: []*resources.TypedArg{
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

	c.addChunk(&llx.Chunk{
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
