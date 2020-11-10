package llx

import (
	"encoding/binary"
	"time"

	"go.mondoo.io/mondoo/types"
)

// NilPrimitive is the empty primitive
var NilPrimitive = &Primitive{Type: types.Nil}

// BoolPrimitive creates a primitive from a boolean value
func BoolPrimitive(v bool) *Primitive {
	return &Primitive{
		Type:  types.Bool,
		Value: bool2bytes(v),
	}
}

// IntPrimitive creates a primitive from an int value
func IntPrimitive(v int64) *Primitive {
	return &Primitive{
		Type:  types.Int,
		Value: int2bytes(v),
	}
}

// FloatPrimitive creates a primitive from a float value
func FloatPrimitive(v float64) *Primitive {
	return &Primitive{
		Type:  types.Float,
		Value: float2bytes(v),
	}
}

// StringPrimitive creates a primitive from a string value
func StringPrimitive(s string) *Primitive {
	return &Primitive{
		Type:  types.String,
		Value: []byte(s),
	}
}

// RegexPrimitive creates a primitive from a regex in string shape
func RegexPrimitive(r string) *Primitive {
	return &Primitive{
		Type:  types.Regex,
		Value: []byte(r),
	}
}

// borrowed from time library.
// these will help with representing everything
const nsecMask = 1<<30 - 1
const nsecShift = 30

// TimePrimitive creates a primitive from a time value
func TimePrimitive(t *time.Time) *Primitive {
	if t == nil {
		return NilPrimitive
	}

	seconds := t.Unix()
	nanos := int32(t.UnixNano() % 1e9)

	v := make([]byte, 12)
	binary.LittleEndian.PutUint64(v, uint64(seconds))
	binary.LittleEndian.PutUint32(v[8:], uint32(nanos))

	return &Primitive{
		Type:  types.Time,
		Value: v,
	}
}

// ScorePrimitive creates a primitive with a numeric score
func ScorePrimitive(num int32) *Primitive {
	v, err := scoreVector(num)
	if err != nil {
		panic(err.Error())
	}

	return &Primitive{
		Type:  types.Score,
		Value: v,
	}
}

// CvssScorePrimitive creates a primitive for a CVSS score
func CvssScorePrimitive(vector string) *Primitive {
	b, err := scoreString(vector)
	if err != nil {
		panic(err.Error())
	}

	return &Primitive{
		Type:  types.Score,
		Value: b,
	}
}

// RefPrimitive creates a primitive from an int value
func RefPrimitive(v int32) *Primitive {
	return &Primitive{
		Type:  types.Ref,
		Value: int2bytes(int64(v)),
	}
}

// ArrayPrimitive creates a primitive from an int value
func ArrayPrimitive(v []*Primitive, childType types.Type) *Primitive {
	return &Primitive{
		Type:  types.Array(childType),
		Array: v,
	}
}

// FunctionPrimitive points to a function in the call stack
func FunctionPrimitive(v int32) *Primitive {
	return &Primitive{
		// TODO: function signature
		Type:  types.Function(0, nil),
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
