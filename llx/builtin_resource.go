package llx

import (
	"errors"
	"strings"
	"time"

	"go.mondoo.io/mondoo/lumi"
)

// resourceFunctions are all the shared handlers for resource calls
var resourceFunctions map[string]chunkHandler

func resourceWhere(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
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
	for i := range list {
		c.runFunctionBlock(&RawData{Type: ct, Value: list[i]}, f, func(res *RawResult) {
			_, ok := filteredList[i]
			if !ok {
				finishedResults++
			}

			isTruthy, _ := res.Data.IsTruthy()
			if isTruthy {
				filteredList[i] = list[i]
			} else {
				filteredList[i] = nil
			}

			// log.Debug().Int("cur", finishedResults).Int("max", len(list)).Msg("finished one where-result")

			if finishedResults == len(list) {
				resList := []interface{}{}
				for j := 0; j < len(filteredList); j++ {
					k := filteredList[j]
					if k != nil {
						resList = append(resList, k)
					}
				}

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
				if err != nil {
					c.cache.Store(ref, &stepCache{Result: &RawData{
						Error: errors.New("Failed to create filter result resource: " + err.Error()),
					}})
				} else {
					c.cache.Store(ref, &stepCache{
						Result: &RawData{
							Type:  bind.Type,
							Value: resResource,
						},
						IsStatic: false,
					})
				}
				c.triggerChain(ref)
			}
		})
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

	return TimeData(&parsed), 0, nil
}
