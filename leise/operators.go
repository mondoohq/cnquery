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
		"+":      nil,
		"-":      nil,
		"*":      nil,
		"/":      nil,
		"%":      nil,
		"=":      nil,
		"||":     compileComparable,
		"&&":     compileComparable,
		"if":     compileIf,
		"expect": compileExpect,
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

func compileComparable(c *compiler, id string, call *parser.Call, res *llx.CodeBundle) (types.Type, error) {
	if call == nil {
		return types.Nil, errors.New("comparable needs a function call")
	}

	if call.Function == nil {
		return types.Nil, errors.New("comparable needs a function call")
	}
	if len(call.Function) != 2 {
		if len(call.Function) != 2 {
			return types.Nil, errors.New("missing arguments")
		}
		return types.Nil, errors.New("too many arguments")
	}

	a := call.Function[0]
	b := call.Function[1]
	if a.Name != "" || b.Name != "" {
		return types.Nil, errors.New("calling operations with named arguments is not supported")
	}

	leftRef, err := c.compileAndAddExpression(a.Value)
	if err != nil {
		return types.Nil, err
	}
	left := c.Result.Code.Code[leftRef-1]

	right, err := c.compileExpression(b.Value)
	if err != nil {
		return types.Nil, err
	}

	if left == nil {
		log.Fatal().Msgf("left is nil: %d %#v", leftRef, c.Result.Code.Code[leftRef-1])
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

func compileIf(c *compiler, id string, call *parser.Call, res *llx.CodeBundle) (types.Type, error) {
	if call == nil {
		return types.Nil, errors.New("need conditional arguments for if-clause")
	}
	if call == nil || len(call.Function) < 1 {
		return types.Nil, errors.New("missing parameters for '" + id + "', it requires 1")
	}

	arg := call.Function[0]
	if arg.Name != "" {
		return types.Nil, errors.New("called '" + id + "' with a named argument, which is not supported")
	}

	argValue, err := c.compileExpression(arg.Value)
	if err != nil {
		return types.Nil, err
	}

	res.Code.AddChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   id,
		Function: &llx.Function{
			Type: string(types.Nil),
			Args: []*llx.Primitive{argValue},
		},
	})
	res.Code.Entrypoints = append(res.Code.Entrypoints, res.Code.ChunkIndex())

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
