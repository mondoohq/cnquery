// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package llx

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/types"
)

// UnsetPrimitive is the unset primitive
var UnsetPrimitive = &Primitive{Type: string(types.Unset)}

// NilPrimitive is the empty primitive
var NilPrimitive = &Primitive{Type: string(types.Nil)}

// BoolPrimitive creates a primitive from a boolean value
func BoolPrimitive(v bool) *Primitive {
	return &Primitive{
		Type:  string(types.Bool),
		Value: bool2bytes(v),
	}
}

// MaxIntPrimitive is the largest integer possible
var MaxIntPrimitive = &Primitive{
	Type:  string(types.Int),
	Value: int2bytes(math.MaxInt64),
}

// MinIntPrimitive is the smallest integer possible
var MinIntPrimitive = &Primitive{
	Type:  string(types.Int),
	Value: int2bytes(math.MinInt64),
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

// TimePrimitive creates a primitive from a time value
func TimePrimitive(t *time.Time) *Primitive {
	if t == nil {
		return NilPrimitive
	}

	seconds := t.Unix()
	nanos := t.Nanosecond()

	v := make([]byte, 12)
	binary.LittleEndian.PutUint64(v, uint64(seconds))
	binary.LittleEndian.PutUint32(v[8:], uint32(nanos))

	return &Primitive{
		Type:  string(types.Time),
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
		Type:  string(types.Score),
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
		Type:  string(types.Score),
		Value: b,
	}
}

// RefPrimitive creates a primitive from an int value
func RefPrimitiveV2(v uint64) *Primitive {
	return &Primitive{
		Type:  string(types.Ref),
		Value: int2bytes(int64(v)),
	}
}

// EmptyPrimitive is the empty value indicator
var EmptyPrimitive = &Primitive{Type: string(types.Empty)}

// ArrayPrimitive creates a primitive from a list of primitives
func ArrayPrimitive(v []*Primitive, childType types.Type) *Primitive {
	return &Primitive{
		Type:  string(types.Array(childType)),
		Array: v,
	}
}

// ArrayPrimitiveT create a primitive from an array of type T
func ArrayPrimitiveT[T any](v []T, f func(T) *Primitive, typ types.Type) *Primitive {
	vt := make([]*Primitive, len(v))
	for i := range v {
		vt[i] = f(v[i])
	}
	return ArrayPrimitive(vt, typ)
}

// MapPrimitive creates a primitive from a map of primitives
func MapPrimitive(v map[string]*Primitive, childType types.Type) *Primitive {
	return &Primitive{
		Type: string(types.Map(types.String, childType)),
		Map:  v,
	}
}

// MapPrimitive creates a primitive from a map of type T
func MapPrimitiveT[T any](v map[string]T, f func(T) *Primitive, typ types.Type) *Primitive {
	vt := make(map[string]*Primitive, len(v))
	for i := range v {
		vt[i] = f(v[i])
	}
	return MapPrimitive(vt, typ)
}

// FunctionPrimitive points to a function in the call stack
func FunctionPrimitiveV1(v int32) *Primitive {
	return &Primitive{
		// TODO: function signature
		Type:  string(types.Function(0, nil)),
		Value: int2bytes(int64(v)),
	}
}

// FunctionPrimitive points to a function in the call stack
func FunctionPrimitive(v uint64) *Primitive {
	return &Primitive{
		// TODO: function signature
		Type:  string(types.Function(0, nil)),
		Value: int2bytes(int64(v)),
	}
}

// RangePrimitive creates a range primitive from the given
// range data. Use the helper functions to initialize and
// combine multiple sets of range data.
func RangePrimitive(data RangeData) *Primitive {
	return &Primitive{
		Type:  string(types.Range),
		Value: data,
	}
}

type RangeData []byte

const (
	// Byte indicators for ranges work like this:
	//
	// Byte1:    version + mode
	// xxxx xxxx
	// VVVV -------> version for the range
	//      MMMM --> 1 = single line
	//               2 = line range
	//               3 = line with column range
	//               4 = line + column range
	//
	// Byte2+:   length indicators
	// xxxx xxxx
	// NNNN -------> length of the first entry (up to 128bit)
	//      MMMM --> length of the second entry (up to 128bit)
	//               note: currently we only support up to 32bit
	//
	rangeVersion1 byte = 0x10
)

func NewRange() RangeData {
	return []byte{}
}

func (r RangeData) AddLine(line uint32) RangeData {
	r = append(r, rangeVersion1|0x01)
	bytes := int2bytes(int64(line))
	r = append(r, byte(len(bytes)<<4))
	r = append(r, bytes...)
	return r
}

func (r RangeData) AddLineRange(line1 uint32, line2 uint32) RangeData {
	r = append(r, rangeVersion1|0x02)
	bytes1 := int2bytes(int64(line1))
	bytes2 := int2bytes(int64(line2))
	r = append(r, byte(len(bytes1)<<4)|byte(len(bytes2)&0x0f))
	r = append(r, bytes1...)
	r = append(r, bytes2...)
	return r
}

func (r RangeData) AddColumnRange(line uint32, column1 uint32, column2 uint32) RangeData {
	r = append(r, rangeVersion1|0x03)
	bytes := int2bytes(int64(line))
	bytes1 := int2bytes(int64(column1))
	bytes2 := int2bytes(int64(column2))

	r = append(r, byte(len(bytes)<<4))
	r = append(r, bytes...)

	r = append(r, byte(len(bytes1)<<4)|byte(len(bytes2)&0xf))
	r = append(r, bytes1...)
	r = append(r, bytes2...)
	return r
}

func (r RangeData) AddLineColumnRange(line1 uint32, line2 uint32, column1 uint32, column2 uint32) RangeData {
	r = append(r, rangeVersion1|0x04)
	bytes1 := int2bytes(int64(line1))
	bytes2 := int2bytes(int64(line2))
	r = append(r, byte(len(bytes1)<<4)|byte(len(bytes2)&0xf))
	r = append(r, bytes1...)
	r = append(r, bytes2...)

	bytes1 = int2bytes(int64(column1))
	bytes2 = int2bytes(int64(column2))
	r = append(r, byte(len(bytes1)<<4)|byte(len(bytes2)&0xf))
	r = append(r, bytes1...)
	r = append(r, bytes2...)

	return r
}

func (r RangeData) ExtractNext() ([]uint32, RangeData) {
	if len(r) == 0 {
		return nil, nil
	}

	version := r[0] & 0xf0
	if version != rangeVersion1 {
		log.Error().Msg("failed to extract range, version is unsupported")
		return nil, nil
	}

	entries := r[0] & 0x0f
	res := []uint32{}
	idx := 1
	switch entries {
	case 3, 4:
		l1 := int((r[idx] & 0xf0) >> 4)
		l2 := int(r[idx] & 0x0f)

		idx++
		if l1 != 0 {
			n := bytes2int(r[idx : idx+l1])
			idx += l1
			res = append(res, uint32(n))
		}
		if l2 != 0 {
			n := bytes2int(r[idx : idx+l1])
			idx += l2
			res = append(res, uint32(n))
		}

		fallthrough

	case 1, 2:
		l1 := int((r[idx] & 0xf0) >> 4)
		l2 := int(r[idx] & 0x0f)

		idx++
		if l1 != 0 {
			n := bytes2int(r[idx : idx+l1])
			idx += l1
			res = append(res, uint32(n))
		}
		if l2 != 0 {
			n := bytes2int(r[idx : idx+l1])
			idx += l2
			res = append(res, uint32(n))
		}

	default:
		log.Error().Msg("failed to extract range, wrong number of entries")
		return nil, nil
	}

	return res, r[idx:]
}

func (r RangeData) ExtractAll() [][]uint32 {
	res := [][]uint32{}
	for {
		cur, rest := r.ExtractNext()
		if len(cur) != 0 {
			res = append(res, cur)
		}
		if len(rest) == 0 {
			break
		}
		r = rest
	}

	return res
}

func (r RangeData) String() string {
	var res strings.Builder

	items := r.ExtractAll()
	for i := range items {
		x := items[i]
		switch len(x) {
		case 1:
			res.WriteString(strconv.Itoa(int(x[0])))
		case 2:
			res.WriteString(strconv.Itoa(int(x[0])))
			res.WriteString("-")
			res.WriteString(strconv.Itoa(int(x[1])))
		case 3:
			res.WriteString(strconv.Itoa(int(x[0])))
			res.WriteString(":")
			res.WriteString(strconv.Itoa(int(x[1])))
			res.WriteString("-")
			res.WriteString(strconv.Itoa(int(x[2])))
		case 4:
			res.WriteString(strconv.Itoa(int(x[0])))
			res.WriteString(":")
			res.WriteString(strconv.Itoa(int(x[2])))
			res.WriteString("-")
			res.WriteString(strconv.Itoa(int(x[1])))
			res.WriteString(":")
			res.WriteString(strconv.Itoa(int(x[3])))
		}

		if i != len(items)-1 {
			res.WriteString(",")
		}
	}

	return res.String()
}

// Ref will return the ref value unless this is not a ref type
func (p *Primitive) RefV1() (int32, bool) {
	typ := types.Type(p.Type)
	if typ != types.Ref && typ.Underlying() != types.FunctionLike {
		return 0, false
	}
	return int32(bytes2int(p.Value)), true
}

// Ref will return the ref value unless this is not a ref type
func (p *Primitive) RefV2() (uint64, bool) {
	typ := types.Type(p.Type)
	if typ != types.Ref && typ.Underlying() != types.FunctionLike {
		return 0, false
	}
	return uint64(bytes2int(p.Value)), true
}

// Label returns a printable label for this primitive
func (p *Primitive) LabelV1(code *CodeV1) string {
	switch types.Type(p.Type).Underlying() {
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

// Label returns a printable label for this primitive
func (p *Primitive) LabelV2(code *CodeV2) string {
	switch types.Type(p.Type).Underlying() {
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

	case types.Range:
		return RangeData(p.Value).String()

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

// Estimation based on https://golang.org/src/runtime/slice.go
const arrayOverhead = 2 * 4

// Estimation based on https://golang.org/src/runtime/slice.go
const mapOverhead = 4 + 1 + 1 + 2 + 4

// Size returns the approximate size of the primitive in bytes
func (p *Primitive) Size() int {
	typ := types.Type(p.Type)

	if typ.NotSet() {
		return 0
	}

	if typ.IsArray() {
		var res int
		for i := range p.Array {
			res += p.Array[i].Size()
		}

		return res + arrayOverhead
	}

	if typ.IsMap() {
		// We are under-estimating the real size of maps in memory, because buckets
		// actually use more room than the calculation below suggests. However,
		// for the sake of approximation, it serves well enough.

		var res int
		for k, v := range p.Map {
			res += len(k)
			res += v.Size()
		}

		return res + mapOverhead
	}

	return len(p.Value)
}

// IsNil returns true if a primitive is nil. A primitive is nil if it's type is nil,
// or if it has no associated value. The exception is the string type. If an empty
// bytes field is serialized (for example for an empty string), that field is nil.
func (p *Primitive) IsNil() bool {
	return p == nil ||
		p.Type == string(types.Nil)
}
