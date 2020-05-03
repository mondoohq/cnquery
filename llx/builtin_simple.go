package llx

import (
	"regexp"
	"strconv"
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

// string vs other types
// string ==/!= bool

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
