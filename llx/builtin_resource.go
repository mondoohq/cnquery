// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package llx

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.mondoo.com/cnquery/v11/types"
	"go.mondoo.com/cnquery/v11/utils/timex"
)

type Resource interface {
	// Name of the resource
	MqlName() string
	// ID of this resource
	MqlID() string
}

type MockResource struct {
	Name string
	ID   string
}

func (m *MockResource) MqlName() string {
	return m.Name
}

func (m *MockResource) MqlID() string {
	return m.ID
}

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

	resource := bind.Value.(Resource)

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
		resList := []*Primitive{}
		for i, res := range results {
			isTruthy := res.isTruthy()
			if isTruthy == !invert {
				prim := (&RawData{Value: list[i], Type: ct}).Result().Data
				resList = append(resList, prim)
			}
		}

		// get all mandatory args
		resourceInfo := e.ctx.runtime.Schema().Lookup(resource.MqlName())
		copyFields := []string{}
		for k, v := range resourceInfo.Fields {
			if k != "list" && v.IsMandatory {
				copyFields = append(copyFields, k)
			}
		}
		args := map[string]*Primitive{
			"list": ArrayPrimitive(resList, ct),
		}

		resResource, err := e.ctx.runtime.CloneResource(resource, blockId, copyFields, args)

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

func resourceSample(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	// sample(resource.list)
	itemsRef := chunk.Function.Args[0]
	items, rref, err := e.resolveValue(itemsRef, ref)
	if err != nil || rref > 0 {
		return nil, rref, err
	}
	list := items.Value.([]any)
	if len(list) == 0 {
		return bind, 0, nil
	}

	cntRef := chunk.Function.Args[1]
	cntRaw, rref, err := e.resolveValue(cntRef, ref)
	if err != nil || rref > 0 {
		return nil, rref, err
	}
	cnt, ok := cntRaw.Value.(int64)
	if !ok {
		return nil, 0, errors.New("failed to get count for sample, incorrect type of value")
	}

	res := _arraySample(list, cnt)
	return ArrayData(res, bind.Type), 0, nil
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

func resourceDateV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	args, rref, err := primitive2array(e, ref, chunk.Function.Args)
	if err != nil || rref != 0 {
		return nil, rref, err
	}

	timestamp, ok := args[0].(string)
	if !ok {
		return nil, 0, errors.New("failed to parse time, timestamp needs to be a string")
	}

	var format string
	if len(args) >= 2 {
		format, ok = args[1].(string)
		if !ok {
			return nil, 0, errors.New("provided time format is not provided as string")
		}
		format = strings.ToLower(format)
	}

	res, err := timex.Parse(timestamp, format)
	if err != nil {
		return nil, 0, errors.New("failed to parse time: " + err.Error())
	}
	return TimeData(res), 0, nil
}

var durationRegex = regexp.MustCompile(`^(\d+|[.])(\w*)$`)

func resourceDuration(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	args, rref, err := primitive2array(e, ref, chunk.Function.Args)
	if err != nil || rref != 0 {
		return nil, rref, err
	}

	// Note: Using the regex is slower than parsing it step by step, so this code can be improved
	m := durationRegex.FindStringSubmatch(args[0].(string))
	if m == nil {
		return nil, 0, errors.New("failed to parse duration")
	}

	num, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return nil, 0, errors.New("failed to parse duration numeric value")
	}

	var t time.Time
	scalar := strings.ToLower(m[2])
	switch scalar {
	case "s", "", "sec", "second", "seconds":
		t = DurationToTime(int64(num))
	case "m", "min", "minute", "minutes":
		t = DurationToTime(int64(num * 60))
	case "h", "hour", "hours":
		t = DurationToTime(int64(num * 60 * 60))
	case "d", "day", "days":
		t = DurationToTime(int64(num * 60 * 60 * 24))
	case "y", "year", "years":
		t = DurationToTime(int64(num * 60 * 60 * 24 * 365))
	default:
		return nil, 0, errors.New("failed to parsee duration (only supports: s/m/h/d/y)")
	}

	return TimeData(t), 0, nil
}
