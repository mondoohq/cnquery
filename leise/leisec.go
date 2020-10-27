package leise

import (
	"errors"
	"regexp"
	"sort"
	"strconv"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo"
	"go.mondoo.io/mondoo/leise/parser"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/types"
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
	Schema  *lumi.Schema
	Result  *llx.CodeBundle
	Binding *binding
	vars    map[string]variable
	parent  *compiler
	props   map[string]*llx.Primitive

	// a standalone code is one that doesn't call any of its bindings
	// examples:
	//   file(xyz).content          is standalone
	//   file(xyz).content == _     is not
	standalone bool

	// helps chaining of builtin calls like `if (..) else if (..) else ..`
	prevID string
}

func addResourceSuggestions(resources map[string]*lumi.ResourceInfo, name string, res *llx.CodeBundle) {
	names := make([]string, len(resources))
	i := 0
	for key := range resources {
		names[i] = key
		i++
	}

	suggestedNames := fuzzy.Find(name, names)
	res.Suggestions = make([]*llx.Documentation, len(suggestedNames))
	var info *lumi.ResourceInfo
	for i := range suggestedNames {
		field := suggestedNames[i]
		info = resources[field]
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
func (c *compiler) compileBlock(expressions []*parser.Expression, typ types.Type) (types.Type, error) {
	fref, _, err := c.blockExpressions(expressions, typ)
	if err != nil {
		return types.Nil, err
	}
	if fref == 0 {
		return typ, nil
	}

	resultType := types.Any
	if typ.IsArray() {
		resultType = types.Array(types.Any)
	}

	chunk := llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "{}",
		Function: &llx.Function{
			Type:    resultType,
			Binding: c.Result.Code.ChunkIndex(),
			Args:    []*llx.Primitive{llx.FunctionPrimitive(fref)},
		},
	}
	c.Result.Code.AddChunk(&chunk)

	return resultType, nil
}

func (c *compiler) compileUnboundBlock(expressions []*parser.Expression, chunk *llx.Chunk) (types.Type, error) {
	if !(chunk.Id == "if") {
		return types.Nil, errors.New("don't know how to compile unbound block on call `" + chunk.Id + "`")
	}

	// if `else { .. }` is called, we reset the prevID to indicate there is no
	// more chaining happening
	if c.prevID == "else" {
		c.prevID = ""
	}

	blockCompiler := &compiler{
		Schema: c.Schema,
		Result: &llx.CodeBundle{
			Code: &llx.Code{
				Id:         chunk.Id,
				Parameters: 0,
				Checksums:  map[int32]string{},
				Code:       []*llx.Chunk{},
			},
			Labels: c.Result.Labels,
			Props:  c.Result.Props,
		},
		vars:       map[string]variable{},
		parent:     c,
		props:      c.props,
		standalone: true,
	}

	err := blockCompiler.compileExpressions(expressions)
	c.Result.Suggestions = append(c.Result.Suggestions, blockCompiler.Result.Suggestions...)
	if err != nil {
		return types.Nil, err
	}

	code := blockCompiler.Result.Code
	code.UpdateID()
	c.Result.Code.Functions = append(c.Result.Code.Functions, code)

	chunk.Function.Args = append(chunk.Function.Args, llx.FunctionPrimitive(c.Result.Code.FunctionsIndex()))
	c.Result.Code.RefreshChunkChecksum(chunk)

	// we set this to true, so that we can decide how to handle all following expressions
	if blockCompiler.Result.Code.SingleValue {
		c.Result.Code.SingleValue = true
	}

	return types.Nil, nil
}

// evaluates the given expressions on a non-array resource
// and creates a function, whose reference is returned
func (c *compiler) blockOnResource(expressions []*parser.Expression, typ types.Type) (int32, bool, error) {
	blockCompiler := &compiler{
		Schema: c.Schema,
		Result: &llx.CodeBundle{
			Code: &llx.Code{
				Id:         "binding",
				Parameters: 1,
				Checksums: map[int32]string{
					// we must provide the first chunk, which is a reference to the caller
					// and which will always be number 1
					1: c.Result.Code.Checksums[c.Result.Code.ChunkIndex()],
				},
				Code: []*llx.Chunk{{
					Call:      llx.Chunk_PRIMITIVE,
					Primitive: &llx.Primitive{Type: typ},
				}},
			},
			Labels: c.Result.Labels,
			Props:  c.Result.Props,
		},
		Binding:    &binding{Type: typ, Ref: 1},
		vars:       map[string]variable{},
		parent:     c,
		props:      c.props,
		standalone: true,
	}

	err := blockCompiler.compileExpressions(expressions)
	c.Result.Suggestions = append(c.Result.Suggestions, blockCompiler.Result.Suggestions...)
	if err != nil {
		return 0, false, err
	}

	code := blockCompiler.Result.Code
	code.UpdateID()
	c.Result.Code.Functions = append(c.Result.Code.Functions, code)
	return c.Result.Code.FunctionsIndex(), blockCompiler.standalone, nil
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
	if len(c.Result.Code.Functions) < int(ref) {
		return types.Nil, errors.New("canot find function block with ref " + strconv.Itoa(int(ref)))
	}

	f := c.Result.Code.Functions[ref-1]
	if len(f.Entrypoints) != 1 {
		return types.Nil, errors.New("function block should only return 1 value (got: " + strconv.Itoa(len(f.Entrypoints)) + ")")
	}

	ep := f.Entrypoints[0]
	chunk := f.Code[ep-1]
	return chunk.Type(c.Result.Code), nil
}

func (c *compiler) dereferenceType(val *llx.Primitive) (types.Type, error) {
	valType := types.Type(val.Type)
	if types.Type(val.Type) != types.Ref {
		return valType, nil
	}

	ref, ok := val.Ref()
	if !ok {
		return types.Nil, errors.New("found a reference type that doesn't return a reference value")
	}

	chunk := c.Result.Code.Code[ref-1]
	if chunk.Primitive == val {
		return types.Nil, errors.New("recursive reference connections detected")
	}

	if chunk.Primitive != nil {
		return c.dereferenceType(chunk.Primitive)
	}

	valType = chunk.Type(c.Result.Code)
	return valType, nil
}

func (c *compiler) unnamedArgs(callerLabel string, init *lumi.Init, args []*parser.Arg) ([]*llx.Primitive, error) {
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
			return nil, errors.New("Incorrect type on argument " + strconv.Itoa(idx) +
				" in " + callerLabel + ": expected " + expectedType.Label() +
				", got: " + vType.Label())
		}

		res[idx*2] = llx.StringPrimitive(expected.Name)
		res[idx*2+1] = v
	}

	return res, nil
}

func (c *compiler) unnamedResourceArgs(resource *lumi.ResourceInfo, args []*parser.Arg) ([]*llx.Primitive, error) {
	if resource.Init == nil {
		return nil, errors.New("cannot find init call for resource " + resource.Id)
	}

	return c.unnamedArgs("resource "+resource.Name, resource.Init, args)
}

// resourceArgs turns the list of arguments for the resource into a list of
// primitives that are used as arguments to initialize that resource
// only works if len(args) > 0 !!
// only works if args are either ALL named or not named !!
func (c *compiler) resourceArgs(resource *lumi.ResourceInfo, args []*parser.Arg) ([]*llx.Primitive, error) {
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
		resType, err := h.compile(c, typ, binding.Ref, id, call)
		return resType, err
	}

	var args []*llx.Primitive

	if call != nil {
		args = make([]*llx.Primitive, len(call.Function))
		var err error
		for idx := range call.Function {
			arg := call.Function[idx]
			args[idx], err = c.compileExpression(arg.Value)
			if err != nil {
				return types.Nil, err
			}
		}
	}

	if err := h.signature.Validate(args); err != nil {
		return types.Nil, err
	}

	resType := h.typ(typ)
	c.Result.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   id,
		Function: &llx.Function{
			Type:    resType,
			Binding: binding.Ref,
			Args:    args,
		},
	})
	return resType, nil
}

// compile a bound identifier to its binding
// example: user { name } , where name is compiled bound to the user
// it will return false if it cannot bind the identifier
func (c *compiler) compileBoundIdentifier(id string, binding *binding, call *parser.Call) (bool, types.Type, error) {
	typ := binding.Type
	if typ.IsResource() {
		resource, ok := c.Schema.Resources[typ.Name()]
		if !ok {
			return true, types.Nil, errors.New("cannot find resource that is called by '" + id + "' of type " + typ.Label())
		}

		fieldinfo, ok := resource.Fields[id]
		if ok {
			if call != nil && len(call.Function) > 0 {
				return true, types.Nil, errors.New("cannot call resource field with arguments yet")
			}
			c.Result.Code.AddChunk(&llx.Chunk{
				Call: llx.Chunk_FUNCTION,
				Id:   id,
				Function: &llx.Function{
					Type:    fieldinfo.Type,
					Binding: binding.Ref,
				},
			})
			return true, types.Type(fieldinfo.Type), nil
		}
	}

	h, _ := builtinFunction(typ, id)
	if h != nil {
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

	typ, err := c.addResource(id, resource, call)
	return true, calls, typ, err
}

func (c *compiler) addResource(id string, resource *lumi.ResourceInfo, call *parser.Call) (types.Type, error) {
	var function *llx.Function
	var err error
	typ := types.Resource(id)

	if call != nil && len(call.Function) > 0 {
		function = &llx.Function{Type: typ}
		function.Args, err = c.resourceArgs(resource, call.Function)
		if err != nil {
			return types.Nil, err
		}
	}

	c.Result.Code.AddChunk(&llx.Chunk{
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
		c.Result.Code.AddChunk(&llx.Chunk{
			Call:      llx.Chunk_PRIMITIVE,
			Primitive: llx.RefPrimitive(variable.ref),
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

	c.Result.Code.AddChunk(&llx.Chunk{
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
			Type:  llx.ArrayType(arr, c.Result.Code),
			Array: arr,
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
			c.Result.Code.AddChunk(&llx.Chunk{
				Call: llx.Chunk_PRIMITIVE,
				// no ID for standalone
				Primitive: res,
			})
			ref = c.Result.Code.ChunkIndex()
			res = llx.RefPrimitive(ref)
		}
	} else {
		id := *operand.Value.Ident
		orgcalls := calls
		calls, typ, err = c.compileIdentifier(id, c.Binding, calls)
		if err != nil {
			return nil, err
		}

		ref = c.Result.Code.ChunkIndex()
		if id == "_" && len(orgcalls) == 0 {
			ref = 1
		}

		res = llx.RefPrimitive(ref)
	}

	// operand:      value [ call | accessor | '.' ident ]+ [ block ]
	// dealing with all call types
	for len(calls) > 0 {
		call := calls[0]
		if call.Function != nil {
			return nil, errors.New("don't know how to compile chained functions just yet")
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
			ref = c.Result.Code.ChunkIndex()
			res = llx.RefPrimitive(ref)
			continue
		}

		if call.Ident != nil {
			var found bool
			var resType types.Type
			id := *call.Ident

			// We get this from the parser if the user called the dot-accessor
			// but didn't provide any values at all. It equates a not found and
			// we can now just suggest all fields
			if id == "." {
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
			ref = c.Result.Code.ChunkIndex()
			res = llx.RefPrimitive(ref)

			continue
		}

		return nil, errors.New("processed a call without any data")
	}

	if operand.Block != nil {
		// for starters, we need the primitive to exist on the stack,
		// so add it if it's missing
		ref := c.Result.Code.ChunkIndex()
		if ref == 0 {
			val, err := c.compileValue(operand.Value)
			if err != nil {
				return nil, err
			}
			c.Result.Code.AddChunk(&llx.Chunk{
				Call: llx.Chunk_PRIMITIVE,
				// no ID for standalone
				Primitive: val,
			})
		}

		if typ == types.Nil {
			_, err = c.compileUnboundBlock(operand.Block, c.Result.Code.LastChunk())
		} else {
			_, err = c.compileBlock(operand.Block, typ)
		}
		if err != nil {
			return nil, err
		}
		ref = c.Result.Code.ChunkIndex()
		res = llx.RefPrimitive(ref)
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
		ref, _ := valc.Ref()
		return ref, nil
		// nothing to do, the last call was added to the compiled chain
	}

	c.Result.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_PRIMITIVE,
		// no id for standalone values
		Primitive: valc,
	})

	return c.Result.Code.ChunkIndex(), nil
}

func (c *compiler) compileExpressions(expressions []*parser.Expression) error {
	var err error
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

		if ident == "return" {
			// A return statement can only be followed by max 1 more expression
			max := len(expressions)
			if idx+2 < max {
				return errors.New("return statement is followed by too many expressions")
			}

			if idx+1 == max {
				// nothing else coming after this, return nil
			}

			c.Result.Code.SingleValue = true
			continue
		}

		// for all other expressions, just compile
		ref, err := c.compileAndAddExpression(expression)
		if err != nil {
			return err
		}

		if prev == "return" {
			prevChunk := c.Result.Code.Code[ref-1]

			c.Result.Code.AddChunk(&llx.Chunk{
				Call: llx.Chunk_FUNCTION,
				Id:   "return",
				Function: &llx.Function{
					Type:    prevChunk.Type(c.Result.Code),
					Binding: 0,
					Args: []*llx.Primitive{
						llx.RefPrimitive(ref),
					},
				},
			})
			c.Result.Code.Entrypoints = append(c.Result.Code.Entrypoints, c.Result.Code.ChunkIndex())
			c.Result.Code.SingleValue = true

			return nil
		}

		if ident == "if" && c.Result.Code.SingleValue {
			// all following expressions need to be compiled in a block which is
			// conditional to this if-statement
			c.prevID = "else"
			rest := expressions[idx+1:]
			_, err := c.compileUnboundBlock(rest, c.Result.Code.LastChunk())
			return err
		}

		l := len(c.Result.Code.Entrypoints)
		// if the last entrypoint already points to this ref, skip it
		if l != 0 && c.Result.Code.Entrypoints[l-1] == ref {
			continue
		}

		c.Result.Code.Entrypoints = append(c.Result.Code.Entrypoints, ref)

		if c.Result.Code.Checksums[ref] == "" {
			return errors.New("failed to compile expression, ref returned empty checksum ID for ref " + strconv.FormatInt(int64(ref), 10))
		}
	}

	return nil
}

// CompileParsed AST into a leiseC structure
func (c *compiler) CompileParsed(ast *parser.AST) error {
	err := c.compileExpressions(ast.Expressions)
	if err != nil {
		return err
	}

	c.Result.Code.UpdateID()
	c.UpdateEntrypoints()

	return nil
}

func (c *compiler) UpdateEntrypoints() {
	// 0. prep: everything that's an entrypoint is a scoringpoint later on
	datapoints := map[int32]struct{}{}

	// 1. remove variable definitions from entrypoints
	varsByRef := make(map[int32]variable, len(c.vars))
	for _, v := range c.vars {
		varsByRef[v.ref] = v
	}

	max := len(c.Result.Code.Entrypoints)
	for i := 0; i < max; i++ {
		ref := c.Result.Code.Entrypoints[i]
		if _, ok := varsByRef[ref]; ok {
			c.Result.Code.Entrypoints[i], c.Result.Code.Entrypoints[max-1] = c.Result.Code.Entrypoints[max-1], c.Result.Code.Entrypoints[i]
			max--
		}
	}
	if max != len(c.Result.Code.Entrypoints) {
		c.Result.Code.Entrypoints = c.Result.Code.Entrypoints[:max]
	}

	// 2. potentially clean up all inherited entrypoints
	// TODO: unclear if this is necessary because the condition may never be met
	entrypoints := map[int32]struct{}{}
	for _, ref := range c.Result.Code.Entrypoints {
		entrypoints[ref] = struct{}{}
		chunk := c.Result.Code.Code[ref-1]
		if chunk.Function != nil {
			delete(entrypoints, chunk.Function.Binding)
		}
	}

	// 3. resolve operators
	for ref := range entrypoints {
		dps := c.Result.Code.RefDatapoints(ref)
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
	c.Result.Code.Datapoints = res
}

// CompileAST with a schema into a chunky code
func CompileAST(ast *parser.AST, schema *lumi.Schema, props map[string]*llx.Primitive) (*llx.CodeBundle, error) {
	if schema == nil {
		return nil, errors.New("leise> please provide a schema to compile this code")
	}

	if props == nil {
		props = map[string]*llx.Primitive{}
	}

	c := compiler{
		Schema: schema,
		Result: &llx.CodeBundle{
			Code: &llx.Code{
				Checksums: map[int32]string{},
			},
			Labels: &llx.Labels{
				Labels: map[string]string{},
			},
			Props:   map[string]string{},
			Version: mondoo.ApiVersion(),
		},
		vars:       map[string]variable{},
		parent:     nil,
		props:      props,
		standalone: true,
	}

	return c.Result, c.CompileParsed(ast)
}

// Compile a code piece against a schema into chunky code
func Compile(input string, schema *lumi.Schema, props map[string]*llx.Primitive) (*llx.CodeBundle, error) {
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
		res, err2 := CompileAST(ast, schema, props)
		if err2 == nil {
			return res, err
		}
		return res, err2
	}

	res, err := CompileAST(ast, schema, props)
	if err != nil {
		return res, err
	}

	err = UpdateLabels(res.Code, res.Labels, schema)
	if err != nil {
		return res, err
	}
	if len(res.Labels.Labels) == 0 {
		res.Labels.Labels = nil
	}

	res.Source = input
	return res, nil
}

// MustCompile a code piece that should not fail (otherwise panic)
func MustCompile(input string, schema *lumi.Schema, props map[string]*llx.Primitive) *llx.CodeBundle {
	res, err := Compile(input, schema, props)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to compile")
	}
	return res
}
