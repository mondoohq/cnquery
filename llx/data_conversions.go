package llx

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/types"
)

type dataConverter func(interface{}, types.Type) (*Primitive, error)
type primitiveConverter func(*Primitive) *RawData

var dataConverters map[types.Type]dataConverter
var primitiveConverters map[types.Type]primitiveConverter

func init() {
	dataConverters = map[types.Type]dataConverter{
		types.Unset:        unset2result,
		types.Nil:          nil2result,
		types.Bool:         bool2result,
		types.Ref:          ref2result,
		types.Int:          int2result,
		types.Float:        float2result,
		types.String:       string2result,
		types.Regex:        regex2result,
		types.Time:         time2result,
		types.Dict:         dict2result,
		types.Score:        score2result,
		types.ArrayLike:    array2result,
		types.MapLike:      map2result,
		types.ResourceLike: resource2result,
		types.FunctionLike: function2result,
	}

	primitiveConverters = map[types.Type]primitiveConverter{
		types.Unset:        punset2raw,
		types.Nil:          pnil2raw,
		types.Bool:         pbool2raw,
		types.Ref:          pref2raw,
		types.Int:          pint2raw,
		types.Float:        pfloat2raw,
		types.String:       pstring2raw,
		types.Regex:        pregex2raw,
		types.Time:         ptime2raw,
		types.Dict:         pdict2raw,
		types.Score:        pscore2raw,
		types.ArrayLike:    parray2raw,
		types.MapLike:      pmap2raw,
		types.ResourceLike: presource2raw,
		types.FunctionLike: pfunction2raw,
	}
}

func unset2result(value interface{}, typ types.Type) (*Primitive, error) {
	return UnsetPrimitive, nil
}

func nil2result(value interface{}, typ types.Type) (*Primitive, error) {
	return NilPrimitive, nil
}

func bool2result(value interface{}, typ types.Type) (*Primitive, error) {
	return BoolPrimitive(value.(bool)), nil
}

func ref2result(value interface{}, typ types.Type) (*Primitive, error) {
	return RefPrimitive(value.(int32)), nil
}

func int2result(value interface{}, typ types.Type) (*Primitive, error) {
	return IntPrimitive(value.(int64)), nil
}

func float2result(value interface{}, typ types.Type) (*Primitive, error) {
	return FloatPrimitive(value.(float64)), nil
}

func string2result(value interface{}, typ types.Type) (*Primitive, error) {
	return StringPrimitive(value.(string)), nil
}

func regex2result(value interface{}, typ types.Type) (*Primitive, error) {
	return RegexPrimitive(value.(string)), nil
}

func time2result(value interface{}, typ types.Type) (*Primitive, error) {
	return TimePrimitive(value.(*time.Time)), nil
}

func dict2result(value interface{}, typ types.Type) (*Primitive, error) {
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
			res[i], err = dict2result(x[i], typ)
			if err != nil {
				return nil, err
			}
		}

		return &Primitive{Type: types.Array(types.Dict), Array: res}, nil
	case map[string]interface{}:
		res := make(map[string]*Primitive, len(x))
		var err error
		for k, v := range x {
			res[k], err = dict2result(v, typ)
			if err != nil {
				return nil, err
			}
		}

		return &Primitive{Type: types.Map(types.String, types.Dict), Map: res}, nil

	default:
		return &Primitive{
			Type: types.Dict,
		}, fmt.Errorf("failed to convert dict to primitive, unsupported child type %T", x)
	}
}

func score2result(value interface{}, typ types.Type) (*Primitive, error) {
	return &Primitive{
		Type:  types.Score,
		Value: value.([]byte),
	}, nil
}

func array2result(value interface{}, typ types.Type) (*Primitive, error) {
	arr := value.([]interface{})
	res := make([]*Primitive, len(arr))
	ct := typ.Child()
	var err error
	for i := range arr {
		res[i], err = raw2primitive(arr[i], ct)
		if err != nil {
			return nil, err
		}
	}
	return &Primitive{Type: typ, Array: res}, nil
}

func stringmap2result(value interface{}, typ types.Type) (*Primitive, error) {
	m := value.(map[string]interface{})
	res := make(map[string]*Primitive)
	ct := typ.Child()
	var err error
	for k, v := range m {
		res[k], err = raw2primitive(v, ct)
		if err != nil {
			return nil, err
		}
	}
	return &Primitive{Type: typ, Map: res}, nil
}

func intmap2result(value interface{}, typ types.Type) (*Primitive, error) {
	m := value.(map[int32]interface{})
	res := make(map[string]*Primitive)
	ct := typ.Child()
	var err error
	for k, v := range m {
		res[strconv.FormatInt(int64(k), 10)], err = raw2primitive(v, ct)
		if err != nil {
			return nil, err
		}
	}
	return &Primitive{Type: typ, Map: res}, nil
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
	m := value.(lumi.ResourceType)
	r := m.LumiResource()
	v := r.Name + "\x00" + r.Id
	return &Primitive{Type: typ, Value: []byte(v)}, nil
}

func function2result(value interface{}, typ types.Type) (*Primitive, error) {
	return FunctionPrimitive(value.(int32)), nil
}

func raw2primitive(value interface{}, typ types.Type) (*Primitive, error) {
	if value == nil {
		return &Primitive{
			Type: typ,
		}, nil
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

	if r.Error != nil {
		errorMsg = r.Error.Error()
	}

	if r.Value == nil {
		return &Result{
			Data:  &Primitive{Type: r.Type},
			Error: errorMsg,
		}
	}

	data, err := raw2primitive(r.Value, r.Type)
	if err != nil {
		return &Result{
			Data:  &Primitive{Type: r.Type},
			Error: err.Error(),
		}
	}
	return &Result{
		Data:  data,
		Error: errorMsg,
	}
}

// Result converts the raw result into a proto-compliant data structure that
// can be sent over the wire. See RawData.Result()
func (r *RawResult) Result() *Result {
	res := r.Data.Result()
	res.CodeId = r.CodeID
	return res
}

func (r *Result) RawResult() *RawResult {
	if r == nil {
		return nil
	}

	data := &RawData{}
	if r.Data != nil {
		data = r.Data.RawData()
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
	return BoolData(bytes2bool(p.Value))
}

func pref2raw(p *Primitive) *RawData {
	return RefData(int32(bytes2int(p.Value)))
}

func pint2raw(p *Primitive) *RawData {
	return IntData(bytes2int(p.Value))
}

func pfloat2raw(p *Primitive) *RawData {
	return FloatData(bytes2float(p.Value))
}

func pstring2raw(p *Primitive) *RawData {
	return StringData(string(p.Value))
}

func pregex2raw(p *Primitive) *RawData {
	return RegexData(string(p.Value))
}

func ptime2raw(p *Primitive) *RawData {
	return TimeData(bytes2time(p.Value))
}

func pdict2raw(p *Primitive) *RawData {
	if p.Value == nil && p.Map == nil && p.Array == nil {
		return NilData
	}

	if p.Map != nil {
		res := make(map[string]interface{}, len(p.Map))
		for k, v := range p.Map {
			res[k] = pdict2raw(v).Value
		}
		return &RawData{Value: res, Error: nil, Type: types.Map(types.String, types.Dict)}
	}

	if p.Array != nil {
		res := make([]interface{}, len(p.Array))
		for i := range p.Array {
			res[i] = pdict2raw(p.Array[i]).Value
		}
		return &RawData{Value: res, Error: nil, Type: types.Array(types.Dict)}
	}

	// FIXME: we can't figure out what the real data is that is embedded if the primitive is dict
	return &RawData{
		Error: errors.New("failed to convert dict to raw, unsupported child type"),
		Type:  types.Dict,
	}
}

func pscore2raw(p *Primitive) *RawData {
	return &RawData{Value: p.Value, Error: nil, Type: types.Score}
}

func parray2raw(p *Primitive) *RawData {
	// FIXME: needs handover for referenced values...
	d, _, err := args2resourceargs(nil, 0, p.Array)
	return &RawData{Value: d, Error: err, Type: types.Type(p.Type)}
}

func pmap2raw(p *Primitive) *RawData {
	d, err := primitive2map(p.Map)
	return &RawData{Value: d, Error: err, Type: types.Type(p.Type)}
}

func presource2raw(p *Primitive) *RawData {
	res := strings.SplitN(string(p.Value), "\x00", 2)
	id := ""
	if len(res) > 1 {
		id = res[1]
	}
	return &RawData{Value: lumi.MockResource{
		StaticResource: &lumi.Resource{
			ResourceID: lumi.ResourceID{Name: res[0], Id: id},
		},
	}, Type: types.Type(p.Type)}
}

func pfunction2raw(p *Primitive) *RawData {
	return &RawData{Value: int32(bytes2int(p.Value)), Type: types.Type(p.Type)}
}

// RawData converts the primitive into the internal go-representation of the
// data that can be used for computations
func (p *Primitive) RawData() *RawData {
	// FIXME: This is a stopgap. It points to an underlying problem that exists and needs fixing.
	if p.Type == "" {
		return &RawData{Error: errors.New("cannot convert primitive with NO type information")}
	}

	typ := types.Type(p.Type)
	c, ok := primitiveConverters[typ.Underlying()]
	if !ok {
		return &RawData{Error: errors.New("cannot convert primitive to value for primitive type " + typ.Label())}
	}
	return c(p)
}

func args2resourceargs(c *LeiseExecutor, ref int32, args []*Primitive) ([]interface{}, int32, error) {
	if args == nil {
		return []interface{}{}, 0, nil
	}

	res := make([]interface{}, len(args))
	for i := range args {
		var cur *RawData

		if types.Type(args[i].Type) == types.Ref {
			var rref int32
			var err error
			cur, rref, err = c.resolveValue(args[i], ref)
			if rref > 0 || err != nil {
				return nil, rref, err
			}
		} else {
			cur = args[i].RawData()
		}

		if cur.Error != nil {
			return nil, 0, cur.Error
		}
		res[i] = cur.Value
	}
	return res, 0, nil
}

func primitive2map(m map[string]*Primitive) (map[string]interface{}, error) {
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

// returns the resolved argument if it's a ref; otherwise just the argument
// returns the reference if something else needs executing before it can be computed
// returns an error otherwise
func (c *LeiseExecutor) resolveValue(arg *Primitive, ref int32) (*RawData, int32, error) {
	typ := types.Type(arg.Type)
	switch typ.Underlying() {
	case types.Ref:
		srcRef := int32(bytes2int(arg.Value))
		// check if the reference exists; if not connect it
		res, ok := c.cache.Load(srcRef)
		if !ok {
			return c.connectRef(srcRef, ref)
		}
		return res.Result, 0, res.Result.Error

	case types.ArrayLike:
		res := make([]interface{}, len(arg.Array))
		for i := range arg.Array {
			c, ref, err := c.resolveValue(arg.Array[i], ref)
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
