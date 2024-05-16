// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package llx

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/types"
)

var testTime = time.Unix(1715874169, 1)

func TestRawData_String(t *testing.T) {
	strVal := "yo"
	intVal := int64(1)
	boolVal := true
	tests := []struct {
		data *RawData
		res  string
	}{
		{NilData, "<null>"},
		{BoolTrue, "true"},
		{BoolFalse, "false"},
		{BoolDataPtr(&boolVal), "true"},
		{BoolDataPtr(nil), "<null>"},
		{IntData(0), "0"},
		{IntDataPtr(&intVal), "1"},
		{IntDataPtr[int](nil), "<null>"},
		{FloatData(123), "123"},
		{StringData("yo"), "\"yo\""},
		{StringDataPtr(nil), "<null>"},
		{StringDataPtr(&strVal), "\"yo\""},
		{RegexData("ex"), "/ex/"},
		{TimeData(testTime), testTime.String()},
		{TimeDataPtr(nil), "<null>"},
		{TimeDataPtr(&testTime), testTime.String()},
		{DictData(int64(1)), "1"},
		{DictData(float64(1.2)), "1.2"},
		{DictData(string("yo")), "\"yo\""},
		{DictData([]interface{}{int64(1)}), "[1]"},
		{DictData(map[string]interface{}{"a": "b"}), "{\"a\":\"b\"}"},
		{ArrayData([]interface{}{"a", "b"}, types.String), "[\"a\",\"b\"]"},
		{MapData(map[string]interface{}{"a": "b"}, types.String), "{\"a\":\"b\"}"},
		// implicit nil:
		{&RawData{types.String, nil, nil}, "<null>"},
	}

	for i := range tests {
		assert.Equal(t, tests[i].res, tests[i].data.String())
	}
}

func TestTruthy(t *testing.T) {
	strVal := "yo"

	tests := []struct {
		data *RawData
		res  bool
	}{
		{NilData, false},
		{BoolTrue, true},
		{BoolFalse, false},
		{IntData(0), false},
		{IntData(123), true},
		{FloatData(0), false},
		{FloatData(1.23), true},
		{StringData(""), false},
		{StringData("b"), true},
		{StringDataPtr(nil), false},
		{StringDataPtr(&strVal), true},
		{RegexData(""), false},
		{RegexData("r"), true},
		{TimeData(time.Time{}), false},
		{TimeData(testTime), true},
		{TimeDataPtr(nil), false},
		{TimeDataPtr(&testTime), true},
		{ArrayData([]interface{}{}, types.Any), false},
		{ArrayData([]interface{}{false}, types.Bool), false},
		{ArrayData([]interface{}{true}, types.Bool), true},
		{MapData(map[string]interface{}{}, types.Any), true},
		{MapData(map[string]interface{}{"a": false}, types.Bool), false},
		{MapData(map[string]interface{}{"a": true}, types.Bool), true},
		{ResourceData(nil, "something"), true},
		// implicit nil:
		{&RawData{types.String, nil, nil}, false},
	}

	for i := range tests {
		o := tests[i]
		t.Run(o.data.String(), func(t *testing.T) {
			is, _ := o.data.IsTruthy()
			assert.Equal(t, o.res, is)
		})
	}
}

func TestSuccess(t *testing.T) {
	tests := []struct {
		data    *RawData
		success bool
		valid   bool
	}{
		{NilData, false, false},
		{BoolTrue, true, true},
		{BoolFalse, false, true},
		{IntData(0), false, false},
		{IntData(123), false, false},
		{FloatData(0), false, false},
		{FloatData(1.23), false, false},
		{StringData(""), false, false},
		{StringData("b"), false, false},
		{RegexData(""), false, false},
		{RegexData("r"), false, false},
		{TimeData(time.Time{}), false, false},
		{TimeDataPtr(nil), false, false},
		{TimeData(testTime), false, false},
		{ArrayData([]interface{}{}, types.Any), false, false},
		{ArrayData([]interface{}{true, false, true}, types.Bool), false, true},
		{ArrayData([]interface{}{true, true}, types.Bool), true, true},
		{ResourceData(nil, "something"), false, false},
		{
			data: &RawData{
				Type: types.Block,
				Value: map[string]interface{}{
					"__s": BoolData(true),
				},
			},
			success: true,
			valid:   true,
		},
		{
			data: &RawData{
				Type: types.Block,
				Value: map[string]interface{}{
					"__s": BoolData(false),
				},
			},
			success: false,
			valid:   true,
		},
		{
			data: &RawData{
				Type: types.Block,
				Value: map[string]interface{}{
					"__s": NilData,
				},
			},
			success: false,
			valid:   false,
		},
		{
			data: &RawData{
				Type:  types.Block,
				Value: map[string]interface{}{},
			},
			success: false,
			valid:   false,
		},
		// implicit nil:
		{&RawData{types.String, nil, nil}, false, false},
	}

	for i := range tests {
		o := tests[i]
		t.Run(o.data.String(), func(t *testing.T) {
			success, valid := o.data.IsSuccess()
			assert.Equal(t, o.success, success)
			assert.Equal(t, o.valid, valid)
		})
	}
}

func TestRawData_JSON(t *testing.T) {
	tests := []*RawData{
		NilData,
		BoolTrue,
		BoolFalse,
		IntData(0),
		IntData(123),
		FloatData(0),
		FloatData(1.23),
		StringData(""),
		StringData("b"),
		RegexData(""),
		RegexData("r"),
		TimeData(time.Time{}.In(time.Local)),
		TimeData(NeverFutureTime),
		TimeData(NeverPastTime),
		// TODO: the raw comparison here does not come out right, because of nano time
		// TimeData(now),
		ArrayData([]interface{}{"a", "b"}, types.String),
		MapData(map[string]interface{}{"a": "b"}, types.String),
		{Error: errors.New("test")},
	}

	for i := range tests {
		o := tests[i]
		t.Run(o.String(), func(t *testing.T) {
			out, err := json.Marshal(o)
			require.NoError(t, err)
			var res RawData
			err = json.Unmarshal(out, &res)
			require.NoError(t, err)
			assert.Equal(t, o, &res)
		})
	}
}
