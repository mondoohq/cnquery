package leise

import (
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/leise/parser"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/types"
)

type fieldCompiler func(*compiler, string, *parser.Call, *llx.CodeBundle) (types.Type, error)

var operatorsCompilers map[string]fieldCompiler

func init() {
	operatorsCompilers = map[string]fieldCompiler{
		"==":     compileComparable,
		"=~":     compileComparable,
		"!=":     compileComparable,
		"!~":     compileComparable,
		">=":     compileComparable,
		">":      compileComparable,
		"<=":     compileComparable,
		"<":      compileComparable,
		"+":      compileTransformation,
		"-":      compileTransformation,
		"*":      compileTransformation,
		"/":      compileTransformation,
		"%":      nil,
		"=":      compileAssignment,
		"||":     compileComparable,
		"&&":     compileComparable,
		"{}":     compileBlock,
		"if":     compileIf,
		"else":   compileElse,
		"expect": compileExpect,
		"score":  compileScore,
		"typeof": compileTypeof,
		"switch": compileSwitch,
		"Never":  compileNever,
	}
}

func resolveType(chunk *llx.Chunk, code *llx.Code) types.Type {
	var typ types.Type
	var ref int32
	if chunk.Function != nil {
		typ = types.Type(chunk.Function.Type)
		ref = chunk.Function.Binding
	} else if chunk.Primitive != nil {
		typ = types.Type(chunk.Primitive.Type)
		ref, _ = chunk.Primitive.Ref()
	} else {
		// if it compiled and we have a name with an ID that is not a ref then
		// it's a resource with that id
		typ = types.Resource(chunk.Id)
	}

	if typ != types.Ref {
		return typ
	}
	return resolveType(code.Code[ref-1], code)
}

// compile the operation between two operands A and B
// examples: A && B, A - B, ...
func compileABOperation(c *compiler, id string, call *parser.Call) (int32, *llx.Chunk, *llx.Primitive, error) {
	if call == nil {
		return 0, nil, nil, errors.New("operation needs a function call")
	}

	if call.Function == nil {
		return 0, nil, nil, errors.New("operation needs a function call")
	}
	if len(call.Function) != 2 {
		if len(call.Function) < 2 {
			return 0, nil, nil, errors.New("missing arguments")
		}
		return 0, nil, nil, errors.New("too many arguments")
	}

	a := call.Function[0]
	b := call.Function[1]
	if a.Name != "" || b.Name != "" {
		return 0, nil, nil, errors.New("calling operations with named arguments is not supported")
	}

	leftRef, err := c.compileAndAddExpression(a.Value)
	if err != nil {
		return 0, nil, nil, err
	}
	left := c.Result.Code.Code[leftRef-1]

	right, err := c.compileExpression(b.Value)
	if err != nil {
		return 0, nil, nil, err
	}

	if left == nil {
		log.Fatal().Msgf("left is nil: %d %#v", leftRef, c.Result.Code.Code[leftRef-1])
	}

	return leftRef, left, right, nil
}

func compileAssignment(c *compiler, id string, call *parser.Call, res *llx.CodeBundle) (types.Type, error) {
	if call == nil {
		return types.Nil, errors.New("assignment needs a function call")
	}

	if call.Function == nil {
		return types.Nil, errors.New("assignment needs a function call")
	}
	if len(call.Function) != 2 {
		if len(call.Function) < 2 {
			return types.Nil, errors.New("missing arguments")
		}
		return types.Nil, errors.New("too many arguments")
	}

	varIdent := call.Function[0]
	varValue := call.Function[1]
	if varIdent.Name != "" || varValue.Name != "" {
		return types.Nil, errors.New("calling operations with named arguments is not supported")
	}

	if varIdent.Value == nil || varIdent.Value.Operand == nil || varIdent.Value.Operand.Value == nil ||
		varIdent.Value.Operand.Value.Ident == nil {
		return types.Nil, errors.New("variable name is not defined")
	}
	name := *varIdent.Value.Operand.Value.Ident

	ref, err := c.compileAndAddExpression(varValue.Value)
	if err != nil {
		return types.Nil, err
	}

	c.vars[name] = variable{
		ref: ref,
		typ: c.Result.Code.Code[ref-1].Type(c.Result.Code),
	}

	return types.Nil, nil
}

func compileComparable(c *compiler, id string, call *parser.Call, res *llx.CodeBundle) (types.Type, error) {
	leftRef, left, right, err := compileABOperation(c, id, call)
	if err != nil {
		return types.Nil, errors.New("failed to compile: " + err.Error())
	}

	for left.Type(res.Code) == types.Ref {
		var ok bool
		leftRef, ok = left.Primitive.Ref()
		if !ok {
			return types.Nil, errors.New("failed to get reference entry of left operand to " + id + ", this should not happen")
		}
		left = res.Code.Code[leftRef-1]
	}

	// find specialized or generalized builtin function
	lt := left.Type(res.Code)
	rt := resolveType(&llx.Chunk{Primitive: right}, res.Code)

	name := id + string(rt)
	h, err := llx.BuiltinFunction(lt, name)
	if err != nil {
		h, err = llx.BuiltinFunction(lt, id)
	}
	if err != nil {
		name = id + string(rt.Underlying())
		h, err = llx.BuiltinFunction(lt, name)
	}
	if err != nil {
		return types.Nil, errors.New("cannot find operator handler: " + lt.Label() + " " + id + " " + types.Type(right.Type).Label())
	}

	if h.Compiler != nil {
		name, err = h.Compiler(left.Type(res.Code), types.Type(right.Type))
		if err != nil {
			return types.Nil, err
		}
	}

	res.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   name,
		Function: &llx.Function{
			Type:    string(types.Bool),
			Binding: leftRef,
			Args:    []*llx.Primitive{right},
		},
	})

	return types.Bool, nil
}

func compileTransformation(c *compiler, id string, call *parser.Call, res *llx.CodeBundle) (types.Type, error) {
	leftRef, left, right, err := compileABOperation(c, id, call)
	if err != nil {
		return types.Nil, err
	}

	// find specialized or generalized builtin function
	lt := left.Type(res.Code).Underlying()
	rt := resolveType(&llx.Chunk{Primitive: right}, res.Code)

	name := id + string(rt)
	h, err := llx.BuiltinFunction(lt, name)
	if err != nil {
		h, err = llx.BuiltinFunction(lt, id)
	}
	if err != nil {
		name = id + string(rt.Underlying())
		h, err = llx.BuiltinFunction(lt, name)
	}
	if err != nil {
		return types.Nil, errors.New("cannot find operator handler: " + lt.Label() + " " + id + " " + types.Type(right.Type).Label())
	}

	if h.Compiler != nil {
		name, err = h.Compiler(left.Type(res.Code), types.Type(right.Type))
		if err != nil {
			return types.Nil, err
		}
	}

	returnType := h.Typ
	if returnType == types.Empty {
		returnType = lt
	}

	res.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   name,
		Function: &llx.Function{
			Type:    string(returnType),
			Binding: leftRef,
			Args:    []*llx.Primitive{right},
		},
	})

	return lt, nil
}

func generateEntrypoints(arg *llx.Primitive, res *llx.CodeBundle) error {
	ref, ok := arg.Ref()
	if !ok {
		return nil
	}

	refobj := res.Code.Code[ref-1]
	if refobj == nil {
		return errors.New("Failed to get code reference on expect call, this shouldn't happen")
	}

	reffunc := refobj.Function
	if reffunc == nil {
		return nil
	}

	// if the left argument is not a primitive but a calculated value
	bind := res.Code.Code[reffunc.Binding-1]
	if bind.Primitive == nil {
		res.Code.Entrypoints = append(res.Code.Entrypoints, int32(reffunc.Binding))
	}

	for i := range reffunc.Args {
		arg := reffunc.Args[i]
		i, ok := arg.Ref()
		if ok {
			// TODO: int32 vs int64
			res.Code.Entrypoints = append(res.Code.Entrypoints, int32(i))
		}
	}
	return nil
}

func compileBlock(c *compiler, id string, call *parser.Call, res *llx.CodeBundle) (types.Type, error) {
	res.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   id,
		Function: &llx.Function{
			Type: string(types.Unset),
			Args: []*llx.Primitive{},
		},
	})
	return types.Unset, nil
}

func compileIf(c *compiler, id string, call *parser.Call, res *llx.CodeBundle) (types.Type, error) {
	if call == nil {
		return types.Nil, errors.New("need conditional arguments for if-clause")
	}
	if len(call.Function) < 1 {
		return types.Nil, errors.New("missing parameters for if-clause, it requires 1")
	}
	arg := call.Function[0]
	if arg.Name != "" {
		return types.Nil, errors.New("called if-clause with a named argument, which is not supported")
	}

	// if we are in a chained if-else call (needs previous if-call)
	if c.prevID == "else" && len(res.Code.Code) != 0 {
		maxRef := len(res.Code.Code) - 1
		prev := res.Code.Code[maxRef]
		if prev.Id == "if" {
			// we need to pop off the last "if" chunk as the new condition needs to
			// be added in front of it
			res.Code.Code = res.Code.Code[0:maxRef]

			argValue, err := c.compileExpression(arg.Value)
			if err != nil {
				return types.Nil, err
			}

			// now add back the last chunk and append the newly compiled condition
			res.Code.AddChunk(prev)
			prev.Function.Args = append(prev.Function.Args, argValue)

			c.prevID = "if"
			return types.Nil, nil
		}
	}

	argValue, err := c.compileExpression(arg.Value)
	if err != nil {
		return types.Nil, err
	}

	res.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   id,
		Function: &llx.Function{
			Type: string(types.Unset),
			Args: []*llx.Primitive{argValue},
		},
	})
	res.Code.Entrypoints = append(res.Code.Entrypoints, res.Code.ChunkIndex())
	c.prevID = "if"

	return types.Nil, nil
}

func compileElse(c *compiler, id string, call *parser.Call, res *llx.CodeBundle) (types.Type, error) {
	if call != nil {
		return types.Nil, errors.New("cannot have conditional arguments for else-clause, use another if-statement")
	}

	if len(res.Code.Code) == 0 {
		return types.Nil, errors.New("can only use else-statement after a preceding if-statement")
	}

	prev := res.Code.Code[len(res.Code.Code)-1]
	if prev.Id != "if" {
		return types.Nil, errors.New("can only use else-statement after a preceding if-statement")
	}

	if c.prevID != "if" {
		return types.Nil, errors.New("can only use else-statement after a preceding if-statement (internal reference is wrong)")
	}

	c.prevID = "else"

	return types.Nil, nil
}

func compileExpect(c *compiler, id string, call *parser.Call, res *llx.CodeBundle) (types.Type, error) {
	if call == nil || len(call.Function) < 1 {
		return types.Nil, errors.New("missing parameter for '" + id + "', it requires 1")
	}
	if len(call.Function) > 1 {
		return types.Nil, errors.New("called '" + id + "' with too many arguments, it requires 1")
	}

	arg := call.Function[0]
	if arg.Name != "" {
		return types.Nil, errors.New("called '" + id + "' with a named argument, which is not supported")
	}

	argValue, err := c.compileExpression(arg.Value)
	if err != nil {
		return types.Nil, err
	}

	if err = generateEntrypoints(argValue, res); err != nil {
		return types.Nil, err
	}

	typ := types.Bool
	res.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   id,
		Function: &llx.Function{
			Type: string(typ),
			Args: []*llx.Primitive{argValue},
		},
	})
	res.Code.Entrypoints = append(res.Code.Entrypoints, res.Code.ChunkIndex())

	return typ, nil
}

func compileScore(c *compiler, id string, call *parser.Call, res *llx.CodeBundle) (types.Type, error) {
	if call == nil || len(call.Function) < 1 {
		return types.Nil, errors.New("missing parameter for '" + id + "', it requires 1")
	}

	arg := call.Function[0]
	if arg == nil || arg.Value == nil || arg.Value.Operand == nil || arg.Value.Operand.Value == nil {
		return types.Nil, errors.New("failed to get parameter for '" + id + "'")
	}

	argValue, err := c.compileExpression(arg.Value)
	if err != nil {
		return types.Nil, err
	}

	res.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "score",
		Function: &llx.Function{
			Type: string(types.Score),
			Args: []*llx.Primitive{argValue},
		},
	})

	return types.Score, nil
}

func compileTypeof(c *compiler, id string, call *parser.Call, res *llx.CodeBundle) (types.Type, error) {
	if call == nil || len(call.Function) < 1 {
		return types.Nil, errors.New("missing parameter for '" + id + "', it requires 1")
	}

	arg := call.Function[0]
	if arg == nil || arg.Value == nil || arg.Value.Operand == nil || arg.Value.Operand.Value == nil {
		return types.Nil, errors.New("failed to get parameter for '" + id + "'")
	}

	argValue, err := c.compileExpression(arg.Value)
	if err != nil {
		return types.Nil, err
	}

	res.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "typeof",
		Function: &llx.Function{
			Type: string(types.String),
			Args: []*llx.Primitive{argValue},
		},
	})

	return types.String, nil
}

func compileSwitch(c *compiler, id string, call *parser.Call, res *llx.CodeBundle) (types.Type, error) {
	var ref *llx.Primitive

	if call != nil && len(call.Function) != 0 {
		arg := call.Function[0]
		if arg.Name != "" {
			return types.Nil, errors.New("called `" + id + "` with a named argument, which is not supported")
		}

		argValue, err := c.compileExpression(arg.Value)
		if err != nil {
			return types.Nil, err
		}

		ref = argValue
	} else {
		ref = &llx.Primitive{Type: string(types.Unset)}
	}

	res.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   id,
		Function: &llx.Function{
			Type: string(types.Unset),
			Args: []*llx.Primitive{ref},
		},
	})
	c.prevID = "switch"

	return types.Nil, nil
}

func compileNever(c *compiler, id string, call *parser.Call, res *llx.CodeBundle) (types.Type, error) {
	res.Code.AddChunk(&llx.Chunk{
		Call:      llx.Chunk_PRIMITIVE,
		Primitive: llx.NeverFuturePrimitive,
	})

	return types.Time, nil
}
