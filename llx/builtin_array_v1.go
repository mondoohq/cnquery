package llx

import (
	"errors"
	"strconv"

	"github.com/hashicorp/go-multierror"
	"go.mondoo.io/mondoo/types"
)

// arrayFunctions are all the handlers for builtin array methods
var arrayFunctionsV1 map[string]chunkHandlerV1

func arrayGetFirstIndexV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: bind.Type[1:]}, 0, nil
	}

	arr, ok := bind.Value.([]interface{})
	if !ok {
		return nil, 0, errors.New("failed to typecast " + bind.Type.Label() + " into array")
	}

	if len(arr) == 0 {
		return nil, 0, errors.New("array index out of bound (trying to access first element on an empty array)")
	}

	return &RawData{
		Type:  bind.Type[1:],
		Value: arr[0],
	}, 0, nil
}

func arrayGetLastIndexV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: bind.Type[1:]}, 0, nil
	}

	arr, ok := bind.Value.([]interface{})
	if !ok {
		return nil, 0, errors.New("failed to typecast " + bind.Type.Label() + " into array")
	}

	if len(arr) == 0 {
		return nil, 0, errors.New("array index out of bound (trying to access last element on an empty array)")
	}

	return &RawData{
		Type:  bind.Type[1:],
		Value: arr[len(arr)-1],
	}, 0, nil
}

func arrayGetIndexV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: bind.Type[1:]}, 0, nil
	}

	args := chunk.Function.Args
	// TODO: all this needs to go into the compile phase
	if len(args) < 1 {
		return nil, 0, errors.New("Called [] with " + strconv.Itoa(len(args)) + " arguments, only 1 supported.")
	}
	if len(args) > 1 {
		return nil, 0, errors.New("called [] with " + strconv.Itoa(len(args)) + " arguments, only 1 supported.")
	}
	t := types.Type(args[0].Type)
	if t != types.Int {
		return nil, 0, errors.New("called [] with wrong type " + t.Label())
	}
	// ^^ TODO

	key := int(bytes2int(args[0].Value))

	arr, ok := bind.Value.([]interface{})
	if !ok {
		return nil, 0, errors.New("failed to typecast " + bind.Type.Label() + " into array")
	}

	if key < 0 {
		if -key > len(arr) {
			return nil, 0, errors.New("array index out of bound (trying to access element " + strconv.Itoa(key) + ", max: " + strconv.Itoa(len(arr)-1) + ")")
		}
		key = len(arr) + key
	}
	if key >= len(arr) {
		return nil, 0, errors.New("array index out of bound (trying to access element " + strconv.Itoa(key) + ", max: " + strconv.Itoa(len(arr)-1) + ")")
	}

	return &RawData{
		Type:  bind.Type[1:],
		Value: arr[key],
	}, 0, nil
}

func arrayBlockListV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: bind.Type[1:]}, 0, nil
	}

	arr, ok := bind.Value.([]interface{})
	if !ok {
		return nil, 0, errors.New("failed to typecast " + bind.Type.Label() + " into array")
	}

	if len(arr) == 0 {
		return bind, 0, nil
	}

	prim := chunk.Function.Args[0]
	if !types.Type(prim.Type).IsFunction() {
		return nil, 0, errors.New("called block with wrong function type")
	}
	fref, ok := prim.RefV1()
	if !ok {
		return nil, 0, errors.New("cannot retrieve function reference on block call")
	}
	fun := c.code.Functions[fref-1]
	if fun == nil {
		return nil, 0, errors.New("block function is nil")
	}

	argList := make([][]*RawData, len(arr))
	for i := range arr {
		argList[i] = []*RawData{
			{
				Type:  bind.Type.Child(),
				Value: arr[i],
			},
		}
	}

	err := c.runFunctionBlocks(argList, false, fun, func(results []*RawData, errors []error) {
		var anyError error
		allResults := make([]interface{}, len(arr))

		for i, rd := range results {
			allResults[i] = rd.Value
		}
		if len(errors) > 0 {
			// This is quite heavy handed. If any of the block calls have an error, the whole
			// thing becomes errored. If we don't do this, then we can have more fine grained
			// errors. For example, if only one item in the list has errors, the block for that
			// item will have an entrypoint with an error
			anyError = multierror.Append(nil, errors...)
		}
		data := &RawData{
			Type:  arrayBlockType,
			Value: allResults,
			Error: anyError,
		}
		c.cache.Store(ref, &stepCache{
			Result:   data,
			IsStatic: true,
		})
		c.triggerChain(ref, data)
	})
	if err != nil {
		return nil, 0, err
	}

	return nil, 0, nil
}

func arrayBlockV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	prim := chunk.Function.Args[0]
	if !types.Type(prim.Type).IsFunction() {
		return nil, 0, errors.New("called block with wrong function type")
	}
	return c.runBlock(bind, prim, chunk.Function.Args[1:], ref)
}

func arrayLengthV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Int, Error: bind.Error}, 0, nil
	}

	arr, ok := bind.Value.([]interface{})
	if !ok {
		return nil, 0, errors.New("failed to typecast " + bind.Type.Label() + " into array")
	}
	return IntData(int64(len(arr))), 0, nil
}

func arrayNotEmptyV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return BoolFalse, 0, nil
	}

	arr, ok := bind.Value.([]interface{})
	if !ok {
		return nil, 0, errors.New("failed to typecast " + bind.Type.Label() + " into array")
	}

	if len(arr) == 0 {
		return BoolFalse, 0, nil
	}
	return BoolTrue, 0, nil
}

func _arrayWhereV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32, invert bool) (*RawData, int32, error) {
	// where(array, function)
	itemsRef := chunk.Function.Args[0]
	items, rref, err := c.resolveValue(itemsRef, ref)
	if err != nil || rref > 0 {
		return nil, rref, err
	}

	if items.Value == nil {
		return &RawData{Type: items.Type}, 0, nil
	}

	list := items.Value.([]interface{})
	if len(list) == 0 {
		return items, 0, nil
	}

	arg1 := chunk.Function.Args[1]
	if types.Type(arg1.Type).Underlying() != types.FunctionLike {
		right := arg1.RawData().Value
		var res []interface{}
		for i := range list {
			left := list[i]
			if left == right {
				res = append(res, left)
			}
		}

		return &RawData{
			Type:  items.Type,
			Value: res,
		}, 0, nil
	}

	fref, ok := arg1.RefV1()
	if !ok {
		return nil, 0, errors.New("Failed to retrieve function reference of 'where' call")
	}

	f := c.code.Functions[fref-1]
	if len(f.Entrypoints) != 1 {
		return nil, 0, errors.New("Expected 'where' block to have 1 entrypoint")
	}
	ct := items.Type.Child()

	argsList := make([][]*RawData, len(list))
	for i := range list {
		argsList[i] = []*RawData{
			{
				Type:  ct,
				Value: list[i],
			},
		}
	}

	err = c.runFunctionBlocks(argsList, true, f, func(results []*RawData, errors []error) {
		resList := []interface{}{}
		for i, rd := range results {
			isTruthy, _ := rd.IsTruthy()
			if isTruthy == !invert {
				resList = append(resList, list[i])
			}
		}

		data := &RawData{
			Type:  bind.Type,
			Value: resList,
		}
		c.cache.Store(ref, &stepCache{
			Result:   data,
			IsStatic: false,
		})
		c.triggerChain(ref, data)
	})
	if err != nil {
		return nil, 0, err
	}

	return nil, 0, nil
}

func arrayWhereV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return _arrayWhereV1(c, bind, chunk, ref, false)
}

func arrayWhereNotV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return _arrayWhereV1(c, bind, chunk, ref, true)
}

func arrayAllV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Bool, Error: errors.New("failed to validate all entries (list is null)")}, 0, nil
	}

	filteredList := bind.Value.([]interface{})

	if len(filteredList) != 0 {
		return BoolFalse, 0, nil
	}
	return BoolTrue, 0, nil
}

func arrayNoneV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Bool, Error: errors.New("failed to validate all entries (list is null)")}, 0, nil
	}

	filteredList := bind.Value.([]interface{})

	if len(filteredList) != 0 {
		return BoolFalse, 0, nil
	}
	return BoolTrue, 0, nil
}

func arrayAnyV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Bool, Error: errors.New("failed to validate all entries (list is null)")}, 0, nil
	}

	filteredList := bind.Value.([]interface{})

	if len(filteredList) == 0 {
		return BoolFalse, 0, nil
	}
	return BoolTrue, 0, nil
}

func arrayOneV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Bool, Error: errors.New("failed to validate all entries (list is null)")}, 0, nil
	}

	filteredList := bind.Value.([]interface{})

	if len(filteredList) != 1 {
		return BoolFalse, 0, nil
	}
	return BoolTrue, 0, nil
}

func arrayMapV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	// map(array, function)
	itemsRef := chunk.Function.Args[0]
	items, rref, err := c.resolveValue(itemsRef, ref)
	if err != nil || rref > 0 {
		return nil, rref, err
	}

	if items.Value == nil {
		return &RawData{Type: items.Type}, 0, nil
	}

	list := items.Value.([]interface{})
	if len(list) == 0 {
		return items, 0, nil
	}

	arg1 := chunk.Function.Args[1]
	fref, ok := arg1.RefV1()
	if !ok {
		return nil, 0, errors.New("Failed to retrieve function reference of 'map' call")
	}

	f := c.code.Functions[fref-1]
	if len(f.Entrypoints) != 1 {
		return nil, 0, errors.New("Expected 'where' block to have 1 entrypoint")
	}

	ct := items.Type.Child()

	argsList := make([][]*RawData, len(list))
	for i := range list {
		argsList[i] = []*RawData{
			{
				Type:  ct,
				Value: list[i],
			},
		}
	}
	err = c.runFunctionBlocks(argsList, true, f, func(results []*RawData, errors []error) {
		mappedType := types.Unset
		resList := []interface{}{}
		epChecksum := f.Checksums[f.Entrypoints[0]]

		for _, rd := range results {
			if rd.Error == nil {
				blockVals := rd.Value.(map[string]interface{})
				if _, ok := blockVals[epChecksum]; ok {
					epVal := blockVals[epChecksum].(*RawData)
					mappedType = epVal.Type
					resList = append(resList, epVal.Value)
				}
			}
		}

		data := &RawData{
			Type:  types.Array(mappedType),
			Value: resList,
		}
		c.cache.Store(ref, &stepCache{
			Result:   data,
			IsStatic: false,
		})
		c.triggerChain(ref, data)
	})
	if err != nil {
		return nil, 0, err
	}

	return nil, 0, nil
}

// Takes an array of resources and a field, identify duplicates of that field value
// Result list is every resource that has duplicates
// (there will be at least resources 2 if there is a duplicate field value)
func arrayFieldDuplicatesV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	// where(array, function)
	itemsRef := chunk.Function.Args[0]
	items, rref, err := c.resolveValue(itemsRef, ref)
	if err != nil || rref > 0 {
		return nil, rref, err
	}

	if items.Value == nil {
		return &RawData{Type: items.Type}, 0, nil
	}

	list := items.Value.([]interface{})
	if len(list) == 0 {
		return items, 0, nil
	}

	arg1 := chunk.Function.Args[1]
	if types.Type(arg1.Type).Underlying() != types.FunctionLike {
		return nil, 0, errors.New("Expected resource field, unable to get field value from " + types.Type(arg1.Type).Label())
	}

	fref, ok := arg1.RefV1()
	if !ok {
		return nil, 0, errors.New("Failed to retrieve function reference of 'field duplicates' call")
	}

	f := c.code.Functions[fref-1]
	ct := items.Type.Child()

	argsList := make([][]*RawData, len(list))
	for i := range list {
		argsList[i] = []*RawData{
			{
				Type:  ct,
				Value: list[i],
			},
		}
	}

	err = c.runFunctionBlocks(argsList, true, f, func(results []*RawData, err []error) {
		epChecksum := f.Checksums[f.Entrypoints[0]]
		filteredList := map[int]*RawData{}

		for i, rd := range results {
			if rd.Error != nil {
				filteredList[i] = &RawData{
					Error: rd.Error,
				}
			} else {
				blockVals := rd.Value.(map[string]interface{})
				epVal := blockVals[epChecksum].(*RawData)
				filteredList[i] = epVal
			}
		}

		resList := []interface{}{}

		equalFunc, ok := types.Equal[filteredList[0].Type]
		if !ok {
			data := &RawData{
				Type:  items.Type,
				Error: errors.New("cannot extract duplicates from array, field must be a basic type"),
			}
			c.cache.Store(ref, &stepCache{
				Result:   data,
				IsStatic: false,
			})
			c.triggerChain(ref, data)
			return
		}

		arr := make([]*RawData, len(list))
		for k, v := range filteredList {
			arr[k] = v
		}

		//to track values of fields
		existing := make(map[int]interface{})
		//to track index of duplicate resources
		duplicateIndices := []int{}
		var found bool
		var added bool
		for i := 0; i < len(arr); i++ {
			left := arr[i].Value

			for j, v := range existing {
				if equalFunc(left, v) {
					found = true
					//Track the index so that we can get the whole resource
					duplicateIndices = append(duplicateIndices, i)
					//check if j was already added to our list of indices
					for di := range duplicateIndices {
						if j == duplicateIndices[di] {
							added = true
						}
					}
					if added == false {
						duplicateIndices = append(duplicateIndices, j)
					}
					break
				}
			}

			//value not found so we add it to list of things to check for dupes
			if !found {
				existing[i] = left
			}
		}

		//Once we collect duplicate indices, make a list of resources
		for i := range duplicateIndices {
			idx := duplicateIndices[i]
			resList = append(resList, list[idx])
		}

		data := &RawData{
			Type:  bind.Type,
			Value: resList,
		}
		c.cache.Store(ref, &stepCache{
			Result:   data,
			IsStatic: false,
		})
		c.triggerChain(ref, data)
	})
	if err != nil {
		return nil, 0, err
	}

	return nil, 0, nil
}

func arrayDuplicatesV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: bind.Type, Error: bind.Error}, 0, nil
	}

	_, dupes, err := detectDupes(bind.Value, bind.Type)
	if err != nil {
		return nil, 0, err
	}

	return &RawData{Type: bind.Type, Value: dupes}, 0, nil
}

func arrayUniqueV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: bind.Type, Error: bind.Error}, 0, nil
	}

	unique, _, err := detectDupes(bind.Value, bind.Type)
	if err != nil {
		return nil, 0, err
	}

	return &RawData{Type: bind.Type, Value: unique}, 0, nil
}

func arrayDifferenceV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: bind.Type, Error: bind.Error}, 0, nil
	}

	args := chunk.Function.Args
	// TODO: all this needs to go into the compile phase
	if len(args) < 1 {
		return nil, 0, errors.New("Called `difference` with " + strconv.Itoa(len(args)) + " arguments, only 1 supported.")
	}
	if len(args) > 1 {
		return nil, 0, errors.New("called `difference` with " + strconv.Itoa(len(args)) + " arguments, only 1 supported.")
	}
	// ^^ TODO

	argRef := args[0]
	arg, rref, err := c.resolveValue(argRef, ref)
	if err != nil || rref > 0 {
		return nil, rref, err
	}

	t := types.Type(arg.Type)
	if t != bind.Type {
		return nil, 0, errors.New("called `difference` with wrong type (got: " + t.Label() + ", expected:" + bind.Type.Label() + ")")
	}

	ct := bind.Type.Child()
	equalFunc, ok := types.Equal[ct]
	if !ok {
		return nil, 0, errors.New("cannot compare array entries")
	}

	org := bind.Value.([]interface{})
	filters := arg.Value.([]interface{})

	var res []interface{}
	var skip bool
	for i := range org {
		skip = false
		for j := range filters {
			if equalFunc(org[i], filters[j]) {
				skip = true
				break
			}
		}

		if !skip {
			res = append(res, org[i])
		}
	}

	return &RawData{Type: bind.Type, Value: res}, 0, nil
}

func arrayContainsNoneV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: bind.Type, Error: bind.Error}, 0, nil
	}

	args := chunk.Function.Args
	// TODO: all this needs to go into the compile phase
	if len(args) < 1 {
		return nil, 0, errors.New("Called `arrayContainsNone` with " + strconv.Itoa(len(args)) + " arguments, only 1 supported.")
	}
	if len(args) > 1 {
		return nil, 0, errors.New("called `arrayContainsNone` with " + strconv.Itoa(len(args)) + " arguments, only 1 supported.")
	}
	// ^^ TODO

	argRef := args[0]
	arg, rref, err := c.resolveValue(argRef, ref)
	if err != nil || rref > 0 {
		return nil, rref, err
	}

	t := types.Type(arg.Type)
	if t != bind.Type {
		return nil, 0, errors.New("called `arrayNone` with wrong type (got: " + t.Label() + ", expected:" + bind.Type.Label() + ")")
	}

	ct := bind.Type.Child()
	equalFunc, ok := types.Equal[ct]
	if !ok {
		return nil, 0, errors.New("cannot compare array entries")
	}

	org := bind.Value.([]interface{})
	filters := arg.Value.([]interface{})

	var res []interface{}
	for i := range org {
		for j := range filters {
			if equalFunc(org[i], filters[j]) {
				res = append(res, org[i])
			}
		}
	}

	return &RawData{Type: bind.Type, Value: res}, 0, nil
}

func compileArrayOpArrayV1(op string) func(types.Type, types.Type) (string, error) {
	return func(left types.Type, right types.Type) (string, error) {
		name := string(left.Child()) + op + string(right)
		af := BuiltinFunctionsV1[types.ArrayLike]
		if _, ok := af[name]; ok {
			return name, nil
		}

		if right.IsArray() {
			return op, nil
		}

		if right == types.Nil {
			return op + string(types.Nil), nil
		}

		return "", errors.New("don't know how to compile " + left.Label() + " " + op + " " + right.Label())
	}
}

func compileLogicalArrayOpV1(underlying types.Type, op string) func(types.Type, types.Type) (string, error) {
	return func(left types.Type, right types.Type) (string, error) {
		name := string(types.Any) + op + string(right.Underlying())
		af := BuiltinFunctionsV1[underlying]
		if _, ok := af[name]; ok {
			return name, nil
		}

		return "", errors.New("cannot find operation for " + left.Label() + " " + op + " " + right.Label())
	}
}

// []T -- []T

func tarrayCmpTarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrays(left, right, tArrayCmp(left, right))
	})
}

func tarrayNotTarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrays(left, right, tArrayCmp(left, right))
	})
}

func boolarrayCmpBoolarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrays(left, right, opBoolCmpBool)
	})
}

func intarrayCmpIntarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrays(left, right, opIntCmpInt)
	})
}

func floatarrayCmpFloatarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrays(left, right, opFloatCmpFloat)
	})
}

func stringarrayCmpStringarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrays(left, right, opStringCmpString)
	})
}

func boolarrayNotBoolarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrays(left, right, opBoolCmpBool)
	})
}

func intarrayNotIntarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrays(left, right, opIntCmpInt)
	})
}

func floatarrayNotFloatarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrays(left, right, opFloatCmpFloat)
	})
}

func stringarrayNotStringarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrays(left, right, opStringCmpString)
	})
}

// []T -- T

func arrayCmpNilV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return BoolTrue, 0, nil
	}
	v := bind.Value.([]interface{})
	if v == nil {
		return BoolTrue, 0, nil
	}
	return BoolFalse, 0, nil
}

func arrayNotNilV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return BoolFalse, 0, nil
	}
	v := bind.Value.([]interface{})
	if v == nil {
		return BoolFalse, 0, nil
	}
	return BoolTrue, 0, nil
}

func boolarrayCmpBoolV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opBoolCmpBool)
	})
}

func boolarrayNotBoolV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opBoolCmpBool)
	})
}

func intarrayCmpIntV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opIntCmpInt)
	})
}

func intarrayNotIntV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opIntCmpInt)
	})
}

func floatarrayCmpFloatV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opFloatCmpFloat)
	})
}

func floatarrayNotFloatV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opFloatCmpFloat)
	})
}

func stringarrayCmpStringV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opStringCmpString)
	})
}

func stringarrayNotStringV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opStringCmpString)
	})
}

// T -- []T

func boolCmpBoolarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opBoolCmpBool)
	})
}

func boolNotBoolarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opBoolCmpBool)
	})
}

func intCmpIntarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opIntCmpInt)
	})
}

func intNotIntarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opIntCmpInt)
	})
}

func floatCmpFloatarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opFloatCmpFloat)
	})
}

func floatNotFloatarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opFloatCmpFloat)
	})
}

func stringCmpStringarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opStringCmpString)
	})
}

func stringNotStringarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opStringCmpString)
	})
}

// int/float -- []T

func intCmpFloatarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opFloatCmpInt)
	})
}

func intNotFloatarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opFloatCmpInt)
	})
}

func floatCmpIntarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opIntCmpFloat)
	})
}

func floatNotIntarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opIntCmpFloat)
	})
}

func intarrayCmpFloatV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opIntCmpFloat)
	})
}

func intarrayNotFloatV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opIntCmpFloat)
	})
}

func floatarrayCmpIntV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opFloatCmpInt)
	})
}

func floatarrayNotIntV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opFloatCmpInt)
	})
}

// string -- []T

func stringCmpBoolarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opBoolCmpString)
	})
}

func stringNotBoolarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opBoolCmpString)
	})
}

func boolarrayCmpStringV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opBoolCmpString)
	})
}

func boolarrayNotStringV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opBoolCmpString)
	})
}

func stringCmpIntarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opIntCmpString)
	})
}

func stringNotIntarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opIntCmpString)
	})
}

func intarrayCmpStringV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opIntCmpString)
	})
}

func intarrayNotStringV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opIntCmpString)
	})
}

func stringCmpFloatarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opFloatCmpString)
	})
}

func stringNotFloatarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opFloatCmpString)
	})
}

func floatarrayCmpStringV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opFloatCmpString)
	})
}

func floatarrayNotStringV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opFloatCmpString)
	})
}

// bool -- []string

func boolCmpStringarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opStringCmpBool)
	})
}

func boolNotStringarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opStringCmpBool)
	})
}

func stringarrayCmpBoolV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opStringCmpBool)
	})
}

func stringarrayNotBoolV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opStringCmpBool)
	})
}

// int -- []string

func intCmpStringarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opStringCmpInt)
	})
}

func intNotStringarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opStringCmpInt)
	})
}

func stringarrayCmpIntV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opStringCmpInt)
	})
}

func stringarrayNotIntV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opStringCmpInt)
	})
}

// float -- []string

func floatCmpStringarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opStringCmpFloat)
	})
}

func floatNotStringarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opStringCmpFloat)
	})
}

func stringarrayCmpFloatV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opStringCmpFloat)
	})
}

func stringarrayNotFloatV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opStringCmpFloat)
	})
}

// regex -- []T

func regexCmpStringarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opStringCmpRegex)
	})
}

func regexNotStringarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opStringCmpRegex)
	})
}

func stringarrayCmpRegexV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opStringCmpRegex)
	})
}

func stringarrayNotRegexV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opStringCmpRegex)
	})
}

func regexCmpIntarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opIntCmpRegex)
	})
}

func regexNotIntarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opIntCmpRegex)
	})
}

func intarrayCmpRegexV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opIntCmpRegex)
	})
}

func intarrayNotRegexV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opIntCmpRegex)
	})
}

func regexCmpFloatarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opFloatCmpRegex)
	})
}

func regexNotFloatarrayV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opFloatCmpRegex)
	})
}

func floatarrayCmpRegexV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opFloatCmpRegex)
	})
}

func floatarrayNotRegexV1(c *LeiseExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opFloatCmpRegex)
	})
}
