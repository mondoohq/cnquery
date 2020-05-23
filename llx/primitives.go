package llx

import "go.mondoo.io/mondoo/types"

// NilPrimitive is the empty primitive
var NilPrimitive = &Primitive{Type: string(types.Nil)}

// BoolPrimitive creates a primitive from a boolean value
func BoolPrimitive(v bool) *Primitive {
	return &Primitive{
		Type:  string(types.Bool),
		Value: bool2bytes(v),
	}
}

// IntPrimitive creates a primitive from an int value
func IntPrimitive(v int64) *Primitive {
	return &Primitive{
		Type:  string(types.Int),
		Value: int2bytes(v),
	}
}

// FloatPrimitive creates a primitive from a float value
func FloatPrimitive(v float64) *Primitive {
	return &Primitive{
		Type:  string(types.Float),
		Value: float2bytes(v),
	}
}

// StringPrimitive creates a primitive from a string value
func StringPrimitive(s string) *Primitive {
	return &Primitive{
		Type:  string(types.String),
		Value: []byte(s),
	}
}

// RegexPrimitive creates a primitive from a regex in string shape
func RegexPrimitive(r string) *Primitive {
	return &Primitive{
		Type:  string(types.Regex),
		Value: []byte(r),
	}
}

// RefPrimitive creates a primitive from an int value
func RefPrimitive(v int32) *Primitive {
	return &Primitive{
		Type:  string(types.Ref),
		Value: int2bytes(int64(v)),
	}
}

// ArrayPrimitive creates a primitive from an int value
func ArrayPrimitive(v []*Primitive, childType types.Type) *Primitive {
	return &Primitive{
		Type:  string(types.Array(childType)),
		Array: v,
	}
}

// FunctionPrimitive points to a function in the call stack
func FunctionPrimitive(v int32) *Primitive {
	return &Primitive{
		// TODO: function signature
		Type:  string(types.Function(0, nil)),
		Value: int2bytes(int64(v)),
	}
}

// Ref will return the ref value unless this is not a ref type
func (p *Primitive) Ref() (int32, bool) {
	typ := types.Type(p.Type)
	if typ != types.Ref && typ.Underlying() != types.FunctionLike {
		return 0, false
	}
	return int32(bytes2int(p.Value)), true
}
