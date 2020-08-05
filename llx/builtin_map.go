package llx

import (
	"errors"
	"strconv"

	"go.mondoo.io/mondoo/types"
)

// mapFunctions are all the handlers for builtin array methods
var mapFunctions map[string]chunkHandler

func mapGetIndex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: bind.Type.Child()}, 0, nil
	}

	args := chunk.Function.Args

	// TODO: all this needs to go into the compile phase
	if len(args) < 1 {
		return nil, 0, errors.New("Called [] with " + strconv.Itoa(len(args)) + " arguments, only 1 supported.")
	}
	if len(args) > 1 {
		return nil, 0, errors.New("Called [] with " + strconv.Itoa(len(args)) + " arguments, only 1 supported.")
	}
	t := types.Type(args[0].Type)
	if t != types.String {
		return nil, 0, errors.New("Called [] with wrong type " + t.Label())
	}
	// ^^ TODO

	key := string(args[0].Value)

	m, ok := bind.Value.(map[string]interface{})
	if !ok {
		return nil, 0, errors.New("Failed to typecast into " + bind.Type.Label())
	}
	childType := bind.Type.Child()
	return &RawData{
		Type:  childType,
		Value: m[key],
	}, 0, nil
}

func mapLength(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Int}, 0, nil
	}

	arr, ok := bind.Value.(map[string]interface{})
	if !ok {
		return nil, 0, errors.New("failed to typecast into " + bind.Type.Label())
	}
	return IntData(int64(len(arr))), 0, nil
}

func mapBlockCall(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return c.runBlock(bind, chunk.Function.Args[0], ref)
}

func dictGetIndex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: bind.Type}, 0, nil
	}

	args := chunk.Function.Args

	// TODO: all this needs to go into the compile phase
	if len(args) < 1 {
		return nil, 0, errors.New("Called [] with " + strconv.Itoa(len(args)) + " arguments, only 1 supported.")
	}
	if len(args) > 1 {
		return nil, 0, errors.New("Called [] with " + strconv.Itoa(len(args)) + " arguments, only 1 supported.")
	}
	t := types.Type(args[0].Type)
	if t != types.String {
		return nil, 0, errors.New("Called [] with wrong type " + t.Label())
	}
	// ^^ TODO

	key := string(args[0].Value)

	m, ok := bind.Value.(map[string]interface{})
	if !ok {
		return nil, 0, errors.New("Failed to typecast into " + bind.Type.Label())
	}

	return &RawData{
		Type:  bind.Type,
		Value: m[key],
	}, 0, nil
}

func dictLength(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: bind.Type}, 0, nil
	}

	arr, ok := bind.Value.(map[string]interface{})
	if !ok {
		return nil, 0, errors.New("failed to typecast into " + bind.Type.Label())
	}
	return IntData(int64(len(arr))), 0, nil
}

func dictBlockCall(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return c.runBlock(bind, chunk.Function.Args[0], ref)
}

// dict ==/!= nil

func opDictCmpNil(left interface{}, right interface{}) bool {
	return left == nil
}

func opNilCmpDict(left interface{}, right interface{}) bool {
	return right == nil
}

func dictCmpNil(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opDictCmpNil)
}

func dictNotNil(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opDictCmpNil)
}

func nilCmpDict(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opNilCmpDict)
}

func nilNotDict(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opNilCmpDict)
}

// dict ==/!= int   (embedded: string + float)

func opDictCmpInt(left interface{}, right interface{}) bool {
	switch x := left.(type) {
	case int64:
		return x == right.(int64)
	case float64:
		return x == float64(left.(int64))
	case string:
		return opStringCmpInt(x, right)
	default:
		return false
	}
}

func opIntCmpDict(left interface{}, right interface{}) bool {
	switch x := right.(type) {
	case int64:
		return left.(int64) == x
	case float64:
		return float64(left.(int64)) == x
	case string:
		return opIntCmpString(left, x)
	default:
		return false
	}
}

func dictCmpInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opDictCmpInt)
}

func dictNotInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opDictCmpInt)
}

func intCmpDict(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opIntCmpDict)
}

func intNotDict(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opIntCmpDict)
}

// dict ==/!= float

func opDictCmpFloat(left interface{}, right interface{}) bool {
	switch x := left.(type) {
	case int64:
		return float64(x) == right.(float64)
	case float64:
		return x == right.(float64)
	case string:
		return opStringCmpFloat(x, right)
	default:
		return false
	}
}

func opFloatCmpDict(left interface{}, right interface{}) bool {
	switch x := right.(type) {
	case int64:
		return left.(float64) == float64(x)
	case float64:
		return left.(float64) == x
	case string:
		return opFloatCmpString(left, x)
	default:
		return false
	}
}

func dictCmpFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opDictCmpFloat)
}

func dictNotFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opDictCmpFloat)
}

func floatCmpDict(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opFloatCmpDict)
}

func floatNotDict(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opFloatCmpDict)
}

// dict ==/!= string

func opDictCmpString(left interface{}, right interface{}) bool {
	l, ok := left.(string)
	if !ok {
		return false
	}
	return l == right.(string)
}

func opStringCmpDict(left interface{}, right interface{}) bool {
	r, ok := right.(string)
	if !ok {
		return false
	}
	return r == left.(string)
}

func dictCmpString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opDictCmpString)
}

func dictNotString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opDictCmpString)
}

func stringCmpDict(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opStringCmpDict)
}

func stringNotDict(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opStringCmpDict)
}
