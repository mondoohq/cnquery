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
