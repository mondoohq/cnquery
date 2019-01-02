package llx

import (
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/types"
)

// RawData is an internal track of raw data that can be cast to the appropriate types
// It cannot be sent over the wire unless serialized (expensive) or
// converted to a proto data structure
type RawData struct {
	Type  types.Type
	Value interface{}
	Error error
}

// NilData for the nil value
var NilData = &RawData{Type: types.Nil}

// BoolData creates a rawdata struct from a go boolean
func BoolData(v bool) *RawData {
	return &RawData{
		Type:  types.Bool,
		Value: v,
	}
}

// BoolFalse is a RawData boolean set to false
var BoolFalse = BoolData(false)

// BoolTrue is a RawData boolean set to true
var BoolTrue = BoolData(true)

// IntData creates a rawdata struct from a go int
func IntData(v int64) *RawData {
	return &RawData{
		Type:  types.Int,
		Value: v,
	}
}

// FloatData creates a rawdata struct from a go float
func FloatData(v float64) *RawData {
	return &RawData{
		Type:  types.Float,
		Value: v,
	}
}

// StringData creates a rawdata struct from a go string
func StringData(s string) *RawData {
	return &RawData{
		Type:  types.String,
		Value: s,
	}
}

// RegexData creates a rawdata struct from a go string
func RegexData(r string) *RawData {
	return &RawData{
		Type:  types.Regex,
		Value: r,
	}
}

// RefData creates a rawdata struct from a go ref
func RefData(v int32) *RawData {
	return &RawData{
		Type:  types.Ref,
		Value: v,
	}
}

// ArrayData creates a rawdata struct from a go array + child data types
func ArrayData(v []interface{}, typ types.Type) *RawData {
	return &RawData{
		Type:  types.Array(typ),
		Value: v,
	}
}

// MapData creates a rawdata struct from a go map + child data types
func MapData(v map[string]interface{}, typ types.Type) *RawData {
	return &RawData{
		Type:  types.Map(types.String, typ),
		Value: v,
	}
}

// MapIntData creates a rawdata struct from a go int map + child data type
func MapIntData(v map[int32]interface{}, typ types.Type) *RawData {
	return &RawData{
		Type:  types.Map(types.Int, typ),
		Value: v,
	}
}

// ResourceData creates a rawdata struct from a resource
func ResourceData(v lumi.ResourceType, name string) *RawData {
	return &RawData{
		Type:  types.Resource(name),
		Value: v,
	}
}

// FunctionData creates a rawdata struct from a function reference
func FunctionData(v int32, sig string) *RawData {
	return &RawData{
		Type:  types.Function(0, nil),
		Value: v,
	}
}

// RawResultByRef is used to sort an array of raw results
type RawResultByRef []*RawResult

func (a RawResultByRef) Len() int           { return len(a) }
func (a RawResultByRef) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a RawResultByRef) Less(i, j int) bool { return a[i].Ref < a[j].Ref }
