package llx

import (
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"go.mondoo.com/cnquery/types"
)

// run an operation that returns true/false on a bind data vs a chunk call.
// Unlike boolOp we don't check if either side is nil
func rawboolOpV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32, f func(*RawData, *RawData) bool) (*RawData, int32, error) {
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
func rawboolNotOpV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32, f func(*RawData, *RawData) bool) (*RawData, int32, error) {
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
func boolOpV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32, f func(interface{}, interface{}) bool) (*RawData, int32, error) {
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
func boolOrOpV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32, fLeft func(interface{}) bool, fRight func(interface{}) bool) (*RawData, int32, error) {
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
func boolAndOpV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32, fLeft func(interface{}) bool, fRight func(interface{}) bool) (*RawData, int32, error) {
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

func boolNotOpV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32, f func(interface{}, interface{}) bool) (*RawData, int32, error) {
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

func dataOpV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32, typ types.Type, f func(interface{}, interface{}) *RawData) (*RawData, int32, error) {
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

func nonNilDataOpV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32, typ types.Type, f func(interface{}, interface{}) *RawData) (*RawData, int32, error) {
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

func chunkEqTrueV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, func(a interface{}, b interface{}) bool {
		return true
	})
}

func chunkEqFalseV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, func(a interface{}, b interface{}) bool {
		return false
	})
}

func chunkNeqFalseV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, func(a interface{}, b interface{}) bool {
		return true
	})
}

func chunkNeqTrueV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, func(a interface{}, b interface{}) bool {
		return false
	})
}

// same operator types
// ==   !=

func boolCmpBoolV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opBoolCmpBool)
}

func boolNotBoolV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opBoolCmpBool)
}

func intCmpIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opIntCmpInt)
}

func intNotIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opIntCmpInt)
}

func floatCmpFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opFloatCmpFloat)
}

func floatNotFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opFloatCmpFloat)
}

func stringCmpStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opStringCmpString)
}

func stringNotStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opStringCmpString)
}

func timeCmpTimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opTimeCmpTime)
}

func timeNotTimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opTimeCmpTime)
}

// int arithmetic

func intPlusIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Int, func(left interface{}, right interface{}) *RawData {
		res := left.(int64) + right.(int64)
		return IntData(res)
	})
}

func intMinusIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Int, func(left interface{}, right interface{}) *RawData {
		res := left.(int64) - right.(int64)
		return IntData(res)
	})
}

func intTimesIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Int, func(left interface{}, right interface{}) *RawData {
		res := left.(int64) * right.(int64)
		return IntData(res)
	})
}

func intDividedIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Int, func(left interface{}, right interface{}) *RawData {
		res := left.(int64) / right.(int64)
		return IntData(res)
	})
}

func intPlusFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := float64(left.(int64)) + right.(float64)
		return FloatData(res)
	})
}

func intMinusFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := float64(left.(int64)) - right.(float64)
		return FloatData(res)
	})
}

func intTimesFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := float64(left.(int64)) * right.(float64)
		return FloatData(res)
	})
}

func intDividedFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := float64(left.(int64)) / right.(float64)
		return FloatData(res)
	})
}

// float arithmetic

func floatPlusIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := left.(float64) + float64(right.(int64))
		return FloatData(res)
	})
}

func floatMinusIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := left.(float64) - float64(right.(int64))
		return FloatData(res)
	})
}

func floatTimesIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := left.(float64) * float64(right.(int64))
		return FloatData(res)
	})
}

func floatDividedIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := left.(float64) / float64(right.(int64))
		return FloatData(res)
	})
}

func floatPlusFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := left.(float64) + right.(float64)
		return FloatData(res)
	})
}

func floatMinusFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := left.(float64) - right.(float64)
		return FloatData(res)
	})
}

func floatTimesFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := left.(float64) * right.(float64)
		return FloatData(res)
	})
}

func floatDividedFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Float, func(left interface{}, right interface{}) *RawData {
		res := left.(float64) / right.(float64)
		return FloatData(res)
	})
}

// int vs float
// int ==/!= float

func intCmpFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opIntCmpFloat)
}

func intNotFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opIntCmpFloat)
}

func floatCmpIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opFloatCmpInt)
}

func floatNotIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opFloatCmpInt)
}

// string vs other types
// string ==/!= nil

func stringCmpNilV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opStringCmpNil)
}

func stringNotNilV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opStringCmpNil)
}

func nilCmpStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opNilCmpString)
}

func nilNotStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opNilCmpString)
}

// string ==/!= bool

func stringCmpBoolV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opStringCmpBool)
}

func stringNotBoolV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opStringCmpBool)
}

func boolCmpStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opBoolCmpString)
}

func boolNotStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opBoolCmpString)
}

// string ==/!= int

func stringCmpIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opStringCmpInt)
}

func stringNotIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opStringCmpInt)
}

func intCmpStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opIntCmpString)
}

func intNotStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opIntCmpString)
}

// string ==/!= float

func stringCmpFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opStringCmpFloat)
}

func stringNotFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opStringCmpFloat)
}

func floatCmpStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opFloatCmpString)
}

func floatNotStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opFloatCmpString)
}

// string ==/!= regex

func stringCmpRegexV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opStringCmpRegex)
}

func stringNotRegexV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opStringCmpRegex)
}

func regexCmpStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opRegexCmpString)
}

func regexNotStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opRegexCmpString)
}

// regex vs other types
// int ==/!= regex

func intCmpRegexV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opIntCmpRegex)
}

func intNotRegexV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opIntCmpRegex)
}

func regexCmpIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opRegexCmpInt)
}

func regexNotIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opRegexCmpInt)
}

// float ==/!= regex

func floatCmpRegexV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opFloatCmpRegex)
}

func floatNotRegexV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opFloatCmpRegex)
}

func regexCmpFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opRegexCmpFloat)
}

func regexNotFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opRegexCmpFloat)
}

// null vs other types
// bool ==/!= nil

func boolCmpNilV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opBoolCmpNil)
}

func boolNotNilV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opBoolCmpNil)
}

func nilCmpBoolV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opNilCmpBool)
}

func nilNotBoolV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opNilCmpBool)
}

// int ==/!= nil

func intCmpNilV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opIntCmpNil)
}

func intNotNilV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opIntCmpNil)
}

func nilCmpIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opNilCmpInt)
}

func nilNotIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opNilCmpInt)
}

// float ==/!= nil

func floatCmpNilV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opFloatCmpNil)
}

func floatNotNilV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opFloatCmpNil)
}

func nilCmpFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opNilCmpFloat)
}

func nilNotFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolNotOpV1(c, bind, chunk, ref, opNilCmpFloat)
}

// time ==/!= nil

func timeCmpNilV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, opTimeCmpNil)
}

func timeNotNilV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, opTimeCmpNil)
}

func nilCmpTimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolOpV1(c, bind, chunk, ref, opNilCmpTime)
}

func nilNotTimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return rawboolNotOpV1(c, bind, chunk, ref, opNilCmpTime)
}

// string </>/<=/>= string

func stringLTStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(string) < right.(string))
	})
}

func stringLTEStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(string) <= right.(string))
	})
}

func stringGTStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(string) > right.(string))
	})
}

func stringGTEStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(string) >= right.(string))
	})
}

// int </>/<=/>= int

func intLTIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(int64) < right.(int64))
	})
}

func intLTEIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(int64) <= right.(int64))
	})
}

func intGTIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(int64) > right.(int64))
	})
}

func intGTEIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(int64) >= right.(int64))
	})
}

// float </>/<=/>= float

func floatLTFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(float64) < right.(float64))
	})
}

func floatLTEFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(float64) <= right.(float64))
	})
}

func floatGTFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(float64) > right.(float64))
	})
}

func floatGTEFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(float64) >= right.(float64))
	})
}

// time </>/<=/>= time

func timeLTTimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
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

func timeLTETimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
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

func timeGTTimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
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

func timeGTETimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
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

func timeMinusTimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
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

func timeTimesIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Time, opTimeTimesInt)
}

func intTimesTimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		return opTimeTimesInt(right, left)
	})
}

func timeTimesFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Time, opTimeTimesFloat)
}

func floatTimesTimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		return opTimeTimesFloat(right, left)
	})
}

// int </>/<=/>= float

func intLTFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(float64(left.(int64)) < right.(float64))
	})
}

func intLTEFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(float64(left.(int64)) <= right.(float64))
	})
}

func intGTFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(float64(left.(int64)) > right.(float64))
	})
}

func intGTEFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(float64(left.(int64)) >= right.(float64))
	})
}

// float </>/<=/>= int

func floatLTIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(float64) < float64(right.(int64)))
	})
}

func floatLTEIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(float64) <= float64(right.(int64)))
	})
}

func floatGTIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(float64) > float64(right.(int64)))
	})
}

func floatGTEIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		return BoolData(left.(float64) >= float64(right.(int64)))
	})
}

// float </>/<=/>= string

func floatLTStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseFloat(right.(string), 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(left.(float64) < f)
	})
}

func floatLTEStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseFloat(right.(string), 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(left.(float64) <= f)
	})
}

func floatGTStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseFloat(right.(string), 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(left.(float64) > f)
	})
}

func floatGTEStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseFloat(right.(string), 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(left.(float64) >= f)
	})
}

// string </>/<=/>= float

func stringLTFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseFloat(left.(string), 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(f < right.(float64))
	})
}

func stringLTEFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseFloat(left.(string), 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(f <= right.(float64))
	})
}

func stringGTFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseFloat(left.(string), 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(f > right.(float64))
	})
}

func stringGTEFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseFloat(left.(string), 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(f >= right.(float64))
	})
}

// int </>/<=/>= string

func intLTStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseInt(right.(string), 10, 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(left.(int64) < f)
	})
}

func intLTEStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseInt(right.(string), 10, 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(left.(int64) <= f)
	})
}

func intGTStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseInt(right.(string), 10, 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(left.(int64) > f)
	})
}

func intGTEStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseInt(right.(string), 10, 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(left.(int64) >= f)
	})
}

// string </>/<=/>= int

func stringLTIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseInt(left.(string), 10, 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(f < right.(int64))
	})
}

func stringLTEIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseInt(left.(string), 10, 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(f <= right.(int64))
	})
}

func stringGTIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
		f, err := strconv.ParseInt(left.(string), 10, 64)
		if err != nil {
			return &RawData{Type: types.Bool, Error: errors.New("failed to convert string to float")}
		}
		return BoolData(f > right.(int64))
	})
}

func stringGTEIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return nonNilDataOpV1(c, bind, chunk, ref, types.Bool, func(left interface{}, right interface{}) *RawData {
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

func boolAndBoolV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOpV1(c, bind, chunk, ref, truthyBool, truthyBool)
}

func boolOrBoolV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOpV1(c, bind, chunk, ref, truthyBool, truthyBool)
}

func intAndIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOpV1(c, bind, chunk, ref, truthyInt, truthyInt)
}

func intOrIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOpV1(c, bind, chunk, ref, truthyInt, truthyInt)
}

func floatAndFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOpV1(c, bind, chunk, ref, truthyFloat, truthyFloat)
}

func floatOrFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOpV1(c, bind, chunk, ref, truthyFloat, truthyFloat)
}

func stringAndStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOpV1(c, bind, chunk, ref, truthyString, truthyString)
}

func stringOrStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOpV1(c, bind, chunk, ref, truthyString, truthyString)
}

func regexAndRegexV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOpV1(c, bind, chunk, ref, truthyString, truthyString)
}

func regexOrRegexV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOpV1(c, bind, chunk, ref, truthyString, truthyString)
}

func arrayAndArrayV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOpV1(c, bind, chunk, ref, truthyArray, truthyArray)
}

func arrayOrArrayV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOpV1(c, bind, chunk, ref, truthyArray, truthyArray)
}

// bool &&/|| T

func boolAndIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOpV1(c, bind, chunk, ref, truthyBool, truthyInt)
}

func boolOrIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOpV1(c, bind, chunk, ref, truthyBool, truthyInt)
}

func intAndBoolV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOpV1(c, bind, chunk, ref, truthyInt, truthyBool)
}

func intOrBoolV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOpV1(c, bind, chunk, ref, truthyInt, truthyBool)
}

func boolAndFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOpV1(c, bind, chunk, ref, truthyBool, truthyFloat)
}

func boolOrFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOpV1(c, bind, chunk, ref, truthyBool, truthyFloat)
}

func floatAndBoolV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOpV1(c, bind, chunk, ref, truthyFloat, truthyBool)
}

func floatOrBoolV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOpV1(c, bind, chunk, ref, truthyFloat, truthyBool)
}

func boolAndStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOpV1(c, bind, chunk, ref, truthyBool, truthyString)
}

func boolOrStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOpV1(c, bind, chunk, ref, truthyBool, truthyString)
}

func stringAndBoolV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOpV1(c, bind, chunk, ref, truthyString, truthyBool)
}

func stringOrBoolV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOpV1(c, bind, chunk, ref, truthyString, truthyBool)
}

func boolAndRegexV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOpV1(c, bind, chunk, ref, truthyBool, truthyString)
}

func boolOrRegexV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOpV1(c, bind, chunk, ref, truthyBool, truthyString)
}

func regexAndBoolV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOpV1(c, bind, chunk, ref, truthyString, truthyBool)
}

func regexOrBoolV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOpV1(c, bind, chunk, ref, truthyString, truthyBool)
}

func boolAndArrayV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOpV1(c, bind, chunk, ref, truthyBool, truthyArray)
}

func boolOrArrayV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOpV1(c, bind, chunk, ref, truthyBool, truthyArray)
}

func arrayAndBoolV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOpV1(c, bind, chunk, ref, truthyArray, truthyBool)
}

func arrayOrBoolV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOpV1(c, bind, chunk, ref, truthyArray, truthyBool)
}

func boolAndMapV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opBoolAndMap)
}

func boolOrMapV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opBoolOrMap)
}

func mapAndBoolV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opMapAndBool)
}

func mapOrBoolV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opMapOrBool)
}

// int &&/|| T

func intAndFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opIntAndFloat)
}

func intOrFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opIntOrFloat)
}

func floatAndIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opFloatAndInt)
}

func floatOrIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opFloatOrInt)
}

func intAndStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opIntAndString)
}

func intOrStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opIntOrString)
}

func stringAndIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opStringAndInt)
}

func stringOrIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opStringOrInt)
}

func intAndRegexV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opIntAndString)
}

func intOrRegexV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opIntOrString)
}

func regexAndIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opStringAndInt)
}

func regexOrIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opStringOrInt)
}

func intAndArrayV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opIntAndArray)
}

func intOrArrayV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opIntOrArray)
}

func arrayAndIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opArrayAndInt)
}

func arrayOrIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opArrayOrInt)
}

func intAndMapV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opIntAndMap)
}

func intOrMapV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opIntOrMap)
}

func mapAndIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opMapAndInt)
}

func mapOrIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opMapOrInt)
}

// float &&/|| T

func floatAndStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opFloatAndString)
}

func floatOrStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opFloatOrString)
}

func stringAndFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opStringAndFloat)
}

func stringOrFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opStringOrFloat)
}

func floatAndRegexV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opFloatAndString)
}

func floatOrRegexV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opFloatOrString)
}

func regexAndFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opStringAndFloat)
}

func regexOrFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opStringOrFloat)
}

func floatAndArrayV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opFloatAndArray)
}

func floatOrArrayV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opFloatOrArray)
}

func arrayAndFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opArrayAndFloat)
}

func arrayOrFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opArrayOrFloat)
}

func floatAndMapV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opFloatAndMap)
}

func floatOrMapV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opFloatOrMap)
}

func mapAndFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opMapAndFloat)
}

func mapOrFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opMapOrFloat)
}

// string &&/|| T

func stringAndRegexV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOpV1(c, bind, chunk, ref, truthyString, truthyString)
}

func stringOrRegexV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOpV1(c, bind, chunk, ref, truthyString, truthyString)
}

func regexAndStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolAndOpV1(c, bind, chunk, ref, truthyString, truthyString)
}

func regexOrStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOrOpV1(c, bind, chunk, ref, truthyString, truthyString)
}

func stringAndArrayV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opStringAndArray)
}

func stringOrArrayV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opStringOrArray)
}

func arrayAndStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opArrayAndString)
}

func arrayOrStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opArrayOrString)
}

func stringAndMapV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opStringAndMap)
}

func stringOrMapV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opStringOrMap)
}

func mapAndStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opMapAndString)
}

func mapOrStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opMapOrString)
}

// string + T

func stringPlusStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOpV1(c, bind, chunk, ref, types.Time, func(left interface{}, right interface{}) *RawData {
		l := left.(string)
		r := right.(string)

		return StringData(l + r)
	})
}

// regex &&/|| array

func regexAndArrayV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opStringAndArray)
}

func regexOrArrayV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opStringOrArray)
}

func arrayAndRegexV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opArrayAndString)
}

func arrayOrRegexV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opArrayOrString)
}

// regex &&/|| map

func regexAndMapV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opStringAndMap)
}

func regexOrMapV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opStringOrMap)
}

func mapAndRegexV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opMapAndString)
}

func mapOrRegexV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opMapOrString)
}

// time &&/|| T
// note: time is always truthy

func boolAndTimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opBoolAndTime)
}

func boolOrTimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func timeAndBoolV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opTimeAndBool)
}

func timeOrBoolV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func intAndTimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opIntAndTime)
}

func intOrTimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func timeAndIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opTimeAndInt)
}

func timeOrIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func floatAndTimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opFloatAndTime)
}

func floatOrTimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func timeAndFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opTimeAndFloat)
}

func timeOrFloatV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func stringAndTimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opStringAndTime)
}

func stringOrTimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func timeAndStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opTimeAndString)
}

func timeOrStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func regexAndTimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opRegexAndTime)
}

func regexOrTimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func timeAndRegexV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opTimeAndRegex)
}

func timeOrRegexV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func timeAndTimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func timeOrTimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func timeAndArrayV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opTimeAndArray)
}

func timeOrArrayV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func arrayAndTimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opArrayAndTime)
}

func arrayOrTimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func timeAndMapV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opTimeAndMap)
}

func timeOrMapV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

func mapAndTimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return boolOpV1(c, bind, chunk, ref, opMapAndTime)
}

func mapOrTimeV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return BoolTrue, 0, nil
}

// string methods

func stringContainsStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
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

func stringContainsIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
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

func stringContainsArrayStringV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
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

func stringContainsArrayIntV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
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

func stringFindV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
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

func stringDowncaseV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: bind.Type}, 0, nil
	}

	res := strings.ToLower(bind.Value.(string))
	return StringData(res), 0, nil
}

func stringCamelcaseV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
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

func stringUpcaseV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: bind.Type}, 0, nil
	}

	res := strings.ToUpper(bind.Value.(string))
	return StringData(res), 0, nil
}

func stringLengthV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: types.Int}, 0, nil
	}

	l := len(bind.Value.(string))
	return IntData(int64(l)), 0, nil
}

func stringLinesV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
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

func stringSplitV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
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

func stringTrimV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
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

func timeSecondsV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
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

func timeMinutesV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
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

func timeHoursV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
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

func timeDaysV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
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

func timeUnixV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	t := bind.Value.(*time.Time)
	if t == nil {
		return &RawData{Type: types.Array(types.Time)}, 0, nil
	}

	raw := t.Unix()
	return IntData(int64(raw)), 0, nil
}
