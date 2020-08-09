package leise

import (
	"errors"
	"sort"

	"go.mondoo.io/mondoo/leise/parser"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/types"
)

type compileHandler struct {
	typ       func(types.Type) types.Type
	signature FunctionSignature
	compile   func(*compiler, types.Type, int32, string, *parser.Call) (types.Type, error)
}

var childType = func(t types.Type) types.Type { return t.Child() }
var arrayBlockType = func(t types.Type) types.Type { return types.Array(types.Map(types.Int, types.Any)) }
var mapBlockType = func(t types.Type) types.Type { return types.Map(types.String, types.Any) }
var boolType = func(t types.Type) types.Type { return types.Bool }
var intType = func(t types.Type) types.Type { return types.Int }
var stringType = func(t types.Type) types.Type { return types.String }
var stringArrayType = func(t types.Type) types.Type { return types.Array(types.String) }
var dictType = func(t types.Type) types.Type { return types.Dict }

var builtinFunctions map[types.Type]map[string]compileHandler

func init() {
	builtinFunctions = map[types.Type]map[string]compileHandler{
		types.String: {
			"contains": {compile: compileStringContains, typ: boolType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.String}}},
			"downcase": {typ: stringType, signature: FunctionSignature{}},
			"length":   {typ: intType, signature: FunctionSignature{}},
			"lines":    {typ: stringArrayType, signature: FunctionSignature{}},
			"split":    {typ: stringArrayType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.String}}},
		},
		types.Time: {
			"seconds": {typ: intType, signature: FunctionSignature{}},
			"minutes": {typ: intType, signature: FunctionSignature{}},
			"hours":   {typ: intType, signature: FunctionSignature{}},
			"days":    {typ: intType, signature: FunctionSignature{}},
			"unix":    {typ: intType, signature: FunctionSignature{}},
		},
		types.Dict: {
			"[]": {typ: dictType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.Any}}},
			"{}": {typ: dictType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			// string-ish
			"length":   {typ: intType, signature: FunctionSignature{}},
			"downcase": {typ: stringType, signature: FunctionSignature{}},
			"lines":    {typ: stringArrayType, signature: FunctionSignature{}},
			"split":    {typ: stringArrayType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.String}}},
			// array-ish
			"where":    {compile: compileWhere, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"contains": {compile: compileStringContains, typ: boolType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.String}}},
			"one":      {compile: compileArrayOne, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"all":      {compile: compileArrayAll, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"any":      {compile: compileArrayAny, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
		},
		types.ArrayLike: {
			"[]":       {typ: childType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.Int}}},
			"{}":       {typ: arrayBlockType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"length":   {typ: intType, signature: FunctionSignature{}},
			"where":    {compile: compileWhere, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"contains": {compile: compileArrayContains, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"one":      {compile: compileArrayOne, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"all":      {compile: compileArrayAll, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"any":      {compile: compileArrayAny, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
		},
		types.MapLike: {
			"[]":     {typ: childType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.String}}},
			"{}":     {typ: mapBlockType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"length": {typ: intType, signature: FunctionSignature{}},
		},
		types.ResourceLike: {
			// "":       compileHandler{compile: compileResourceDefault},
			"length":   {compile: compileResourceLength, signature: FunctionSignature{}},
			"where":    {compile: compileResourceWhere, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"contains": {compile: compileResourceContains, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"one":      {compile: compileResourceOne, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"all":      {compile: compileResourceAll, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"any":      {compile: compileResourceAny, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
		},
		// TODO: [#32] unique builtin fields that need a long-term support in LR
		types.Resource("parse"): {
			"date": {compile: compileResourceParseDate, signature: FunctionSignature{Required: 1, Args: []types.Type{types.String, types.String}}},
		},
	}
}

// Note: Call it with the full type, not just the underlying type
func builtinFunction(typ types.Type, id string) (*compileHandler, error) {
	// TODO: [#32] special handlers for specific types, which are builtin and should
	// be removed long-term, one the resource is native in LR
	fh, ok := builtinFunctions[typ]
	if ok {
		c, ok := fh[id]
		if ok {
			return &c, nil
		}
	}

	fh, ok = builtinFunctions[typ.Underlying()]
	if ok {
		c, ok := fh[id]
		if ok {
			return &c, nil
		}
	} else {
		return nil, errors.New("Cannot find any functions for type '" + typ.Label() + "' during compile")
	}

	return nil, errors.New("Cannot find function '" + id + "' for type '" + typ.Label() + "' during compile")
}

func fieldNames(resource *lumi.ResourceInfo) []string {
	res := make([]string, len(resource.Fields))
	idx := 0
	for k := range resource.Fields {
		res[idx] = k
		idx++
	}
	return res
}

func availableFields(c *compiler, typ types.Type) []string {
	m, ok := builtinFunctions[typ.Underlying()]
	if !ok {
		return nil
	}

	res := make([]string, len(m))
	idx := 0
	for k := range m {
		res[idx] = k
		idx++
	}

	if typ.IsResource() {
		fieldNames := fieldNames(c.Schema.Resources[typ.Name()])
		res = append(res, fieldNames...)
	}
	sort.Strings(res)

	return res
}
