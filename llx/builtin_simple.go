package llx

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.mondoo.io/mondoo/types"
)

func rawdataOp(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32, f func(*RawData, *RawData) bool) (*RawData, int32, error) {
	v, dref, err := c.resolveValue(chunk.Function.Args[0], ref)
	if err != nil {
		return nil, 0, err
	}
	if dref != 0 {
		return nil, dref, nil
	}
	return BoolData(f(bind, v)), 0, nil
}

func dataOp(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32, f func(interface{}, interface{}) bool) (*RawData, int32, error) {
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

func rawdataNotOp(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32, f func(*RawData, *RawData) bool) (*RawData, int32, error) {
	v, dref, err := c.resolveValue(chunk.Function.Args[0], ref)
	if err != nil {
		return nil, 0, err
	}
	if dref != 0 {
		return nil, dref, nil
	}
	return BoolData(!f(bind, v)), 0, nil
}

func dataNotOp(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32, f func(interface{}, interface{}) bool) (*RawData, int32, error) {
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
	if v.Value == nil {
		return BoolData(true), 0, nil
	}

	return BoolData(!f(bind.Value, v.Value)), 0, nil
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
	l := left.(time.Time)
	r := right.(time.Time)
	return l.Equal(r)
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

func opRegexCmpBool(left interface{}, right interface{}) bool {
	if right.(bool) == true {
		return opStringCmpRegex("true", left.(string))
	}
	return opStringCmpRegex("false", left.(string))
}

func opBoolCmpRegex(left interface{}, right interface{}) bool {
	if left.(bool) == true {
		return opStringCmpRegex("true", right.(string))
	}
	return opStringCmpRegex("false", right.(string))
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
	return dataOp(c, bind, chunk, ref, opBoolCmpBool)
}

func boolNotBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opBoolCmpBool)
}

func intCmpInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opIntCmpInt)
}

func intNotInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opIntCmpInt)
}

func floatCmpFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opFloatCmpFloat)
}

func floatNotFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opFloatCmpFloat)
}

func stringCmpString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opStringCmpString)
}

func stringNotString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opStringCmpString)
}

func timeCmpTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opTimeCmpTime)
}

func timeNotTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opTimeCmpTime)
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
	return dataOp(c, bind, chunk, ref, opStringCmpNil)
}

func stringNotNil(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opStringCmpNil)
}

func nilCmpString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opNilCmpString)
}

func nilNotString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opNilCmpString)
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
	return dataOp(c, bind, chunk, ref, opStringCmpBool)
}

func stringNotBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opStringCmpBool)
}

func boolCmpString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opBoolCmpString)
}

func boolNotString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opBoolCmpString)
}

// string ==/!= int

func stringCmpInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opStringCmpInt)
}

func stringNotInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opStringCmpInt)
}

func intCmpString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opIntCmpString)
}

func intNotString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opIntCmpString)
}

// string ==/!= float

func stringCmpFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opStringCmpFloat)
}

func stringNotFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opStringCmpFloat)
}

func floatCmpString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opFloatCmpString)
}

func floatNotString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opFloatCmpString)
}

// string ==/!= regex

func stringCmpRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opStringCmpRegex)
}

func stringNotRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opStringCmpRegex)
}

func regexCmpString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opRegexCmpString)
}

func regexNotString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opRegexCmpString)
}

// regex vs other types
// bool ==/!= regex

func boolCmpRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opBoolCmpRegex)
}

func boolNotRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opBoolCmpRegex)
}

func regexCmpBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opRegexCmpBool)
}

func regexNotBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opRegexCmpBool)
}

// int ==/!= regex

func intCmpRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opIntCmpRegex)
}

func intNotRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opIntCmpRegex)
}

func regexCmpInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opRegexCmpInt)
}

func regexNotInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opRegexCmpInt)
}

// float ==/!= regex

func floatCmpRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opFloatCmpRegex)
}

func floatNotRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opFloatCmpRegex)
}

func regexCmpFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opRegexCmpFloat)
}

func regexNotFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opRegexCmpFloat)
}

// bool ==/!= nil

func opBoolCmpNil(left interface{}, right interface{}) bool {
	return left == nil
}

func opNilCmpBool(left interface{}, right interface{}) bool {
	return right == nil
}

func boolCmpNil(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opBoolCmpNil)
}

func boolNotNil(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opBoolCmpNil)
}

func nilCmpBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opNilCmpBool)
}

func nilNotBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opNilCmpBool)
}

// int ==/!= nil

func opIntCmpNil(left interface{}, right interface{}) bool {
	return left == nil
}

func opNilCmpInt(left interface{}, right interface{}) bool {
	return right == nil
}

func intCmpNil(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opIntCmpNil)
}

func intNotNil(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opIntCmpNil)
}

func nilCmpInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opNilCmpInt)
}

func nilNotInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opNilCmpInt)
}

// float ==/!= nil

func opFloatCmpNil(left interface{}, right interface{}) bool {
	return left == nil
}

func opNilCmpFloat(left interface{}, right interface{}) bool {
	return right == nil
}

func floatCmpNil(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opFloatCmpNil)
}

func floatNotNil(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opFloatCmpNil)
}

func nilCmpFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opNilCmpFloat)
}

func nilNotFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opNilCmpFloat)
}

// time ==/!= nil

func opTimeCmpNil(left interface{}, right interface{}) bool {
	return left == nil
}

func opNilCmpTime(left interface{}, right interface{}) bool {
	return right == nil
}

func timeCmpNil(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opTimeCmpNil)
}

func timeNotNil(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opTimeCmpNil)
}

func nilCmpTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opNilCmpTime)
}

func nilNotTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataNotOp(c, bind, chunk, ref, opNilCmpTime)
}

// string </>/<=/>= string

func stringLTString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		return left.(string) < right.(string)
	})
}

func stringLTEString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		return left.(string) <= right.(string)
	})
}

func stringGTString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		return left.(string) > right.(string)
	})
}

func stringGTEString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		return left.(string) >= right.(string)
	})
}

// int </>/<=/>= int

func intLTInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		return left.(int64) < right.(int64)
	})
}

func intLTEInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		return left.(int64) <= right.(int64)
	})
}

func intGTInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		return left.(int64) > right.(int64)
	})
}

func intGTEInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		return left.(int64) >= right.(int64)
	})
}

// float </>/<=/>= float

func floatLTFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		return left.(float64) < right.(float64)
	})
}

func floatLTEFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		return left.(float64) <= right.(float64)
	})
}

func floatGTFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		return left.(float64) > right.(float64)
	})
}

func floatGTEFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		return left.(float64) >= right.(float64)
	})
}

// time </>/<=/>= time

func timeLTTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		l := left.(time.Time)
		r := right.(time.Time)
		return l.Before(r)
	})
}

func timeLTETime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		l := left.(time.Time)
		r := right.(time.Time)
		return l.Before(r) || l.Equal(r)
	})
}

func timeGTTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		l := left.(time.Time)
		r := right.(time.Time)
		return l.After(r)
	})
}

func timeGTETime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		l := left.(time.Time)
		r := right.(time.Time)
		return l.After(r) || l.Equal(r)
	})
}

// int </>/<=/>= float

func intLTFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		return float64(left.(int64)) < right.(float64)
	})
}

func intLTEFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		return float64(left.(int64)) <= right.(float64)
	})
}

func intGTFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		return float64(left.(int64)) > right.(float64)
	})
}

func intGTEFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		return float64(left.(int64)) >= right.(float64)
	})
}

// float </>/<=/>= int

func floatLTInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		return left.(float64) < float64(right.(int64))
	})
}

func floatLTEInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		return left.(float64) <= float64(right.(int64))
	})
}

func floatGTInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		return left.(float64) > float64(right.(int64))
	})
}

func floatGTEInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		return left.(float64) >= float64(right.(int64))
	})
}

// float </>/<=/>= string

func floatLTString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		f, err := strconv.ParseFloat(right.(string), 64)
		return err == nil && left.(float64) < f
	})
}

func floatLTEString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		f, err := strconv.ParseFloat(right.(string), 64)
		return err == nil && left.(float64) <= f
	})
}

func floatGTString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		f, err := strconv.ParseFloat(right.(string), 64)
		return err == nil && left.(float64) > f
	})
}

func floatGTEString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		f, err := strconv.ParseFloat(right.(string), 64)
		return err == nil && left.(float64) >= f
	})
}

// string </>/<=/>= float

func stringLTFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		f, err := strconv.ParseFloat(left.(string), 64)
		return err == nil && f < right.(float64)
	})
}

func stringLTEFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		f, err := strconv.ParseFloat(left.(string), 64)
		return err == nil && f <= right.(float64)
	})
}

func stringGTFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		f, err := strconv.ParseFloat(left.(string), 64)
		return err == nil && f > right.(float64)
	})
}

func stringGTEFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		f, err := strconv.ParseFloat(left.(string), 64)
		return err == nil && f >= right.(float64)
	})
}

// int </>/<=/>= string

func intLTString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		f, err := strconv.ParseInt(right.(string), 10, 64)
		return err == nil && left.(int64) < f
	})
}

func intLTEString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		f, err := strconv.ParseInt(right.(string), 10, 64)
		return err == nil && left.(int64) <= f
	})
}

func intGTString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		f, err := strconv.ParseInt(right.(string), 10, 64)
		return err == nil && left.(int64) > f
	})
}

func intGTEString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		f, err := strconv.ParseInt(right.(string), 10, 64)
		return err == nil && left.(int64) >= f
	})
}

// string </>/<=/>= int

func stringLTInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		f, err := strconv.ParseInt(left.(string), 10, 64)
		return err == nil && f < right.(int64)
	})
}

func stringLTEInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		f, err := strconv.ParseInt(left.(string), 10, 64)
		return err == nil && f <= right.(int64)
	})
}

func stringGTInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		f, err := strconv.ParseInt(left.(string), 10, 64)
		return err == nil && f > right.(int64)
	})
}

func stringGTEInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, func(left interface{}, right interface{}) bool {
		f, err := strconv.ParseInt(left.(string), 10, 64)
		return err == nil && f >= right.(int64)
	})
}

// ---------------------------------
//       &&  AND        ||  OR
// ---------------------------------

// T &&/|| T

func opBoolAndBool(left interface{}, right interface{}) bool {
	return left.(bool) && right.(bool)
}

func boolAndBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opBoolAndBool)
}

func opBoolOrBool(left interface{}, right interface{}) bool {
	return left.(bool) || right.(bool)
}

func boolOrBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opBoolOrBool)
}

func opIntAndInt(left interface{}, right interface{}) bool {
	return (left.(int64) != 0) && (right.(int64) != 0)
}

func intAndInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opIntAndInt)
}

func opIntOrInt(left interface{}, right interface{}) bool {
	return (left.(int64) != 0) || (right.(int64) != 0)
}

func intOrInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opIntOrInt)
}

func opFloatAndFloat(left interface{}, right interface{}) bool {
	return (left.(float64) != 0) && (right.(float64) != 0)
}

func floatAndFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opFloatAndFloat)
}

func opFloatOrFloat(left interface{}, right interface{}) bool {
	return (left.(float64) != 0) || (right.(float64) != 0)
}

func floatOrFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opFloatOrFloat)
}

func opStringAndString(left interface{}, right interface{}) bool {
	return (left.(string) != "") && (right.(string) != "")
}

func stringAndString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opStringAndString)
}

func opStringOrString(left interface{}, right interface{}) bool {
	return (left.(string) != "") || (right.(string) != "")
}

func stringOrString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opStringOrString)
}

func regexAndRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opStringAndString)
}

func regexOrRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opStringOrString)
}

func opArrayAndArray(left interface{}, right interface{}) bool {
	return (len(left.([]interface{})) != 0) && (len(right.([]interface{})) != 0)
}

func arrayAndArray(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opArrayAndArray)
}

func opArrayOrArray(left interface{}, right interface{}) bool {
	return (len(left.([]interface{})) != 0) || (len(right.([]interface{})) != 0)
}

func arrayOrArray(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opArrayOrArray)
}

// bool &&/|| T

func opBoolAndInt(left interface{}, right interface{}) bool {
	return left.(bool) && (right.(int64) != 0)
}

func opIntAndBool(left interface{}, right interface{}) bool {
	return right.(bool) && (left.(int64) != 0)
}

func opBoolOrInt(left interface{}, right interface{}) bool {
	return left.(bool) || (right.(int64) != 0)
}

func opIntOrBool(left interface{}, right interface{}) bool {
	return right.(bool) || (left.(int64) != 0)
}

func boolAndInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opBoolAndInt)
}

func boolOrInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opBoolOrInt)
}

func intAndBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opIntAndBool)
}

func intOrBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opIntOrBool)
}

func opBoolAndFloat(left interface{}, right interface{}) bool {
	return left.(bool) && (right.(float64) != 0)
}

func opFloatAndBool(left interface{}, right interface{}) bool {
	return right.(bool) && (left.(float64) != 0)
}

func opBoolOrFloat(left interface{}, right interface{}) bool {
	return left.(bool) || (right.(float64) != 0)
}

func opFloatOrBool(left interface{}, right interface{}) bool {
	return right.(bool) || (left.(float64) != 0)
}

func boolAndFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opBoolAndFloat)
}

func boolOrFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opBoolOrFloat)
}

func floatAndBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opFloatAndBool)
}

func floatOrBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opFloatOrBool)
}

func opBoolAndString(left interface{}, right interface{}) bool {
	return left.(bool) && (right.(string) != "")
}

func opStringAndBool(left interface{}, right interface{}) bool {
	return right.(bool) && (left.(string) != "")
}

func opBoolOrString(left interface{}, right interface{}) bool {
	return left.(bool) || (right.(string) != "")
}

func opStringOrBool(left interface{}, right interface{}) bool {
	return right.(bool) || (left.(string) != "")
}

func boolAndString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opBoolAndString)
}

func boolOrString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opBoolOrString)
}

func stringAndBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opStringAndBool)
}

func stringOrBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opStringOrBool)
}

func boolAndRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opBoolAndString)
}

func boolOrRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opBoolOrString)
}

func regexAndBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opStringAndBool)
}

func regexOrBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opStringOrBool)
}

func opBoolAndTime(left interface{}, right interface{}) bool {
	return left.(bool) && (!right.(time.Time).IsZero())
}

func opTimeAndBool(left interface{}, right interface{}) bool {
	return right.(bool) && (!left.(time.Time).IsZero())
}

func opBoolOrTime(left interface{}, right interface{}) bool {
	return left.(bool) || (!right.(time.Time).IsZero())
}

func opTimeOrBool(left interface{}, right interface{}) bool {
	return right.(bool) || (!left.(time.Time).IsZero())
}

func boolAndTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opBoolAndTime)
}

func boolOrTime(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opBoolOrTime)
}

func timeAndBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opTimeAndBool)
}

func timeOrBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opTimeOrBool)
}

func opBoolAndArray(left interface{}, right interface{}) bool {
	return left.(bool) && (len(right.([]interface{})) != 0)
}

func opArrayAndBool(left interface{}, right interface{}) bool {
	return right.(bool) && (len(left.([]interface{})) != 0)
}

func opBoolOrArray(left interface{}, right interface{}) bool {
	return left.(bool) || (len(right.([]interface{})) != 0)
}

func opArrayOrBool(left interface{}, right interface{}) bool {
	return right.(bool) || (len(left.([]interface{})) != 0)
}

func boolAndArray(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opBoolAndArray)
}

func boolOrArray(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opBoolOrArray)
}

func arrayAndBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opArrayAndBool)
}

func arrayOrBool(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opArrayOrBool)
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
	return dataOp(c, bind, chunk, ref, opIntAndFloat)
}

func intOrFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opIntOrFloat)
}

func floatAndInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opFloatAndInt)
}

func floatOrInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opFloatOrInt)
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
	return dataOp(c, bind, chunk, ref, opIntAndString)
}

func intOrString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opIntOrString)
}

func stringAndInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opStringAndInt)
}

func stringOrInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opStringOrInt)
}

func intAndRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opIntAndString)
}

func intOrRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opIntOrString)
}

func regexAndInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opStringAndInt)
}

func regexOrInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opStringOrInt)
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
	return dataOp(c, bind, chunk, ref, opIntAndArray)
}

func intOrArray(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opIntOrArray)
}

func arrayAndInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opArrayAndInt)
}

func arrayOrInt(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opArrayOrInt)
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
	return dataOp(c, bind, chunk, ref, opFloatAndString)
}

func floatOrString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opFloatOrString)
}

func stringAndFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opStringAndFloat)
}

func stringOrFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opStringOrFloat)
}

func floatAndRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opFloatAndString)
}

func floatOrRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opFloatOrString)
}

func regexAndFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opStringAndFloat)
}

func regexOrFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opStringOrFloat)
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
	return dataOp(c, bind, chunk, ref, opFloatAndArray)
}

func floatOrArray(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opFloatOrArray)
}

func arrayAndFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opArrayAndFloat)
}

func arrayOrFloat(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opArrayOrFloat)
}

// string &&/|| T

func stringAndRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opStringAndString)
}

func stringOrRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opStringOrString)
}

func regexAndString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opStringAndString)
}

func regexOrString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opStringOrString)
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
	return dataOp(c, bind, chunk, ref, opStringAndArray)
}

func stringOrArray(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opStringOrArray)
}

func arrayAndString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opArrayAndString)
}

func arrayOrString(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opArrayOrString)
}

// regex &&/|| T

func regexAndArray(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opStringAndArray)
}

func regexOrArray(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opStringOrArray)
}

func arrayAndRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opArrayAndString)
}

func arrayOrRegex(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	return dataOp(c, bind, chunk, ref, opArrayOrString)
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

func stringDowncase(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	if bind.Value == nil {
		return &RawData{Type: bind.Type}, 0, nil
	}

	res := strings.ToLower(bind.Value.(string))
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
