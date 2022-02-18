package llx

import (
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"go.mondoo.io/mondoo/types"
)

// run an operation that returns true/false on a bind data vs a chunk call.
// Unlike boolOp we don't check if either side is nil
func rawboolOpV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64, f func(*RawData, *RawData) bool) (*RawData, uint64, error) {
	v, dref, err := e.resolveValue(chunk.Function.Args[0], ref)
	if err != nil {
		return nil, 0, err
	}
	if dref != 0 {
		return nil, dref, nil
	}
	return BoolData(f(bind, v)), 0, nil
}

// run an operation that returns true/false on a bind data vs a chunk call.
// Unlike boolOp we don't check if either side is nil. It inverts the
// returned boolean from the child function.
func rawboolNotOpV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64, f func(*RawData, *RawData) bool) (*RawData, uint64, error) {
	v, dref, err := e.resolveValue(chunk.Function.Args[0], ref)
	if err != nil {
		return nil, 0, err
	}
	if dref != 0 {
		return nil, dref, nil
	}
	return BoolData(!f(bind, v)), 0, nil
}

// run an operation that returns true/false on a bind data vs a chunk call.
// this includes handling for the case where either side is nil, i.e.
// - if both sides are nil we return true
// - if either side is nil but not the other we return false
func boolOpV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64, f func(interface{}, interface{}) bool) (*RawData, uint64, error) {
	v, dref, err := e.resolveValue(chunk.Function.Args[0], ref)
	if err != nil {
		return nil, 0, err
	}
	if dref != 0 {
		return nil, dref, nil
	}

	if bind.Value == nil {
		return BoolData(v.Value == nil), 0, nil
	}
	if v == nil || v.Value == nil {
		return BoolData(false), 0, nil
	}

	return BoolData(f(bind.Value, v.Value)), 0, nil
}

// boolOrOp behaves like boolOp, but checks if the left argument is true first
// (and stop if it is). Only then proceeds to check the right argument.
func boolOrOpV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64, fLeft func(interface{}) bool, fRight func(interface{}) bool) (*RawData, uint64, error) {
	if bind.Value != nil && fLeft(bind.Value) {
		return BoolData(true), 0, nil
	}

	v, dref, err := e.resolveValue(chunk.Function.Args[0], ref)
	if err != nil {
		return nil, 0, err
	}
	if dref != 0 {
		return nil, dref, nil
	}

	if bind.Value == nil {
		return BoolData(v.Value == nil), 0, nil
	}
	if v == nil || v.Value == nil {
		return BoolData(false), 0, nil
	}

	return BoolData(fRight(v.Value)), 0, nil
}

// boolAndOp behaves like boolOp, but checks if the left argument is false first
// (and stop if it is). Only then proceeds to check the right argument.
func boolAndOpV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64, fLeft func(interface{}) bool, fRight func(interface{}) bool) (*RawData, uint64, error) {
	if bind.Value != nil && !fLeft(bind.Value) {
		return BoolData(false), 0, nil
	}

	v, dref, err := e.resolveValue(chunk.Function.Args[0], ref)
	if err != nil {
		return nil, 0, err
	}
	if dref != 0 {
		return nil, dref, nil
	}

	if bind.Value == nil {
		return BoolData(v.Value == nil), 0, nil
	}
	if v == nil || v.Value == nil {
		return BoolData(false), 0, nil
	}
	return BoolData(fRight(v.Value)), 0, nil
}

func boolNotOpV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64, f func(interface{}, interface{}) bool) (*RawData, uint64, error) {
	v, dref, err := e.resolveValue(chunk.Function.Args[0], ref)
	if err != nil {
		return nil, 0, err
	}
	if dref != 0 {
		return nil, dref, nil
	}

	if bind.Value == nil {
		return BoolData(v.Value != nil), 0, nil
	}
	if v == nil || v.Value == nil {
		return BoolData(true), 0, nil
	}

	return BoolData(!f(bind.Value, v.Value)), 0, nil
}

func dataOpV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64, typ types.Type, f func(interface{}, interface{}) *RawData) (*RawData, uint64, error) {
	v, dref, err := e.resolveValue(chunk.Function.Args[0], ref)
	if err != nil {
		return nil, 0, err
	}
	if dref != 0 {
		return nil, dref, nil
	}

	if bind.Value == nil {
		return &RawData{Type: typ}, 0, nil
	}
	if v == nil || v.Value == nil {
		return &RawData{Type: typ}, 0, nil
	}

	return f(bind.Value, v.Value), 0, nil
}

func nonNilDataOpV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64, typ types.Type, f func(interface{}, interface{}) *RawData) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: typ, Error: errors.New("left side of operation is null")}, 0, nil
	}

	v, dref, err := e.resolveValue(chunk.Function.Args[0], ref)
	if err != nil {
		return nil, 0, err
	}
	if dref != 0 {
		return nil, dref, nil
	}

	if v == nil || v.Value == nil {
		return &RawData{Type: typ, Error: errors.New("right side of operation is null")}, 0, nil
	}

	return f(bind.Value, v.Value), 0, nil
}

// for equality and inequality checks that are pre-determined
// we need to catch the case where both values end up nil

func chunkEqTrueV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, func(a interface{}, b interface{}) bool {
		return true
	})
}

func chunkEqFalseV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, func(a interface{}, b interface{}) bool {
		return false
	})
}

func chunkNeqFalseV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, func(a interface{}, b interface{}) bool {
		return true
	})
}

func chunkNeqTrueV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, func(a interface{}, b interface{}) bool {
		return false
	})
}

// raw operator handling
// ==   !=

func opBoolCmpBool(left interface{}, right interface{}) bool {
	return left.(bool) == right.(bool)
}

func opIntCmpInt(left interface{}, right interface{}) bool {
	return left.(int64) == right.(int64)
}

func opFloatCmpFloat(left interface{}, right interface{}) bool {
	return left.(float64) == right.(float64)
}

func opStringCmpString(left interface{}, right interface{}) bool {
	return left.(string) == right.(string)
}

func opTimeCmpTime(left interface{}, right interface{}) bool {
	l := left.(*time.Time)
	r := right.(*time.Time)
	if l == nil {
		return r == nil
	}
	if r == nil {
		return false
	}

	if (*l == NeverPastTime || *l == NeverFutureTime) && (*r == NeverPastTime || *r == NeverFutureTime) {
		return true
	}

	return (*l).Equal(*r)
}

func opStringCmpInt(left interface{}, right interface{}) bool {
	return left.(string) == strconv.FormatInt(right.(int64), 10)
}

func opIntCmpString(left interface{}, right interface{}) bool {
	return right.(string) == strconv.FormatInt(left.(int64), 10)
}

func opStringCmpFloat(left interface{}, right interface{}) bool {
	return left.(string) == strconv.FormatFloat(right.(float64), 'f', -1, 64)
}

func opFloatCmpString(left interface{}, right interface{}) bool {
	return right.(string) == strconv.FormatFloat(left.(float64), 'f', -1, 64)
}

func opStringCmpRegex(left interface{}, right interface{}) bool {
	r := regexp.MustCompile(right.(string))
	return r.Match([]byte(left.(string)))
}

func opRegexCmpString(left interface{}, right interface{}) bool {
	r := regexp.MustCompile(left.(string))
	return r.Match([]byte(right.(string)))
}

func opRegexCmpInt(left interface{}, right interface{}) bool {
	return opStringCmpRegex(strconv.FormatInt(right.(int64), 10), left.(string))
}

func opIntCmpRegex(left interface{}, right interface{}) bool {
	return opStringCmpRegex(strconv.FormatInt(left.(int64), 10), right.(string))
}

func opRegexCmpFloat(left interface{}, right interface{}) bool {
	return opStringCmpRegex(strconv.FormatFloat(right.(float64), 'f', -1, 64), left.(string))
}

func opFloatCmpRegex(left interface{}, right interface{}) bool {
	return opStringCmpRegex(strconv.FormatFloat(left.(float64), 'f', -1, 64), right.(string))
}

// same operator types
// ==   !=

func boolCmpBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opBoolCmpBool)
}

func boolNotBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opBoolCmpBool)
}

func intCmpIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opIntCmpInt)
}

func intNotIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opIntCmpInt)
}

func floatCmpFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opFloatCmpFloat)
}

func floatNotFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opFloatCmpFloat)
}

func stringCmpStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opStringCmpString)
}

func stringNotStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opStringCmpString)
}

func timeCmpTimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opTimeCmpTime)
}

func timeNotTimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opTimeCmpTime)
}

// int arithmetic

func intPlusIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Int, func(left interface{}, right interface{}) *RawData {
		res := left.(int64) + right.(int64)
		return IntData(res)
	})
}

func intMinusIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Int, func(left interface{}, right interface{}) *RawData {
		res := left.(int64) - right.(int64)
		return IntData(res)
	})
}

func intTimesIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Int, func(left interface{}, right interface{}) *RawData {
		res := left.(int64) * right.(int64)
		return IntData(res)
	})
}

func intDividedIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Int, func(left interface{}, right interface{}) *RawData {
		res := left.(int64) / right.(int64)
		return IntData(res)
	})
}

func intPlusFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := float64(left.(int64)) + right.(float64)
		return FloatData(res)
	})
}

func intMinusFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := float64(left.(int64)) - right.(float64)
		return FloatData(res)
	})
}

func intTimesFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := float64(left.(int64)) * right.(float64)
		return FloatData(res)
	})
}

func intDividedFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := float64(left.(int64)) / right.(float64)
		return FloatData(res)
	})
}

// float arithmetic

func floatPlusIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := left.(float64) + float64(right.(int64))
		return FloatData(res)
	})
}

func floatMinusIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := left.(float64) - float64(right.(int64))
		return FloatData(res)
	})
}

func floatTimesIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := left.(float64) * float64(right.(int64))
		return FloatData(res)
	})
}

func floatDividedIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := left.(float64) / float64(right.(int64))
		return FloatData(res)
	})
}

func floatPlusFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := left.(float64) + right.(float64)
		return FloatData(res)
	})
}

func floatMinusFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := left.(float64) - right.(float64)
		return FloatData(res)
	})
}

func floatTimesFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := left.(float64) * right.(float64)
		return FloatData(res)
	})
}

func floatDividedFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := left.(float64) / right.(float64)
		return FloatData(res)
	})
}

// int vs float
// int ==/!= float

func opIntCmpFloat(left interface{}, right interface{}) bool {
	return float64(left.(int64)) == right.(float64)
}

func opFloatCmpInt(left interface{}, right interface{}) bool {
	return left.(float64) == float64(right.(int64))
}

func intCmpFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opIntCmpFloat)
}

func intNotFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opIntCmpFloat)
}

func floatCmpIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opFloatCmpInt)
}

func floatNotIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opFloatCmpInt)
}

// string vs other types
// string ==/!= nil

func opStringCmpNil(left interface{}, right interface{}) bool {
	return left == nil
}

func opNilCmpString(left interface{}, right interface{}) bool {
	return right == nil
}

func stringCmpNilV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opStringCmpNil)
}

func stringNotNilV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opStringCmpNil)
}

func nilCmpStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opNilCmpString)
}

func nilNotStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opNilCmpString)
}

// string ==/!= bool

func opStringCmpBool(left interface{}, right interface{}) bool {
	if right.(bool) == true {
		return left.(string) == "true"
	}
	return left.(string) == "false"
}

func opBoolCmpString(left interface{}, right interface{}) bool {
	if left.(bool) == true {
		return right.(string) == "true"
	}
	return right.(string) == "false"
}

func stringCmpBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opStringCmpBool)
}

func stringNotBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opStringCmpBool)
}

func boolCmpStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opBoolCmpString)
}

func boolNotStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opBoolCmpString)
}

// string ==/!= int

func stringCmpIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opStringCmpInt)
}

func stringNotIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opStringCmpInt)
}

func intCmpStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opIntCmpString)
}

func intNotStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opIntCmpString)
}

// string ==/!= float

func stringCmpFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opStringCmpFloat)
}

func stringNotFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opStringCmpFloat)
}

func floatCmpStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opFloatCmpString)
}

func floatNotStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opFloatCmpString)
}

// string ==/!= regex

func stringCmpRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opStringCmpRegex)
}

func stringNotRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opStringCmpRegex)
}

func regexCmpStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opRegexCmpString)
}

func regexNotStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opRegexCmpString)
}

// regex vs other types
// int ==/!= regex

func intCmpRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opIntCmpRegex)
}

func intNotRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opIntCmpRegex)
}

func regexCmpIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opRegexCmpInt)
}

func regexNotIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opRegexCmpInt)
}

// float ==/!= regex

func floatCmpRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opFloatCmpRegex)
}

func floatNotRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opFloatCmpRegex)
}

func regexCmpFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opRegexCmpFloat)
}

func regexNotFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opRegexCmpFloat)
}

// null vs other types
// bool ==/!= nil

func opBoolCmpNil(left interface{}, right interface{}) bool {
	return left == nil
}

func opNilCmpBool(left interface{}, right interface{}) bool {
	return right == nil
}

func boolCmpNilV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opBoolCmpNil)
}

func boolNotNilV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opBoolCmpNil)
}

func nilCmpBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opNilCmpBool)
}

func nilNotBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opNilCmpBool)
}

// int ==/!= nil

func opIntCmpNil(left interface{}, right interface{}) bool {
	return left == nil
}

func opNilCmpInt(left interface{}, right interface{}) bool {
	return right == nil
}

func intCmpNilV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opIntCmpNil)
}

func intNotNilV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opIntCmpNil)
}

func nilCmpIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opNilCmpInt)
}

func nilNotIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opNilCmpInt)
}

// float ==/!= nil

func opFloatCmpNil(left interface{}, right interface{}) bool {
	return left == nil
}

func opNilCmpFloat(left interface{}, right interface{}) bool {
	return right == nil
}

func floatCmpNilV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opFloatCmpNil)
}

func floatNotNilV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opFloatCmpNil)
}

func nilCmpFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opNilCmpFloat)
}

func nilNotFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolNotOpV2(e, bind, chunk, ref, opNilCmpFloat)
}

// time ==/!= nil

func opTimeCmpNil(left *RawData, right *RawData) bool {
	return left.Value == nil || left.Value.(*time.Time) == nil
}

func opNilCmpTime(left *RawData, right *RawData) bool {
	return right.Value == nil || right.Value.(*time.Time) == nil
}

func timeCmpNilV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, opTimeCmpNil)
}

func timeNotNilV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, opTimeCmpNil)
}

func nilCmpTimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolOpV2(e, bind, chunk, ref, opNilCmpTime)
}

func nilNotTimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return rawboolNotOpV2(e, bind, chunk, ref, opNilCmpTime)
}

// string </>/<=/>= string

func stringLTStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(string) < right.(string))
	})
}

func stringLTEStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(string) <= right.(string))
	})
}

func stringGTStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(string) > right.(string))
	})
}

func stringGTEStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(string) >= right.(string))
	})
}

// int </>/<=/>= int

func intLTIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(int64) < right.(int64))
	})
}

func intLTEIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(int64) <= right.(int64))
	})
}

func intGTIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(int64) > right.(int64))
	})
}

func intGTEIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(int64) >= right.(int64))
	})
}

// float </>/<=/>= float

func floatLTFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(float64) < right.(float64))
	})
}

func floatLTEFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(float64) <= right.(float64))
	})
}

func floatGTFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(float64) > right.(float64))
	})
}

func floatGTEFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(float64) >= right.(float64))
	})
}

// time </>/<=/>= time

func timeLTTimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		l := left.(*time.Time)
		if l == nil {
			return &RawData{Type: types.Bool, Error: errors.New("left side of operation is null")}
		}
		r := right.(*time.Time)
		if r == nil {
			return &RawData{Type: types.Bool, Error: errors.New("right side of operation is null")}
		}

		return BoolData((*l).Before(*r))
	})
}

func timeLTETimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		l := left.(*time.Time)
		if l == nil {
			return &RawData{Type: types.Bool, Error: errors.New("left side of operation is null")}
		}
		r := right.(*time.Time)
		if r == nil {
			return &RawData{Type: types.Bool, Error: errors.New("right side of operation is null")}
		}

		return BoolData((*l).Before(*r) || (*l).Equal(*r))
	})
}

func timeGTTimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		l := left.(*time.Time)
		if l == nil {
			return &RawData{Type: types.Bool, Error: errors.New("left side of operation is null")}
		}
		r := right.(*time.Time)
		if r == nil {
			return &RawData{Type: types.Bool, Error: errors.New("right side of operation is null")}
		}

		return BoolData((*l).After(*r))
	})
}

func timeGTETimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		l := left.(*time.Time)
		if l == nil {
			return &RawData{Type: types.Bool, Error: errors.New("left side of operation is null")}
		}
		r := right.(*time.Time)
		if r == nil {
			return &RawData{Type: types.Bool, Error: errors.New("right side of operation is null")}
		}

		return BoolData((*l).After(*r) || (*l).Equal(*r))
	})
}

// time arithmetic

func timeMinusTimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		l := left.(*time.Time)
		r := right.(*time.Time)
		if l == nil || r == nil {
			return &RawData{Type: types.Time}
		}

		if *r == NeverPastTime {
			return NeverFuturePrimitive.RawData()
		}
		if *r == NeverFutureTime {
			return NeverPastPrimitive.RawData()
		}
		if *l == NeverPastTime {
			return NeverPastPrimitive.RawData()
		}
		if *l == NeverFutureTime {
			return NeverFuturePrimitive.RawData()
		}

		diff := l.Unix() - r.Unix()
		res := DurationToTime(diff)
		return TimeData(res)
	})
}

func opTimeTimesInt(left interface{}, right interface{}) *RawData {
	l := left.(*time.Time)
	if l == nil {
		return &RawData{Type: types.Time}
	}

	if *l == NeverPastTime {
		return NeverPastPrimitive.RawData()
	}
	if *l == NeverFutureTime {
		return NeverFuturePrimitive.RawData()
	}

	diff := TimeToDuration(l) * right.(int64)
	res := DurationToTime(diff)
	return TimeData(res)
}

func opTimeTimesFloat(left interface{}, right interface{}) *RawData {
	l := left.(*time.Time)
	if l == nil {
		return &RawData{Type: types.Time}
	}

	if *l == NeverPastTime {
		return NeverPastPrimitive.RawData()
	}
	if *l == NeverFutureTime {
		return NeverFuturePrimitive.RawData()
	}

	diff := float64(TimeToDuration(l)) * right.(float64)
	res := DurationToTime(int64(diff))
	return TimeData(res)
}

func timeTimesIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Time, opTimeTimesInt)
}

func intTimesTimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		return opTimeTimesInt(right, left)
	})
}

func timeTimesFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Time, opTimeTimesFloat)
}

func floatTimesTimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		return opTimeTimesFloat(right, left)
	})
}

// int </>/<=/>= float

func intLTFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(float64(left.(int64)) < right.(float64))
	})
}

func intLTEFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(float64(left.(int64)) <= right.(float64))
	})
}

func intGTFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(float64(left.(int64)) > right.(float64))
	})
}

func intGTEFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(float64(left.(int64)) >= right.(float64))
	})
}

// float </>/<=/>= int

func floatLTIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(float64) < float64(right.(int64)))
	})
}

func floatLTEIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(float64) <= float64(right.(int64)))
	})
}

func floatGTIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(float64) > float64(right.(int64)))
	})
}

func floatGTEIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(float64) >= float64(right.(int64)))
	})
}

// float </>/<=/>= string

func floatLTStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseFloat(right.(string), 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(left.(float64) < f)
	})
}

func floatLTEStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseFloat(right.(string), 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(left.(float64) <= f)
	})
}

func floatGTStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseFloat(right.(string), 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(left.(float64) > f)
	})
}

func floatGTEStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseFloat(right.(string), 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(left.(float64) >= f)
	})
}

// string </>/<=/>= float

func stringLTFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseFloat(left.(string), 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(f < right.(float64))
	})
}

func stringLTEFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseFloat(left.(string), 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(f <= right.(float64))
	})
}

func stringGTFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseFloat(left.(string), 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(f > right.(float64))
	})
}

func stringGTEFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseFloat(left.(string), 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(f >= right.(float64))
	})
}

// int </>/<=/>= string

func intLTStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseInt(right.(string), 10, 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(left.(int64) < f)
	})
}

func intLTEStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseInt(right.(string), 10, 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(left.(int64) <= f)
	})
}

func intGTStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseInt(right.(string), 10, 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(left.(int64) > f)
	})
}

func intGTEStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseInt(right.(string), 10, 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(left.(int64) >= f)
	})
}

// string </>/<=/>= int

func stringLTIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseInt(left.(string), 10, 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(f < right.(int64))
	})
}

func stringLTEIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseInt(left.(string), 10, 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(f <= right.(int64))
	})
}

func stringGTIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseInt(left.(string), 10, 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(f > right.(int64))
	})
}

func stringGTEIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return nonNilDataOpV2(e, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseInt(left.(string), 10, 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(f >= right.(int64))
	})
}

// ---------------------------------
//       &&  AND        ||  OR
// ---------------------------------

// T &&/|| T

func truthyBool(val interface{}) bool {
	return val.(bool)
}

func truthyInt(val interface{}) bool {
	return val.(int64) != 0
}

func truthyFloat(val interface{}) bool {
	return val.(float64) != 0
}

func truthyString(val interface{}) bool {
	return val.(string) != ""
}

func truthyArray(val interface{}) bool {
	return true
}

func truthyMap(val interface{}) bool {
	return true
}

func boolAndBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolAndOpV2(e, bind, chunk, ref, truthyBool, truthyBool)
}

func boolOrBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOrOpV2(e, bind, chunk, ref, truthyBool, truthyBool)
}

func intAndIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolAndOpV2(e, bind, chunk, ref, truthyInt, truthyInt)
}

func intOrIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOrOpV2(e, bind, chunk, ref, truthyInt, truthyInt)
}

func floatAndFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolAndOpV2(e, bind, chunk, ref, truthyFloat, truthyFloat)
}

func floatOrFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOrOpV2(e, bind, chunk, ref, truthyFloat, truthyFloat)
}

func stringAndStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolAndOpV2(e, bind, chunk, ref, truthyString, truthyString)
}

func stringOrStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOrOpV2(e, bind, chunk, ref, truthyString, truthyString)
}

func regexAndRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolAndOpV2(e, bind, chunk, ref, truthyString, truthyString)
}

func regexOrRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOrOpV2(e, bind, chunk, ref, truthyString, truthyString)
}

func arrayAndArrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolAndOpV2(e, bind, chunk, ref, truthyArray, truthyArray)
}

func arrayOrArrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOrOpV2(e, bind, chunk, ref, truthyArray, truthyArray)
}

// bool &&/|| T

func boolAndIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolAndOpV2(e, bind, chunk, ref, truthyBool, truthyInt)
}

func boolOrIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOrOpV2(e, bind, chunk, ref, truthyBool, truthyInt)
}

func intAndBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolAndOpV2(e, bind, chunk, ref, truthyInt, truthyBool)
}

func intOrBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOrOpV2(e, bind, chunk, ref, truthyInt, truthyBool)
}

func boolAndFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolAndOpV2(e, bind, chunk, ref, truthyBool, truthyFloat)
}

func boolOrFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOrOpV2(e, bind, chunk, ref, truthyBool, truthyFloat)
}

func floatAndBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolAndOpV2(e, bind, chunk, ref, truthyFloat, truthyBool)
}

func floatOrBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOrOpV2(e, bind, chunk, ref, truthyFloat, truthyBool)
}

func boolAndStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolAndOpV2(e, bind, chunk, ref, truthyBool, truthyString)
}

func boolOrStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOrOpV2(e, bind, chunk, ref, truthyBool, truthyString)
}

func stringAndBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolAndOpV2(e, bind, chunk, ref, truthyString, truthyBool)
}

func stringOrBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOrOpV2(e, bind, chunk, ref, truthyString, truthyBool)
}

func boolAndRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolAndOpV2(e, bind, chunk, ref, truthyBool, truthyString)
}

func boolOrRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOrOpV2(e, bind, chunk, ref, truthyBool, truthyString)
}

func regexAndBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolAndOpV2(e, bind, chunk, ref, truthyString, truthyBool)
}

func regexOrBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOrOpV2(e, bind, chunk, ref, truthyString, truthyBool)
}

func boolAndArrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolAndOpV2(e, bind, chunk, ref, truthyBool, truthyArray)
}

func boolOrArrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOrOpV2(e, bind, chunk, ref, truthyBool, truthyArray)
}

func arrayAndBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolAndOpV2(e, bind, chunk, ref, truthyArray, truthyBool)
}

func arrayOrBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOrOpV2(e, bind, chunk, ref, truthyArray, truthyBool)
}

func opBoolAndMap(left interface{}, right interface{}) bool {
	return left.(bool) && (len(right.([]interface{})) != 0)
}

func opMapAndBool(left interface{}, right interface{}) bool {
	return right.(bool) && (len(left.([]interface{})) != 0)
}

func opBoolOrMap(left interface{}, right interface{}) bool {
	return left.(bool) || (len(right.([]interface{})) != 0)
}

func opMapOrBool(left interface{}, right interface{}) bool {
	return right.(bool) || (len(left.([]interface{})) != 0)
}

func boolAndMapV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opBoolAndMap)
}

func boolOrMapV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opBoolOrMap)
}

func mapAndBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opMapAndBool)
}

func mapOrBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opMapOrBool)
}

// int &&/|| T

func opIntAndFloat(left interface{}, right interface{}) bool {
	return (left.(int64) != 0) && (right.(float64) != 0)
}

func opFloatAndInt(left interface{}, right interface{}) bool {
	return (right.(int64) != 0) && (left.(float64) != 0)
}

func opIntOrFloat(left interface{}, right interface{}) bool {
	return (left.(int64) != 0) || (right.(float64) != 0)
}

func opFloatOrInt(left interface{}, right interface{}) bool {
	return (right.(int64) != 0) || (left.(float64) != 0)
}

func intAndFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opIntAndFloat)
}

func intOrFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opIntOrFloat)
}

func floatAndIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opFloatAndInt)
}

func floatOrIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opFloatOrInt)
}

func opIntAndString(left interface{}, right interface{}) bool {
	return (left.(int64) != 0) && (right.(string) != "")
}

func opStringAndInt(left interface{}, right interface{}) bool {
	return (right.(int64) != 0) && (left.(string) != "")
}

func opIntOrString(left interface{}, right interface{}) bool {
	return (left.(int64) != 0) || (right.(string) != "")
}

func opStringOrInt(left interface{}, right interface{}) bool {
	return (right.(int64) != 0) || (left.(string) != "")
}

func intAndStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opIntAndString)
}

func intOrStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opIntOrString)
}

func stringAndIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opStringAndInt)
}

func stringOrIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opStringOrInt)
}

func intAndRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opIntAndString)
}

func intOrRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opIntOrString)
}

func regexAndIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opStringAndInt)
}

func regexOrIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opStringOrInt)
}

func opIntAndArray(left interface{}, right interface{}) bool {
	return (left.(int64) != 0) && (len(right.([]interface{})) != 0)
}

func opArrayAndInt(left interface{}, right interface{}) bool {
	return (right.(int64) != 0) && (len(left.([]interface{})) != 0)
}

func opIntOrArray(left interface{}, right interface{}) bool {
	return (left.(int64) != 0) || (len(right.([]interface{})) != 0)
}

func opArrayOrInt(left interface{}, right interface{}) bool {
	return (right.(int64) != 0) || (len(left.([]interface{})) != 0)
}

func intAndArrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opIntAndArray)
}

func intOrArrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opIntOrArray)
}

func arrayAndIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opArrayAndInt)
}

func arrayOrIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opArrayOrInt)
}

func opIntAndMap(left interface{}, right interface{}) bool {
	return (left.(int64) != 0) && (len(right.(map[string]interface{})) != 0)
}

func opMapAndInt(left interface{}, right interface{}) bool {
	return (right.(int64) != 0) && (len(left.(map[string]interface{})) != 0)
}

func opIntOrMap(left interface{}, right interface{}) bool {
	return (left.(int64) != 0) || (len(right.(map[string]interface{})) != 0)
}

func opMapOrInt(left interface{}, right interface{}) bool {
	return (right.(int64) != 0) || (len(left.(map[string]interface{})) != 0)
}

func intAndMapV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opIntAndMap)
}

func intOrMapV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opIntOrMap)
}

func mapAndIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opMapAndInt)
}

func mapOrIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opMapOrInt)
}

// float &&/|| T

func opFloatAndString(left interface{}, right interface{}) bool {
	return (left.(float64) != 0) && (right.(string) != "")
}

func opStringAndFloat(left interface{}, right interface{}) bool {
	return (right.(float64) != 0) && (left.(string) != "")
}

func opFloatOrString(left interface{}, right interface{}) bool {
	return (left.(float64) != 0) || (right.(string) != "")
}

func opStringOrFloat(left interface{}, right interface{}) bool {
	return (right.(float64) != 0) || (left.(string) != "")
}

func floatAndStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opFloatAndString)
}

func floatOrStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opFloatOrString)
}

func stringAndFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opStringAndFloat)
}

func stringOrFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opStringOrFloat)
}

func floatAndRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opFloatAndString)
}

func floatOrRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opFloatOrString)
}

func regexAndFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opStringAndFloat)
}

func regexOrFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opStringOrFloat)
}

func opFloatAndArray(left interface{}, right interface{}) bool {
	return (left.(float64) != 0) && (len(right.([]interface{})) != 0)
}

func opArrayAndFloat(left interface{}, right interface{}) bool {
	return (right.(float64) != 0) && (len(left.([]interface{})) != 0)
}

func opFloatOrArray(left interface{}, right interface{}) bool {
	return (left.(float64) != 0) || (len(right.([]interface{})) != 0)
}

func opArrayOrFloat(left interface{}, right interface{}) bool {
	return (right.(float64) != 0) || (len(left.([]interface{})) != 0)
}

func floatAndArrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opFloatAndArray)
}

func floatOrArrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opFloatOrArray)
}

func arrayAndFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opArrayAndFloat)
}

func arrayOrFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opArrayOrFloat)
}

func opFloatAndMap(left interface{}, right interface{}) bool {
	return (left.(float64) != 0) && (len(right.(map[string]interface{})) != 0)
}

func opMapAndFloat(left interface{}, right interface{}) bool {
	return (right.(float64) != 0) && (len(left.(map[string]interface{})) != 0)
}

func opFloatOrMap(left interface{}, right interface{}) bool {
	return (left.(float64) != 0) || (len(right.(map[string]interface{})) != 0)
}

func opMapOrFloat(left interface{}, right interface{}) bool {
	return (right.(float64) != 0) || (len(left.(map[string]interface{})) != 0)
}

func floatAndMapV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opFloatAndMap)
}

func floatOrMapV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opFloatOrMap)
}

func mapAndFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opMapAndFloat)
}

func mapOrFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opMapOrFloat)
}

// string &&/|| T

func stringAndRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolAndOpV2(e, bind, chunk, ref, truthyString, truthyString)
}

func stringOrRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOrOpV2(e, bind, chunk, ref, truthyString, truthyString)
}

func regexAndStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolAndOpV2(e, bind, chunk, ref, truthyString, truthyString)
}

func regexOrStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOrOpV2(e, bind, chunk, ref, truthyString, truthyString)
}

func opStringAndArray(left interface{}, right interface{}) bool {
	return (left.(float64) != 0) && (len(right.([]interface{})) != 0)
}

func opArrayAndString(left interface{}, right interface{}) bool {
	return (right.(float64) != 0) && (len(left.([]interface{})) != 0)
}

func opStringOrArray(left interface{}, right interface{}) bool {
	return (left.(float64) != 0) || (len(right.([]interface{})) != 0)
}

func opArrayOrString(left interface{}, right interface{}) bool {
	return (right.(float64) != 0) || (len(left.([]interface{})) != 0)
}

func stringAndArrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opStringAndArray)
}

func stringOrArrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opStringOrArray)
}

func arrayAndStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opArrayAndString)
}

func arrayOrStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opArrayOrString)
}

func opStringAndMap(left interface{}, right interface{}) bool {
	return (left.(float64) != 0) && (len(right.(map[string]interface{})) != 0)
}

func opMapAndString(left interface{}, right interface{}) bool {
	return (right.(float64) != 0) && (len(left.(map[string]interface{})) != 0)
}

func opStringOrMap(left interface{}, right interface{}) bool {
	return (left.(float64) != 0) || (len(right.(map[string]interface{})) != 0)
}

func opMapOrString(left interface{}, right interface{}) bool {
	return (right.(float64) != 0) || (len(left.(map[string]interface{})) != 0)
}

func stringAndMapV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opStringAndMap)
}

func stringOrMapV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opStringOrMap)
}

func mapAndStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opMapAndString)
}

func mapOrStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opMapOrString)
}

// string + T

func stringPlusStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return dataOpV2(e, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		l := left.(string)
		r := right.(string)

		return StringData(l + r)
	})
}

// regex &&/|| array

func regexAndArrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opStringAndArray)
}

func regexOrArrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opStringOrArray)
}

func arrayAndRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opArrayAndString)
}

func arrayOrRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opArrayOrString)
}

// regex &&/|| map

func regexAndMapV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opStringAndMap)
}

func regexOrMapV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opStringOrMap)
}

func mapAndRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opMapAndString)
}

func mapOrRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opMapOrString)
}

// time &&/|| T
// note: time is always truthy

func opBoolAndTime(left interface{}, right interface{}) bool {
	return left.(bool)
}

func opTimeAndBool(left interface{}, right interface{}) bool {
	return right.(bool)
}

func boolAndTimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opBoolAndTime)
}

func boolOrTimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return BoolTrue, 0, nil
}

func timeAndBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opTimeAndBool)
}

func timeOrBoolV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return BoolTrue, 0, nil
}

func opIntAndTime(left interface{}, right interface{}) bool {
	return left.(int64) != 0
}

func opTimeAndInt(left interface{}, right interface{}) bool {
	return right.(int64) != 0
}

func intAndTimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opIntAndTime)
}

func intOrTimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return BoolTrue, 0, nil
}

func timeAndIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opTimeAndInt)
}

func timeOrIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return BoolTrue, 0, nil
}

func opFloatAndTime(left interface{}, right interface{}) bool {
	return left.(float64) != 0
}

func opTimeAndFloat(left interface{}, right interface{}) bool {
	return right.(float64) != 0
}

func floatAndTimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opFloatAndTime)
}

func floatOrTimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return BoolTrue, 0, nil
}

func timeAndFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opTimeAndFloat)
}

func timeOrFloatV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return BoolTrue, 0, nil
}

func opStringAndTime(left interface{}, right interface{}) bool {
	return left.(string) != ""
}

func opTimeAndString(left interface{}, right interface{}) bool {
	return right.(string) != ""
}

func stringAndTimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opStringAndTime)
}

func stringOrTimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return BoolTrue, 0, nil
}

func timeAndStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opTimeAndString)
}

func timeOrStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return BoolTrue, 0, nil
}

func opRegexAndTime(left interface{}, right interface{}) bool {
	return left.(string) != ""
}

func opTimeAndRegex(left interface{}, right interface{}) bool {
	return right.(string) != ""
}

func regexAndTimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opRegexAndTime)
}

func regexOrTimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return BoolTrue, 0, nil
}

func timeAndRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opTimeAndRegex)
}

func timeOrRegexV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return BoolTrue, 0, nil
}

func timeAndTimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return BoolTrue, 0, nil
}

func timeOrTimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return BoolTrue, 0, nil
}

func opTimeAndArray(left interface{}, right interface{}) bool {
	return len(right.([]interface{})) != 0
}

func opArrayAndTime(left interface{}, right interface{}) bool {
	return len(left.([]interface{})) != 0
}

func timeAndArrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opTimeAndArray)
}

func timeOrArrayV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return BoolTrue, 0, nil
}

func arrayAndTimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opArrayAndTime)
}

func arrayOrTimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return BoolTrue, 0, nil
}

func opTimeAndMap(left interface{}, right interface{}) bool {
	return len(right.(map[string]interface{})) != 0
}

func opMapAndTime(left interface{}, right interface{}) bool {
	return len(left.(map[string]interface{})) != 0
}

func timeAndMapV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opTimeAndMap)
}

func timeOrMapV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return BoolTrue, 0, nil
}

func mapAndTimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return boolOpV2(e, bind, chunk, ref, opMapAndTime)
}

func mapOrTimeV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	return BoolTrue, 0, nil
}

// string methods

func stringContainsStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return BoolFalse, 0, nil
	}

	argRef := chunk.Function.Args[0]
	arg, rref, err := e.resolveValue(argRef, ref)
	if err != nil || rref > 0 {
		return nil, rref, err
	}

	if arg.Value == nil {
		return BoolFalse, 0, nil
	}

	ok := strings.Contains(bind.Value.(string), arg.Value.(string))
	return BoolData(ok), 0, nil
}

func stringContainsIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return BoolFalse, 0, nil
	}

	argRef := chunk.Function.Args[0]
	arg, rref, err := e.resolveValue(argRef, ref)
	if err != nil || rref > 0 {
		return nil, rref, err
	}

	if arg.Value == nil {
		return BoolFalse, 0, nil
	}

	val := strconv.FormatInt(arg.Value.(int64), 10)

	ok := strings.Contains(bind.Value.(string), val)
	return BoolData(ok), 0, nil
}

func stringContainsArrayStringV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return BoolFalse, 0, nil
	}

	argRef := chunk.Function.Args[0]
	arg, rref, err := e.resolveValue(argRef, ref)
	if err != nil || rref > 0 {
		return nil, rref, err
	}

	if arg.Value == nil {
		return BoolFalse, 0, nil
	}

	var ok bool
	arr := arg.Value.([]interface{})
	for i := range arr {
		v := arr[i].(string)
		ok = strings.Contains(bind.Value.(string), v)
		if ok {
			return BoolData(ok), 0, nil
		}
	}

	return BoolData(false), 0, nil
}

func stringContainsArrayIntV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return BoolFalse, 0, nil
	}

	argRef := chunk.Function.Args[0]
	arg, rref, err := e.resolveValue(argRef, ref)
	if err != nil || rref > 0 {
		return nil, rref, err
	}

	if arg.Value == nil {
		return BoolFalse, 0, nil
	}

	var ok bool
	arr := arg.Value.([]interface{})
	for i := range arr {
		v := arr[i].(int64)
		val := strconv.FormatInt(v, 10)
		ok = strings.Contains(bind.Value.(string), val)
		if ok {
			return BoolData(ok), 0, nil
		}
	}

	return BoolData(false), 0, nil
}

func stringFindV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return ArrayData([]interface{}{}, types.String), 0, nil
	}

	argRef := chunk.Function.Args[0]
	arg, rref, err := e.resolveValue(argRef, ref)
	if err != nil || rref > 0 {
		return nil, rref, err
	}

	if arg.Value == nil {
		return ArrayData([]interface{}{}, types.String), 0, nil
	}

	reContent := arg.Value.(string)
	re, err := regexp.Compile(reContent)
	if err != nil {
		return nil, 0, errors.New("Failed to compile regular expression: " + reContent)
	}

	list := re.FindAllString(bind.Value.(string), -1)
	res := make([]interface{}, len(list))
	for i := range list {
		res[i] = list[i]
	}

	return ArrayData(res, types.String), 0, nil
}

func stringDowncaseV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: bind.Type}, 0, nil
	}

	res := strings.ToLower(bind.Value.(string))
	return StringData(res), 0, nil
}

var camelCaseRe = regexp.MustCompile(`([[:punct:]]|\s)+[\p{L}]`)

func stringCamelcaseV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: bind.Type}, 0, nil
	}

	res := camelCaseRe.ReplaceAllStringFunc(bind.Value.(string), func(in string) string {
		reader := strings.NewReader(in)
		var last rune
		for {
			r, _, err := reader.ReadRune()
			if err == io.EOF {
				break
			}
			if err != nil {
				return in
			}
			last = r
		}

		return string(unicode.ToTitle(last))
	})

	return StringData(res), 0, nil
}

func stringUpcaseV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: bind.Type}, 0, nil
	}

	res := strings.ToUpper(bind.Value.(string))
	return StringData(res), 0, nil
}

func stringLengthV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Int}, 0, nil
	}

	l := len(bind.Value.(string))
	return IntData(int64(l)), 0, nil
}

func stringLinesV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Array(types.String)}, 0, nil
	}

	s := bind.Value.(string)
	lines := strings.Split(s, "\n")
	res := make([]interface{}, len(lines))
	for i := range lines {
		res[i] = lines[i]
	}

	return ArrayData(res, types.String), 0, nil
}

func stringSplitV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Array(types.String)}, 0, nil
	}

	argRef := chunk.Function.Args[0]
	arg, rref, err := e.resolveValue(argRef, ref)
	if err != nil || rref > 0 {
		return nil, rref, err
	}

	if arg.Value == nil {
		return &RawData{
			Type:  types.Array(types.String),
			Value: nil,
			Error: errors.New("failed to split string, seperator was null"),
		}, 0, nil
	}

	splits := strings.Split(bind.Value.(string), arg.Value.(string))
	res := make([]interface{}, len(splits))
	for i := range splits {
		res[i] = splits[i]
	}

	return ArrayData(res, types.String), 0, nil
}

func stringTrimV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	if bind.Value == nil {
		return &RawData{Type: bind.Type}, 0, nil
	}

	cutset := " \t\n\r"

	if len(chunk.Function.Args) != 0 {
		argRef := chunk.Function.Args[0]
		arg, rref, err := e.resolveValue(argRef, ref)
		if err != nil || rref > 0 {
			return nil, rref, err
		}

		if arg.Value == nil {
			return &RawData{
				Type:  bind.Type,
				Value: nil,
				Error: errors.New("failed to trim string, cutset was null"),
			}, 0, nil
		}

		cutset = arg.Value.(string)
	}

	res := strings.Trim(bind.Value.(string), cutset)

	return StringData(res), 0, nil
}

// time methods

// zeroTimeOffset to help convert unix times into base times that start at the year 0
const zeroTimeOffset int64 = -62167219200

// TimeToDuration takes a regular time object and treats it as a duration and gets the duration in seconds
func TimeToDuration(t *time.Time) int64 {
	return t.Unix() - zeroTimeOffset
}

// DurationToTime takes a duration in seconds and turns it into a time object
func DurationToTime(i int64) time.Time {
	return time.Unix(i+zeroTimeOffset, 0)
}

func timeSecondsV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	t := bind.Value.(*time.Time)
	if t == nil {
		return &RawData{Type: types.Array(types.Time)}, 0, nil
	}

	if *t == NeverPastTime {
		return MinIntPrimitive.RawData(), 0, nil
	}
	if *t == NeverFutureTime {
		return MaxIntPrimitive.RawData(), 0, nil
	}

	raw := TimeToDuration(t)
	return IntData(int64(raw)), 0, nil
}

func timeMinutesV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	t := bind.Value.(*time.Time)
	if t == nil {
		return &RawData{Type: types.Array(types.Time)}, 0, nil
	}

	if *t == NeverPastTime {
		return MinIntPrimitive.RawData(), 0, nil
	}
	if *t == NeverFutureTime {
		return MaxIntPrimitive.RawData(), 0, nil
	}

	raw := TimeToDuration(t) / 60
	return IntData(int64(raw)), 0, nil
}

func timeHoursV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	t := bind.Value.(*time.Time)
	if t == nil {
		return &RawData{Type: types.Array(types.Time)}, 0, nil
	}

	if *t == NeverPastTime {
		return MinIntPrimitive.RawData(), 0, nil
	}
	if *t == NeverFutureTime {
		return MaxIntPrimitive.RawData(), 0, nil
	}

	raw := TimeToDuration(t) / (60 * 60)
	return IntData(int64(raw)), 0, nil
}

func timeDaysV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	t := bind.Value.(*time.Time)
	if t == nil {
		return &RawData{Type: types.Array(types.Time)}, 0, nil
	}

	if *t == NeverPastTime {
		return MinIntPrimitive.RawData(), 0, nil
	}
	if *t == NeverFutureTime {
		return MaxIntPrimitive.RawData(), 0, nil
	}

	raw := TimeToDuration(t) / (60 * 60 * 24)
	return IntData(int64(raw)), 0, nil
}

func timeUnixV2(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	t := bind.Value.(*time.Time)
	if t == nil {
		return &RawData{Type: types.Array(types.Time)}, 0, nil
	}

	raw := t.Unix()
	return IntData(int64(raw)), 0, nil
}
