package llx

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/types"
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
		rawResult := result.RawResultV2()
		reResult := rawResult.Result()

		assert.Equal(t, result.GetData().GetType(), reResult.GetData().GetType())
		assert.Equal(t, result.GetData().GetArray(), reResult.GetData().GetArray())
		assert.Equal(t, result.GetData().GetMap(), reResult.GetData().GetMap())
		assert.Equal(t, result.GetData().GetValue(), reResult.GetData().GetValue())
	}
}

func TestErrorConversion(t *testing.T) {
	// test error conversion
	rawData := StringPrimitive("hello").RawData()
	rawData.Error = errors.New("cannot do x")
	rawResult := &RawResult{Data: rawData, CodeID: "fakeid"}

	convertedRawResult := rawResult.Result().RawResultV2()
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

	convertedRawResult := rawResult.Result().RawResultV2()
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

	convertedRawResult := rawResult.Result().RawResultV2()
	assert.Equal(t, rawResult.Data.Type, convertedRawResult.Data.Type)
	assert.Equal(t, rawResult.Data.Value, convertedRawResult.Data.Value)
}

func TestCastResult(t *testing.T) {
	t.Run("to bool", func(t *testing.T) {
		t.Run("from legacy block", func(t *testing.T) {
			// Previously, blocks did not specify a __t field, which says
			// if the block is truthy based only on the evaluation of of
			// the entrypoints. Allow falling back
			t.Run("from block truthy", func(t *testing.T) {
				rawData := &RawData{
					Type: types.Block,
					Value: map[string]interface{}{
						"yUHOZ/pJzgQ3FLcnKAPphE4TgWqFptqPWA8GYl4e5Dqg0/YzQWcDml2cbrTEj3nj1rm0azm9povOYMRjTgSvZg==": &RawData{
							Type:  types.String,
							Value: "8.2.2004",
						},
					},
				}
				rawResult := &RawResult{Data: rawData, CodeID: "fakeid"}
				casted := rawResult.CastResult(types.Bool).RawResultV2()
				require.NoError(t, casted.Data.Error)
				require.Equal(t, types.Bool, casted.Data.Type)
				require.Equal(t, true, casted.Data.Value)
			})

			t.Run("from block not truthy", func(t *testing.T) {
				rawData := &RawData{
					Type: types.Block,
					Value: map[string]interface{}{
						"yUHOZ/pJzgQ3FLcnKAPphE4TgWqFptqPWA8GYl4e5Dqg0/YzQWcDml2cbrTEj3nj1rm0azm9povOYMRjTgSvZg==": &RawData{
							Type:  types.String,
							Value: "",
						},
					},
				}
				rawResult := &RawResult{Data: rawData, CodeID: "fakeid"}
				casted := rawResult.CastResult(types.Bool).RawResultV2()
				require.NoError(t, casted.Data.Error)
				require.Equal(t, types.Bool, casted.Data.Type)
				require.Equal(t, false, casted.Data.Value)
			})
		})
		t.Run("from block with __t", func(t *testing.T) {
			t.Run("from block not truthy", func(t *testing.T) {
				rawData := &RawData{
					Type: types.Block,
					Value: map[string]interface{}{
						"__t": BoolFalse,
						"yUHOZ/pJzgQ3FLcnKAPphE4TgWqFptqPWA8GYl4e5Dqg0/YzQWcDml2cbrTEj3nj1rm0azm9povOYMRjTgSvZg==": &RawData{
							Type:  types.String,
							Value: "8.2.2004",
						},
					},
				}
				rawResult := &RawResult{Data: rawData, CodeID: "fakeid"}
				casted := rawResult.CastResult(types.Bool).RawResultV2()
				require.NoError(t, casted.Data.Error)
				require.Equal(t, types.Bool, casted.Data.Type)
				require.Equal(t, false, casted.Data.Value)
			})

			t.Run("from block truthy", func(t *testing.T) {
				rawData := &RawData{
					Type: types.Block,
					Value: map[string]interface{}{
						"__t": BoolTrue,
						"yUHOZ/pJzgQ3FLcnKAPphE4TgWqFptqPWA8GYl4e5Dqg0/YzQWcDml2cbrTEj3nj1rm0azm9povOYMRjTgSvZg==": &RawData{
							Type:  types.String,
							Value: "",
						},
					},
				}
				rawResult := &RawResult{Data: rawData, CodeID: "fakeid"}
				casted := rawResult.CastResult(types.Bool).RawResultV2()
				require.NoError(t, casted.Data.Error)
				require.Equal(t, types.Bool, casted.Data.Type)
				require.Equal(t, true, casted.Data.Value)
			})
		})

		t.Run("from string truthy", func(t *testing.T) {
			rawData := &RawData{
				Type:  types.String,
				Value: "asdf",
			}
			rawResult := &RawResult{Data: rawData, CodeID: "fakeid"}
			casted := rawResult.CastResult(types.Bool).RawResultV2()
			require.NoError(t, casted.Data.Error)
			require.Equal(t, types.Bool, casted.Data.Type)
			require.Equal(t, true, casted.Data.Value)
		})

		t.Run("from string not truthy", func(t *testing.T) {
			rawData := &RawData{
				Type:  types.String,
				Value: "",
			}
			rawResult := &RawResult{Data: rawData, CodeID: "fakeid"}
			casted := rawResult.CastResult(types.Bool).RawResultV2()
			require.NoError(t, casted.Data.Error)
			require.Equal(t, types.Bool, casted.Data.Type)
			require.Equal(t, false, casted.Data.Value)
		})
	})

	t.Run("null-RawResults", func(t *testing.T) {
		testCases := []struct {
			Type types.Type
		}{
			{Type: types.String},
			{Type: types.Int},
			{Type: types.Float},
			{Type: types.Block},
			{Type: types.Array(types.String)},
			{Type: types.Map(types.String, types.String)},
		}

		for _, tc := range testCases {
			t.Run(tc.Type.Label()+" with type casting 📸", func(t *testing.T) {
				rawResult := &RawResult{Data: &RawData{Type: types.Nil}, CodeID: "fakeid"}
				casted := rawResult.CastResult(tc.Type)
				require.Empty(t, casted.Error)
				require.Equal(t, tc.Type.Label(), types.Type(casted.Data.Type).Label())
				require.True(t, !casted.Data.IsNil())
			})

			t.Run(tc.Type.Label()+" without casting 📵", func(t *testing.T) {
				rawResult := &RawResult{Data: &RawData{Type: tc.Type}, CodeID: "fakeid"}
				res := rawResult.Result()
				require.Empty(t, res.Error)
				require.Equal(t, tc.Type.Label(), types.Type(res.Data.Type).Label())
				require.True(t, !res.Data.IsNil())
			})

			t.Run(tc.Type.Label()+" convert RawResult => Result => RawResult creates non-nil values", func(t *testing.T) {
				rawResult := &RawResult{Data: &RawData{Type: tc.Type}, CodeID: "fakeid"}
				res := rawResult.Result().RawResultV2()
				require.NotNil(t, res.Data.Value)
			})
		}
	})
}

func TestResultFromNilConversion(t *testing.T) {
	t.Run("nil-Primitive => RawData => Result is non-nil", func(t *testing.T) {
		zeroTime := time.Unix(0, 0)
		tests := []struct {
			Primitive *Primitive
			Data      []byte
		}{
			{
				NilPrimitive,
				nil,
			},
			{
				&Primitive{Type: string(types.Bool)},
				BoolPrimitive(false).Value,
			},
			{
				&Primitive{Type: string(types.Int)},
				IntPrimitive(0).Value,
			},
			{
				&Primitive{Type: string(types.Float)},
				FloatPrimitive(0).Value,
			},
			{
				&Primitive{Type: string(types.String)},
				StringPrimitive("").Value,
			},
			{
				&Primitive{Type: string(types.Regex)},
				StringPrimitive("").Value,
			},
			{
				&Primitive{Type: string(types.Time)},
				TimePrimitive(&zeroTime).Value,
			},
		}

		for i := range tests {
			t.Run(types.Type(tests[i].Primitive.Type).Label(), func(t *testing.T) {
				result := tests[i].Primitive.RawData().Result()
				assert.Equal(t, tests[i].Primitive.Type, result.GetData().GetType())
				assert.Equal(t, tests[i].Data, result.GetData().GetValue())
			})
		}
	})

	// We have code that assumes this types return empty types instead
	// of nil types. This should be made more consistent, but I'm putting
	// these tests here to make sure other things are not broken
	t.Run("container types", func(t *testing.T) {
		t.Run("map", func(t *testing.T) {
			p := &Primitive{
				Type: string(types.Map(types.String, types.Int)),
			}
			rawData := p.RawData()
			result := rawData.Result()

			assert.Equal(t, p.Type, result.GetData().GetType())

			assert.NotNil(t, result.GetData().GetMap())
			assert.Empty(t, result.GetData().GetMap())
		})
		t.Run("array", func(t *testing.T) {
			p := &Primitive{
				Type: string(types.Array(types.String)),
			}
			rawData := p.RawData()
			result := rawData.Result()

			assert.Equal(t, p.Type, result.GetData().GetType())

			assert.NotNil(t, result.GetData().GetArray())
			assert.Empty(t, result.GetData().GetArray())
		})
	})

}
