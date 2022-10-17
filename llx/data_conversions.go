package llx

import (
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/types"
	"google.golang.org/protobuf/proto"
)

type (
	dataConverter      func(interface{}, types.Type) (*Primitive, error)
	primitiveConverter func(*Primitive) *RawData
)

var (
	dataConverters      map[types.Type]dataConverter
	primitiveConverters map[types.Type]primitiveConverter
)

func init() {
	dataConverters = map[types.Type]dataConverter{
		types.Unset:        unset2result,
		types.Nil:          nil2result,
		types.Bool:         bool2result,
		types.Int:          int2result,
		types.Float:        float2result,
		types.String:       string2result,
		types.Regex:        regex2result,
		types.Time:         time2result,
		types.Dict:         dict2result,
		types.Score:        score2result,
		types.Block:        block2result,
		types.ArrayLike:    array2result,
		types.MapLike:      map2result,
		types.ResourceLike: resource2result,
		types.FunctionLike: function2result,
	}

	primitiveConverters = map[types.Type]primitiveConverter{
		types.Unset:        punset2raw,
		types.Nil:          pnil2raw,
		types.Bool:         pbool2raw,
		types.Int:          pint2raw,
		types.Float:        pfloat2raw,
		types.String:       pstring2raw,
		types.Regex:        pregex2raw,
		types.Time:         ptime2raw,
		types.Dict:         pdict2raw,
		types.Score:        pscore2raw,
		types.Block:        pblock2rawV2,
		types.ArrayLike:    parray2raw,
		types.MapLike:      pmap2raw,
		types.ResourceLike: presource2raw,
		types.FunctionLike: pfunction2raw,
		types.Ref:          pref2raw,
	}
}

func dict2primitive(value interface{}) (*Primitive, error) {
	if value == nil {
		return NilPrimitive, nil
	}

	switch x := value.(type) {
	case bool:
		return BoolPrimitive(x), nil
	case int64:
		return IntPrimitive(x), nil
	case float64:
		return FloatPrimitive(x), nil
	case string:
		return StringPrimitive(x), nil
	case []interface{}:
		res := make([]*Primitive, len(x))
		var err error
		for i := range x {
			res[i], err = dict2primitive(x[i])
			if err != nil {
				return nil, err
			}
		}
		return &Primitive{Type: string(types.Array(types.Dict)), Array: res}, nil

	case map[string]interface{}:
		res := make(map[string]*Primitive, len(x))
		var err error
		for k, v := range x {
			res[k], err = dict2primitive(v)
			if err != nil {
				return nil, err
			}
		}
		return &Primitive{Type: string(types.Map(types.String, types.Dict)), Map: res}, nil

	default:
		return nil, errors.New("failed to convert dict to primitive, unsupported child type: " + reflect.TypeOf(x).String())
	}
}

func primitive2dictV2(p *Primitive) (interface{}, error) {
	switch types.Type(p.Type).Underlying() {
	case types.Nil:
		return nil, nil
	case types.Bool:
		return bytes2bool(p.Value), nil
	case types.Int:
		return bytes2int(p.Value), nil
	case types.Float:
		return bytes2float(p.Value), nil
	case types.String:
		return string(p.Value), nil
	case types.ArrayLike:
		d, _, err := args2resourceargsV2(nil, 0, p.Array)
		return d, err
	case types.MapLike:
		m, err := primitive2mapV2(p.Map)
		return m, err
	default:
		hexType := make([]byte, hex.EncodedLen(len(p.Type)))
		hex.Encode(hexType, []byte(p.Type))
		return nil, errors.New("unknown type to convert dict primitive back to raw data (" + string(hexType) + ")")
	}
}

func unset2result(value interface{}, typ types.Type) (*Primitive, error) {
	return UnsetPrimitive, nil
}

func nil2result(value interface{}, typ types.Type) (*Primitive, error) {
	return NilPrimitive, nil
}

func errInvalidConversion(value interface{}, expectedType types.Type) error {
	return fmt.Errorf("could not convert %T to %s", value, expectedType.Label())
}

func bool2result(value interface{}, typ types.Type) (*Primitive, error) {
	v, ok := value.(bool)
	if !ok {
		return nil, errInvalidConversion(value, typ)
	}
	return BoolPrimitive(v), nil
}

func ref2resultV2(value interface{}, typ types.Type) (*Primitive, error) {
	v, ok := value.(uint64)
	if !ok {
		return nil, errInvalidConversion(value, typ)
	}
	return RefPrimitiveV2(v), nil
}

func int2result(value interface{}, typ types.Type) (*Primitive, error) {
	v, ok := value.(int64)
	if !ok {
		return nil, errInvalidConversion(value, typ)
	}
	return IntPrimitive(v), nil
}

func float2result(value interface{}, typ types.Type) (*Primitive, error) {
	v, ok := value.(float64)
	if !ok {
		return nil, errInvalidConversion(value, typ)
	}
	return FloatPrimitive(v), nil
}

func string2result(value interface{}, typ types.Type) (*Primitive, error) {
	v, ok := value.(string)
	if !ok {
		return nil, errInvalidConversion(value, typ)
	}
	return StringPrimitive(v), nil
}

func regex2result(value interface{}, typ types.Type) (*Primitive, error) {
	v, ok := value.(string)
	if !ok {
		return nil, errInvalidConversion(value, typ)
	}
	return RegexPrimitive(v), nil
}

func time2result(value interface{}, typ types.Type) (*Primitive, error) {
	v, ok := value.(*time.Time)
	if !ok {
		return nil, errInvalidConversion(value, typ)
	}
	return TimePrimitive(v), nil
}

func dict2result(value interface{}, typ types.Type) (*Primitive, error) {
	prim, err := dict2primitive(value)
	if err != nil {
		return nil, err
	}

	raw, err := proto.MarshalOptions{Deterministic: true}.Marshal(prim)
	if err != nil {
		return nil, err
	}

	return &Primitive{Type: string(types.Dict), Value: raw}, nil
}

func score2result(value interface{}, typ types.Type) (*Primitive, error) {
	v, ok := value.([]byte)
	if !ok {
		return nil, errInvalidConversion(value, typ)
	}
	return &Primitive{
		Type:  string(types.Score),
		Value: v,
	}, nil
}

func block2result(value interface{}, typ types.Type) (*Primitive, error) {
	m, ok := value.(map[string]interface{})
	if !ok {
		return nil, errInvalidConversion(value, typ)
	}
	res := make(map[string]*Primitive)

	for k, v := range m {
		raw, ok := v.(*RawData)
		if !ok {
			return nil, errInvalidConversion(value, typ)
		}
		res[k] = raw.Result().Data
	}
	return &Primitive{Type: string(typ), Map: res}, nil
}

func array2result(value interface{}, typ types.Type) (*Primitive, error) {
	arr, ok := value.([]interface{})
	if !ok {
		return nil, errInvalidConversion(value, typ)
	}
	res := make([]*Primitive, len(arr))
	ct := typ.Child()
	var err error
	for i := range arr {
		res[i], err = raw2primitive(arr[i], ct)
		if err != nil {
			return nil, err
		}
	}
	return &Primitive{Type: string(typ), Array: res}, nil
}

func stringmap2result(value interface{}, typ types.Type) (*Primitive, error) {
	m, ok := value.(map[string]interface{})
	if !ok {
		return nil, errInvalidConversion(value, typ)
	}
	res := make(map[string]*Primitive)
	ct := typ.Child()
	var err error
	for k, v := range m {
		res[k], err = raw2primitive(v, ct)
		if err != nil {
			return nil, err
		}
	}
	return &Primitive{Type: string(typ), Map: res}, nil
}

func intmap2result(value interface{}, typ types.Type) (*Primitive, error) {
	m, ok := value.(map[int32]interface{})
	if !ok {
		return nil, errInvalidConversion(value, typ)
	}
	res := make(map[string]*Primitive)
	ct := typ.Child()
	var err error
	for k, v := range m {
		res[strconv.FormatInt(int64(k), 10)], err = raw2primitive(v, ct)
		if err != nil {
			return nil, err
		}
	}
	return &Primitive{Type: string(typ), Map: res}, nil
}

func map2result(value interface{}, typ types.Type) (*Primitive, error) {
	switch typ.Key() {
	case types.String:
		return stringmap2result(value, typ)
	case types.Int:
		return intmap2result(value, typ)
	default:
		return nil, errors.New("only supports turning string or int maps into primitives, not " + typ.Label())
	}
}

func resource2result(value interface{}, typ types.Type) (*Primitive, error) {
	m, ok := value.(resources.ResourceType)
	if !ok {
		return nil, errInvalidConversion(value, typ)
	}
	r := m.MqlResource()
	return &Primitive{Type: string(typ), Value: []byte(r.Id)}, nil
}

func function2result(value interface{}, typ types.Type) (*Primitive, error) {
	v, ok := value.(uint64)
	if ok {
		return FunctionPrimitive(v), nil
	}
	return nil, errInvalidConversion(value, typ)
}

func raw2primitive(value interface{}, typ types.Type) (*Primitive, error) {
	if value == nil {
		// there are only few types whose value is allowed to be nil
		switch typ {
		case types.Unset:
			return UnsetPrimitive, nil
		default:
			return NilPrimitive, nil
		}
	}

	utyp := typ.Underlying()
	c, ok := dataConverters[utyp]
	if !ok {
		rdata, ok := value.(*RawData)
		if ok {
			return raw2primitive(rdata.Value, rdata.Type)
		}
		return nil, errors.New("cannot serialize data type " + typ.Label())
	}
	return c(value, typ)
}

// Result converts the raw data into a proto-compliant data structure that
// can be sent over the wire. It converts the interface{} value of RawData
// into a []byte structure that is easily serializable
func (r *RawData) Result() *Result {
	errorMsg := ""

	// In case we encounter an error we need to still construct the result object
	// with the type information so it can be processed by the server
	if r.Error != nil {
		errorMsg = r.Error.Error()

		// if the value is nil, we don't want to loose the type information,
		// so we return it early before raw2primitive has a chance to change the
		// type to nil
		if r.Value == nil {
			return &Result{
				Data:  &Primitive{Type: string(r.Type)},
				Error: errorMsg,
			}
		}
	}

	data, err := raw2primitive(r.Value, r.Type)
	if err != nil {
		return &Result{
			Data:  &Primitive{Type: string(r.Type)},
			Error: err.Error(),
		}
	}
	return &Result{
		Data:  data,
		Error: errorMsg,
	}
}

func (r *RawData) CastResult(t types.Type) (*Result, error) {
	errorMsg := ""

	// In case we encounter an error we need to still construct the result object
	// with the type information so it can be processed by the server
	if r.Error != nil {
		errorMsg = r.Error.Error()
	}

	// Allow any type to take on nil values
	if r.Value == nil {
		return &Result{
			Data:  &Primitive{Type: string(t)},
			Error: errorMsg,
		}, nil
	}

	if t == types.Bool {
		truthy, castable := r.IsTruthy()
		if !castable {
			return nil, fmt.Errorf("cannot cast from %s to %s", r.Type.Label(), t.Label())
		}
		return &Result{
			Data:  BoolPrimitive(truthy),
			Error: errorMsg,
		}, nil
	}

	data, err := raw2primitive(r.Value, t)
	if err != nil {
		return nil, err
	}
	return &Result{
		Data:  data,
		Error: errorMsg,
	}, nil
}

func (r *RawResult) CastResult(t types.Type) *Result {
	res, err := r.Data.CastResult(t)
	if err != nil {
		return &Result{
			CodeId: r.CodeID,
			Data:   &Primitive{Type: string(t)},
			Error:  err.Error(),
		}
	}
	res.CodeId = r.CodeID
	return res
}

// Result converts the raw result into a proto-compliant data structure that
// can be sent over the wire. See RawData.Result()
func (r *RawResult) Result() *Result {
	res := r.Data.Result()
	res.CodeId = r.CodeID
	return res
}

func (r *Result) RawResultV2() *RawResult {
	if r == nil {
		return nil
	}

	data := &RawData{}
	if r.Data != nil {
		if r.Data.IsNil() {
			data.Type = types.Nil
		} else {
			data = r.Data.RawData()
		}
	}
	if len(r.Error) > 0 {
		data.Error = errors.New(r.Error)
	}
	return &RawResult{
		Data:   data,
		CodeID: r.CodeId,
	}
}

func punset2raw(p *Primitive) *RawData {
	return UnsetData
}

func pnil2raw(p *Primitive) *RawData {
	return NilData
}

func pbool2raw(p *Primitive) *RawData {
	if len(p.Value) == 0 {
		return &RawData{
			Type:  types.Type(p.Type),
			Value: false,
		}
	}
	return BoolData(bytes2bool(p.Value))
}

func pint2raw(p *Primitive) *RawData {
	if len(p.Value) == 0 {
		return &RawData{
			Type:  types.Type(p.Type),
			Value: int64(0),
		}
	}
	return IntData(bytes2int(p.Value))
}

func pfloat2raw(p *Primitive) *RawData {
	if len(p.Value) == 0 {
		return &RawData{
			Type:  types.Type(p.Type),
			Value: float64(0),
		}
	}
	return FloatData(bytes2float(p.Value))
}

func pstring2raw(p *Primitive) *RawData {
	return StringData(string(p.Value))
}

func pregex2raw(p *Primitive) *RawData {
	return RegexData(string(p.Value))
}

func ptime2raw(p *Primitive) *RawData {
	if len(p.Value) == 0 {
		t := time.Unix(0, 0)
		return &RawData{
			Type:  types.Type(p.Type),
			Value: &t,
		}
	}
	return TimeData(bytes2time(p.Value))
}

func pdict2raw(p *Primitive) *RawData {
	if p.Value == nil {
		return &RawData{
			Type:  types.Dict,
			Value: nil,
		}
	}

	res := Primitive{} // unmarshal placeholder
	err := proto.Unmarshal(p.Value, &res)
	if err != nil {
		return &RawData{Error: err, Type: types.Dict}
	}

	raw, err := primitive2dictV2(&res)
	return &RawData{Error: err, Type: types.Dict, Value: raw}
}

func pscore2raw(p *Primitive) *RawData {
	if len(p.Value) == 0 {
		return &RawData{
			Value: int64(0),
			Type:  types.Score,
		}
	}
	return &RawData{Value: p.Value, Type: types.Score}
}

func pblock2rawV2(p *Primitive) *RawData {
	d, err := primitive2rawdataMapV2(p.Map)
	return &RawData{Value: d, Error: err, Type: types.Type(p.Type)}
}

func parray2raw(p *Primitive) *RawData {
	// Note: We don't hand over the compiler here. Reason is that if you have
	// primitives that have refs in them, you should properly resolve them
	// during the execution of the code. This function is really only applicable
	// much later when you try to just get to the values of the returned data.
	d, _, err := args2resourceargsV2(nil, 0, p.Array)
	if d == nil {
		d = []interface{}{}
	}
	return &RawData{Value: d, Error: err, Type: types.Type(p.Type)}
}

func pmap2raw(p *Primitive) *RawData {
	d, err := primitive2mapV2(p.Map)
	return &RawData{Value: d, Error: err, Type: types.Type(p.Type)}
}

func presource2raw(p *Primitive) *RawData {
	id := string(p.Value)

	return &RawData{Value: resources.MockResource{
		StaticResource: &resources.Resource{
			ResourceID: resources.ResourceID{Name: types.Type(p.Type).ResourceName(), Id: id},
		},
	}, Type: types.Type(p.Type)}
}

func pfunction2raw(p *Primitive) *RawData {
	// note: function pointers can never have a value that is nil
	rv := bytes2int(p.Value)
	if rv>>32 != 0 {
		return &RawData{Value: uint64(bytes2int(p.Value)), Type: types.Type(p.Type)}
	} else {
		return &RawData{Value: int32(bytes2int(p.Value)), Type: types.Type(p.Type)}
	}
}

func pref2raw(p *Primitive) *RawData {
	// note: refs can never have a value that is nil
	rv := bytes2int(p.Value)
	if rv>>32 != 0 {
		return &RawData{Value: uint64(bytes2int(p.Value)), Type: types.Type(p.Type)}
	} else {
		return &RawData{Value: int32(bytes2int(p.Value)), Type: types.Type(p.Type)}
	}
}

// Tries to resolve primitives; returns refs if they don't exist yet.
// Returns errors and ref=0 if there was an error.
// Note: Returned array can be nil.
func args2resourceargsV2(b *blockExecutor, ref uint64, args []*Primitive) ([]interface{}, uint64, error) {
	if args == nil {
		return []interface{}{}, 0, nil
	}

	res := make([]interface{}, len(args))
	for i := range args {
		var cur *RawData

		if b != nil && types.Type(args[i].Type) == types.Ref {
			var rref uint64
			var err error
			cur, rref, err = b.resolveValue(args[i], ref)
			if rref > 0 || err != nil {
				return nil, rref, err
			}
		} else {
			cur = args[i].RawData()
		}

		if cur != nil {
			if cur.Error != nil {
				return nil, 0, cur.Error
			}
			res[i] = cur.Value
		}
	}
	return res, 0, nil
}

// Converts a map of primitives into a map of go data (no type info).
// Return map is never nil.
func primitive2mapV2(m map[string]*Primitive) (map[string]interface{}, error) {
	if m == nil {
		return map[string]interface{}{}, nil
	}

	res := make(map[string]interface{})
	for k, v := range m {
		if v == nil {
			res[k] = nil
			continue
		}
		cur := v.RawData()
		if cur.Error != nil {
			return nil, cur.Error
		}
		res[k] = cur.Value
	}
	return res, nil
}

// Converts a map of primitives into a map of RawData (to preserve type-info).
// Return map is never nil.
func primitive2rawdataMapV2(m map[string]*Primitive) (map[string]interface{}, error) {
	if m == nil {
		return map[string]interface{}{}, nil
	}

	res := make(map[string]interface{})
	for k, v := range m {
		if v == nil {
			res[k] = nil
			continue
		}
		cur := v.RawData()
		if cur.Error != nil {
			return nil, cur.Error
		}
		res[k] = cur
	}
	return res, nil
}

// RawData converts the primitive into the internal go-representation of the
// data that can be used for computations
func (p *Primitive) RawData() *RawData {
	// FIXME: This is a stopgap. It points to an underlying problem that exists and needs fixing.
	if p.GetType() == "" {
		return &RawData{Error: errors.New("cannot convert primitive with NO type information")}
	}

	typ := types.Type(p.Type)
	c, ok := primitiveConverters[typ.Underlying()]
	if !ok {
		return &RawData{Error: errors.New("cannot convert primitive to value for primitive type " + typ.Label())}
	}
	return c(p)
}

func (b *blockExecutor) lookupValue(ref uint64) (*RawData, uint64, error) {
	if b == nil {
		panic("value not computed")
	}

	res, ok := b.cache.Load(ref)
	if !ok {
		return b.parent.lookupValue(ref)
	}
	return res.Result, 0, res.Result.Error
}

func (b *blockExecutor) resolveRef(srcRef uint64, ref uint64) (*RawData, uint64, error) {
	if !b.isInMyBlock(srcRef) {
		// the value is provided by a parent
		return b.parent.lookupValue(srcRef)
	} else {
		// check if the reference exists; if not connect it
		res, ok := b.cache.Load(srcRef)
		if !ok {
			return b.connectRef(srcRef, ref)
		}
		return res.Result, 0, res.Result.Error
	}
}

// returns the resolved argument if it's a ref; otherwise just the argument
// returns the reference if something else needs executing before it can be computed
// returns an error otherwise
func (b *blockExecutor) resolveValue(arg *Primitive, ref uint64) (*RawData, uint64, error) {
	typ := types.Type(arg.Type)
	switch typ.Underlying() {
	case types.Ref:
		srcRef := uint64(bytes2int(arg.Value))
		return b.resolveRef(srcRef, ref)
	case types.ArrayLike:
		res := make([]interface{}, len(arg.Array))
		for i := range arg.Array {
			c, ref, err := b.resolveValue(arg.Array[i], ref)
			if ref != 0 || err != nil {
				return nil, ref, err
			}
			res[i] = c.Value
		}

		// type is in arg.Value
		return &RawData{
			Type:  typ,
			Value: res,
		}, 0, nil
	}

	v := arg.RawData()
	return v, 0, v.Error
}
