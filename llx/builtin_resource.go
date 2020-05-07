package llx

import (
	"errors"

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

	resource := bind.Value.(lumi.ResourceType).LumiResource()

	arg1 := chunk.Function.Args[1]
	fref, ok := arg1.Ref()
	if !ok {
		return nil, 0, errors.New("Failed to retrieve function reference of 'where' call")
	}

	f := c.code.Functions[fref-1]
	list := items.Value.([]interface{})
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

			if finishedResults == len(list) {
				resList := []interface{}{}
				for j := 0; j < len(filteredList); j++ {
					k := filteredList[j]
					if k != nil {
						resList = append(resList, k)
					}
				}

				resResource, err := c.runtime.CreateResourceWithID(resource.Name, f.Id, "list", resList)
				if err != nil {
					c.cache.Store(ref, &stepCache{Result: &RawData{
						Error: errors.New("Failed to create filter result resource: " + err.Error()),
					}})
					return
				}
				c.cache.Store(ref, &stepCache{
					Result: &RawData{
						Type:  bind.Type,
						Value: resResource,
					},
					IsStatic: false,
				})
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
