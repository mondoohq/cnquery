package llx

import (
	"errors"
	"strconv"
	"strings"
	"sync"

	"go.mondoo.io/mondoo/types"
)

// mapFunctions are all the handlers for builtin array methods
var mapFunctions map[string]chunkHandlerV2

func mapGetIndexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
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

func mapLengthV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Int}, 0, nil
	}

	arr, ok := bind.Value.(map[string]interface{})
	if !ok {
		return nil, 0, errors.New("failed to typecast " + bind.Type.Label() + " into map")
	}
	return IntData(int64(len(arr))), 0, nil
}

func _mapWhereV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64, invert bool) (*RawData, uint64, error) {
	// where(array, function)
	itemsRef := chunk.Function.Args[0]
	items, rref, err := e.resolveValue(itemsRef, ref)
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

	fref, ok := arg1.RefV2()
	if !ok {
		return nil, 0, errors.New("failed to retrieve function reference of 'where' call")
	}

	dref, err := e.ensureArgsResolved(chunk.Function.Args[2:], ref)
	if dref != 0 || err != nil {
		return nil, dref, err
	}

	valueType := items.Type.Child()
	resMap := map[string]interface{}{}
	found := map[string]struct{}{}
	finishedResults := 0
	l := sync.Mutex{}
	for it := range list {
		key := it
		err := e.runFunctionBlock([]*RawData{{Type: types.String, Value: key}, {Type: valueType, Value: list[key]}}, fref, func(res *RawResult) {
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
				e.cache.Store(ref, &stepCache{
					Result:   data,
					IsStatic: false,
				})
				e.triggerChain(ref, data)
			}
		})
		if err != nil {
			return nil, 0, err
		}
	}

	return nil, 0, nil
}

func mapWhereV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return _mapWhereV2(e, bind, chunk, ref, false)
}

func mapWhereNotV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return _mapWhereV2(e, bind, chunk, ref, true)
}

func mapBlockCallV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return e.runBlock(bind, chunk.Function.Args[0], nil, ref)
}

func mapKeysV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
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

func mapValuesV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
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

func dictGetIndexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
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

func dictLengthV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
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

func dictNotEmptyV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
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

func dictBlockCallV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	switch bind.Value.(type) {
	case []interface{}:
		return arrayBlockListV2(e, bind, chunk, ref)
	default:
		return e.runBlock(bind, chunk.Function.Args[0], chunk.Function.Args[1:], ref)
	}
}

func dictCamelcaseV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	_, ok := bind.Value.(string)
	if !ok {
		return nil, 0, errors.New("dict value does not support field `downcase`")
	}

	return stringCamelcaseV2(e, bind, chunk, ref)
}

func dictDowncaseV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	_, ok := bind.Value.(string)
	if !ok {
		return nil, 0, errors.New("dict value does not support field `downcase`")
	}

	return stringDowncaseV2(e, bind, chunk, ref)
}

func dictUpcaseV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	_, ok := bind.Value.(string)
	if !ok {
		return nil, 0, errors.New("dict value does not support field `upcase`")
	}

	return stringUpcaseV2(e, bind, chunk, ref)
}

func dictLinesV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	_, ok := bind.Value.(string)
	if !ok {
		return nil, 0, errors.New("dict value does not support field `lines`")
	}

	return stringLinesV2(e, bind, chunk, ref)
}

func dictSplitV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	_, ok := bind.Value.(string)
	if !ok {
		return nil, 0, errors.New("dict value does not support field `split`")
	}

	return stringSplitV2(e, bind, chunk, ref)
}

func dictTrimV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	_, ok := bind.Value.(string)
	if !ok {
		return nil, 0, errors.New("dict value does not support field `trim`")
	}

	return stringTrimV2(e, bind, chunk, ref)
}

func dictKeysV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
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

func dictValuesV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
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

func _dictWhereV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64, inverted bool) (*RawData, uint64, error) {
	// where(array, function)
	itemsRef := chunk.Function.Args[0]
	items, rref, err := e.resolveValue(itemsRef, ref)
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
	fref, ok := arg1.RefV2()
	if !ok {
		return nil, 0, errors.New("Failed to retrieve function reference of 'where' call")
	}

	dref, err := e.ensureArgsResolved(chunk.Function.Args[2:], ref)
	if dref != 0 || err != nil {
		return nil, dref, err
	}

	ct := items.Type.Child()
	filteredList := map[int]interface{}{}
	finishedResults := 0
	l := sync.Mutex{}
	for it := range list {
		i := it
		err := e.runFunctionBlock([]*RawData{{Type: ct, Value: list[i]}}, fref, func(res *RawResult) {
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
				e.cache.Store(ref, &stepCache{
					Result:   data,
					IsStatic: false,
				})
				e.triggerChain(ref, data)
			}
		})
		if err != nil {
			return nil, 0, err
		}
	}

	return nil, 0, nil
}

func dictWhereV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return _dictWhereV2(e, bind, chunk, ref, false)
}

func dictWhereNotV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return _dictWhereV2(e, bind, chunk, ref, true)
}

func dictAllV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
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

func dictNoneV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
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

func dictAnyV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
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

func dictOneV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
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

func dictMapV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	// map(array, function)
	itemsRef := chunk.Function.Args[0]
	items, rref, err := e.resolveValue(itemsRef, ref)
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
	fref, ok := arg1.RefV2()
	if !ok {
		return nil, 0, errors.New("Failed to retrieve function reference of 'map' call")
	}

	dref, err := e.ensureArgsResolved(chunk.Function.Args[2:], ref)
	if dref != 0 || err != nil {
		return nil, dref, err
	}

	ct := items.Type.Child()
	mappedType := types.Unset
	resMap := map[int]interface{}{}
	finishedResults := 0
	l := sync.Mutex{}
	for it := range list {
		i := it
		err := e.runFunctionBlock([]*RawData{{Type: ct, Value: list[i]}}, fref, func(res *RawResult) {
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
				e.cache.Store(ref, &stepCache{
					Result:   data,
					IsStatic: false,
				})
				e.triggerChain(ref, data)
			}
		})
		if err != nil {
			return nil, 0, err
		}
	}

	return nil, 0, nil
}

func anyContainsString(an interface{}, s string) bool {
	if an == nil {
		return false
	}

	switch x := an.(type) {
	case string:
		return strings.Contains(x, s)
	case []interface{}:
		for i := range x {
			if anyContainsString(x[i], s) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func dictContainsStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	argRef := chunk.Function.Args[0]
	arg, rref, err := e.resolveValue(argRef, ref)
	if err != nil || rref > 0 {
		return nil, rref, err
	}

	if arg.Value == nil {
		return BoolFalse, 0, nil
	}

	ok := anyContainsString(bind.Value, arg.Value.(string))
	return BoolData(ok), 0, nil
}

func dictContainsIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	argRef := chunk.Function.Args[0]
	arg, rref, err := e.resolveValue(argRef, ref)
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

func dictContainsArrayStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	switch bind.Value.(type) {
	case string:
		return stringContainsArrayStringV2(e, bind, chunk, ref)
	default:
		return nil, 0, errors.New("dict value does not support field `contains`")
	}
}

func dictContainsArrayIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	switch bind.Value.(type) {
	case string:
		return stringContainsArrayIntV2(e, bind, chunk, ref)
	default:
		return nil, 0, errors.New("dict value does not support field `contains`")
	}
}

func dictFindV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	switch bind.Value.(type) {
	case string:
		return stringFindV2(e, bind, chunk, ref)
	default:
		return nil, 0, errors.New("dict value does not support field `find`")
	}
}

// map &&/||

func opArrayAndMap(left interface{}, right interface{}) bool {
	return (len(left.([]interface{})) != 0) && (len(right.(map[string]interface{})) != 0)
}

func opArrayOrMap(left interface{}, right interface{}) bool {
	return (len(left.([]interface{})) != 0) || (len(right.(map[string]interface{})) != 0)
}

func opMapAndArray(left interface{}, right interface{}) bool {
	return (len(right.(map[string]interface{})) != 0) && (len(left.([]interface{})) != 0)
}

func opMapOrArray(left interface{}, right interface{}) bool {
	return (len(right.(map[string]interface{})) != 0) || (len(left.([]interface{})) != 0)
}

func arrayAndMapV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opArrayAndMap)
}

func arrayOrMapV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opArrayOrMap)
}

func mapAndArrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opMapAndArray)
}

func mapOrArrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opMapOrArray)
}

func opMapAndMap(left interface{}, right interface{}) bool {
	return (len(left.(map[string]interface{})) != 0) && (len(right.(map[string]interface{})) != 0)
}

func opMapOrMap(left interface{}, right interface{}) bool {
	return (len(left.(map[string]interface{})) != 0) || (len(right.(map[string]interface{})) != 0)
}

func mapAndMapV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opMapAndMap)
}

func mapOrMapV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opMapOrMap)
}

// dict ==/!= nil

func opDictCmpNil(left interface{}, right interface{}) bool {
	return left == nil
}

func opNilCmpDict(left interface{}, right interface{}) bool {
	return right == nil
}

func dictCmpNilV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opDictCmpNil)
}

func dictNotNilV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opDictCmpNil)
}

func nilCmpDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opNilCmpDict)
}

func nilNotDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opNilCmpDict)
}

// dict ==/!= bool

func opDictCmpBool(left interface{}, right interface{}) bool {
	switch x := left.(type) {
	case bool:
		return x == right.(bool)
	case string:
		return opStringCmpBool(x, right)
	default:
		return false
	}
}

func opBoolCmpDict(left interface{}, right interface{}) bool {
	switch x := right.(type) {
	case bool:
		return left.(bool) == x
	case string:
		return opBoolCmpString(left, x)
	default:
		return false
	}
}

func dictCmpBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opDictCmpBool)
}

func dictNotBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opDictCmpBool)
}

func boolCmpDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opBoolCmpDict)
}

func boolNotDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opBoolCmpDict)
}

// dict ==/!= int   (embedded: string + float)

func opDictCmpInt(left interface{}, right interface{}) bool {
	switch x := left.(type) {
	case int64:
		return x == right.(int64)
	case float64:
		return x == float64(right.(int64))
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

func dictCmpIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opDictCmpInt)
}

func dictNotIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opDictCmpInt)
}

func intCmpDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opIntCmpDict)
}

func intNotDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opIntCmpDict)
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

func dictCmpFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opDictCmpFloat)
}

func dictNotFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opDictCmpFloat)
}

func floatCmpDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opFloatCmpDict)
}

func floatNotDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opFloatCmpDict)
}

// dict ==/!= string

func opDictCmpString(left interface{}, right interface{}) bool {
	switch x := left.(type) {
	case string:
		return x == right.(string)
	case bool:
		return opBoolCmpString(x, right)
	case int64:
		return opIntCmpString(x, right)
	case float64:
		return opFloatCmpString(x, right)
	default:
		return false
	}
}

func opStringCmpDict(left interface{}, right interface{}) bool {
	switch x := right.(type) {
	case string:
		return left.(string) == x
	case bool:
		return opStringCmpBool(left, x)
	case int64:
		return opStringCmpInt(left, x)
	case float64:
		return opStringCmpFloat(left, x)
	default:
		return false
	}
}

func dictCmpStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opDictCmpString)
}

func dictNotStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opDictCmpString)
}

func stringCmpDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opStringCmpDict)
}

func stringNotDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opStringCmpDict)
}

// dict ==/!= regex

func opDictCmpRegex(left interface{}, right interface{}) bool {
	switch x := left.(type) {
	case string:
		return opStringCmpRegex(x, right)
	case int64:
		return opIntCmpRegex(x, right)
	case float64:
		return opFloatCmpRegex(x, right)
	default:
		return false
	}
}

func opRegexCmpDict(left interface{}, right interface{}) bool {
	switch x := right.(type) {
	case string:
		return opRegexCmpString(left, x)
	case int64:
		return opRegexCmpInt(left, x)
	case float64:
		return opRegexCmpFloat(left, x)
	default:
		return false
	}
}

func dictCmpRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opDictCmpRegex)
}

func dictNotRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opDictCmpRegex)
}

func regexCmpDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opRegexCmpDict)
}

func regexNotDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opRegexCmpDict)
}

// dict ==/!= arrays

func opDictCmpArray(left interface{}, right interface{}) bool {
	switch left.(type) {
	case string:
		return false
	default:
		return false
	}
}

func dictCmpArrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opDictCmpArray)
}

func dictNotArrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opDictCmpArray)
}

func opDictCmpStringarray(left *RawData, right *RawData) bool {
	switch left.Value.(type) {
	case string:
		return cmpArrayOne(right, left, opStringCmpString)
	default:
		return false
	}
}

func dictCmpStringarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, opDictCmpStringarray)
}

func dictNotStringarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, opDictCmpStringarray)
}

func opDictCmpBoolarray(left *RawData, right *RawData) bool {
	switch left.Value.(type) {
	case string:
		return cmpArrayOne(right, left, opBoolCmpString)
	default:
		return false
	}
}

func dictCmpBoolarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, opDictCmpStringarray)
}

func dictNotBoolarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, opDictCmpStringarray)
}

func opDictCmpIntarray(left *RawData, right *RawData) bool {
	switch left.Value.(type) {
	case string:
		return cmpArrayOne(right, left, opIntCmpString)
	default:
		return false
	}
}

func dictCmpIntarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, opDictCmpIntarray)
}

func dictNotIntarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, opDictCmpIntarray)
}

func opDictCmpFloatarray(left *RawData, right *RawData) bool {
	switch left.Value.(type) {
	case string:
		return cmpArrayOne(right, left, opFloatCmpString)
	default:
		return false
	}
}

func dictCmpFloatarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, opDictCmpFloatarray)
}

func dictNotFloatarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, opDictCmpFloatarray)
}

// dict ==/!= dict

func opDictCmpDict(left interface{}, right interface{}) bool {
	switch x := left.(type) {
	case bool:
		return opBoolCmpDict(x, right)
	case int64:
		return opIntCmpDict(x, right)
	case float64:
		return opFloatCmpDict(x, right)
	case string:
		return opStringCmpDict(x, right)
	default:
		return false
	}
}

func dictCmpDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opDictCmpDict)
}

func dictNotDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opDictCmpDict)
}

// dict </>/<=/>= int

func opDictLTInt(left interface{}, right interface{}) *RawData {
	switch x := left.(type) {
	case int64:
		return BoolData(x < right.(int64))
	case float64:
		return BoolData(x < float64(right.(int64)))
	case string:
		f, err := strconv.ParseInt(x, 10, 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + x + " as number"), Type: types.Bool}
		}
		return BoolData(f < right.(int64))
	default:
		return &RawData{Error: errors.New("type conflict for '<'"), Type: types.Bool}
	}
}

func opDictLTEInt(left interface{}, right interface{}) *RawData {
	switch x := left.(type) {
	case int64:
		return BoolData(x <= right.(int64))
	case float64:
		return BoolData(x <= float64(right.(int64)))
	case string:
		f, err := strconv.ParseInt(x, 10, 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + x + " as number"), Type: types.Bool}
		}
		return BoolData(f <= right.(int64))
	default:
		return &RawData{Error: errors.New("type conflict for '<='"), Type: types.Bool}
	}
}

func opDictGTInt(left interface{}, right interface{}) *RawData {
	switch x := left.(type) {
	case int64:
		return BoolData(x > right.(int64))
	case float64:
		return BoolData(x > float64(right.(int64)))
	case string:
		f, err := strconv.ParseInt(x, 10, 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + x + " as number"), Type: types.Bool}
		}
		return BoolData(f > right.(int64))
	default:
		return &RawData{Error: errors.New("type conflict for '>'"), Type: types.Bool}
	}
}

func opDictGTEInt(left interface{}, right interface{}) *RawData {
	switch x := left.(type) {
	case int64:
		return BoolData(x >= right.(int64))
	case float64:
		return BoolData(x >= float64(right.(int64)))
	case string:
		f, err := strconv.ParseInt(x, 10, 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + x + " as number"), Type: types.Bool}
		}
		return BoolData(f >= right.(int64))
	default:
		return &RawData{Error: errors.New("type conflict for '>='"), Type: types.Bool}
	}
}

func dictLTIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, opDictLTInt)
}

func dictLTEIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, opDictLTEInt)
}

func dictGTIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, opDictGTInt)
}

func dictGTEIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, opDictGTEInt)
}

func opIntLTDict(left interface{}, right interface{}) *RawData {
	switch x := right.(type) {
	case int64:
		return BoolData(left.(int64) < x)
	case float64:
		return BoolData(float64(left.(int64)) < x)
	case string:
		f, err := strconv.ParseInt(x, 10, 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + x + " as number"), Type: types.Bool}
		}
		return BoolData(left.(int64) < f)
	default:
		return &RawData{Error: errors.New("type conflict for '<'"), Type: types.Bool}
	}
}

func opIntLTEDict(left interface{}, right interface{}) *RawData {
	switch x := right.(type) {
	case int64:
		return BoolData(left.(int64) <= x)
	case float64:
		return BoolData(float64(left.(int64)) <= x)
	case string:
		f, err := strconv.ParseInt(x, 10, 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + x + " as number"), Type: types.Bool}
		}
		return BoolData(left.(int64) <= f)
	default:
		return &RawData{Error: errors.New("type conflict for '<='"), Type: types.Bool}
	}
}

func opIntGTDict(left interface{}, right interface{}) *RawData {
	switch x := right.(type) {
	case int64:
		return BoolData(left.(int64) > x)
	case float64:
		return BoolData(float64(left.(int64)) > x)
	case string:
		f, err := strconv.ParseInt(x, 10, 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + x + " as number"), Type: types.Bool}
		}
		return BoolData(left.(int64) > f)
	default:
		return &RawData{Error: errors.New("type conflict for '>'"), Type: types.Bool}
	}
}

func opIntGTEDict(left interface{}, right interface{}) *RawData {
	switch x := right.(type) {
	case int64:
		return BoolData(left.(int64) >= x)
	case float64:
		return BoolData(float64(left.(int64)) >= x)
	case string:
		f, err := strconv.ParseInt(x, 10, 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + x + " as number"), Type: types.Bool}
		}
		return BoolData(left.(int64) >= f)
	default:
		return &RawData{Error: errors.New("type conflict for '>='"), Type: types.Bool}
	}
}

func intLTDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, opIntLTDict)
}

func intLTEDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, opIntLTEDict)
}

func intGTDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, opIntLTEDict)
}

func intGTEDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, opIntLTDict)
}

// dict </>/<=/>= float

func opDictLTFloat(left interface{}, right interface{}) *RawData {
	switch x := left.(type) {
	case int64:
		return BoolData(float64(x) < right.(float64))
	case float64:
		return BoolData(x < right.(float64))
	case string:
		f, err := strconv.ParseFloat(x, 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + x + " as number"), Type: types.Bool}
		}
		return BoolData(f < right.(float64))
	default:
		return &RawData{Error: errors.New("type conflict for '<'"), Type: types.Bool}
	}
}

func opDictLTEFloat(left interface{}, right interface{}) *RawData {
	switch x := left.(type) {
	case int64:
		return BoolData(float64(x) <= right.(float64))
	case float64:
		return BoolData(x <= right.(float64))
	case string:
		f, err := strconv.ParseFloat(x, 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + x + " as number"), Type: types.Bool}
		}
		return BoolData(f <= right.(float64))
	default:
		return &RawData{Error: errors.New("type conflict for '<='"), Type: types.Bool}
	}
}

func opDictGTFloat(left interface{}, right interface{}) *RawData {
	switch x := left.(type) {
	case int64:
		return BoolData(float64(x) > right.(float64))
	case float64:
		return BoolData(x > right.(float64))
	case string:
		f, err := strconv.ParseFloat(x, 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + x + " as number"), Type: types.Bool}
		}
		return BoolData(f > right.(float64))
	default:
		return &RawData{Error: errors.New("type conflict for '>'"), Type: types.Bool}
	}
}

func opDictGTEFloat(left interface{}, right interface{}) *RawData {
	switch x := left.(type) {
	case int64:
		return BoolData(float64(x) >= right.(float64))
	case float64:
		return BoolData(x >= right.(float64))
	case string:
		f, err := strconv.ParseFloat(x, 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + x + " as number"), Type: types.Bool}
		}
		return BoolData(f >= right.(float64))
	default:
		return &RawData{Error: errors.New("type conflict for '>='"), Type: types.Bool}
	}
}

func dictLTFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, opDictLTFloat)
}

func dictLTEFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, opDictLTEFloat)
}

func dictGTFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, opDictGTFloat)
}

func dictGTEFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, opDictGTEFloat)
}

func opFloatLTDict(left interface{}, right interface{}) *RawData {
	switch x := right.(type) {
	case int64:
		return BoolData(left.(float64) < float64(x))
	case float64:
		return BoolData(left.(float64) < x)
	case string:
		f, err := strconv.ParseFloat(x, 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + x + " as number"), Type: types.Bool}
		}
		return BoolData(left.(float64) < f)
	default:
		return &RawData{Error: errors.New("type conflict for '<'"), Type: types.Bool}
	}
}

func opFloatLTEDict(left interface{}, right interface{}) *RawData {
	switch x := right.(type) {
	case int64:
		return BoolData(left.(float64) <= float64(x))
	case float64:
		return BoolData(left.(float64) <= x)
	case string:
		f, err := strconv.ParseFloat(x, 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + x + " as number"), Type: types.Bool}
		}
		return BoolData(left.(float64) <= f)
	default:
		return &RawData{Error: errors.New("type conflict for '<='"), Type: types.Bool}
	}
}

func opFloatGTDict(left interface{}, right interface{}) *RawData {
	switch x := right.(type) {
	case int64:
		return BoolData(left.(float64) > float64(x))
	case float64:
		return BoolData(left.(float64) > x)
	case string:
		f, err := strconv.ParseFloat(x, 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + x + " as number"), Type: types.Bool}
		}
		return BoolData(left.(float64) > f)
	default:
		return &RawData{Error: errors.New("type conflict for '>'"), Type: types.Bool}
	}
}

func opFloatGTEDict(left interface{}, right interface{}) *RawData {
	switch x := right.(type) {
	case int64:
		return BoolData(left.(float64) >= float64(x))
	case float64:
		return BoolData(left.(float64) >= x)
	case string:
		f, err := strconv.ParseFloat(x, 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + x + " as number"), Type: types.Bool}
		}
		return BoolData(left.(float64) >= f)
	default:
		return &RawData{Error: errors.New("type conflict for '>='"), Type: types.Bool}
	}
}

func floatLTDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, opFloatLTDict)
}

func floatLTEDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, opFloatLTEDict)
}

func floatGTDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, opFloatGTDict)
}

func floatGTEDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, opFloatGTEDict)
}

// dict </>/<=/>= string

func opDictLTString(left interface{}, right interface{}) *RawData {
	switch x := left.(type) {
	case int64:
		f, err := strconv.ParseInt(right.(string), 10, 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + right.(string) + " as number"), Type: types.Bool}
		}
		return BoolData(x < f)
	case float64:
		f, err := strconv.ParseFloat(right.(string), 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + right.(string) + " as number"), Type: types.Bool}
		}
		return BoolData(x < f)
	case string:
		return BoolData(x < right.(string))
	default:
		return &RawData{Error: errors.New("type conflict for '<'"), Type: types.Bool}
	}
}

func opDictLTEString(left interface{}, right interface{}) *RawData {
	switch x := left.(type) {
	case int64:
		f, err := strconv.ParseInt(right.(string), 10, 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + right.(string) + " as number"), Type: types.Bool}
		}
		return BoolData(x <= f)
	case float64:
		f, err := strconv.ParseFloat(right.(string), 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + right.(string) + " as number"), Type: types.Bool}
		}
		return BoolData(x <= f)
	case string:
		return BoolData(x <= right.(string))
	default:
		return &RawData{Error: errors.New("type conflict for '<='"), Type: types.Bool}
	}
}

func opDictGTString(left interface{}, right interface{}) *RawData {
	switch x := left.(type) {
	case int64:
		f, err := strconv.ParseInt(right.(string), 10, 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + right.(string) + " as number"), Type: types.Bool}
		}
		return BoolData(x > f)
	case float64:
		f, err := strconv.ParseFloat(right.(string), 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + right.(string) + " as number"), Type: types.Bool}
		}
		return BoolData(x > f)
	case string:
		return BoolData(x > right.(string))
	default:
		return &RawData{Error: errors.New("type conflict for '>'"), Type: types.Bool}
	}
}

func opDictGTEString(left interface{}, right interface{}) *RawData {
	switch x := left.(type) {
	case int64:
		f, err := strconv.ParseInt(right.(string), 10, 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + right.(string) + " as number"), Type: types.Bool}
		}
		return BoolData(x >= f)
	case float64:
		f, err := strconv.ParseFloat(right.(string), 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + right.(string) + " as number"), Type: types.Bool}
		}
		return BoolData(x >= f)
	case string:
		return BoolData(x >= right.(string))
	default:
		return &RawData{Error: errors.New("type conflict for '>='"), Type: types.Bool}
	}
}

func dictLTStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, opDictLTString)
}

func dictLTEStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, opDictLTEString)
}

func dictGTStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, opDictGTString)
}

func dictGTEStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, opDictGTEString)
}

func opStringLTDict(left interface{}, right interface{}) *RawData {
	switch x := right.(type) {
	case int64:
		f, err := strconv.ParseInt(left.(string), 10, 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + left.(string) + " as number"), Type: types.Bool}
		}
		return BoolData(f < x)
	case float64:
		f, err := strconv.ParseFloat(left.(string), 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + left.(string) + " as number"), Type: types.Bool}
		}
		return BoolData(f < x)
	case string:
		return BoolData(left.(string) < x)
	default:
		return &RawData{Error: errors.New("type conflict for '<'"), Type: types.Bool}
	}
}

func opStringLTEDict(left interface{}, right interface{}) *RawData {
	switch x := right.(type) {
	case int64:
		f, err := strconv.ParseInt(left.(string), 10, 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + left.(string) + " as number"), Type: types.Bool}
		}
		return BoolData(f <= x)
	case float64:
		f, err := strconv.ParseFloat(left.(string), 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + left.(string) + " as number"), Type: types.Bool}
		}
		return BoolData(f <= x)
	case string:
		return BoolData(left.(string) <= x)
	default:
		return &RawData{Error: errors.New("type conflict for '<='"), Type: types.Bool}
	}
}

func opStringGTDict(left interface{}, right interface{}) *RawData {
	switch x := right.(type) {
	case int64:
		f, err := strconv.ParseInt(left.(string), 10, 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + left.(string) + " as number"), Type: types.Bool}
		}
		return BoolData(f > x)
	case float64:
		f, err := strconv.ParseFloat(left.(string), 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + left.(string) + " as number"), Type: types.Bool}
		}
		return BoolData(f > x)
	case string:
		return BoolData(left.(string) > x)
	default:
		return &RawData{Error: errors.New("type conflict for '>'"), Type: types.Bool}
	}
}

func opStringGTEDict(left interface{}, right interface{}) *RawData {
	switch x := right.(type) {
	case int64:
		f, err := strconv.ParseInt(left.(string), 10, 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + left.(string) + " as number"), Type: types.Bool}
		}
		return BoolData(f >= x)
	case float64:
		f, err := strconv.ParseFloat(left.(string), 64)
		if err != nil {
			return &RawData{Error: errors.New("cannot parse " + left.(string) + " as number"), Type: types.Bool}
		}
		return BoolData(f >= x)
	case string:
		return BoolData(left.(string) >= x)
	default:
		return &RawData{Error: errors.New("type conflict for '>='"), Type: types.Bool}
	}
}

func stringLTDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, opStringLTDict)
}

func stringLTEDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, opStringLTEDict)
}

func stringGTDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, opStringGTDict)
}

func stringGTEDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, opStringGTEDict)
}

// dict </>/<=/>= dict

func dictLTDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
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

func dictLTEDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
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

func dictGTDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
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

func dictGTEDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
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

func truthyDict(value interface{}) bool {
	switch x := value.(type) {
	case bool:
		return x
	case int64:
		return x != 0
	case float64:
		return x != 0
	case string:
		return x != ""
	case []interface{}:
		return len(x) != 0
	case map[string]interface{}:
		return len(x) != 0
	default:
		return false
	}
}

// ... bool

func opBoolAndDict(left interface{}, right interface{}) bool {
	return left.(bool) && truthyDict(right)
}

func opBoolOrDict(left interface{}, right interface{}) bool {
	return left.(bool) || truthyDict(right)
}

func opDictAndBool(left interface{}, right interface{}) bool {
	return truthyDict(left) && right.(bool)
}

func opDictOrBool(left interface{}, right interface{}) bool {
	return truthyDict(left) || right.(bool)
}

func boolAndDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opBoolAndDict)
}

func boolOrDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opBoolOrDict)
}

func dictAndBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opDictAndBool)
}

func dictOrBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opDictOrBool)
}

// ... int

func opIntAndDict(left interface{}, right interface{}) bool {
	return left.(int64) != 0 && truthyDict(right)
}

func opIntOrDict(left interface{}, right interface{}) bool {
	return left.(int64) != 0 || truthyDict(right)
}

func opDictAndInt(left interface{}, right interface{}) bool {
	return truthyDict(left) && right.(int64) != 0
}

func opDictOrInt(left interface{}, right interface{}) bool {
	return truthyDict(left) || right.(int64) != 0
}

func intAndDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opIntAndDict)
}

func intOrDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opIntOrDict)
}

func dictAndIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opDictAndInt)
}

func dictOrIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opDictOrInt)
}

// ... float

func opFloatAndDict(left interface{}, right interface{}) bool {
	return left.(float64) != 0 && truthyDict(right)
}

func opFloatOrDict(left interface{}, right interface{}) bool {
	return left.(float64) != 0 || truthyDict(right)
}

func opDictAndFloat(left interface{}, right interface{}) bool {
	return truthyDict(left) && right.(float64) != 0
}

func opDictOrFloat(left interface{}, right interface{}) bool {
	return truthyDict(left) || right.(float64) != 0
}

func floatAndDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opFloatAndDict)
}

func floatOrDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opFloatOrDict)
}

func dictAndFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opDictAndFloat)
}

func dictOrFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opDictOrFloat)
}

// ... string

func opStringAndDict(left interface{}, right interface{}) bool {
	return left.(string) != "" && truthyDict(right)
}

func opStringOrDict(left interface{}, right interface{}) bool {
	return left.(string) != "" || truthyDict(right)
}

func opDictAndString(left interface{}, right interface{}) bool {
	return truthyDict(left) && right.(string) != ""
}

func opDictOrString(left interface{}, right interface{}) bool {
	return truthyDict(left) || right.(string) != ""
}

func stringAndDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opStringAndDict)
}

func stringOrDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opStringOrDict)
}

func dictAndStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opDictAndString)
}

func dictOrStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opDictOrString)
}

// ... regex

func opRegexAndDict(left interface{}, right interface{}) bool {
	return left.(string) != "" && truthyDict(right)
}

func opRegexOrDict(left interface{}, right interface{}) bool {
	return left.(string) != "" || truthyDict(right)
}

func opDictAndRegex(left interface{}, right interface{}) bool {
	return truthyDict(left) && right.(string) != ""
}

func opDictOrRegex(left interface{}, right interface{}) bool {
	return truthyDict(left) || right.(string) != ""
}

func regexAndDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opRegexAndDict)
}

func regexOrDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opRegexOrDict)
}

func dictAndRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opDictAndRegex)
}

func dictOrRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opDictOrRegex)
}

// ... time
// note: time cannot be falsy

func opTimeAndDict(left interface{}, right interface{}) bool {
	return truthyDict(right)
}

func opTimeOrDict(left interface{}, right interface{}) bool {
	return true
}

func opDictAndTime(left interface{}, right interface{}) bool {
	return truthyDict(left)
}

func opDictOrTime(left interface{}, right interface{}) bool {
	return true
}

func timeAndDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opTimeAndDict)
}

func timeOrDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opTimeOrDict)
}

func dictAndTimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opDictAndTime)
}

func dictOrTimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opDictOrTime)
}

// ... dict

func opDictAndDict(left interface{}, right interface{}) bool {
	return truthyDict(left) && truthyDict(right)
}

func opDictOrDict(left interface{}, right interface{}) bool {
	return truthyDict(left) || truthyDict(right)
}

func dictAndDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opDictAndDict)
}

func dictOrDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opDictOrDict)
}

// ... array

func opDictAndArray(left interface{}, right interface{}) bool {
	return truthyDict(left) && (len(right.([]interface{})) != 0)
}

func opArrayAndDict(left interface{}, right interface{}) bool {
	return truthyDict(right) && (len(left.([]interface{})) != 0)
}

func opDictOrArray(left interface{}, right interface{}) bool {
	return truthyDict(left) || (len(right.([]interface{})) != 0)
}

func opArrayOrDict(left interface{}, right interface{}) bool {
	return truthyDict(right) || (len(left.([]interface{})) != 0)
}

func dictAndArrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opDictAndArray)
}

func dictOrArrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opDictOrArray)
}

func arrayAndDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opArrayAndDict)
}

func arrayOrDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opArrayOrDict)
}

// ... map

func opDictAndMap(left interface{}, right interface{}) bool {
	return truthyDict(left) && (len(right.(map[string]interface{})) != 0)
}

func opMapAndDict(left interface{}, right interface{}) bool {
	return truthyDict(right) && (len(left.(map[string]interface{})) != 0)
}

func opDictOrMap(left interface{}, right interface{}) bool {
	return truthyDict(left) || (len(right.(map[string]interface{})) != 0)
}

func opMapOrDict(left interface{}, right interface{}) bool {
	return truthyDict(right) || (len(left.(map[string]interface{})) != 0)
}

func dictAndMapV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opDictAndMap)
}

func dictOrMapV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opDictOrMap)
}

func mapAndDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opMapAndDict)
}

func mapOrDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opMapOrDict)
}

// dict + - * /

func dictPlusStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
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

func stringPlusDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
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

func intPlusDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
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

func dictPlusIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
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

func floatPlusDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
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

func dictPlusFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
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

func intMinusDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
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

func dictMinusIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
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

func floatMinusDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
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

func dictMinusFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
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

func intTimesDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
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

func dictTimesIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
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

func floatTimesDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
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

func dictTimesFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
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

func intDividedDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
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

func dictDividedIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
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

func floatDividedDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
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

func dictDividedFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
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

func dictTimesTimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
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

func timeTimesDictV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
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
