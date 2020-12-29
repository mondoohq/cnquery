package llx

import (
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
		ArrayPrimitive([]*Primitive{StringPrimitive("hello"), IntPrimitive(42)}, types.Any),
	}

	for i := range tests {
		rawData := tests[i].RawData()
		result := rawData.Result()
		rawResult := result.RawResult()
		reResult := rawResult.Result()
		assert.Equal(t, result, reResult)
	}
}
