package llx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/types"
)

func TestRawData_String(t *testing.T) {
	tests := []struct {
		data *RawData
		res  string
	}{
		{BoolTrue, "true"},
		{BoolFalse, "false"},
		{IntData(0), "0"},
		{FloatData(123), "123"},
		{StringData("yo"), "\"yo\""},
		{RegexData("ex"), "/ex/"},
		{ArrayData([]interface{}{"a", "b"}, types.String), "[\"a\",\"b\"]"},
		{MapData(map[string]interface{}{"a": "b"}, types.String), "{\"a\":\"b\"}"},
	}

	for i := range tests {
		assert.Equal(t, tests[i].res, tests[i].data.String())
	}
}
