package llx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/logger"
	"go.mondoo.io/mondoo/types"
)

func init() {
	logger.InitTestEnv()
}

func TestPrimitiveBool(t *testing.T) {
	a := &Primitive{Type: string(types.Bool), Value: bool2bytes(true)}
	b := &Primitive{Type: string(types.Bool), Value: bool2bytes(false)}
	assert.Equal(t, a, BoolPrimitive(true))
	assert.Equal(t, b, BoolPrimitive(false))
}

func TestPrimitiveFloat(t *testing.T) {
	a := &Primitive{Type: string(types.Float), Value: []byte{0x9a, 0x99, 0x99, 0x99, 0x99, 0x99, 0x28, 0x40}}
	assert.Equal(t, a, FloatPrimitive(12.3))
}

func TestPrimitiveInt(t *testing.T) {
	a := &Primitive{Type: string(types.Int), Value: []byte{0xf6, 0x01}}
	assert.Equal(t, a, IntPrimitive(123))
}

func TestPrimitiveString(t *testing.T) {
	a := &Primitive{Type: string(types.String), Value: []byte("hi")}
	assert.Equal(t, a, StringPrimitive("hi"))
}

func TestPrimitiveRegex(t *testing.T) {
	a := &Primitive{Type: string(types.Regex), Value: []byte(".*")}
	assert.Equal(t, a, RegexPrimitive(".*"))
}

func TestPrimitiveTime(t *testing.T) {
	a := &Primitive{Type: string(types.Time), Value: []byte{0x15, 0xcd, 0x5b, 0x07, 0x00, 0x00, 0x00, 0x00, 0x15, 0xcd, 0x5b, 0x07}}
	ut := time.Unix(123456789, 123456789)
	assert.Equal(t, a, TimePrimitive(&ut))

	assert.Equal(t, NilPrimitive, TimePrimitive(nil))
}

func TestPrimitiveRef(t *testing.T) {
	a := &Primitive{Type: string(types.Ref), Value: []byte{0xf6, 0x01}}
	assert.Equal(t, a, RefPrimitive(123))
}

func TestPrimitiveFunction(t *testing.T) {
	a := &Primitive{
		Type:  string(types.Function(0, nil)),
		Value: []byte{0xf6, 0x01},
	}
	assert.Equal(t, a, FunctionPrimitive(123))
}
