package plugin

import (
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/types"
)

type Runtime struct {
	Connection   Connection
	Resources    map[string]Resource
	Callback     ProviderCallback
	HasRecording bool
}

type Connection interface{}

type Resource interface {
	MqlID() string
	MqlName() string
}

func (r *Runtime) ResourceFromRecording(name string, id string) (map[string]*llx.RawData, error) {
	data, err := r.Callback.GetRecording(&DataReq{
		Resource:   name,
		ResourceId: id,
	})
	if err != nil {
		return nil, err
	}

	return ProtoArgsToRawDataArgs(data.Fields)
}

type TValue[T any] struct {
	Data  T
	State State
	Error error
}

func (x *TValue[T]) ToDataRes(typ types.Type) *DataRes {
	if x.State&StateIsSet == 0 {
		return &DataRes{}
	}
	if x.State&StateIsNull != 0 {
		res := &DataRes{
			Data: &llx.Primitive{Type: string(typ)},
		}
		if x.Error != nil {
			res.Error = x.Error.Error()
		}
		return res
	}
	raw := llx.RawData{Type: typ, Value: x.Data, Error: x.Error}
	res := raw.Result()
	return &DataRes{Data: res.Data, Error: res.Error}
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
func RawToTValue[T any](value interface{}, err error) (TValue[T], bool) {
	if value == nil {
		return TValue[T]{State: StateIsNull | StateIsSet, Error: err}, true
	}

	tv, ok := value.(T)
	if !ok {
		return TValue[T]{}, false
	}

	return TValue[T]{Data: tv, State: StateIsSet, Error: err}, true
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

func GetOrCompute[T any](cached *TValue[T], compute func() (T, error)) *TValue[T] {
	if cached.State&StateIsSet != 0 {
		return cached
	}

	x, err := compute()
	if err != nil {
		res := &TValue[T]{Data: x, Error: err}
		if err != NotReady {
			res.State = StateIsSet | StateIsNull
		}
		return res
	}

	// this only happens if the function set the field proactively, in which
	// case we grab the value from the cached entry for consistancy
	if cached.State&StateIsSet != 0 {
		return cached
	}

	(*cached) = TValue[T]{Data: x, State: StateIsSet, Error: err}
	return cached
}

func PrimitiveArgsToRawDataArgs(pargs map[string]*llx.Primitive) map[string]*llx.RawData {
	res := make(map[string]*llx.RawData, len(pargs))
	for k, v := range pargs {
		res[k] = v.RawData()
	}
	return res
}

func ProtoArgsToRawDataArgs(pargs map[string]*llx.Result) (map[string]*llx.RawData, error) {
	res := make(map[string]*llx.RawData, len(pargs))
	var err error
	for k, v := range pargs {
		res[k] = v.RawData()
	}

	return res, err
}

func NonErrorArgs(pargs map[string]*llx.RawData) map[string]*llx.RawData {
	if len(pargs) == 0 {
		return map[string]*llx.RawData{}
	}

	res := map[string]*llx.RawData{}
	for k, v := range pargs {
		if v.Error != nil {
			continue
		}
		res[k] = v
	}
	return res
}
