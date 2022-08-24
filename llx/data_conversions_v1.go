package llx

import (
	"go.mondoo.com/cnquery/types"
)

// returns the resolved argument if it's a ref; otherwise just the argument
// returns the reference if something else needs executing before it can be computed
// returns an error otherwise
func (c *MQLExecutorV1) resolveValue(arg *Primitive, ref int32) (*RawData, int32, error) {
	typ := types.Type(arg.Type)
	switch typ.Underlying() {
	case types.Ref:
		srcRef := int32(bytes2int(arg.Value))
		// check if the reference exists; if not connect it
		res, ok := c.cache.Load(srcRef)
		if !ok {
			return c.connectRef(srcRef, ref)
		}
		return res.Result, 0, res.Result.Error

	case types.ArrayLike:
		res := make([]interface{}, len(arg.Array))
		for i := range arg.Array {
			c, ref, err := c.resolveValue(arg.Array[i], ref)
			if ref != 0 || err != nil {
				return nil, ref, err
			}
			res[i] = c.Value
		}

		// type is in arg.Value
		return &RawData{
			Type:  typ,
			Value: res,
		}, 0, nil
	}

	v := arg.RawData()
	return v, 0, v.Error
}

func args2resourceargsV1(c *MQLExecutorV1, ref int32, args []*Primitive) ([]interface{}, int32, error) {
	if args == nil {
		return []interface{}{}, 0, nil
	}

	res := make([]interface{}, len(args))
	for i := range args {
		var cur *RawData

		if types.Type(args[i].Type) == types.Ref {
			var rref int32
			var err error
			cur, rref, err = c.resolveValue(args[i], ref)
			if rref > 0 || err != nil {
				return nil, rref, err
			}
		} else {
			cur = args[i].RawData()
		}

		if cur != nil {
			if cur.Error != nil {
				return nil, 0, cur.Error
			}
			res[i] = cur.Value
		}
	}
	return res, 0, nil
}
