package plugin

import (
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers/os/connection"
	"go.mondoo.com/cnquery/providers/proto"
	"go.mondoo.com/cnquery/types"
)

type Runtime struct {
	Connection connection.Connection
	// ...
}

type Resource interface {
	MqlID() (string, error)
	MqlName() string
}

type TValue[T any] struct {
	Data  T
	State State
	Error error
}

func (x *TValue[T]) ToDataRes(typ types.Type) *proto.DataRes {
	if x.State&StateIsSet == 0 {
		return &proto.DataRes{}
	}
	if x.State&StateIsNull != 0 {
		return &proto.DataRes{Data: &llx.Primitive{Type: string(typ)}}
	}
	raw := llx.RawData{Type: typ, Value: x.Data, Error: x.Error}
	res := raw.Result()
	return &proto.DataRes{Data: res.Data, Error: res.Error}
}

func PrimitiveToTValue[T any](p *llx.Primitive) TValue[T] {
	raw := p.RawData()
	if raw.Value == nil {
		return TValue[T]{State: StateIsNull}
	}
	return TValue[T]{Data: raw.Value.(T), State: StateIsSet}
}

// RawToTValue converts a raw (interface{}) value into a typed value
// and returns true if the type was correct.
func RawToTValue[T any](value interface{}) (TValue[T], bool) {
	if value == nil {
		return TValue[T]{State: StateIsNull}, true
	}

	tv, ok := value.(T)
	if !ok {
		return TValue[T]{}, false
	}

	return TValue[T]{Data: tv, State: StateIsSet}, true
}

type State byte

type notReady struct{}

func (n notReady) Error() string {
	return "NotReady"
}

var NotReady = notReady{}

const (
	StateIsSet State = 0x1 << iota
	StateIsNull
)

func GetOrCompute[T any](cached *TValue[T], compute func() (T, error)) (T, error) {
	if cached.State&StateIsSet != 0 {
		return cached.Data, cached.Error
	}

	x, err := compute()
	if err != nil {
		return x, err
	}

	// this only happens if the function set the field proactively, in which
	// case we grab the value from the cached entry for consistancy
	if cached.State&StateIsSet != 0 {
		return cached.Data, cached.Error
	}

	cached = &TValue[T]{Data: x, Error: err}
	return x, err
}
