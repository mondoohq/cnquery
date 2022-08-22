package llx

import (
	"errors"
	"strings"
	"time"

	"go.mondoo.io/mondoo/resources"
	"go.mondoo.io/mondoo/types"
)

// resourceFunctions are all the shared handlers for resource calls
var resourceFunctionsV1 map[string]chunkHandlerV1

func _resourceWhereV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32, invert bool) (*RawData, int32, error) {
	// where(resource.list, function)
	itemsRef := chunk.Function.Args[0]
	items, rref, err := c.resolveValue(itemsRef, ref)
	if err != nil || rref > 0 {
		return nil, rref, err
	}
	list := items.Value.([]interface{})
	if len(list) == 0 {
		return bind, 0, nil
	}

	resource := bind.Value.(resources.ResourceType)

	arg1 := chunk.Function.Args[1]
	fref, ok := arg1.RefV2()
	if !ok {
		return nil, 0, errors.New("Failed to retrieve function reference of 'where' call")
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

	err = c.runFunctionBlocks(argsList, f, func(results []arrayBlockCallResult, errs []error) {
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
			"list", resList, "__id", f.Id,
		}
		for k, v := range resourceInfo.Fields {
			if k != "list" && v.IsMandatory {
				if v, err := resource.Field(k); err == nil {
					args = append(args, k, v)
				}
			}
		}

		resResource, err := c.runtime.CreateResourceWithID(mqlResource.Name, f.Id, args...)
		var data *RawData
		if err != nil {
			data = &RawData{
				Error: errors.New("Failed to create filter result resource: " + err.Error()),
			}
			c.cache.Store(ref, &stepCache{
				Result: data,
			})
		} else {
			data = &RawData{
				Type:  bind.Type,
				Value: resResource,
			}
			c.cache.Store(ref, &stepCache{
				Result:   data,
				IsStatic: false,
			})
		}

		c.triggerChain(ref, data)
	})

	if err != nil {
		return nil, 0, err
	}

	return nil, 0, nil
}

func resourceWhereV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return _resourceWhereV1(c, bind, chunk, ref, false)
}

func resourceWhereNotV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return _resourceWhereV1(c, bind, chunk, ref, true)
}

func resourceMapV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	// map(resource.list, function)
	itemsRef := chunk.Function.Args[0]
	items, rref, err := c.resolveValue(itemsRef, ref)
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

	err = c.runFunctionBlocks(argsList, f, func(results []arrayBlockCallResult, errs []error) {
		mappedType := types.Unset
		resList := []interface{}{}
		epChecksum := f.Checksums[f.Entrypoints[0]]

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

		c.cache.Store(ref, &stepCache{
			Result: data,
		})

		c.triggerChain(ref, data)
	})

	if err != nil {
		return nil, 0, err
	}

	return nil, 0, nil
}

func resourceLengthV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	// length(resource.list)
	itemsRef := chunk.Function.Args[0]
	items, rref, err := c.resolveValue(itemsRef, ref)
	if err != nil || rref > 0 {
		return nil, rref, err
	}

	list := items.Value.([]interface{})
	return IntData(int64(len(list))), 0, nil
}

func resourceDateV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	args, rref, err := args2resourceargsV1(c, ref, chunk.Function.Args)
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
