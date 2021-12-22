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
func rawboolOp(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32, f func(*RawData, *RawData) bool) (*RawData, int32, error) {
	v, dref, err := c.resolveValue(chunk.Function.Args[0], ref)
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
func rawboolNotOp(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32, f func(*RawData, *RawData) bool) (*RawData, int32, error) {
	v, dref, err := c.resolveValue(chunk.Function.Args[0], ref)
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
func boolOp(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32, f func(interface{}, interface{}) bool) (*RawData, int32, error) {
	v, dref, err := c.resolveValue(chunk.Function.Args[0], ref)
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
func boolOrOp(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32, fLeft func(interface{}) bool, fRight func(interface{}) bool) (*RawData, int32, error) {
	if bind.Value != nil && fLeft(bind.Value) {
		return BoolData(true), 0, nil
	}

	v, dref, err := c.resolveValue(chunk.Function.Args[0], ref)
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
func boolAndOp(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32, fLeft func(interface{}) bool, fRight func(interface{}) bool) (*RawData, int32, error) {
	if bind.Value != nil && !fLeft(bind.Value) {
		return BoolData(false), 0, nil
	}

	v, dref, err := c.resolveValue(chunk.Function.Args[0], ref)
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

func boolNotOp(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32, f func(interface{}, interface{}) bool) (*RawData, int32, error) {
	v, dref, err := c.resolveValue(chunk.Function.Args[0], ref)
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

func dataOp(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32, typ types.Type, f func(interface{}, interface{}) *RawData) (*RawData, int32, error) {
	v, dref, err := c.resolveValue(chunk.Function.Args[0], ref)
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

func nonNilDataOp(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32, typ types.Type, f func(interface{}, interface{}) *RawData) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: typ, Error: errors.New("left side of operation is null")}, 0, nil
	}

	v, dref, err := c.resolveValue(chunk.Function.Args[0], ref)
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

func chunkEqTrue(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, func(a interface{}, b interface{}) bool {
		return true
	})
}

func chunkEqFalse(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, func(a interface{}, b interface{}) bool {
		return false
	})
}

func chunkNeqFalse(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOp(c, bind, chunk, ref, func(a interface{}, b interface{}) bool {
		return true
	})
}

func chunkNeqTrue(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOp(c, bind, chunk, ref, func(a interface{}, b interface{}) bool {
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

func boolCmpBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opBoolCmpBool)
}

func boolNotBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOp(c, bind, chunk, ref, opBoolCmpBool)
}

func intCmpInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opIntCmpInt)
}

func intNotInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOp(c, bind, chunk, ref, opIntCmpInt)
}

func floatCmpFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opFloatCmpFloat)
}

func floatNotFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOp(c, bind, chunk, ref, opFloatCmpFloat)
}

func stringCmpString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opStringCmpString)
}

func stringNotString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOp(c, bind, chunk, ref, opStringCmpString)
}

func timeCmpTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opTimeCmpTime)
}

func timeNotTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOp(c, bind, chunk, ref, opTimeCmpTime)
}

// int arithmetic

func intPlusInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, types.Int, func(left interface{}, right interface{}) *RawData {
		res := left.(int64) + right.(int64)
		return IntData(res)
	})
}

func intMinusInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, types.Int, func(left interface{}, right interface{}) *RawData {
		res := left.(int64) - right.(int64)
		return IntData(res)
	})
}

func intTimesInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, types.Int, func(left interface{}, right interface{}) *RawData {
		res := left.(int64) * right.(int64)
		return IntData(res)
	})
}

func intDividedInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, types.Int, func(left interface{}, right interface{}) *RawData {
		res := left.(int64) / right.(int64)
		return IntData(res)
	})
}

func intPlusFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := float64(left.(int64)) + right.(float64)
		return FloatData(res)
	})
}

func intMinusFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := float64(left.(int64)) - right.(float64)
		return FloatData(res)
	})
}

func intTimesFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := float64(left.(int64)) * right.(float64)
		return FloatData(res)
	})
}

func intDividedFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := float64(left.(int64)) / right.(float64)
		return FloatData(res)
	})
}

// float arithmetic

func floatPlusInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := left.(float64) + float64(right.(int64))
		return FloatData(res)
	})
}

func floatMinusInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := left.(float64) - float64(right.(int64))
		return FloatData(res)
	})
}

func floatTimesInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := left.(float64) * float64(right.(int64))
		return FloatData(res)
	})
}

func floatDividedInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := left.(float64) / float64(right.(int64))
		return FloatData(res)
	})
}

func floatPlusFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := left.(float64) + right.(float64)
		return FloatData(res)
	})
}

func floatMinusFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := left.(float64) - right.(float64)
		return FloatData(res)
	})
}

func floatTimesFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := left.(float64) * right.(float64)
		return FloatData(res)
	})
}

func floatDividedFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
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

func intCmpFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opIntCmpFloat)
}

func intNotFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOp(c, bind, chunk, ref, opIntCmpFloat)
}

func floatCmpInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opFloatCmpInt)
}

func floatNotInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOp(c, bind, chunk, ref, opFloatCmpInt)
}

// string vs other types
// string ==/!= nil

func opStringCmpNil(left interface{}, right interface{}) bool {
	return left == nil
}

func opNilCmpString(left interface{}, right interface{}) bool {
	return right == nil
}

func stringCmpNil(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opStringCmpNil)
}

func stringNotNil(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOp(c, bind, chunk, ref, opStringCmpNil)
}

func nilCmpString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opNilCmpString)
}

func nilNotString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOp(c, bind, chunk, ref, opNilCmpString)
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

func stringCmpBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opStringCmpBool)
}

func stringNotBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOp(c, bind, chunk, ref, opStringCmpBool)
}

func boolCmpString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opBoolCmpString)
}

func boolNotString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOp(c, bind, chunk, ref, opBoolCmpString)
}

// string ==/!= int

func stringCmpInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opStringCmpInt)
}

func stringNotInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOp(c, bind, chunk, ref, opStringCmpInt)
}

func intCmpString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opIntCmpString)
}

func intNotString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOp(c, bind, chunk, ref, opIntCmpString)
}

// string ==/!= float

func stringCmpFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opStringCmpFloat)
}

func stringNotFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOp(c, bind, chunk, ref, opStringCmpFloat)
}

func floatCmpString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opFloatCmpString)
}

func floatNotString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOp(c, bind, chunk, ref, opFloatCmpString)
}

// string ==/!= regex

func stringCmpRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opStringCmpRegex)
}

func stringNotRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOp(c, bind, chunk, ref, opStringCmpRegex)
}

func regexCmpString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opRegexCmpString)
}

func regexNotString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOp(c, bind, chunk, ref, opRegexCmpString)
}

// regex vs other types
// int ==/!= regex

func intCmpRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opIntCmpRegex)
}

func intNotRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOp(c, bind, chunk, ref, opIntCmpRegex)
}

func regexCmpInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opRegexCmpInt)
}

func regexNotInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOp(c, bind, chunk, ref, opRegexCmpInt)
}

// float ==/!= regex

func floatCmpRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opFloatCmpRegex)
}

func floatNotRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOp(c, bind, chunk, ref, opFloatCmpRegex)
}

func regexCmpFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opRegexCmpFloat)
}

func regexNotFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOp(c, bind, chunk, ref, opRegexCmpFloat)
}

// null vs other types
// bool ==/!= nil

func opBoolCmpNil(left interface{}, right interface{}) bool {
	return left == nil
}

func opNilCmpBool(left interface{}, right interface{}) bool {
	return right == nil
}

func boolCmpNil(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opBoolCmpNil)
}

func boolNotNil(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOp(c, bind, chunk, ref, opBoolCmpNil)
}

func nilCmpBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opNilCmpBool)
}

func nilNotBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOp(c, bind, chunk, ref, opNilCmpBool)
}

// int ==/!= nil

func opIntCmpNil(left interface{}, right interface{}) bool {
	return left == nil
}

func opNilCmpInt(left interface{}, right interface{}) bool {
	return right == nil
}

func intCmpNil(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opIntCmpNil)
}

func intNotNil(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOp(c, bind, chunk, ref, opIntCmpNil)
}

func nilCmpInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opNilCmpInt)
}

func nilNotInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOp(c, bind, chunk, ref, opNilCmpInt)
}

// float ==/!= nil

func opFloatCmpNil(left interface{}, right interface{}) bool {
	return left == nil
}

func opNilCmpFloat(left interface{}, right interface{}) bool {
	return right == nil
}

func floatCmpNil(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opFloatCmpNil)
}

func floatNotNil(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOp(c, bind, chunk, ref, opFloatCmpNil)
}

func nilCmpFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opNilCmpFloat)
}

func nilNotFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOp(c, bind, chunk, ref, opNilCmpFloat)
}

// time ==/!= nil

func opTimeCmpNil(left *RawData, right *RawData) bool {
	return left.Value == nil || left.Value.(*time.Time) == nil
}

func opNilCmpTime(left *RawData, right *RawData) bool {
	return right.Value == nil || right.Value.(*time.Time) == nil
}

func timeCmpNil(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOp(c, bind, chunk, ref, opTimeCmpNil)
}

func timeNotNil(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOp(c, bind, chunk, ref, opTimeCmpNil)
}

func nilCmpTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOp(c, bind, chunk, ref, opNilCmpTime)
}

func nilNotTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOp(c, bind, chunk, ref, opNilCmpTime)
}

// string </>/<=/>= string

func stringLTString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(string) < right.(string))
	})
}

func stringLTEString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(string) <= right.(string))
	})
}

func stringGTString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(string) > right.(string))
	})
}

func stringGTEString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(string) >= right.(string))
	})
}

// int </>/<=/>= int

func intLTInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(int64) < right.(int64))
	})
}

func intLTEInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(int64) <= right.(int64))
	})
}

func intGTInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(int64) > right.(int64))
	})
}

func intGTEInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(int64) >= right.(int64))
	})
}

// float </>/<=/>= float

func floatLTFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(float64) < right.(float64))
	})
}

func floatLTEFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(float64) <= right.(float64))
	})
}

func floatGTFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(float64) > right.(float64))
	})
}

func floatGTEFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(float64) >= right.(float64))
	})
}

// time </>/<=/>= time

func timeLTTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
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

func timeLTETime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
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

func timeGTTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
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

func timeGTETime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
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

func timeMinusTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
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

func timeTimesInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, types.Time, opTimeTimesInt)
}

func intTimesTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		return opTimeTimesInt(right, left)
	})
}

func timeTimesFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, types.Time, opTimeTimesFloat)
}

func floatTimesTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		return opTimeTimesFloat(right, left)
	})
}

// int </>/<=/>= float

func intLTFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(float64(left.(int64)) < right.(float64))
	})
}

func intLTEFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(float64(left.(int64)) <= right.(float64))
	})
}

func intGTFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(float64(left.(int64)) > right.(float64))
	})
}

func intGTEFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(float64(left.(int64)) >= right.(float64))
	})
}

// float </>/<=/>= int

func floatLTInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(float64) < float64(right.(int64)))
	})
}

func floatLTEInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(float64) <= float64(right.(int64)))
	})
}

func floatGTInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(float64) > float64(right.(int64)))
	})
}

func floatGTEInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(float64) >= float64(right.(int64)))
	})
}

// float </>/<=/>= string

func floatLTString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseFloat(right.(string), 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(left.(float64) < f)
	})
}

func floatLTEString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseFloat(right.(string), 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(left.(float64) <= f)
	})
}

func floatGTString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseFloat(right.(string), 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(left.(float64) > f)
	})
}

func floatGTEString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseFloat(right.(string), 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(left.(float64) >= f)
	})
}

// string </>/<=/>= float

func stringLTFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseFloat(left.(string), 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(f < right.(float64))
	})
}

func stringLTEFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseFloat(left.(string), 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(f <= right.(float64))
	})
}

func stringGTFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseFloat(left.(string), 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(f > right.(float64))
	})
}

func stringGTEFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseFloat(left.(string), 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(f >= right.(float64))
	})
}

// int </>/<=/>= string

func intLTString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseInt(right.(string), 10, 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(left.(int64) < f)
	})
}

func intLTEString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseInt(right.(string), 10, 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(left.(int64) <= f)
	})
}

func intGTString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseInt(right.(string), 10, 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(left.(int64) > f)
	})
}

func intGTEString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseInt(right.(string), 10, 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(left.(int64) >= f)
	})
}

// string </>/<=/>= int

func stringLTInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseInt(left.(string), 10, 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(f < right.(int64))
	})
}

func stringLTEInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseInt(left.(string), 10, 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(f <= right.(int64))
	})
}

func stringGTInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseInt(left.(string), 10, 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(f > right.(int64))
	})
}

func stringGTEInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOp(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
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

func boolAndBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOp(c, bind, chunk, ref, truthyBool, truthyBool)
}

func boolOrBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOp(c, bind, chunk, ref, truthyBool, truthyBool)
}

func intAndInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOp(c, bind, chunk, ref, truthyInt, truthyInt)
}

func intOrInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOp(c, bind, chunk, ref, truthyInt, truthyInt)
}

func floatAndFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOp(c, bind, chunk, ref, truthyFloat, truthyFloat)
}

func floatOrFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOp(c, bind, chunk, ref, truthyFloat, truthyFloat)
}

func stringAndString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOp(c, bind, chunk, ref, truthyString, truthyString)
}

func stringOrString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOp(c, bind, chunk, ref, truthyString, truthyString)
}

func regexAndRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOp(c, bind, chunk, ref, truthyString, truthyString)
}

func regexOrRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOp(c, bind, chunk, ref, truthyString, truthyString)
}

func arrayAndArray(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOp(c, bind, chunk, ref, truthyArray, truthyArray)
}

func arrayOrArray(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOp(c, bind, chunk, ref, truthyArray, truthyArray)
}

// bool &&/|| T

func boolAndInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOp(c, bind, chunk, ref, truthyBool, truthyInt)
}

func boolOrInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOp(c, bind, chunk, ref, truthyBool, truthyInt)
}

func intAndBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOp(c, bind, chunk, ref, truthyInt, truthyBool)
}

func intOrBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOp(c, bind, chunk, ref, truthyInt, truthyBool)
}

func boolAndFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOp(c, bind, chunk, ref, truthyBool, truthyFloat)
}

func boolOrFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOp(c, bind, chunk, ref, truthyBool, truthyFloat)
}

func floatAndBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOp(c, bind, chunk, ref, truthyFloat, truthyBool)
}

func floatOrBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOp(c, bind, chunk, ref, truthyFloat, truthyBool)
}

func boolAndString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOp(c, bind, chunk, ref, truthyBool, truthyString)
}

func boolOrString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOp(c, bind, chunk, ref, truthyBool, truthyString)
}

func stringAndBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOp(c, bind, chunk, ref, truthyString, truthyBool)
}

func stringOrBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOp(c, bind, chunk, ref, truthyString, truthyBool)
}

func boolAndRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOp(c, bind, chunk, ref, truthyBool, truthyString)
}

func boolOrRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOp(c, bind, chunk, ref, truthyBool, truthyString)
}

func regexAndBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOp(c, bind, chunk, ref, truthyString, truthyBool)
}

func regexOrBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOp(c, bind, chunk, ref, truthyString, truthyBool)
}

func boolAndArray(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOp(c, bind, chunk, ref, truthyBool, truthyArray)
}

func boolOrArray(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOp(c, bind, chunk, ref, truthyBool, truthyArray)
}

func arrayAndBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOp(c, bind, chunk, ref, truthyArray, truthyBool)
}

func arrayOrBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOp(c, bind, chunk, ref, truthyArray, truthyBool)
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

func boolAndMap(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opBoolAndMap)
}

func boolOrMap(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opBoolOrMap)
}

func mapAndBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opMapAndBool)
}

func mapOrBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opMapOrBool)
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

func intAndFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opIntAndFloat)
}

func intOrFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opIntOrFloat)
}

func floatAndInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opFloatAndInt)
}

func floatOrInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opFloatOrInt)
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

func intAndString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opIntAndString)
}

func intOrString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opIntOrString)
}

func stringAndInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opStringAndInt)
}

func stringOrInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opStringOrInt)
}

func intAndRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opIntAndString)
}

func intOrRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opIntOrString)
}

func regexAndInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opStringAndInt)
}

func regexOrInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opStringOrInt)
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

func intAndArray(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opIntAndArray)
}

func intOrArray(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opIntOrArray)
}

func arrayAndInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opArrayAndInt)
}

func arrayOrInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opArrayOrInt)
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

func intAndMap(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opIntAndMap)
}

func intOrMap(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opIntOrMap)
}

func mapAndInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opMapAndInt)
}

func mapOrInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opMapOrInt)
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

func floatAndString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opFloatAndString)
}

func floatOrString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opFloatOrString)
}

func stringAndFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opStringAndFloat)
}

func stringOrFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opStringOrFloat)
}

func floatAndRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opFloatAndString)
}

func floatOrRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opFloatOrString)
}

func regexAndFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opStringAndFloat)
}

func regexOrFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opStringOrFloat)
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

func floatAndArray(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opFloatAndArray)
}

func floatOrArray(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opFloatOrArray)
}

func arrayAndFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opArrayAndFloat)
}

func arrayOrFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opArrayOrFloat)
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

func floatAndMap(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opFloatAndMap)
}

func floatOrMap(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opFloatOrMap)
}

func mapAndFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opMapAndFloat)
}

func mapOrFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opMapOrFloat)
}

// string &&/|| T

func stringAndRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOp(c, bind, chunk, ref, truthyString, truthyString)
}

func stringOrRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOp(c, bind, chunk, ref, truthyString, truthyString)
}

func regexAndString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOp(c, bind, chunk, ref, truthyString, truthyString)
}

func regexOrString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOp(c, bind, chunk, ref, truthyString, truthyString)
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

func stringAndArray(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opStringAndArray)
}

func stringOrArray(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opStringOrArray)
}

func arrayAndString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opArrayAndString)
}

func arrayOrString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opArrayOrString)
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

func stringAndMap(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opStringAndMap)
}

func stringOrMap(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opStringOrMap)
}

func mapAndString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opMapAndString)
}

func mapOrString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opMapOrString)
}

// string + T

func stringPlusString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		l := left.(string)
		r := right.(string)

		return StringData(l + r)
	})
}

// regex &&/|| array

func regexAndArray(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opStringAndArray)
}

func regexOrArray(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opStringOrArray)
}

func arrayAndRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opArrayAndString)
}

func arrayOrRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opArrayOrString)
}

// regex &&/|| map

func regexAndMap(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opStringAndMap)
}

func regexOrMap(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opStringOrMap)
}

func mapAndRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opMapAndString)
}

func mapOrRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opMapOrString)
}

// time &&/|| T
// note: time is always truthy

func opBoolAndTime(left interface{}, right interface{}) bool {
	return left.(bool)
}

func opTimeAndBool(left interface{}, right interface{}) bool {
	return right.(bool)
}

func boolAndTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opBoolAndTime)
}

func boolOrTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func timeAndBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opTimeAndBool)
}

func timeOrBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func opIntAndTime(left interface{}, right interface{}) bool {
	return left.(int64) != 0
}

func opTimeAndInt(left interface{}, right interface{}) bool {
	return right.(int64) != 0
}

func intAndTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opIntAndTime)
}

func intOrTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func timeAndInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opTimeAndInt)
}

func timeOrInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func opFloatAndTime(left interface{}, right interface{}) bool {
	return left.(float64) != 0
}

func opTimeAndFloat(left interface{}, right interface{}) bool {
	return right.(float64) != 0
}

func floatAndTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opFloatAndTime)
}

func floatOrTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func timeAndFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opTimeAndFloat)
}

func timeOrFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func opStringAndTime(left interface{}, right interface{}) bool {
	return left.(string) != ""
}

func opTimeAndString(left interface{}, right interface{}) bool {
	return right.(string) != ""
}

func stringAndTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opStringAndTime)
}

func stringOrTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func timeAndString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opTimeAndString)
}

func timeOrString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func opRegexAndTime(left interface{}, right interface{}) bool {
	return left.(string) != ""
}

func opTimeAndRegex(left interface{}, right interface{}) bool {
	return right.(string) != ""
}

func regexAndTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opRegexAndTime)
}

func regexOrTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func timeAndRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opTimeAndRegex)
}

func timeOrRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func timeAndTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func timeOrTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func opTimeAndArray(left interface{}, right interface{}) bool {
	return len(right.([]interface{})) != 0
}

func opArrayAndTime(left interface{}, right interface{}) bool {
	return len(left.([]interface{})) != 0
}

func timeAndArray(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opTimeAndArray)
}

func timeOrArray(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func arrayAndTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opArrayAndTime)
}

func arrayOrTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func opTimeAndMap(left interface{}, right interface{}) bool {
	return len(right.(map[string]interface{})) != 0
}

func opMapAndTime(left interface{}, right interface{}) bool {
	return len(left.(map[string]interface{})) != 0
}

func timeAndMap(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opTimeAndMap)
}

func timeOrMap(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func mapAndTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOp(c, bind, chunk, ref, opMapAndTime)
}

func mapOrTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

// string methods

func stringContainsString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return BoolFalse, 0, nil
	}

	argRef := chunk.Function.Args[0]
	arg, rref, err := c.resolveValue(argRef, ref)
	if err != nil || rref > 0 {
		return nil, rref, err
	}

	if arg.Value == nil {
		return BoolFalse, 0, nil
	}

	ok := strings.Contains(bind.Value.(string), arg.Value.(string))
	return BoolData(ok), 0, nil
}

func stringContainsInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return BoolFalse, 0, nil
	}

	argRef := chunk.Function.Args[0]
	arg, rref, err := c.resolveValue(argRef, ref)
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

func stringContainsArrayString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return BoolFalse, 0, nil
	}

	argRef := chunk.Function.Args[0]
	arg, rref, err := c.resolveValue(argRef, ref)
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

func stringContainsArrayInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return BoolFalse, 0, nil
	}

	argRef := chunk.Function.Args[0]
	arg, rref, err := c.resolveValue(argRef, ref)
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

func stringFind(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return ArrayData([]interface{}{}, types.String), 0, nil
	}

	argRef := chunk.Function.Args[0]
	arg, rref, err := c.resolveValue(argRef, ref)
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

func stringDowncase(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: bind.Type}, 0, nil
	}

	res := strings.ToLower(bind.Value.(string))
	return StringData(res), 0, nil
}

var camelCaseRe = regexp.MustCompile(`([[:punct:]]|\s)+[\p{L}]`)

func stringCamelcase(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
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

func stringUpcase(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: bind.Type}, 0, nil
	}

	res := strings.ToUpper(bind.Value.(string))
	return StringData(res), 0, nil
}

func stringLength(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Int}, 0, nil
	}

	l := len(bind.Value.(string))
	return IntData(int64(l)), 0, nil
}

func stringLines(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
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

func stringSplit(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Array(types.String)}, 0, nil
	}

	argRef := chunk.Function.Args[0]
	arg, rref, err := c.resolveValue(argRef, ref)
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

func stringTrim(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: bind.Type}, 0, nil
	}

	cutset := " \t\n\r"

	if len(chunk.Function.Args) != 0 {
		argRef := chunk.Function.Args[0]
		arg, rref, err := c.resolveValue(argRef, ref)
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

func timeSeconds(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
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

func timeMinutes(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
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

func timeHours(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
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

func timeDays(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
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

func timeUnix(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	t := bind.Value.(*time.Time)
	if t == nil {
		return &RawData{Type: types.Array(types.Time)}, 0, nil
	}

	raw := t.Unix()
	return IntData(int64(raw)), 0, nil
}
