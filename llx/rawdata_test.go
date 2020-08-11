package llx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/types"
)

var now = time.Now()

func TestRawData_String(t *testing.T) {
	tests := []struct {
		data *RawData
		res  string
	}{
		{NilData, "<null>"},
		{BoolTrue, "true"},
		{BoolFalse, "false"},
		{IntData(0), "0"},
		{FloatData(123), "123"},
		{StringData("yo"), "\"yo\""},
		{RegexData("ex"), "/ex/"},
		{TimeData(&now), now.String()},
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
		{RegexData(""), false},
		{RegexData("r"), true},
		{TimeData(&time.Time{}), false},
		{TimeData(&now), true},
		{ArrayData([]interface{}{}, types.Any), true},
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
