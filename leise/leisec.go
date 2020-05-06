package leise

import (
	"errors"
	"regexp"
	"sort"
	"strconv"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/leise/parser"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/types"
)

type binding struct {
	Type types.Type
	Ref  int32
}

type compiler struct {
	Schema  *lumi.Schema
	Result  *llx.CodeBundle
	Binding *binding
}

func addResourceSuggestions(resources map[string]*lumi.ResourceInfo, name string, res *llx.CodeBundle) {
	names := make([]string, len(resources))
	i := 0
	for key := range resources {
		names[i] = key
		i++
	}

	res.Suggestions = fuzzy.Find(name, names)
	sort.Strings(res.Suggestions)
}

func addFieldSuggestions(fields []string, fieldName string, res *llx.CodeBundle) {
	res.Suggestions = fuzzy.Find(fieldName, fields)
	sort.Strings(res.Suggestions)
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
	fref, err := c.blockExpressions(expressions, typ)
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
			Type:    string(resultType),
			Binding: c.Result.Code.ChunkIndex(),
			Args:    []*llx.Primitive{llx.FunctionPrimitive(fref)},
		},
	}
	c.Result.Code.AddChunk(&chunk)

	return resultType, nil
}

// evaluates the given expressions on a non-array resource
// and creates a function, whose reference is returned
func (c *compiler) blockOnResource(expressions []*parser.Expression, typ types.Type) (int32, error) {
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
					Primitive: &llx.Primitive{Type: string(typ)},
				}},
			},
			Labels: c.Result.Labels,
		},
		Binding: &binding{Type: typ, Ref: 1},
	}

	err := blockCompiler.compileExpressions(expressions)
	c.Result.Suggestions = append(c.Result.Suggestions, blockCompiler.Result.Suggestions...)
	if err != nil {
		return 0, err
	}

	code := blockCompiler.Result.Code
	code.UpdateID()
	c.Result.Code.Functions = append(c.Result.Code.Functions, code)
	return int32(len(c.Result.Code.Functions)), nil
}

// blockExpressions evaluates the given expressions as if called by a block and
// returns the compiled function reference
func (c *compiler) blockExpressions(expressions []*parser.Expression, typ types.Type) (int32, error) {
	if len(expressions) == 0 {
		return 0, nil
	}

	if typ.IsArray() {
		return c.blockOnResource(expressions, typ.Child())
	}

	return c.blockOnResource(expressions, typ)
}

func (c *compiler) unnamedResourceArgs(resource *lumi.ResourceInfo, args []*parser.Arg) ([]*llx.Primitive, error) {
	init := resource.Init
	if init == nil {
		return nil, errors.New("cannot find init call for resource " + resource.Id)
	}

	if len(args) > len(init.Args) {
		return nil, errors.New("Called resource " + resource.Name +
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
		if types.Type(v.Type) == types.Ref {
			return nil, errors.New("Cannot handle refs in function calls just yet")
		}

		expected := init.Args[idx]
		expectedType := types.Type(expected.Type)
		typ := types.Type(v.Type)
		if typ != expectedType {
			return nil, errors.New("Incorrect type on argument " + strconv.Itoa(idx) +
				" in resource " + resource.Name + ": expected " + expectedType.Label() +
				", got: " + typ.Label())
		}

		res[idx*2] = llx.StringPrimitive(expected.Name)
		res[idx*2+1] = v
	}

	return res, nil
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
		vt := types.Type(v.Type)
		if vt == types.Ref {
			return nil, errors.New("Cannot handle refs in function calls just yet")
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
			Type:    string(resType),
			Binding: c.Result.Code.ChunkIndex(),
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
		function = &llx.Function{Type: string(typ)}
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
// 1. global f(): expect, ...
// 2. global rsc: sshd, sshd.config
// 3. bound field: user { name }
// x. called field: user.name <= not in this scope
func (c *compiler) compileIdentifier(id string, binding *binding, calls []*parser.Call) ([]*parser.Call, types.Type, error) {
	var call *parser.Call
	restCalls := calls
	if len(calls) > 0 && calls[0].Function != nil {
		call = calls[0]
		restCalls = calls[1:]
	}

	var typ types.Type
	var err error
	var found bool
	if binding != nil {
		// special handling for the `self` operator
		if id == "_" {
			if len(restCalls) == 0 {
				// TODO: something is missing
				return restCalls, binding.Type, nil
			}

			nextCall := restCalls[0]
			found, typ, err = c.compileBoundIdentifier(*nextCall.Ident, binding, nextCall)
			if found {
				return restCalls[1:], typ, err
			}
			// return restCalls, binding.Type, nil
		}

		found, typ, err = c.compileBoundIdentifier(id, binding, call)
		if found {
			return restCalls, typ, err
		}
	}

	f := operatorsCompilers[id]
	if f != nil {
		typ, err := f(c, id, call, c.Result)
		return restCalls, typ, err
	}

	found, restCalls, typ, err = c.compileResource(id, calls)
	if found {
		return restCalls, typ, err
	}

	// suggestions
	if binding == nil {
		addResourceSuggestions(c.Schema.Resources, id, c.Result)
		return nil, types.Nil, errors.New("Cannot find resource for identifier '" + id + "'")
	}
	addFieldSuggestions(availableFields(c, binding.Type), id, c.Result)
	return nil, types.Nil, errors.New("Cannot find field or resource '" + id + "' in block for type '" + c.Binding.Type.Label() + "'")
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
			Type:  string(llx.ArrayType(arr, c.Result.Code)),
			Array: arr,
		}, nil
	}

	return llx.NilPrimitive, nil
}

func (c *compiler) compileOperand(operand *parser.Operand) (*llx.Primitive, error) {
	calls := operand.Calls
	var err error
	var res *llx.Primitive
	var typ types.Type
	var ref int32

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
		calls, typ, err = c.compileIdentifier(id, c.Binding, calls)
		if err != nil {
			return nil, err
		}
		ref = c.Result.Code.ChunkIndex()
		res = llx.RefPrimitive(ref)
	}

	// operand:      value [ call | accessor | '.' ident ]+ [ block ]
	// dealing with all call types
	for len(calls) > 0 {
		call := calls[0]
		if call.Function != nil {
			return nil, errors.New("Don't know how to compile chained functions just yet")
		}

		if call.Accessor != nil {
			// turn accessor into a regular function and call that
			fCall := &parser.Call{Function: []*parser.Arg{&parser.Arg{Value: call.Accessor}}}
			// accessors are aways builtin functions
			h, _ := builtinFunction(typ.Underlying(), "[]")
			if h == nil {
				return nil, errors.New("Cannot find '[]' function on type " + typ.Label())
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

			calls = calls[1:]
			call = nil
			if len(calls) > 0 && calls[0].Function != nil {
				call = calls[0]
			}

			found, resType, err = c.compileBoundIdentifier(id, &binding{Type: typ, Ref: ref}, call)
			if !found {
				addFieldSuggestions(availableFields(c, typ), id, c.Result)
				return nil, errors.New("Cannot find field '" + id + "' in " + typ.Label() + "")
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
		if types.Type(res.Type) != types.Ref {
			return nil, errors.New("Cannot call block on simple type '" + types.Type(res.Type).Label() + "' yet")
		}
		typ, err = c.compileBlock(operand.Block, typ)
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
		// nothing to do, the last call was added to the compiled chain
	} else {
		c.Result.Code.AddChunk(&llx.Chunk{
			Call: llx.Chunk_PRIMITIVE,
			// no id for standalone values
			Primitive: valc,
		})
	}

	return c.Result.Code.ChunkIndex(), nil
}

func (c *compiler) compileExpressions(expressions []*parser.Expression) error {
	var err error
	for idx := range expressions {
		if err = expressions[idx].ProcessOperators(); err != nil {
			return err
		}
	}

	for idx := range expressions {
		ref, err := c.compileAndAddExpression(expressions[idx])
		if err != nil {
			return err
		}

		l := len(c.Result.Code.Entrypoints)
		// if the last entrypoint already points to this ref, skip it
		if l != 0 && c.Result.Code.Entrypoints[l-1] == ref {
			continue
		}

		c.Result.Code.Entrypoints = append(c.Result.Code.Entrypoints, ref)

		if c.Result.Code.Checksums[ref] == "" {
			return errors.New("Failed to compile expression, ref returned empty checksum ID for ref " + strconv.FormatInt(int64(ref), 10))
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

	return nil
}

// CompileAST with a schema into a chunky code
func CompileAST(ast *parser.AST, schema *lumi.Schema) (*llx.CodeBundle, error) {
	if schema == nil {
		return nil, errors.New("leise> please provide a schema to compile this code")
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
		},
	}

	return c.Result, c.CompileParsed(ast)
}

// Compile a code piece against a schema into chunky code
func Compile(input string, schema *lumi.Schema) (*llx.CodeBundle, error) {
	ast, err := parser.Parse(input)
	if err != nil {
		return nil, err
	}

	res, err := CompileAST(ast, schema)
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
func MustCompile(input string, schema *lumi.Schema) *llx.CodeBundle {
	res, err := Compile(input, schema)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to compile")
	}
	return res
}
