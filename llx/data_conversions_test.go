package llx

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/types"
)

func TestResultConversion(t *testing.T) {
	now := time.Now()
	tests := []*Primitive{
		BoolPrimitive(true),
		BoolPrimitive(false),
		IntPrimitive(42),
		FloatPrimitive(1.2),
		ScorePrimitive(100),
		StringPrimitive("hello"),
		NilPrimitive,
		TimePrimitive(&now),
		ArrayPrimitive([]*Primitive{StringPrimitive("hello")}, types.String),
		ArrayPrimitive([]*Primitive{IntPrimitive(42)}, types.Int),

		// TODO: any is not supported for arrays in serialization
		//ArrayPrimitive([]*Primitive{StringPrimitive("hello"), IntPrimitive(42)}, types.Any),
	}

	for i := range tests {
		rawData := tests[i].RawData()
		result := rawData.Result()
		rawResult := result.RawResult()
		reResult := rawResult.Result()
		assert.Equal(t, result, reResult)
	}
}

func TestErrorConversion(t *testing.T) {
	// test error conversion
	rawData := StringPrimitive("hello").RawData()
	rawData.Error = errors.New("cannot do x")
	rawResult := &RawResult{Data: rawData, CodeID: "fakeid"}

	convertedRawResult := rawResult.Result().RawResult()
	assert.Equal(t, rawResult.Data.Type, convertedRawResult.Data.Type)
	assert.Equal(t, rawResult.Data.Value, convertedRawResult.Data.Value)
	assert.Equal(t, rawResult.Data.Error, convertedRawResult.Data.Error)
	assert.Equal(t, rawResult.CodeID, convertedRawResult.CodeID)
}

func TestDictConversion(t *testing.T) {
	rawData := &RawData{
		Type:  types.Dict,
		Value: "hello",
	}
	rawResult := &RawResult{Data: rawData, CodeID: "fakeid"}

	convertedRawResult := rawResult.Result().RawResult()
	assert.Equal(t, rawResult.Data.Type, convertedRawResult.Data.Type)
	assert.Equal(t, rawResult.Data.Value, convertedRawResult.Data.Value)
}

func TestResourceConversion(t *testing.T) {
	// this checks that the result of a resource is properly converted. in this case  platform { name title release }
	rawData := &RawData{
		Type: types.Block,
		Value: map[string]interface{}{
			"yUHOZ/pJzgQ3FLcnKAPphE4TgWqFptqPWA8GYl4e5Dqg0/YzQWcDml2cbrTEj3nj1rm0azm9povOYMRjTgSvZg==": &RawData{
				Type:  types.String,
				Value: "8.2.2004",
			},
			"eXSx690ws3fjmTRXKjSBqpgounx3VRr3RKSaBo1mmPnW7+D2NSjYD9W5uNGiageTGQh37XHomdUvF4iSMON9yQ==": &RawData{
				Type:  types.String,
				Value: "CentOS Linux",
			},
			"EpnHIF31KeNgY/3Z4KyBuKHQ0kk/i+MyYbTX+ZWiQIAvK6lv4P2Nlf9CKAIrn2KOfCWICteI96BN1e8GA6sNZA==": &RawData{
				Type:  types.String,
				Value: "centos",
			},
		},
	}
	rawResult := &RawResult{Data: rawData, CodeID: "fakeid"}

	convertedRawResult := rawResult.Result().RawResult()
	assert.Equal(t, rawResult.Data.Type, convertedRawResult.Data.Type)
	assert.Equal(t, rawResult.Data.Value, convertedRawResult.Data.Value)
}
