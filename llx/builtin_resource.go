package llx

import (
	"errors"
	"strings"
	"sync"
	"time"

	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/types"
)

// resourceFunctions are all the shared handlers for resource calls
var resourceFunctions map[string]chunkHandler

func _resourceWhere(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32, invert bool) (*RawData, int32, error) {
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

	resource := bind.Value.(lumi.ResourceType)

	arg1 := chunk.Function.Args[1]
	fref, ok := arg1.Ref()
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
		err := c.runFunctionBlock([]*RawData{&RawData{Type: ct, Value: list[i]}}, f, func(res *RawResult) {
			resList := func() []interface{} {
				l.Lock()
				defer l.Unlock()

				_, ok := filteredList[i]
				if !ok {
					finishedResults++
				}

				isTruthy, _ := res.Data.IsTruthy()
				if isTruthy == !invert {
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
				// get all mandatory args
				lumiResource := resource.LumiResource()
				resourceInfo := lumiResource.Runtime.Registry.Resources[lumiResource.Name]
				args := []interface{}{
					"list", resList, "__id", f.Id,
				}
				for k, v := range resourceInfo.Fields {
					if k != "list" && v.Mandatory {
						if v, err := resource.Field(k); err == nil {
							args = append(args, k, v)
						}
					}
				}

				resResource, err := c.runtime.CreateResourceWithID(lumiResource.Name, f.Id, args...)
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
			}
		})
		if err != nil {
			return nil, 0, err
		}
	}

	return nil, 0, nil
}

func resourceWhere(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return _resourceWhere(c, bind, chunk, ref, false)
}

func resourceWhereNot(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return _resourceWhere(c, bind, chunk, ref, true)
}

func resourceMap(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
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
	fref, ok := arg1.Ref()
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
					Result: data,
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

func resourceLength(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	// length(resource.list)
	itemsRef := chunk.Function.Args[0]
	items, rref, err := c.resolveValue(itemsRef, ref)
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

func resourceDate(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	args, rref, err := args2resourceargs(c, ref, chunk.Function.Args)
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
