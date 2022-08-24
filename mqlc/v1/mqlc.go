package v1

import (
	"errors"
	"regexp"
	"sort"
	"strconv"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/mqlc/parser"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/types"
)

type binding struct {
	Type types.Type
	Ref  int32
}

type variable struct {
	ref int32
	typ types.Type
}

type compiler struct {
	Schema  *resources.Schema
	Result  *llx.CodeBundle
	Binding *binding
	vars    map[string]variable
	parent  *compiler
	props   map[string]*llx.Primitive
	comment string

	// a standalone code is one that doesn't call any of its bindings
	// examples:
	//   file(xyz).content          is standalone
	//   file(xyz).content == _     is not
	standalone bool

	// helps chaining of builtin calls like `if (..) else if (..) else ..`
	prevID string
}

func (c *compiler) newBlockCompiler(code *llx.CodeV1, binding *binding) compiler {
	return compiler{
		Schema: c.Schema,
		Result: &llx.CodeBundle{
			DeprecatedV5Code: code,
			Labels:           c.Result.Labels,
			Props:            c.Result.Props,
		},
		Binding:    binding,
		vars:       map[string]variable{},
		parent:     c,
		props:      c.props,
		standalone: true,
	}
}

func addResourceSuggestions(resourceInfos map[string]*resources.ResourceInfo, name string, res *llx.CodeBundle) {
	names := make([]string, len(resourceInfos))
	i := 0
	for key := range resourceInfos {
		names[i] = key
		i++
	}

	suggestedNames := fuzzy.Find(name, names)
	res.Suggestions = make([]*llx.Documentation, len(suggestedNames))
	var info *resources.ResourceInfo
	for i := range suggestedNames {
		field := suggestedNames[i]
		info = resourceInfos[field]
		if info != nil {
			res.Suggestions[i] = &llx.Documentation{
				Field: field,
				Title: info.Title,
				Desc:  info.Desc,
			}
		} else {
			res.Suggestions[i] = &llx.Documentation{
				Field: field,
			}
		}
	}

	sort.SliceStable(res.Suggestions, func(i, j int) bool { return res.Suggestions[i].Field < res.Suggestions[j].Field })
}

func addFieldSuggestions(fields map[string]llx.Documentation, fieldName string, res *llx.CodeBundle) {
	names := make([]string, len(fields))
	i := 0
	for key := range fields {
		names[i] = key
		i++
	}

	suggestedNames := fuzzy.Find(fieldName, names)
	res.Suggestions = make([]*llx.Documentation, len(suggestedNames))
	for i := range suggestedNames {
		info := fields[suggestedNames[i]]
		res.Suggestions[i] = &info
	}

	sort.SliceStable(res.Suggestions, func(i, j int) bool { return res.Suggestions[i].Field < res.Suggestions[j].Field })
}

// func (c *compiler) addAccessor(call *Call, typ types.Type) types.Type {
// 	binding := c.Result.Code.ChunkIndex()
// 	ownerType := c.Result.Code.LastChunk().Type(c.Result.Code)

// 	if call.Accessors != nil {
// 		arg, err := c.compileValue(call.Accessors)
// 		if err != nil {
// 			panic(err.Error())
// 		}

// 		c.Result.Code.AddChunk(&llx.Chunk{
// 			Call: llx.Chunk_FUNCTION,
// 			Id:   "[]",
// 			Function: &llx.Function{
// 				Type:    string(ownerType.Child()),
// 				Binding: binding,
// 				Args:    []*llx.Primitive{arg},
// 			},
// 		})

// 		return ownerType.Child()
// 	}

// 	if call.Params != nil {
// 		panic("We have not yet implemented adding more unnamed function calls")
// 	}

// 	panic("Tried to add accessor calls for a call that has no accessors or params")
// }

// func (c *compiler) addAccessorCalls(calls []*Call, typ types.Type) types.Type {
// 	if calls == nil || len(calls) == 0 {
// 		return typ
// 	}
// 	for i := range calls {
// 		typ = c.addAccessorCall(calls[i], typ)
// 	}
// 	return typ
// }

// compileBlock on a context
func (c *compiler) compileBlock(expressions []*parser.Expression, typ types.Type, bindingRef int32) (types.Type, error) {
	// For resource, users may indicate to query all fields. It also works for list of resources.
	// This is a special case which is handled here:
	if len(expressions) == 1 && (typ.IsResource() || (typ.IsArray() && typ.Child().IsResource())) {
		x := expressions[0]
		if x.Operand != nil && x.Operand.Value != nil && x.Operand.Value.Ident != nil && *(x.Operand.Value.Ident) == "*" {
			var fields map[string]llx.Documentation
			if typ.IsArray() {
				fields = availableGlobFields(c, typ.Child())
			} else {
				fields = availableGlobFields(c, typ)
			}

			fieldNames := make([]string, len(fields))
			var i int
			for k := range fields {
				fieldNames[i] = k
				i++
			}
			sort.Strings(fieldNames)

			expressions = []*parser.Expression{}
			for _, v := range fieldNames {
				name := v
				expressions = append(expressions, &parser.Expression{
					Operand: &parser.Operand{
						Value: &parser.Value{Ident: &name},
					},
				})
			}
		}
	}

	fref, _, err := c.blockExpressions(expressions, typ)
	if err != nil {
		return types.Nil, err
	}
	if fref == 0 {
		return typ, nil
	}

	var resultType types.Type
	if typ.IsArray() {
		resultType = types.Array(types.Block)
	} else {
		resultType = types.Block
	}

	chunk := llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "{}",
		Function: &llx.Function{
			Type:                string(resultType),
			DeprecatedV5Binding: bindingRef,
			Args:                []*llx.Primitive{llx.FunctionPrimitiveV1(fref)},
		},
	}
	c.Result.DeprecatedV5Code.AddChunk(&chunk)

	return resultType, nil
}

func (c *compiler) compileIfBlock(expressions []*parser.Expression, chunk *llx.Chunk) (types.Type, error) {
	// if `else { .. }` is called, we reset the prevID to indicate there is no
	// more chaining happening
	if c.prevID == "else" {
		c.prevID = ""
	}

	code := c.Result.DeprecatedV5Code

	blockCompiler := c.newBlockCompiler(&llx.CodeV1{
		Id:         chunk.Id,
		Parameters: 0,
		Checksums:  map[int32]string{},
		Code:       []*llx.Chunk{},
	}, nil)

	err := blockCompiler.compileExpressions(expressions)
	c.Result.Suggestions = append(c.Result.Suggestions, blockCompiler.Result.Suggestions...)
	if err != nil {
		return types.Nil, err
	}

	block := blockCompiler.Result.DeprecatedV5Code

	// insert a body if we are in standalone mode to return a value
	if len(block.Code) == 0 && c.standalone {
		block.AddChunk(&llx.Chunk{
			Call:      llx.Chunk_PRIMITIVE,
			Primitive: llx.NilPrimitive,
		})
		block.AddChunk(&llx.Chunk{
			Call: llx.Chunk_FUNCTION,
			Id:   "return",
			Function: &llx.Function{
				Type: string(types.Nil),
				Args: []*llx.Primitive{llx.RefPrimitiveV1(1)},
			},
		})
		block.SingleValue = true
		block.Entrypoints = []int32{2}
	}

	block.UpdateID()
	code.Functions = append(code.Functions, block)

	// the last chunk in this case is the `if` function call
	chunk.Function.Args = append(chunk.Function.Args,
		llx.FunctionPrimitiveV1(code.FunctionsIndex()),
	)

	if len(block.Code) != 0 {
		var typeToEnforce types.Type
		if block.SingleValue || code.SingleValue {
			last := block.Code[block.ChunkIndex()-1]
			typeToEnforce = last.Type()
		} else {
			typeToEnforce = types.Block
		}

		t, ok := types.Enforce(types.Type(chunk.Function.Type), typeToEnforce)
		if !ok {
			return types.Nil, errors.New("mismatched return type for child block of if-function; make sure all return types are the same")
		}
		chunk.Function.Type = string(t)
	}

	code.RefreshChunkChecksum(chunk)

	// we set this to true, so that we can decide how to handle all following expressions
	if block.SingleValue {
		code.SingleValue = true
	}

	return types.Nil, nil
}

func (c *compiler) compileSwitchCase(expression *parser.Expression, bind *binding, chunk *llx.Chunk) error {
	// for the default case, we get a nil expression
	if expression == nil {
		chunk.Function.Args = append(chunk.Function.Args, llx.BoolPrimitive(true))
		return nil
	}

	prevBind := c.Binding
	c.Binding = bind
	defer func() {
		c.Binding = prevBind
	}()

	argValue, err := c.compileExpression(expression)
	if err != nil {
		return err
	}
	chunk.Function.Args = append(chunk.Function.Args, argValue)
	return nil
}

func (c *compiler) compileSwitchBlock(expressions []*parser.Expression, chunk *llx.Chunk) (types.Type, error) {
	// determine if there is a binding
	// i.e. something inside of those `switch( ?? )` calls
	var bind *binding
	arg := chunk.Function.Args[0]

	// we have to pop the switch chunk from the compiler stack, because it needs
	// to be the last item on the stack. otherwise the last reference (top of stack)
	// will not be pointing to it and an additional entrypoint will be generated

	code := c.Result.DeprecatedV5Code

	last := len(code.Code) - 1
	if code.Code[last] != chunk {
		return types.Nil, errors.New("failed to compile switch statement, it wasn't on the top of the compile stack")
	}
	code.Code = code.Code[:last]
	checksum := code.Checksums[int32(last+1)]
	code.Checksums[int32(last+1)] = ""
	defer func() {
		code.Code = append(code.Code, chunk)
		code.Checksums[code.ChunkIndex()] = checksum
	}()

	if types.Type(arg.Type) != types.Unset {
		if types.Type(arg.Type) == types.Ref {
			val, ok := arg.RefV1()
			if !ok {
				return types.Nil, errors.New("could not resolve references of switch argument")
			}
			bind = &binding{
				Type: types.Type(arg.Type),
				Ref:  val,
			}
		} else {
			code.AddChunk(&llx.Chunk{
				Call:      llx.Chunk_PRIMITIVE,
				Primitive: arg,
			})
			ref := code.ChunkIndex()
			bind = &binding{
				Type: types.Type(arg.Type),
				Ref:  ref,
			}
		}
	}

	for i := 0; i < len(expressions); i += 2 {
		err := c.compileSwitchCase(expressions[i], bind, chunk)
		if err != nil {
			return types.Nil, err
		}

		// compile the block of this case/default
		if i+1 >= len(expressions) {
			return types.Nil, errors.New("missing block expression in calling `case`/`default` statement")
		}

		blockExp := expressions[i+1]
		if *blockExp.Operand.Value.Ident != "{}" {
			return types.Nil, errors.New("expected block inside case/default statement")
		}

		expressions := blockExp.Operand.Block

		var blockCompiler compiler
		if bind != nil {
			blockCompiler = c.newBlockCompiler(&llx.CodeV1{
				Id:         chunk.Id,
				Parameters: 1,
				Checksums: map[int32]string{
					// we must provide the first chunk, which is a reference to the caller
					// and which will always be number 1
					1: code.Checksums[code.ChunkIndex()],
				},
				Code: []*llx.Chunk{{
					Call:      llx.Chunk_PRIMITIVE,
					Primitive: &llx.Primitive{Type: string(bind.Type)},
				}},
				SingleValue: true,
			}, bind)
		} else {
			blockCompiler = c.newBlockCompiler(&llx.CodeV1{
				Id:          chunk.Id,
				Parameters:  0,
				Checksums:   map[int32]string{},
				Code:        []*llx.Chunk{},
				SingleValue: true,
			}, nil)
		}

		err = blockCompiler.compileExpressions(expressions)
		c.Result.Suggestions = append(c.Result.Suggestions, blockCompiler.Result.Suggestions...)
		if err != nil {
			return types.Nil, err
		}

		block := blockCompiler.Result.DeprecatedV5Code
		block.UpdateID()
		code.Functions = append(code.Functions, block)
		chunk.Function.Args = append(chunk.Function.Args, llx.FunctionPrimitiveV1(code.FunctionsIndex()))
	}

	code.RefreshChunkChecksum(chunk)

	return types.Nil, nil
}

func (c *compiler) compileUnboundBlock(expressions []*parser.Expression, chunk *llx.Chunk) (types.Type, error) {
	switch chunk.Id {
	case "if":
		return c.compileIfBlock(expressions, chunk)
	case "switch":
		return c.compileSwitchBlock(expressions, chunk)
	default:
		return types.Nil, errors.New("don't know how to compile unbound block on call `" + chunk.Id + "`")
	}
}

// evaluates the given expressions on a non-array resource
// and creates a function, whose reference is returned
func (c *compiler) blockOnResource(expressions []*parser.Expression, typ types.Type) (int32, bool, error) {
	code := c.Result.DeprecatedV5Code

	blockCompiler := c.newBlockCompiler(&llx.CodeV1{
		Id:         "binding",
		Parameters: 1,
		Checksums: map[int32]string{
			// we must provide the first chunk, which is a reference to the caller
			// and which will always be number 1
			1: code.Checksums[code.ChunkIndex()],
		},
		Code: []*llx.Chunk{{
			Call:      llx.Chunk_PRIMITIVE,
			Primitive: &llx.Primitive{Type: string(typ)},
		}},
	}, &binding{Type: typ, Ref: 1})

	err := blockCompiler.compileExpressions(expressions)
	c.Result.Suggestions = append(c.Result.Suggestions, blockCompiler.Result.Suggestions...)
	if err != nil {
		return 0, false, err
	}

	block := blockCompiler.Result.DeprecatedV5Code
	block.UpdateID()
	code.Functions = append(code.Functions, block)
	return code.FunctionsIndex(), blockCompiler.standalone, nil
}

// blockExpressions evaluates the given expressions as if called by a block and
// returns the compiled function reference
func (c *compiler) blockExpressions(expressions []*parser.Expression, typ types.Type) (int32, bool, error) {
	if len(expressions) == 0 {
		return 0, false, nil
	}

	if typ.IsArray() {
		return c.blockOnResource(expressions, typ.Child())
	}

	return c.blockOnResource(expressions, typ)
}

// returns the type of the given funciton block references
// error if the block has multiple entrypoints
func (c *compiler) functionBlockType(ref int32) (types.Type, error) {
	functions := c.Result.DeprecatedV5Code.Functions
	if len(functions) < int(ref) {
		return types.Nil, errors.New("canot find function block with ref " + strconv.Itoa(int(ref)))
	}

	f := functions[ref-1]
	if len(f.Entrypoints) != 1 {
		return types.Nil, errors.New("function block should only return 1 value (got: " + strconv.Itoa(len(f.Entrypoints)) + ")")
	}

	ep := f.Entrypoints[0]
	chunk := f.Code[ep-1]
	return chunk.Type(), nil
}

func (c *compiler) dereferenceType(val *llx.Primitive) (types.Type, error) {
	valType := types.Type(val.Type)
	if types.Type(val.Type) != types.Ref {
		return valType, nil
	}

	ref, ok := val.RefV1()
	if !ok {
		return types.Nil, errors.New("found a reference type that doesn't return a reference value")
	}

	code := c.Result.DeprecatedV5Code
	chunk := code.Code[ref-1]
	if chunk.Primitive == val {
		return types.Nil, errors.New("recursive reference connections detected")
	}

	if chunk.Primitive != nil {
		return c.dereferenceType(chunk.Primitive)
	}

	valType = chunk.Type()
	return valType, nil
}

func (c *compiler) unnamedArgs(callerLabel string, init *resources.Init, args []*parser.Arg) ([]*llx.Primitive, error) {
	if len(args) > len(init.Args) {
		return nil, errors.New("Called " + callerLabel +
			" with too many arguments (expected " + strconv.Itoa(len(init.Args)) +
			" but got " + strconv.Itoa(len(args)) + ")")
	}

	// add all calls to the chunk stack
	// collect all their types and call references
	res := make([]*llx.Primitive, len(args)*2)

	for idx := range args {
		arg := args[idx]

		v, err := c.compileExpression(arg.Value)
		if err != nil {
			return nil, errors.New("addResourceCall error: " + err.Error())
		}

		vType := types.Type(v.Type)
		if vType == types.Ref {
			vType, err = c.dereferenceType(v)
			if err != nil {
				return nil, err
			}
		}

		expected := init.Args[idx]
		expectedType := types.Type(expected.Type)
		if vType != expectedType {
			// TODO: We are looking for dict types to see if we can type-cast them
			// This needs massive improvements to dynamically cast them in LLX.
			// For a full description see: https://gitlab.com/mondoolabs/mondoo/-/issues/241
			// This is ONLY a temporary workaround which works in a few cases:
			if vType == types.Dict && expectedType == types.String {
				// we are good, LLX will handle it
			} else {
				return nil, errors.New("Incorrect type on argument " + strconv.Itoa(idx) +
					" in " + callerLabel + ": expected " + expectedType.Label() +
					", got: " + vType.Label())
			}
		}

		res[idx*2] = llx.StringPrimitive(expected.Name)
		res[idx*2+1] = v
	}

	return res, nil
}

func (c *compiler) unnamedResourceArgs(resource *resources.ResourceInfo, args []*parser.Arg) ([]*llx.Primitive, error) {
	if resource.Init == nil {
		return nil, errors.New("cannot find init call for resource " + resource.Id)
	}

	return c.unnamedArgs("resource "+resource.Name, resource.Init, args)
}

// resourceArgs turns the list of arguments for the resource into a list of
// primitives that are used as arguments to initialize that resource
// only works if len(args) > 0 !!
// only works if args are either ALL named or not named !!
func (c *compiler) resourceArgs(resource *resources.ResourceInfo, args []*parser.Arg) ([]*llx.Primitive, error) {
	if args[0].Name == "" {
		return c.unnamedResourceArgs(resource, args)
	}

	res := make([]*llx.Primitive, len(args)*2)
	for idx := range args {
		arg := args[idx]
		field, ok := resource.Fields[arg.Name]
		if !ok {
			return nil, errors.New("resource " + resource.Name + " does not have a field named " + arg.Name)
		}

		v, err := c.compileExpression(arg.Value)
		if err != nil {
			return nil, errors.New("resourceArgs error: " + err.Error())
		}

		vt, err := c.dereferenceType(v)
		if err != nil {
			return nil, err
		}

		ft := types.Type(field.Type)
		if vt != ft {
			return nil, errors.New("Wrong type for field " + arg.Name + " in resource " + resource.Name + ": expected " + ft.Label() + ", got " + vt.Label())
		}

		res[idx*2] = llx.StringPrimitive(arg.Name)
		res[idx*2+1] = v
	}

	return res, nil
}

func (c *compiler) compileBuiltinFunction(h *compileHandler, id string, binding *binding, call *parser.Call) (types.Type, error) {
	typ := binding.Type

	if h.compile != nil {
		return h.compile(c, typ, binding.Ref, id, call)
	}

	var args []*llx.Primitive

	if call != nil {
		for idx := range call.Function {
			arg := call.Function[idx]
			x, err := c.compileExpression(arg.Value)
			if err != nil {
				return types.Nil, err
			}
			if x != nil {
				args = append(args, x)
			}
		}
	}

	if err := h.signature.Validate(args, c); err != nil {
		return types.Nil, err
	}

	resType := h.typ(typ)
	c.Result.DeprecatedV5Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   id,
		Function: &llx.Function{
			Type:                string(resType),
			DeprecatedV5Binding: binding.Ref,
			Args:                args,
		},
	})
	return resType, nil
}

func filterTrailingNullArgs(call *parser.Call) *parser.Call {
	if call == nil {
		return call
	}

	res := parser.Call{
		Comments: call.Comments,
		Ident:    call.Ident,
		Function: call.Function,
		Accessor: call.Accessor,
	}

	args := call.Function
	if len(args) == 0 {
		return &res
	}

	lastIdx := len(args) - 1
	x := args[lastIdx]
	if x.Value.IsEmpty() {
		res.Function = args[0:lastIdx]
	}

	return &res
}

func filterEmptyExpressions(expressions []*parser.Expression) []*parser.Expression {
	res := []*parser.Expression{}
	for i := range expressions {
		exp := expressions[i]
		if exp.IsEmpty() {
			continue
		}
		res = append(res, exp)
	}

	return res
}

// compile a bound identifier to its binding
// example: user { name } , where name is compiled bound to the user
// it will return false if it cannot bind the identifier
func (c *compiler) compileBoundIdentifier(id string, binding *binding, call *parser.Call) (bool, types.Type, error) {
	typ := binding.Type
	if typ.IsResource() {
		resource, ok := c.Schema.Resources[typ.ResourceName()]
		if !ok {
			return true, types.Nil, errors.New("cannot find resource that is called by '" + id + "' of type " + typ.Label())
		}

		fieldinfo, ok := resource.Fields[id]
		if ok {
			if call != nil && len(call.Function) > 0 {
				return true, types.Nil, errors.New("cannot call resource field with arguments yet")
			}

			c.Result.MinMondooVersion = getMinMondooVersion(c.Result.MinMondooVersion, typ.ResourceName(), id)

			c.Result.DeprecatedV5Code.AddChunk(&llx.Chunk{
				Call: llx.Chunk_FUNCTION,
				Id:   id,
				Function: &llx.Function{
					Type:                fieldinfo.Type,
					DeprecatedV5Binding: binding.Ref,
				},
			})
			return true, types.Type(fieldinfo.Type), nil
		}
	}

	h, _ := builtinFunction(typ, id)
	if h != nil {
		call = filterTrailingNullArgs(call)
		typ, err := c.compileBuiltinFunction(h, id, binding, call)
		return true, typ, err
	}

	return false, types.Nil, nil
}

// compile a resource from an identifier, trying to find the longest matching resource
// and execute all call functions if there are any
func (c *compiler) compileResource(id string, calls []*parser.Call) (bool, []*parser.Call, types.Type, error) {
	resource, ok := c.Schema.Resources[id]
	if !ok {
		return false, nil, types.Nil, nil
	}

	for len(calls) > 0 && calls[0].Ident != nil {
		nuID := id + "." + (*calls[0].Ident)
		nuResource, ok := c.Schema.Resources[nuID]
		if !ok {
			break
		}
		resource, id = nuResource, nuID
		calls = calls[1:]
	}

	var call *parser.Call
	if len(calls) > 0 && calls[0].Function != nil {
		call = calls[0]
		calls = calls[1:]
	}

	c.Result.MinMondooVersion = getMinMondooVersion(c.Result.MinMondooVersion, id, "")

	typ, err := c.addResource(id, resource, call)
	return true, calls, typ, err
}

func (c *compiler) addResource(id string, resource *resources.ResourceInfo, call *parser.Call) (types.Type, error) {
	var function *llx.Function
	var err error
	typ := types.Resource(id)

	if call != nil && len(call.Function) > 0 {
		function = &llx.Function{Type: string(typ)}
		function.Args, err = c.resourceArgs(resource, call.Function)
		if err != nil {
			return types.Nil, err
		}
	}

	c.Result.DeprecatedV5Code.AddChunk(&llx.Chunk{
		Call:     llx.Chunk_FUNCTION,
		Id:       id,
		Function: function,
	})
	return typ, nil
}

// compileIdentifier within a context of a binding
// 1. global f(): 			expect, ...
// 2. global resource: 	sshd, sshd.config
// 3. bound field: 			user { name }
// x. called field: 		user.name <= not in this scope
func (c *compiler) compileIdentifier(id string, callBinding *binding, calls []*parser.Call) ([]*parser.Call, types.Type, error) {
	var call *parser.Call
	restCalls := calls
	if len(calls) > 0 && calls[0].Function != nil {
		call = calls[0]
		restCalls = calls[1:]
	}

	var typ types.Type
	var err error
	var found bool
	if callBinding != nil {
		// special handling for the `self` operator
		if id == "_" {
			c.standalone = false

			if len(restCalls) == 0 {
				return restCalls, callBinding.Type, nil
			}

			nextCall := restCalls[0]

			if nextCall.Ident != nil {
				calls = restCalls[1:]
				call = nil
				if len(calls) > 0 && calls[0].Function != nil {
					call = calls[0]
				}

				found, typ, err = c.compileBoundIdentifier(*nextCall.Ident, callBinding, call)
				if found {
					if call != nil {
						return restCalls[2:], typ, err
					}
					return restCalls[1:], typ, err
				}
				return nil, types.Nil, errors.New("could not find call _." + (*nextCall.Ident))
			}

			if nextCall.Accessor != nil {
				// turn accessor into a regular function and call that
				fCall := &parser.Call{Function: []*parser.Arg{{Value: nextCall.Accessor}}}
				// accessors are aways builtin functions
				h, _ := builtinFunction(callBinding.Type.Underlying(), "[]")
				if h == nil {
					return nil, types.Nil, errors.New("cannot find '[]' function on type " + callBinding.Type.Label())
				}
				typ, err = c.compileBuiltinFunction(h, "[]", &binding{Type: callBinding.Type, Ref: callBinding.Ref}, fCall)
				if err != nil {
					return nil, types.Nil, err
				}

				if call != nil && len(calls) > 0 {
					calls = calls[1:]
				}

				return restCalls[1:], typ, nil
			}

			return nil, types.Nil, errors.New("not sure how to handle implicit calls around `_`")
		}

		found, typ, err = c.compileBoundIdentifier(id, callBinding, call)
		if found {
			c.standalone = false
			return restCalls, typ, err
		}
	} // end bound functions

	if id == "props" {
		return c.compileProps(call, restCalls, c.Result)
	}

	f := operatorsCompilers[id]
	if f != nil {
		typ, err := f(c, id, call, c.Result)
		return restCalls, typ, err
	}

	variable, ok := c.vars[id]
	if ok {
		c.Result.DeprecatedV5Code.AddChunk(&llx.Chunk{
			Call:      llx.Chunk_PRIMITIVE,
			Primitive: llx.RefPrimitiveV1(variable.ref),
		})
		return restCalls, variable.typ, nil
	}

	found, restCalls, typ, err = c.compileResource(id, calls)
	if found {
		return restCalls, typ, err
	}

	// suggestions
	if callBinding == nil {
		addResourceSuggestions(c.Schema.Resources, id, c.Result)
		return nil, types.Nil, errors.New("cannot find resource for identifier '" + id + "'")
	}
	addFieldSuggestions(availableFields(c, callBinding.Type), id, c.Result)
	return nil, types.Nil, errors.New("cannot find field or resource '" + id + "' in block for type '" + c.Binding.Type.Label() + "'")
}

// compileProps handles built-in properties for this code
// we will use any properties defined at the compiler-level as type-indicators
func (c *compiler) compileProps(call *parser.Call, calls []*parser.Call, res *llx.CodeBundle) ([]*parser.Call, types.Type, error) {
	if call != nil && len(call.Function) != 0 {
		return nil, types.Nil, errors.New("'props' is not a function")
	}

	if len(calls) == 0 {
		return nil, types.Nil, errors.New("called 'props' without a property, please provide the name you are trying to access")
	}

	nextCall := calls[0]
	restCalls := calls[1:]

	if nextCall.Ident == nil {
		return nil, types.Nil, errors.New("please call 'props' with the name of the property you are trying to access")
	}

	name := *nextCall.Ident
	prim, ok := c.props[name]
	if !ok {
		keys := make(map[string]llx.Documentation, len(c.props))
		for key, prim := range c.props {
			keys[key] = llx.Documentation{
				Field: key,
				Title: key + " (" + types.Type(prim.Type).Label() + ")",
			}
		}

		addFieldSuggestions(keys, name, res)

		return nil, types.Nil, errors.New("cannot find property '" + name + "', please define it first")
	}

	c.Result.DeprecatedV5Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_PROPERTY,
		Id:   name,
		Primitive: &llx.Primitive{
			Type: prim.Type,
		},
	})

	res.Props[name] = string(prim.Type)

	return restCalls, types.Type(prim.Type), nil
}

// compileValue takes an AST value and compiles it
func (c *compiler) compileValue(val *parser.Value) (*llx.Primitive, error) {
	if val.Bool != nil {
		return llx.BoolPrimitive(bool(*val.Bool)), nil
	}

	if val.Int != nil {
		return llx.IntPrimitive(int64(*val.Int)), nil
	}

	if val.Float != nil {
		return llx.FloatPrimitive(float64(*val.Float)), nil
	}

	if val.String != nil {
		return llx.StringPrimitive(*val.String), nil
	}

	if val.Regex != nil {
		re := string(*val.Regex)
		_, err := regexp.Compile(re)
		if err != nil {
			return nil, errors.New("failed to compile regular expression '" + re + "': " + err.Error())
		}
		return llx.RegexPrimitive(re), nil
	}

	if val.Array != nil {
		arr := make([]*llx.Primitive, len(val.Array))
		var err error
		for i := range val.Array {
			e := val.Array[i]
			arr[i], err = c.compileExpression(e)
			if err != nil {
				return nil, err
			}
		}

		return &llx.Primitive{
			Type:  string(llx.ArrayTypeV1(arr, c.Result.DeprecatedV5Code)),
			Array: arr,
		}, nil
	}

	if val.Map != nil {
		mapRes := make(map[string]*llx.Primitive, len(val.Map))
		var resType types.Type

		for k, v := range val.Map {
			vv, err := c.compileExpression(v)
			if err != nil {
				return nil, err
			}
			if types.Type(vv.Type) != resType {
				if resType == "" {
					resType = types.Type(vv.Type)
				} else if resType != types.Any {
					resType = types.Any
				}
			}
			mapRes[k] = vv
		}

		if resType == "" {
			resType = types.Unset
		}

		return &llx.Primitive{
			Type: string(types.Map(types.String, resType)),
			Map:  mapRes,
		}, nil
	}

	return llx.NilPrimitive, nil
}

func (c *compiler) compileOperand(operand *parser.Operand) (*llx.Primitive, error) {
	var err error
	var res *llx.Primitive
	var typ types.Type
	var ref int32

	calls := operand.Calls
	c.comment = operand.Comments
	code := c.Result.DeprecatedV5Code

	// value:        bool | string | regex | number | array | map | ident
	// so all simple values are compiled into primitives and identifiers
	// into function calls
	if operand.Value.Ident == nil {
		res, err = c.compileValue(operand.Value)
		if err != nil {
			return nil, err
		}
		typ = types.Type(res.Type)

		if len(calls) > 0 {
			code.AddChunk(&llx.Chunk{
				Call: llx.Chunk_PRIMITIVE,
				// no ID for standalone
				Primitive: res,
			})
			ref = code.ChunkIndex()
			res = llx.RefPrimitiveV1(ref)
		}
	} else {
		id := *operand.Value.Ident
		orgcalls := calls
		calls, typ, err = c.compileIdentifier(id, c.Binding, calls)
		if err != nil {
			return nil, err
		}

		ref = code.ChunkIndex()
		if id == "_" && len(orgcalls) == 0 {
			ref = c.Binding.Ref
		}

		res = llx.RefPrimitiveV1(ref)
	}

	// operand:      value [ call | accessor | '.' ident ]+ [ block ]
	// dealing with all call types
	for len(calls) > 0 {
		call := calls[0]
		if call.Function != nil {
			return nil, errors.New("don't know how to compile chained functions just yet")
		}

		if call.Comments != "" {
			c.comment = call.Comments
		}

		if call.Accessor != nil {
			// turn accessor into a regular function and call that
			fCall := &parser.Call{Function: []*parser.Arg{{Value: call.Accessor}}}
			// accessors are aways builtin functions
			h, _ := builtinFunction(typ.Underlying(), "[]")
			if h == nil {
				return nil, errors.New("cannot find '[]' function on type " + typ.Label())
			}
			typ, err = c.compileBuiltinFunction(h, "[]", &binding{Type: typ, Ref: ref}, fCall)
			if err != nil {
				return nil, err
			}

			if call != nil && len(calls) > 0 {
				calls = calls[1:]
			}
			ref = code.ChunkIndex()
			res = llx.RefPrimitiveV1(ref)
			continue
		}

		if call.Ident != nil {
			var found bool
			var resType types.Type
			id := *call.Ident

			if id == "." {
				// We get this from the parser if the user called the dot-accessor
				// but didn't provide any values at all. It equates a not found and
				// we can now just suggest all fields
				addFieldSuggestions(availableFields(c, typ), "", c.Result)
				return nil, errors.New("missing field name in accessing " + typ.Label())
			}

			calls = calls[1:]
			call = nil
			if len(calls) > 0 && calls[0].Function != nil {
				call = calls[0]
			}

			found, resType, err = c.compileBoundIdentifier(id, &binding{Type: typ, Ref: ref}, call)
			if err != nil {
				return nil, err
			}
			if !found {
				addFieldSuggestions(availableFields(c, typ), id, c.Result)
				return nil, errors.New("cannot find field '" + id + "' in " + typ.Label())
			}

			typ = resType
			if call != nil && len(calls) > 0 {
				calls = calls[1:]
			}
			ref = code.ChunkIndex()
			res = llx.RefPrimitiveV1(ref)

			continue
		}

		return nil, errors.New("processed a call without any data")
	}

	if operand.Block != nil {
		// for starters, we need the primitive to exist on the stack,
		// so add it if it's missing
		if x := code.ChunkIndex(); x == 0 {
			val, err := c.compileValue(operand.Value)
			if err != nil {
				return nil, err
			}
			code.AddChunk(&llx.Chunk{
				Call: llx.Chunk_PRIMITIVE,
				// no ID for standalone
				Primitive: val,
			})
			ref = code.ChunkIndex()
		}

		if typ == types.Nil {
			_, err = c.compileUnboundBlock(operand.Block, code.LastChunk())
		} else {
			_, err = c.compileBlock(operand.Block, typ, ref)
		}
		if err != nil {
			return nil, err
		}
		ref = code.ChunkIndex()
		res = llx.RefPrimitiveV1(ref)
	}

	return res, nil
}

func (c *compiler) compileExpression(expression *parser.Expression) (*llx.Primitive, error) {
	if len(expression.Operations) > 0 {
		panic("ran into an expression that wasn't pre-compiled. It has more than 1 value attached to it")
	}
	return c.compileOperand(expression.Operand)
}

func (c *compiler) compileAndAddExpression(expression *parser.Expression) (int32, error) {
	valc, err := c.compileExpression(expression)
	if err != nil {
		return 0, err
	}

	if types.Type(valc.Type) == types.Ref {
		ref, _ := valc.RefV1()
		return ref, nil
		// nothing to do, the last call was added to the compiled chain
	}

	code := c.Result.DeprecatedV5Code
	code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_PRIMITIVE,
		// no id for standalone values
		Primitive: valc,
	})

	return code.ChunkIndex(), nil
}

func (c *compiler) compileExpressions(expressions []*parser.Expression) error {
	var err error

	// we may have comment-only expressions
	expressions = filterEmptyExpressions(expressions)

	for idx := range expressions {
		if err = expressions[idx].ProcessOperators(); err != nil {
			return err
		}
	}

	var ident string
	var prev string
	for idx := range expressions {
		expression := expressions[idx]
		prev = ident
		ident = ""
		if expression.Operand != nil && expression.Operand.Value != nil && expression.Operand.Value.Ident != nil {
			ident = *expression.Operand.Value.Ident
		}

		code := c.Result.DeprecatedV5Code

		if ident == "return" {
			// A return statement can only be followed by max 1 more expression
			max := len(expressions)
			if idx+2 < max {
				return errors.New("return statement is followed by too many expressions")
			}

			if idx+1 == max {
				// nothing else coming after this, return nil
			}

			code.SingleValue = true
			continue
		}

		// for all other expressions, just compile
		ref, err := c.compileAndAddExpression(expression)
		if err != nil {
			return err
		}

		if prev == "return" {
			prevChunk := code.Code[ref-1]

			code.AddChunk(&llx.Chunk{
				Call: llx.Chunk_FUNCTION,
				Id:   "return",
				Function: &llx.Function{
					Type:                string(prevChunk.Type()),
					DeprecatedV5Binding: 0,
					Args: []*llx.Primitive{
						llx.RefPrimitiveV1(ref),
					},
				},
			})
			code.Entrypoints = []int32{code.ChunkIndex()}
			code.SingleValue = true

			return nil
		}

		if ident == "if" && code.SingleValue {
			// all following expressions need to be compiled in a block which is
			// conditional to this if-statement
			c.prevID = "else"
			rest := expressions[idx+1:]
			_, err := c.compileUnboundBlock(rest, code.LastChunk())
			return err
		}

		l := len(code.Entrypoints)
		// if the last entrypoint already points to this ref, skip it
		if l != 0 && code.Entrypoints[l-1] == ref {
			continue
		}

		code.Entrypoints = append(code.Entrypoints, ref)

		if code.Checksums[ref] == "" {
			return errors.New("failed to compile expression, ref returned empty checksum ID for ref " + strconv.FormatInt(int64(ref), 10))
		}
	}

	return nil
}

// CompileParsed AST into an executable structure
func (c *compiler) CompileParsed(ast *parser.AST) error {
	err := c.compileExpressions(ast.Expressions)
	if err != nil {
		return err
	}

	c.Result.DeprecatedV5Code.UpdateID()
	c.updateEntrypoints()
	return nil
}

func (c *compiler) updateEntrypoints() {
	code := c.Result.DeprecatedV5Code

	// 0. prep: everything that's an entrypoint is a scoringpoint later on
	datapoints := map[int32]struct{}{}

	// 1. remove variable definitions from entrypoints
	varsByRef := make(map[int32]variable, len(c.vars))
	for _, v := range c.vars {
		varsByRef[v.ref] = v
	}

	max := len(code.Entrypoints)
	for i := 0; i < max; i++ {
		ref := code.Entrypoints[i]
		if _, ok := varsByRef[ref]; ok {
			code.Entrypoints[i], code.Entrypoints[max-1] = code.Entrypoints[max-1], code.Entrypoints[i]
			max--
		}
	}
	if max != len(code.Entrypoints) {
		code.Entrypoints = code.Entrypoints[:max]
	}

	// 2. potentially clean up all inherited entrypoints
	// TODO: unclear if this is necessary because the condition may never be met
	entrypoints := map[int32]struct{}{}
	for _, ref := range code.Entrypoints {
		entrypoints[ref] = struct{}{}
		chunk := code.Code[ref-1]
		if chunk.Function != nil {
			delete(entrypoints, chunk.Function.DeprecatedV5Binding)
		}
	}

	// 3. resolve operators
	for ref := range entrypoints {
		dps := code.RefDatapoints(ref)
		if dps != nil {
			for i := range dps {
				datapoints[dps[i]] = struct{}{}
			}
		}
	}

	// done
	res := make([]int32, len(datapoints))
	var idx int
	for ref := range datapoints {
		res[idx] = ref
		idx++
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i] < res[j]
	})
	code.Datapoints = append(code.Datapoints, res...)
}

// CompileAST with a schema into a chunky code
func CompileAST(ast *parser.AST, schema *resources.Schema, props map[string]*llx.Primitive) (*llx.CodeBundle, error) {
	if schema == nil {
		return nil, errors.New("mqlc> please provide a schema to compile this code")
	}

	if props == nil {
		props = map[string]*llx.Primitive{}
	}

	c := compiler{
		Schema: schema,
		Result: &llx.CodeBundle{
			DeprecatedV5Code: &llx.CodeV1{
				Checksums: map[int32]string{},
			},
			Labels: &llx.Labels{
				Labels: map[string]string{},
			},
			Props:            map[string]string{},
			Version:          cnquery.ApiVersion(),
			MinMondooVersion: "",
		},
		vars:       map[string]variable{},
		parent:     nil,
		props:      props,
		standalone: true,
	}

	return c.Result, c.CompileParsed(ast)
}

// Compile a code piece against a schema into chunky code
func Compile(input string, schema *resources.Schema, props map[string]*llx.Primitive) (*llx.CodeBundle, error) {
	// remove leading whitespace
	input = Dedent(input)

	ast, err := parser.Parse(input)
	if ast == nil {
		return nil, err
	}

	// Special handling for parser errors: We still try to compile it because
	// we want to get any compiler suggestions for auto-complete / fixing it.
	// That said, we must return an error either way.
	if err != nil {
		res, _ := CompileAST(ast, schema, props)
		return res, err
	}

	res, err := CompileAST(ast, schema, props)
	if err != nil {
		return res, err
	}

	err = UpdateLabels(res.DeprecatedV5Code, res.Labels, schema)
	if err != nil {
		return res, err
	}
	if len(res.Labels.Labels) == 0 {
		res.Labels.Labels = nil
	}

	err = UpdateAssertions(res)
	if err != nil {
		return res, err
	}

	res.Source = input
	return res, nil
}

// MustCompile a code piece that should not fail (otherwise panic)
func MustCompile(input string, schema *resources.Schema, props map[string]*llx.Primitive) *llx.CodeBundle {
	res, err := Compile(input, schema, props)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to compile")
	}
	return res
}

func getMinMondooVersion(current string, resource string, field string) string {
	return "0.0.0"
}

//func getMinMondooVersion(current string, resource string, field string) string {
//	rd := all.ResourceDocs.Resources[resource]
//	var minverDocs string
//	if rd != nil {
//		minverDocs = rd.MinMondooVersion
//		if field != "" {
//			f := rd.Fields[field]
//			if f != nil && f.MinMondooVersion != "" {
//				minverDocs = f.MinMondooVersion
//			}
//		}
//		if current != "" {
//			// If the field has a newer version requirement than the current code bundle
//			// then update the version requirement to the newest version required.
//			docMin, err := vrs.NewVersion(minverDocs)
//			curMin, err1 := vrs.NewVersion(current)
//			if docMin != nil && err == nil && err1 == nil && docMin.LessThan(curMin) {
//				return current
//			}
//		}
//	}
//	return minverDocs
//}
