package llx

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/types"
)

// resourceFunctions are all the shared handlers for resource calls
var resourceFunctionsV2 map[string]chunkHandlerV2

func _resourceWhereV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64, invert bool) (*RawData, uint64, error) {
	// where(resource.list, function)
	itemsRef := chunk.Function.Args[0]
	items, rref, err := e.resolveValue(itemsRef, ref)
	if err != nil || rref > 0 {
		return nil, rref, err
	}
	list := items.Value.([]interface{})
	if len(list) == 0 {
		return bind, 0, nil
	}

	resource := bind.Value.(resources.ResourceType)

	arg1 := chunk.Function.Args[1]
	blockRef, ok := arg1.RefV2()
	if !ok {
		return nil, 0, errors.New("Failed to retrieve function reference of 'where' call")
	}

	dref, err := e.ensureArgsResolved(chunk.Function.Args[2:], ref)
	if dref != 0 || err != nil {
		return nil, dref, err
	}

	blockId := e.ctx.code.Id + strconv.FormatUint(blockRef>>32, 10)

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

	err = e.runFunctionBlocks(argsList, blockRef, func(results []arrayBlockCallResult, errs []error) {
		resList := []interface{}{}

		for i, res := range results {
			isTruthy := res.isTruthy()
			if isTruthy == !invert {
				resList = append(resList, list[i])
			}
		}

		// get all mandatory args
		mqlResource := resource.MqlResource()
		resourceInfo := mqlResource.MotorRuntime.Registry.Resources[mqlResource.Name]

		args := []interface{}{
			"list", resList, "__id", blockId,
		}
		for k, v := range resourceInfo.Fields {
			if k != "list" && v.IsMandatory {
				if v, err := resource.Field(k); err == nil {
					args = append(args, k, v)
				}
			}
		}

		resResource, err := e.ctx.runtime.CreateResourceWithID(mqlResource.Name, blockId, args...)
		var data *RawData
		if err != nil {
			data = &RawData{
				Error: errors.New("Failed to create filter result resource: " + err.Error()),
			}
			e.cache.Store(ref, &stepCache{
				Result: data,
			})
		} else {
			data = &RawData{
				Type:  bind.Type,
				Value: resResource,
			}
			e.cache.Store(ref, &stepCache{
				Result:   data,
				IsStatic: false,
			})
		}

		e.triggerChain(ref, data)
	})

	if err != nil {
		return nil, 0, err
	}

	return nil, 0, nil
}

func resourceWhereV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return _resourceWhereV2(e, bind, chunk, ref, false)
}

func resourceWhereNotV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return _resourceWhereV2(e, bind, chunk, ref, true)
}

func resourceMapV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	// map(resource.list, function)
	itemsRef := chunk.Function.Args[0]
	items, rref, err := e.resolveValue(itemsRef, ref)
	if err != nil || rref > 0 {
		return nil, rref, err
	}
	list := items.Value.([]interface{})
	if len(list) == 0 {
		return bind, 0, nil
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
			Result: data,
		})

		e.triggerChain(ref, data)
	})

	if err != nil {
		return nil, 0, err
	}

	return nil, 0, nil
}

func resourceLengthV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	// length(resource.list)
	itemsRef := chunk.Function.Args[0]
	items, rref, err := e.resolveValue(itemsRef, ref)
	if err != nil || rref > 0 {
		return nil, rref, err
	}

	list := items.Value.([]interface{})
	return IntData(int64(len(list))), 0, nil
}

var timeFormats = map[string]string{
	"ansic":    time.ANSIC,
	"rfc822":   time.RFC822,
	"rfc822z":  time.RFC822Z,
	"rfc850":   time.RFC850,
	"rfc1123":  time.RFC1123,
	"rfc1123z": time.RFC1123Z,
	"rfc3339":  time.RFC3339,
	"kitchen":  time.Kitchen,
	"stamp":    time.Stamp,
}

func resourceDateV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	args, rref, err := args2resourceargsV2(e, ref, chunk.Function.Args)
	if err != nil || rref != 0 {
		return nil, rref, err
	}

	format := time.RFC3339
	if len(args) >= 2 {
		format = args[1].(string)
	}

	var timeParseFormat string
	timeParseFormat, ok := timeFormats[strings.ToLower(format)]
	if !ok {
		timeParseFormat = format
	}

	parsed, err := time.Parse(timeParseFormat, args[0].(string))
	if err != nil {
		return nil, 0, errors.New("failed to parse time: " + err.Error())
	}

	return TimeData(parsed), 0, nil
}
