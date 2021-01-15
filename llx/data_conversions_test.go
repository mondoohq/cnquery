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
