package llx

import (
	"errors"
	"strconv"
	"sync"

	"go.mondoo.com/cnquery/types"
)

// mapFunctions are all the handlers for builtin array methods
var mapFunctionsV1 map[string]chunkHandlerV1

func mapGetIndexV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
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
	childType := bind.Type.Child()

	if bind.Value == nil {
		return &RawData{
			Type:  childType,
			Value: nil,
		}, 0, nil
	}

	m, ok := bind.Value.(map[string]interface{})
	if !ok {
		return nil, 0, errors.New("failed to typecast " + bind.Type.Label() + " into map")
	}
	return &RawData{
		Type:  childType,
		Value: m[key],
	}, 0, nil
}

func mapLengthV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Int}, 0, nil
	}

	arr, ok := bind.Value.(map[string]interface{})
	if !ok {
		return nil, 0, errors.New("failed to typecast " + bind.Type.Label() + " into map")
	}
	return IntData(int64(len(arr))), 0, nil
}

func _mapWhereV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32, invert bool) (*RawData, int32, error) {
	// where(array, function)
	itemsRef := chunk.Function.Args[0]
	items, rref, err := c.resolveValue(itemsRef, ref)
	if err != nil || rref > 0 {
		return nil, rref, err
	}

	if items.Value == nil {
		return &RawData{Type: items.Type}, 0, nil
	}

	list := items.Value.(map[string]interface{})
	if len(list) == 0 {
		return items, 0, nil
	}

	arg1 := chunk.Function.Args[1]
	if types.Type(arg1.Type).Underlying() != types.FunctionLike {
		return nil, 0, errors.New("cannot call 'where' on a map without a filter function")
	}

	fref, ok := arg1.RefV1()
	if !ok {
		return nil, 0, errors.New("failed to retrieve function reference of 'where' call")
	}

	f := c.code.Functions[fref-1]
	valueType := items.Type.Child()
	resMap := map[string]interface{}{}
	found := map[string]struct{}{}
	finishedResults := 0
	l := sync.Mutex{}
	for it := range list {
		key := it
		err := c.runFunctionBlock([]*RawData{{Type: types.String, Value: key}, {Type: valueType, Value: list[key]}}, f, func(res *RawResult) {
			done := func() bool {
				l.Lock()
				defer l.Unlock()

				if _, ok := found[key]; ok {
					return false
				}
				found[key] = struct{}{}
				finishedResults++

				isTruthy, _ := res.Data.IsTruthy()
				if isTruthy == !invert {
					resMap[key] = list[key]
				}

				return finishedResults == len(list)
			}()

			if done {
				data := &RawData{
					Type:  bind.Type,
					Value: resMap,
				}
				c.cache.Store(ref, &stepCache{
					Result:   data,
					IsStatic: false,
				})
				c.triggerChain(ref, data)
			}
		})
		if err != nil {
			return nil, 0, err
		}
	}

	return nil, 0, nil
}

func mapWhereV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return _mapWhereV1(c, bind, chunk, ref, false)
}

func mapWhereNotV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return _mapWhereV1(c, bind, chunk, ref, true)
}

func mapBlockCallV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return c.runBlock(bind, chunk.Function.Args[0], nil, ref)
}

func mapKeysV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{
			Type:  types.Array(types.Dict),
			Error: errors.New("Failed to get keys of `null`"),
		}, 0, nil
	}

	m, ok := bind.Value.(map[string]interface{})
	if !ok {
		return nil, 0, errors.New("failed to typecast " + bind.Type.Label() + " into map")
	}

	res := make([]interface{}, len(m))
	var i int
	for key := range m {
		res[i] = key
		i++
	}

	return ArrayData(res, types.String), 0, nil
}

func mapValuesV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{
			Type:  types.Array(types.Dict),
			Error: errors.New("Failed to get values of `null`"),
		}, 0, nil
	}

	m, ok := bind.Value.(map[string]interface{})
	if !ok {
		return nil, 0, errors.New("failed to typecast " + bind.Type.Label() + " into map")
	}

	res := make([]interface{}, len(m))
	var i int
	for _, value := range m {
		res[i] = value
		i++
	}

	return ArrayData(res, types.Dict), 0, nil
}

func dictGetIndexV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: bind.Type}, 0, nil
	}

	switch x := bind.Value.(type) {
	case []interface{}:
		args := chunk.Function.Args

		// TODO: all this needs to go into the compile phase
		if len(args) < 1 {
			return nil, 0, errors.New("Called [] with " + strconv.Itoa(len(args)) + " arguments, only 1 supported.")
		}
		if len(args) > 1 {
			return nil, 0, errors.New("Called [] with " + strconv.Itoa(len(args)) + " arguments, only 1 supported.")
		}
		t := types.Type(args[0].Type)
		if t != types.Int {
			return nil, 0, errors.New("Called [] with wrong type " + t.Label())
		}
		// ^^ TODO

		key := int(bytes2int(args[0].Value))
		return &RawData{
			Value: x[key],
			Type:  bind.Type,
		}, 0, nil

	case map[string]interface{}:
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
		return &RawData{
			Value: x[key],
			Type:  bind.Type,
		}, 0, nil
	default:
		return nil, 0, errors.New("dict value does not support accessor `[]`")
	}
}

func dictLengthV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: bind.Type}, 0, nil
	}

	switch x := bind.Value.(type) {
	case string:
		return IntData(int64(len(x))), 0, nil
	case []interface{}:
		return IntData(int64(len(x))), 0, nil
	case map[string]interface{}:
		return IntData(int64(len(x))), 0, nil
	default:
		return nil, 0, errors.New("dict value does not support field `length`")
	}
}

func dictNotEmptyV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return BoolFalse, 0, nil
	}

	switch x := bind.Value.(type) {
	case string:
		return BoolData(len(x) != 0), 0, nil
	case []interface{}:
		return BoolData(len(x) != 0), 0, nil
	case map[string]interface{}:
		return BoolData(len(x) != 0), 0, nil
	default:
		return nil, 0, errors.New("dict value does not support field `notEmpty`")
	}
}

func dictBlockCallV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	switch bind.Value.(type) {
	case []interface{}:
		return arrayBlockListV1(c, bind, chunk, ref)
	default:
		return c.runBlock(bind, chunk.Function.Args[0], nil, ref)
	}
}

func dictCamelcaseV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	_, ok := bind.Value.(string)
	if !ok {
		return nil, 0, errors.New("dict value does not support field `downcase`")
	}

	return stringCamelcaseV1(c, bind, chunk, ref)
}

func dictDowncaseV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	_, ok := bind.Value.(string)
	if !ok {
		return nil, 0, errors.New("dict value does not support field `downcase`")
	}

	return stringDowncaseV1(c, bind, chunk, ref)
}

func dictUpcaseV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	_, ok := bind.Value.(string)
	if !ok {
		return nil, 0, errors.New("dict value does not support field `upcase`")
	}

	return stringUpcaseV1(c, bind, chunk, ref)
}

func dictLinesV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	_, ok := bind.Value.(string)
	if !ok {
		return nil, 0, errors.New("dict value does not support field `lines`")
	}

	return stringLinesV1(c, bind, chunk, ref)
}

func dictSplitV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	_, ok := bind.Value.(string)
	if !ok {
		return nil, 0, errors.New("dict value does not support field `split`")
	}

	return stringSplitV1(c, bind, chunk, ref)
}

func dictTrimV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	_, ok := bind.Value.(string)
	if !ok {
		return nil, 0, errors.New("dict value does not support field `trim`")
	}

	return stringTrimV1(c, bind, chunk, ref)
}

func dictKeysV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{
			Type:  types.Array(types.Dict),
			Error: errors.New("Failed to get keys of `null`"),
		}, 0, nil
	}

	m, ok := bind.Value.(map[string]interface{})
	if !ok {
		return nil, 0, errors.New("dict value does not support field `keys`")
	}

	res := make([]interface{}, len(m))
	var i int
	for key := range m {
		res[i] = key
		i++
	}

	return ArrayData(res, types.String), 0, nil
}

func dictValuesV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{
			Type:  types.Array(types.Dict),
			Error: errors.New("Failed to get values of `null`"),
		}, 0, nil
	}

	m, ok := bind.Value.(map[string]interface{})
	if !ok {
		return nil, 0, errors.New("dict value does not support field `values`")
	}

	res := make([]interface{}, len(m))
	var i int
	for _, value := range m {
		res[i] = value
		i++
	}

	return ArrayData(res, types.Dict), 0, nil
}

func _dictWhereV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32, inverted bool) (*RawData, int32, error) {
	// where(array, function)
	itemsRef := chunk.Function.Args[0]
	items, rref, err := c.resolveValue(itemsRef, ref)
	if err != nil || rref > 0 {
		return nil, rref, err
	}

	if items.Value == nil {
		return &RawData{Type: items.Type}, 0, nil
	}

	list, ok := items.Value.([]interface{})
	if !ok {
		return nil, 0, errors.New("failed to call dict.where on a non-list value")
	}

	if len(list) == 0 {
		return items, 0, nil
	}

	arg1 := chunk.Function.Args[1]
	fref, ok := arg1.RefV1()
	if !ok {
		return nil, 0, errors.New("Failed to retrieve function reference of 'where' call")
	}

	f := c.code.Functions[fref-1]
	ct := items.Type.Child()
	filteredList := map[int]interface{}{}
	finishedResults := 0
	l := sync.Mutex{}
	for it := range list {
		i := it
		err := c.runFunctionBlock([]*RawData{{Type: ct, Value: list[i]}}, f, func(res *RawResult) {
			resList := func() []interface{} {
				l.Lock()
				defer l.Unlock()

				_, ok := filteredList[i]
				if !ok {
					finishedResults++
				}

				isTruthy, _ := res.Data.IsTruthy()
				if isTruthy == !inverted {
					filteredList[i] = list[i]
				} else {
					filteredList[i] = nil
				}

				if finishedResults == len(list) {
					resList := []interface{}{}
					for j := 0; j < len(filteredList); j++ {
						k := filteredList[j]
						if k != nil {
							resList = append(resList, k)
						}
					}
					return resList
				}
				return nil
			}()

			if resList != nil {
				data := &RawData{
					Type:  bind.Type,
					Value: resList,
				}
				c.cache.Store(ref, &stepCache{
					Result:   data,
					IsStatic: false,
				})
				c.triggerChain(ref, data)
			}
		})
		if err != nil {
			return nil, 0, err
		}
	}

	return nil, 0, nil
}

func dictWhereV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return _dictWhereV1(c, bind, chunk, ref, false)
}

func dictWhereNotV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return _dictWhereV1(c, bind, chunk, ref, true)
}

func dictAllV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Bool}, 0, nil
	}

	filteredList, ok := bind.Value.([]interface{})
	if !ok {
		return nil, 0, errors.New("failed to call dict assertion on a non-list value")
	}

	if len(filteredList) != 0 {
		return BoolFalse, 0, nil
	}
	return BoolTrue, 0, nil
}

func dictNoneV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Bool}, 0, nil
	}

	filteredList, ok := bind.Value.([]interface{})
	if !ok {
		return nil, 0, errors.New("failed to call dict assertion on a non-list value")
	}

	if len(filteredList) != 0 {
		return BoolFalse, 0, nil
	}
	return BoolTrue, 0, nil
}

func dictAnyV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Bool}, 0, nil
	}

	filteredList, ok := bind.Value.([]interface{})
	if !ok {
		return nil, 0, errors.New("failed to call dict assertion on a non-list value")
	}

	if len(filteredList) == 0 {
		return BoolFalse, 0, nil
	}
	return BoolTrue, 0, nil
}

func dictOneV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Bool}, 0, nil
	}

	filteredList, ok := bind.Value.([]interface{})
	if !ok {
		return nil, 0, errors.New("failed to call dict assertion on a non-list value")
	}

	if len(filteredList) != 1 {
		return BoolFalse, 0, nil
	}
	return BoolTrue, 0, nil
}

func dictMapV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	// map(array, function)
	itemsRef := chunk.Function.Args[0]
	items, rref, err := c.resolveValue(itemsRef, ref)
	if err != nil || rref > 0 {
		return nil, rref, err
	}

	if items.Value == nil {
		return &RawData{Type: items.Type}, 0, nil
	}

	list, ok := items.Value.([]interface{})
	if !ok {
		return nil, 0, errors.New("failed to call dict.map on a non-list value")
	}

	if len(list) == 0 {
		return items, 0, nil
	}

	arg1 := chunk.Function.Args[1]
	fref, ok := arg1.RefV1()
	if !ok {
		return nil, 0, errors.New("Failed to retrieve function reference of 'map' call")
	}

	f := c.code.Functions[fref-1]
	ct := items.Type.Child()
	mappedType := types.Unset
	resMap := map[int]interface{}{}
	finishedResults := 0
	l := sync.Mutex{}
	for it := range list {
		i := it
		err := c.runFunctionBlock([]*RawData{{Type: ct, Value: list[i]}}, f, func(res *RawResult) {
			resList := func() []interface{} {
				l.Lock()
				defer l.Unlock()

				_, ok := resMap[i]
				if !ok {
					finishedResults++
					resMap[i] = res.Data.Value
					mappedType = res.Data.Type
				}

				if finishedResults == len(list) {
					resList := []interface{}{}
					for j := 0; j < len(resMap); j++ {
						k := resMap[j]
						if k != nil {
							resList = append(resList, k)
						}
					}
					return resList
				}
				return nil
			}()

			if resList != nil {
				data := &RawData{
					Type:  types.Array(mappedType),
					Value: resList,
				}
				c.cache.Store(ref, &stepCache{
					Result:   data,
					IsStatic: false,
				})
				c.triggerChain(ref, data)
			}
		})
		if err != nil {
			return nil, 0, err
		}
	}

	return nil, 0, nil
}

func dictContainsStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	argRef := chunk.Function.Args[0]
	arg, rref, err := c.resolveValue(argRef, ref)
	if err != nil || rref > 0 {
		return nil, rref, err
	}

	if arg.Value == nil {
		return BoolFalse, 0, nil
	}

	ok := anyContainsString(bind.Value, arg.Value.(string))
	return BoolData(ok), 0, nil
}

func dictContainsIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	argRef := chunk.Function.Args[0]
	arg, rref, err := c.resolveValue(argRef, ref)
	if err != nil || rref > 0 {
		return nil, rref, err
	}

	if arg.Value == nil {
		return BoolFalse, 0, nil
	}

	val := strconv.FormatInt(arg.Value.(int64), 10)

	ok := anyContainsString(bind.Value, val)
	return BoolData(ok), 0, nil
}

func dictContainsArrayStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	switch bind.Value.(type) {
	case string:
		return stringContainsArrayStringV1(c, bind, chunk, ref)
	default:
		return nil, 0, errors.New("dict value does not support field `contains`")
	}
}

func dictContainsArrayIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	switch bind.Value.(type) {
	case string:
		return stringContainsArrayIntV1(c, bind, chunk, ref)
	default:
		return nil, 0, errors.New("dict value does not support field `contains`")
	}
}

func dictFindV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	switch bind.Value.(type) {
	case string:
		return stringFindV1(c, bind, chunk, ref)
	default:
		return nil, 0, errors.New("dict value does not support field `find`")
	}
}

// map &&/||

func arrayAndMapV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opArrayAndMap)
}

func arrayOrMapV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opArrayOrMap)
}

func mapAndArrayV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opMapAndArray)
}

func mapOrArrayV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opMapOrArray)
}

func mapAndMapV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opMapAndMap)
}

func mapOrMapV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opMapOrMap)
}

// dict ==/!= nil

func dictCmpNilV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opDictCmpNil)
}

func dictNotNilV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opDictCmpNil)
}

func nilCmpDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opNilCmpDict)
}

func nilNotDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opNilCmpDict)
}

// dict ==/!= bool

func dictCmpBoolV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opDictCmpBool)
}

func dictNotBoolV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opDictCmpBool)
}

func boolCmpDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opBoolCmpDict)
}

func boolNotDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opBoolCmpDict)
}

// dict ==/!= int   (embedded: string + float)

func dictCmpIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opDictCmpInt)
}

func dictNotIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opDictCmpInt)
}

func intCmpDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opIntCmpDict)
}

func intNotDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opIntCmpDict)
}

// dict ==/!= float

func dictCmpFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opDictCmpFloat)
}

func dictNotFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opDictCmpFloat)
}

func floatCmpDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opFloatCmpDict)
}

func floatNotDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opFloatCmpDict)
}

// dict ==/!= string

func dictCmpStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opDictCmpString)
}

func dictNotStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opDictCmpString)
}

func stringCmpDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opStringCmpDict)
}

func stringNotDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opStringCmpDict)
}

// dict ==/!= regex

func dictCmpRegexV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opDictCmpRegex)
}

func dictNotRegexV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opDictCmpRegex)
}

func regexCmpDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opRegexCmpDict)
}

func regexNotDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opRegexCmpDict)
}

// dict ==/!= arrays

func dictCmpArrayV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opDictCmpArray)
}

func dictNotArrayV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opDictCmpArray)
}

func dictCmpStringarrayV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, opDictCmpStringarray)
}

func dictNotStringarrayV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, opDictCmpStringarray)
}

func dictCmpBoolarrayV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, opDictCmpStringarray)
}

func dictNotBoolarrayV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, opDictCmpStringarray)
}

func dictCmpIntarrayV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, opDictCmpIntarray)
}

func dictNotIntarrayV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, opDictCmpIntarray)
}

func dictCmpFloatarrayV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, opDictCmpFloatarray)
}

func dictNotFloatarrayV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, opDictCmpFloatarray)
}

// dict ==/!= dict

func dictCmpDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opDictCmpDict)
}

func dictNotDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opDictCmpDict)
}

// dict </>/<=/>= int

func dictLTIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, opDictLTInt)
}

func dictLTEIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, opDictLTEInt)
}

func dictGTIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, opDictGTInt)
}

func dictGTEIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, opDictGTEInt)
}

func intLTDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, opIntLTDict)
}

func intLTEDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, opIntLTEDict)
}

func intGTDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, opIntLTEDict)
}

func intGTEDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, opIntLTDict)
}

// dict </>/<=/>= float

func dictLTFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, opDictLTFloat)
}

func dictLTEFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, opDictLTEFloat)
}

func dictGTFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, opDictGTFloat)
}

func dictGTEFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, opDictGTEFloat)
}

func floatLTDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, opFloatLTDict)
}

func floatLTEDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, opFloatLTEDict)
}

func floatGTDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, opFloatGTDict)
}

func floatGTEDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, opFloatGTEDict)
}

// dict </>/<=/>= string

func dictLTStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, opDictLTString)
}

func dictLTEStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, opDictLTEString)
}

func dictGTStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, opDictGTString)
}

func dictGTEStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, opDictGTEString)
}

func stringLTDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, opStringLTDict)
}

func stringLTEDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, opStringLTEDict)
}

func stringGTDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, opStringGTDict)
}

func stringGTEDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, opStringGTEDict)
}

// dict </>/<=/>= dict

func dictLTDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		switch x := right.(type) {
		case int64:
			return opDictLTInt(left, x)
		case float64:
			return opDictLTFloat(left, x)
		case string:
			return opDictLTString(left, x)
		default:
			return &RawData{Error: errors.New("type conflict for '<'"), Type: types.Bool}
		}
	})
}

func dictLTEDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		switch x := right.(type) {
		case int64:
			return opDictLTEInt(left, x)
		case float64:
			return opDictLTEFloat(left, x)
		case string:
			return opDictLTEString(left, x)
		default:
			return &RawData{Error: errors.New("type conflict for '<='"), Type: types.Bool}
		}
	})
}

func dictGTDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		switch x := right.(type) {
		case int64:
			return opDictLTEInt(left, x)
		case float64:
			return opDictLTEFloat(left, x)
		case string:
			return opDictLTString(left, x)
		default:
			return &RawData{Error: errors.New("type conflict for '>'"), Type: types.Bool}
		}
	})
}

func dictGTEDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		switch x := right.(type) {
		case int64:
			return opDictLTInt(left, x)
		case float64:
			return opDictLTFloat(left, x)
		case string:
			return opDictLTString(left, x)
		default:
			return &RawData{Error: errors.New("type conflict for '>='"), Type: types.Bool}
		}
	})
}

// dict && / || ...

// ... bool

func boolAndDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opBoolAndDict)
}

func boolOrDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opBoolOrDict)
}

func dictAndBoolV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opDictAndBool)
}

func dictOrBoolV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opDictOrBool)
}

// ... int

func intAndDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opIntAndDict)
}

func intOrDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opIntOrDict)
}

func dictAndIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opDictAndInt)
}

func dictOrIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opDictOrInt)
}

// ... float

func floatAndDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opFloatAndDict)
}

func floatOrDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opFloatOrDict)
}

func dictAndFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opDictAndFloat)
}

func dictOrFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opDictOrFloat)
}

// ... string

func stringAndDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opStringAndDict)
}

func stringOrDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opStringOrDict)
}

func dictAndStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opDictAndString)
}

func dictOrStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opDictOrString)
}

// ... regex

func regexAndDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opRegexAndDict)
}

func regexOrDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opRegexOrDict)
}

func dictAndRegexV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opDictAndRegex)
}

func dictOrRegexV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opDictOrRegex)
}

// ... time
// note: time cannot be falsy

func timeAndDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opTimeAndDict)
}

func timeOrDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opTimeOrDict)
}

func dictAndTimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opDictAndTime)
}

func dictOrTimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opDictOrTime)
}

// ... dict

func dictAndDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opDictAndDict)
}

func dictOrDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opDictOrDict)
}

// ... array

func dictAndArrayV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opDictAndArray)
}

func dictOrArrayV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opDictOrArray)
}

func arrayAndDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opArrayAndDict)
}

func arrayOrDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opArrayOrDict)
}

// ... map

func dictAndMapV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opDictAndMap)
}

func dictOrMapV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opDictOrMap)
}

func mapAndDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opMapAndDict)
}

func mapOrDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opMapOrDict)
}

// dict + - * /

func dictPlusStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		r := right.(string)

		switch l := left.(type) {
		case string:
			return StringData(l + r)
		default:
			return &RawData{
				Type:  types.Nil,
				Value: nil,
				Error: errors.New("dict value does not support `+` operation with string"),
			}
		}
	})
}

func stringPlusDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		l := left.(string)

		switch r := right.(type) {
		case string:
			return StringData(l + r)
		default:
			return &RawData{
				Type:  types.Nil,
				Value: nil,
				Error: errors.New("dict value does not support `+` operation with string"),
			}
		}
	})
}

func intPlusDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		l := left.(int64)

		switch r := right.(type) {
		case int64:
			return IntData(l + r)
		case float64:
			return FloatData(float64(l) + r)
		default:
			return &RawData{
				Type:  types.Nil,
				Value: nil,
				Error: errors.New("right side of `+` operation is not number"),
			}
		}
	})
}

func dictPlusIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		r := right.(int64)

		switch l := left.(type) {
		case int64:
			return IntData(l + r)
		case float64:
			return FloatData(l + float64(r))
		default:
			return &RawData{
				Type:  types.Nil,
				Value: nil,
				Error: errors.New("left side of `+` operation is not number"),
			}
		}
	})
}

func floatPlusDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		l := left.(float64)

		switch r := right.(type) {
		case int64:
			return FloatData(l + float64(r))
		case float64:
			return FloatData(l + r)
		default:
			return &RawData{
				Type:  types.Nil,
				Value: nil,
				Error: errors.New("right side of `+` operation is not number"),
			}
		}
	})
}

func dictPlusFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		r := right.(float64)

		switch l := left.(type) {
		case int64:
			return FloatData(float64(l) + r)
		case float64:
			return FloatData(l + r)
		default:
			return &RawData{
				Type:  types.Nil,
				Value: nil,
				Error: errors.New("left side of `+` operation is not number"),
			}
		}
	})
}

func intMinusDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		l := left.(int64)

		switch r := right.(type) {
		case int64:
			return IntData(l - r)
		case float64:
			return FloatData(float64(l) - r)
		default:
			return &RawData{
				Type:  types.Nil,
				Value: nil,
				Error: errors.New("right side of `-` operation is not number"),
			}
		}
	})
}

func dictMinusIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		r := right.(int64)

		switch l := left.(type) {
		case int64:
			return IntData(l - r)
		case float64:
			return FloatData(l - float64(r))
		default:
			return &RawData{
				Type:  types.Nil,
				Value: nil,
				Error: errors.New("left side of `-` operation is not number"),
			}
		}
	})
}

func floatMinusDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		l := left.(float64)

		switch r := right.(type) {
		case int64:
			return FloatData(l - float64(r))
		case float64:
			return FloatData(l - r)
		default:
			return &RawData{
				Type:  types.Nil,
				Value: nil,
				Error: errors.New("right side of `-` operation is not number"),
			}
		}
	})
}

func dictMinusFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		r := right.(float64)

		switch l := left.(type) {
		case int64:
			return FloatData(float64(l) - r)
		case float64:
			return FloatData(l - r)
		default:
			return &RawData{
				Type:  types.Nil,
				Value: nil,
				Error: errors.New("left side of `-` operation is not number"),
			}
		}
	})
}

func intTimesDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		l := left.(int64)

		switch r := right.(type) {
		case int64:
			return IntData(l * r)
		case float64:
			return FloatData(float64(l) * r)
		default:
			return &RawData{
				Type:  types.Nil,
				Value: nil,
				Error: errors.New("right side of `*` operation is not number"),
			}
		}
	})
}

func dictTimesIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		r := right.(int64)

		switch l := left.(type) {
		case int64:
			return IntData(l * r)
		case float64:
			return FloatData(l * float64(r))
		default:
			return &RawData{
				Type:  types.Nil,
				Value: nil,
				Error: errors.New("left side of `*` operation is not number"),
			}
		}
	})
}

func floatTimesDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		l := left.(float64)

		switch r := right.(type) {
		case int64:
			return FloatData(l * float64(r))
		case float64:
			return FloatData(l * r)
		default:
			return &RawData{
				Type:  types.Nil,
				Value: nil,
				Error: errors.New("right side of `*` operation is not number"),
			}
		}
	})
}

func dictTimesFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		r := right.(float64)

		switch l := left.(type) {
		case int64:
			return FloatData(float64(l) * r)
		case float64:
			return FloatData(l * r)
		default:
			return &RawData{
				Type:  types.Nil,
				Value: nil,
				Error: errors.New("left side of `*` operation is not number"),
			}
		}
	})
}

func intDividedDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		l := left.(int64)

		switch r := right.(type) {
		case int64:
			return IntData(l / r)
		case float64:
			return FloatData(float64(l) / r)
		default:
			return &RawData{
				Type:  types.Nil,
				Value: nil,
				Error: errors.New("right side of `/` operation is not number"),
			}
		}
	})
}

func dictDividedIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		r := right.(int64)

		switch l := left.(type) {
		case int64:
			return IntData(l / r)
		case float64:
			return FloatData(l / float64(r))
		default:
			return &RawData{
				Type:  types.Nil,
				Value: nil,
				Error: errors.New("left side of `/` operation is not number"),
			}
		}
	})
}

func floatDividedDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		l := left.(float64)

		switch r := right.(type) {
		case int64:
			return FloatData(l / float64(r))
		case float64:
			return FloatData(l / r)
		default:
			return &RawData{
				Type:  types.Nil,
				Value: nil,
				Error: errors.New("right side of `/` operation is not number"),
			}
		}
	})
}

func dictDividedFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		r := right.(float64)

		switch l := left.(type) {
		case int64:
			return FloatData(float64(l) / r)
		case float64:
			return FloatData(l / r)
		default:
			return &RawData{
				Type:  types.Nil,
				Value: nil,
				Error: errors.New("left side of `/` operation is not number"),
			}
		}
	})
}

func dictTimesTimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		switch l := left.(type) {
		case int64:
			return opTimeTimesInt(right, l)
		case float64:
			return opTimeTimesFloat(right, l)
		default:
			return &RawData{
				Type:  types.Nil,
				Value: nil,
				Error: errors.New("left side of `*` operation is not compatible with `time`"),
			}
		}
	})
}

func timeTimesDictV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		switch r := right.(type) {
		case int64:
			return opTimeTimesInt(left, r)
		case float64:
			return opTimeTimesFloat(left, r)
		default:
			return &RawData{
				Type:  types.Nil,
				Value: nil,
				Error: errors.New("left side of `*` operation is not compatible with `time`"),
			}
		}
	})
}
