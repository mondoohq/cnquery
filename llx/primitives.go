package llx

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"time"

	"go.mondoo.io/mondoo/types"
)

// UnsetPrimitive is the unset primitive
var UnsetPrimitive = &Primitive{Type: types.Unset}

// NilPrimitive is the empty primitive
var NilPrimitive = &Primitive{Type: types.Nil}

// BoolPrimitive creates a primitive from a boolean value
func BoolPrimitive(v bool) *Primitive {
	return &Primitive{
		Type:  types.Bool,
		Value: bool2bytes(v),
	}
}

// MaxIntPrimitive is the largest integer possible
var MaxIntPrimitive = &Primitive{
	Type:  types.Int,
	Value: int2bytes(math.MaxInt64),
}

// MinIntPrimitive is the smallest integer possible
var MinIntPrimitive = &Primitive{
	Type:  types.Int,
	Value: int2bytes(math.MinInt64),
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

// NeverFutureTime is an indicator for what we consider infinity when looking at time
var NeverFutureTime = time.Unix(1<<63-1, 0)

// NeverPastTime is an indicator for what we consider negative infinity when looking at time
var NeverPastTime = time.Unix(-(1<<63 - 1), 0)

// NeverFuturePrimitive is the special time primitive for the infinite future time
var NeverFuturePrimitive = TimePrimitive(&NeverFutureTime)

// NeverPastPrimitive is the special time primitive for the infinite future time
var NeverPastPrimitive = TimePrimitive(&NeverPastTime)

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

// ArrayPrimitive creates a primitive from a list of primitives
func ArrayPrimitive(v []*Primitive, childType types.Type) *Primitive {
	return &Primitive{
		Type:  types.Array(childType),
		Array: v,
	}
}

// MapPrimitive creates a primitive from a map of primitives
func MapPrimitive(v map[string]*Primitive, childType types.Type) *Primitive {
	return &Primitive{
		Type: types.Map(types.String, childType),
		Map:  v,
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

// Label returns a printable label for this primitive
func (p *Primitive) Label(code *Code) string {
	switch p.Type.Underlying() {
	case types.Any:
		return string(p.Value)
	case types.Ref:
		return "<ref>"
	case types.Nil:
		return "null"
	case types.Bool:
		if len(p.Value) == 0 {
			return "null"
		}
		if bytes2bool(p.Value) {
			return "true"
		}
		return "false"
	case types.Int:
		if len(p.Value) == 0 {
			return "null"
		}
		data := bytes2int(p.Value)
		if data == math.MaxInt64 {
			return "Infinity"
		}
		if data == math.MinInt64 {
			return "-Infinity"
		}
		return fmt.Sprintf("%d", data)
	case types.Float:
		if len(p.Value) == 0 {
			return "null"
		}
		data := bytes2float(p.Value)
		if math.IsInf(data, 1) {
			return "Infinity"
		}
		if math.IsInf(data, -1) {
			return "-Infinity"
		}
		return fmt.Sprintf("%f", data)
	case types.String:
		if len(p.Value) == 0 {
			return "null"
		}
		return PrettyPrintString(string(p.Value))
	case types.Regex:
		if len(p.Value) == 0 {
			return "null"
		}
		return fmt.Sprintf("/%s/", string(p.Value))
	case types.Time:
		return "<...>"
	case types.Dict:
		return "<...>"
	case types.Score:
		return ScoreString(p.Value)
	case types.ArrayLike:
		if len(p.Array) == 0 {
			return "[]"
		}
		return "[..]"

	case types.MapLike:
		if len(p.Map) == 0 {
			return "{}"
		}
		return "{..}"

	case types.ResourceLike:
		return ""

	default:
		return ""
	}
}

func PrettyPrintString(s string) string {
	res := fmt.Sprintf("%#v", s)
	res = strings.ReplaceAll(res, "\\n", "\n")
	res = strings.ReplaceAll(res, "\\t", "\t")
	return res
}
