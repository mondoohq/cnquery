package llx

import (
	"errors"
	"strconv"

	"github.com/hashicorp/go-multierror"
	"go.mondoo.com/cnquery/types"
)

var arrayBlockType = types.Array(types.Block)

// arrayFunctions are all the handlers for builtin array methods
var arrayFunctions map[string]chunkHandlerV2

func arrayGetFirstIndexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
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

func arrayGetLastIndexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
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

func arrayGetIndexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
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

func arrayBlockListV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
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
	fref, ok := prim.RefV2()
	if !ok {
		return nil, 0, errors.New("cannot retrieve function reference on block call")
	}
	block := e.ctx.code.Block(fref)
	if block == nil {
		return nil, 0, errors.New("block function is nil")
	}

	dref, err := e.ensureArgsResolved(chunk.Function.Args[1:], ref)
	if dref != 0 || err != nil {
		return nil, dref, err
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

	err = e.runFunctionBlocks(argList, fref, func(results []arrayBlockCallResult, errs []error) {
		var anyError error
		allResults := make([]interface{}, len(arr))

		for i, rd := range results {
			allResults[i] = rd.toRawData().Value
		}
		if len(errs) > 0 {
			// This is quite heavy handed. If any of the block calls have an error, the whole
			// thing becomes errored. If we don't do this, then we can have more fine grained
			// errors. For example, if only one item in the list has errors, the block for that
			// item will have an entrypoint with an error
			anyError = multierror.Append(nil, errs...)
		}
		data := &RawData{
			Type:  arrayBlockType,
			Value: allResults,
			Error: anyError,
		}
		e.cache.Store(ref, &stepCache{
			Result:   data,
			IsStatic: true,
		})
		e.triggerChain(ref, data)
	})

	if err != nil {
		return nil, 0, err
	}

	return nil, 0, nil
}

func arrayBlockV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	prim := chunk.Function.Args[0]
	if !types.Type(prim.Type).IsFunction() {
		return nil, 0, errors.New("called block with wrong function type")
	}
	return e.runBlock(bind, prim, chunk.Function.Args[1:], ref)
}

func arrayLengthV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Int, Error: bind.Error}, 0, nil
	}

	arr, ok := bind.Value.([]interface{})
	if !ok {
		return nil, 0, errors.New("failed to typecast " + bind.Type.Label() + " into array")
	}
	return IntData(int64(len(arr))), 0, nil
}

func arrayNotEmptyV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
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

func _arrayWhereV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64, invert bool) (*RawData, uint64, error) {
	// where(array, function)
	itemsRef := chunk.Function.Args[0]
	items, rref, err := e.resolveValue(itemsRef, ref)
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

	fref, ok := arg1.RefV2()
	if !ok {
		return nil, 0, errors.New("Failed to retrieve function reference of 'where' call")
	}

	dref, err := e.ensureArgsResolved(chunk.Function.Args[2:], ref)
	if dref != 0 || err != nil {
		return nil, dref, err
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
	err = e.runFunctionBlocks(argsList, fref, func(results []arrayBlockCallResult, errors []error) {
		resList := []interface{}{}
		for i, res := range results {
			isTruthy := res.isTruthy()
			if isTruthy == !invert {
				resList = append(resList, list[i])
			}
		}

		data := &RawData{
			Type:  bind.Type,
			Value: resList,
		}
		e.cache.Store(ref, &stepCache{
			Result:   data,
			IsStatic: false,
		})
		e.triggerChain(ref, data)
	})

	if err != nil {
		return nil, 0, err
	}

	return nil, 0, nil
}

func arrayWhereV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return _arrayWhereV2(e, bind, chunk, ref, false)
}

func arrayWhereNotV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return _arrayWhereV2(e, bind, chunk, ref, true)
}

func arrayAllV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Bool, Error: errors.New("failed to validate all entries (list is null)")}, 0, nil
	}

	filteredList := bind.Value.([]interface{})

	if len(filteredList) != 0 {
		return BoolFalse, 0, nil
	}
	return BoolTrue, 0, nil
}

func arrayNoneV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Bool, Error: errors.New("failed to validate all entries (list is null)")}, 0, nil
	}

	filteredList := bind.Value.([]interface{})

	if len(filteredList) != 0 {
		return BoolFalse, 0, nil
	}
	return BoolTrue, 0, nil
}

func arrayAnyV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Bool, Error: errors.New("failed to validate all entries (list is null)")}, 0, nil
	}

	filteredList := bind.Value.([]interface{})

	if len(filteredList) == 0 {
		return BoolFalse, 0, nil
	}
	return BoolTrue, 0, nil
}

func arrayOneV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Bool, Error: errors.New("failed to validate all entries (list is null)")}, 0, nil
	}

	filteredList := bind.Value.([]interface{})

	if len(filteredList) != 1 {
		return BoolFalse, 0, nil
	}
	return BoolTrue, 0, nil
}

func arrayMapV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	// map(array, function)
	itemsRef := chunk.Function.Args[0]
	items, rref, err := e.resolveValue(itemsRef, ref)
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
	fref, ok := arg1.RefV2()
	if !ok {
		return nil, 0, errors.New("Failed to retrieve function reference of 'map' call")
	}

	dref, err := e.ensureArgsResolved(chunk.Function.Args[2:], ref)
	if dref != 0 || err != nil {
		return nil, dref, err
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

	err = e.runFunctionBlocks(argsList, fref, func(results []arrayBlockCallResult, errs []error) {
		mappedType := types.Unset
		resList := []interface{}{}
		f := e.ctx.code.Block(fref)

		epChecksum := e.ctx.code.Checksums[f.Entrypoints[0]]

		for _, res := range results {
			if epValIface, ok := res.entrypoints[epChecksum]; ok {
				epVal := epValIface.(*RawData)
				mappedType = epVal.Type
				resList = append(resList, epVal.Value)
			}
		}

		data := &RawData{
			Type:  types.Array(mappedType),
			Value: resList,
		}
		e.cache.Store(ref, &stepCache{
			Result:   data,
			IsStatic: false,
		})
		e.triggerChain(ref, data)
	})

	if err != nil {
		return nil, 0, err
	}

	return nil, 0, nil
}

// Take an array and separate it into a list of unique entries and another
// list of only duplicates. The latter list only has every entry appear only
// once.
func detectDupes(array interface{}, typ types.Type) ([]interface{}, []interface{}, error) {
	if array == nil {
		return nil, nil, nil
	}
	arr, ok := array.([]interface{})
	if !ok {
		return nil, nil, errors.New("failed to typecast " + typ.Label() + " into array")
	}

	ct := typ.Child()
	equalFunc, ok := types.Equal[ct]
	if !ok {
		return nil, nil, errors.New("cannot extract duplicates from array, must be a basic type. Try using a field argument.")
	}

	existing := []interface{}{}
	duplicates := []interface{}{}
	var found bool
	for i := 0; i < len(arr); i++ {
		left := arr[i]

		for j := range existing {
			if equalFunc(left, existing[j]) {
				found = true
				break
			}
		}

		if !found {
			existing = append(existing, left)
			continue
		}

		found = false
		for j := range duplicates {
			if equalFunc(left, duplicates[j]) {
				found = true
				break
			}
		}

		if found {
			found = false
		} else {
			duplicates = append(duplicates, left)
		}
	}

	return existing, duplicates, nil
}

// Takes an array of resources and a field, identify duplicates of that field value
// Result list is every resource that has duplicates
// (there will be at least resources 2 if there is a duplicate field value)
func arrayFieldDuplicatesV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	// where(array, function)
	itemsRef := chunk.Function.Args[0]
	items, rref, err := e.resolveValue(itemsRef, ref)
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

	fref, ok := arg1.RefV2()
	if !ok {
		return nil, 0, errors.New("Failed to retrieve function reference of 'field duplicates' call")
	}

	dref, err := e.ensureArgsResolved(chunk.Function.Args[2:], ref)
	if dref != 0 || err != nil {
		return nil, dref, err
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

	err = e.runFunctionBlocks(argsList, fref, func(results []arrayBlockCallResult, errs []error) {
		f := e.ctx.code.Block(fref)
		epChecksum := e.ctx.code.Checksums[f.Entrypoints[0]]
		filteredList := map[int]*RawData{}

		for i, res := range results {
			rd := res.toRawData()
			if rd.Error != nil {
				filteredList[i] = &RawData{
					Error: rd.Error,
				}
			} else {
				epVal := res.entrypoints[epChecksum].(*RawData)
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
			e.cache.Store(ref, &stepCache{
				Result:   data,
				IsStatic: false,
			})
			e.triggerChain(ref, data)
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
		e.cache.Store(ref, &stepCache{
			Result:   data,
			IsStatic: false,
		})
		e.triggerChain(ref, data)
	})

	if err != nil {
		return nil, 0, err
	}

	return nil, 0, nil
}

func arrayDuplicatesV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: bind.Type, Error: bind.Error}, 0, nil
	}

	_, dupes, err := detectDupes(bind.Value, bind.Type)
	if err != nil {
		return nil, 0, err
	}

	return &RawData{Type: bind.Type, Value: dupes}, 0, nil
}

func arrayUniqueV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: bind.Type, Error: bind.Error}, 0, nil
	}

	unique, _, err := detectDupes(bind.Value, bind.Type)
	if err != nil {
		return nil, 0, err
	}

	return &RawData{Type: bind.Type, Value: unique}, 0, nil
}

func arrayDifferenceV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
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
	arg, rref, err := e.resolveValue(argRef, ref)
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

func arrayContainsNoneV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
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
	arg, rref, err := e.resolveValue(argRef, ref)
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

func compileArrayOpArray(op string) func(types.Type, types.Type) (string, error) {
	return func(left types.Type, right types.Type) (string, error) {
		name := string(left.Child()) + op + string(right)
		af := BuiltinFunctionsV2[types.ArrayLike]
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

func compileLogicalArrayOp(underlying types.Type, op string) func(types.Type, types.Type) (string, error) {
	return func(left types.Type, right types.Type) (string, error) {
		name := string(types.Any) + op + string(right.Underlying())
		af := BuiltinFunctionsV2[underlying]
		if _, ok := af[name]; ok {
			return name, nil
		}

		return "", errors.New("cannot find operation for " + left.Label() + " " + op + " " + right.Label())
	}
}

func cmpArrays(left *RawData, right *RawData, f func(interface{}, interface{}) bool) bool {
	if left.Value == nil {
		if right.Value == nil {
			return true
		}
		return false
	}
	if right == nil || right.Value == nil {
		return false
	}

	l := left.Value.([]interface{})
	r := right.Value.([]interface{})

	if len(l) != len(r) {
		return false
	}

	for i := range l {
		if !f(l[i], r[i]) {
			return false
		}
	}

	return true
}

func cmpArrayOne(leftArray *RawData, right *RawData, f func(interface{}, interface{}) bool) bool {
	l := leftArray.Value.([]interface{})
	if len(l) != 1 {
		return false
	}
	return f(l[0], right.Value)
}

// []T -- []T

func tArrayCmp(left *RawData, right *RawData) func(interface{}, interface{}) bool {
	return func(a interface{}, b interface{}) bool {
		if left.Type.Child() != right.Type.Child() {
			return false
		}
		return a == b
	}
}

func tarrayCmpTarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrays(left, right, tArrayCmp(left, right))
	})
}

func tarrayNotTarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrays(left, right, tArrayCmp(left, right))
	})
}

func boolarrayCmpBoolarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrays(left, right, opBoolCmpBool)
	})
}

func intarrayCmpIntarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrays(left, right, opIntCmpInt)
	})
}

func floatarrayCmpFloatarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrays(left, right, opFloatCmpFloat)
	})
}

func stringarrayCmpStringarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrays(left, right, opStringCmpString)
	})
}

func boolarrayNotBoolarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrays(left, right, opBoolCmpBool)
	})
}

func intarrayNotIntarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrays(left, right, opIntCmpInt)
	})
}

func floatarrayNotFloatarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrays(left, right, opFloatCmpFloat)
	})
}

func stringarrayNotStringarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrays(left, right, opStringCmpString)
	})
}

// []T -- T

func arrayCmpNilV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return BoolTrue, 0, nil
	}
	v := bind.Value.([]interface{})
	if v == nil {
		return BoolTrue, 0, nil
	}
	return BoolFalse, 0, nil
}

func arrayNotNilV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return BoolFalse, 0, nil
	}
	v := bind.Value.([]interface{})
	if v == nil {
		return BoolFalse, 0, nil
	}
	return BoolTrue, 0, nil
}

func boolarrayCmpBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opBoolCmpBool)
	})
}

func boolarrayNotBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opBoolCmpBool)
	})
}

func intarrayCmpIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opIntCmpInt)
	})
}

func intarrayNotIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opIntCmpInt)
	})
}

func floatarrayCmpFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opFloatCmpFloat)
	})
}

func floatarrayNotFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opFloatCmpFloat)
	})
}

func stringarrayCmpStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opStringCmpString)
	})
}

func stringarrayNotStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opStringCmpString)
	})
}

// T -- []T

func boolCmpBoolarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opBoolCmpBool)
	})
}

func boolNotBoolarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opBoolCmpBool)
	})
}

func intCmpIntarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opIntCmpInt)
	})
}

func intNotIntarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opIntCmpInt)
	})
}

func floatCmpFloatarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opFloatCmpFloat)
	})
}

func floatNotFloatarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opFloatCmpFloat)
	})
}

func stringCmpStringarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opStringCmpString)
	})
}

func stringNotStringarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opStringCmpString)
	})
}

// int/float -- []T

func intCmpFloatarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opFloatCmpInt)
	})
}

func intNotFloatarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opFloatCmpInt)
	})
}

func floatCmpIntarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opIntCmpFloat)
	})
}

func floatNotIntarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opIntCmpFloat)
	})
}

func intarrayCmpFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opIntCmpFloat)
	})
}

func intarrayNotFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opIntCmpFloat)
	})
}

func floatarrayCmpIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opFloatCmpInt)
	})
}

func floatarrayNotIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opFloatCmpInt)
	})
}

// string -- []T

func stringCmpBoolarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opBoolCmpString)
	})
}

func stringNotBoolarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opBoolCmpString)
	})
}

func boolarrayCmpStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opBoolCmpString)
	})
}

func boolarrayNotStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opBoolCmpString)
	})
}

func stringCmpIntarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opIntCmpString)
	})
}

func stringNotIntarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opIntCmpString)
	})
}

func intarrayCmpStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opIntCmpString)
	})
}

func intarrayNotStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opIntCmpString)
	})
}

func stringCmpFloatarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opFloatCmpString)
	})
}

func stringNotFloatarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opFloatCmpString)
	})
}

func floatarrayCmpStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opFloatCmpString)
	})
}

func floatarrayNotStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opFloatCmpString)
	})
}

// bool -- []string

func boolCmpStringarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opStringCmpBool)
	})
}

func boolNotStringarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opStringCmpBool)
	})
}

func stringarrayCmpBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opStringCmpBool)
	})
}

func stringarrayNotBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opStringCmpBool)
	})
}

// int -- []string

func intCmpStringarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opStringCmpInt)
	})
}

func intNotStringarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opStringCmpInt)
	})
}

func stringarrayCmpIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opStringCmpInt)
	})
}

func stringarrayNotIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opStringCmpInt)
	})
}

// float -- []string

func floatCmpStringarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opStringCmpFloat)
	})
}

func floatNotStringarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opStringCmpFloat)
	})
}

func stringarrayCmpFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opStringCmpFloat)
	})
}

func stringarrayNotFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opStringCmpFloat)
	})
}

// regex -- []T

func regexCmpStringarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opStringCmpRegex)
	})
}

func regexNotStringarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opStringCmpRegex)
	})
}

func stringarrayCmpRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opStringCmpRegex)
	})
}

func stringarrayNotRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opStringCmpRegex)
	})
}

func regexCmpIntarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opIntCmpRegex)
	})
}

func regexNotIntarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opIntCmpRegex)
	})
}

func intarrayCmpRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opIntCmpRegex)
	})
}

func intarrayNotRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opIntCmpRegex)
	})
}

func regexCmpFloatarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opFloatCmpRegex)
	})
}

func regexNotFloatarrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(right, left, opFloatCmpRegex)
	})
}

func floatarrayCmpRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opFloatCmpRegex)
	})
}

func floatarrayNotRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, func(left *RawData, right *RawData) bool {
		return cmpArrayOne(left, right, opFloatCmpRegex)
	})
}
