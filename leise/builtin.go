package leise

import (
	"errors"

	"go.mondoo.io/mondoo/leise/parser"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/types"
)

type compileHandler struct {
	typ       func(types.Type) types.Type
	signature FunctionSignature
	compile   func(*compiler, types.Type, int32, string, *parser.Call) (types.Type, error)
}

var childType = func(t types.Type) types.Type { return t.Child() }
var arrayBlockType = func(t types.Type) types.Type { return types.Array(types.Map(types.Int, types.Block)) }
var boolType = func(t types.Type) types.Type { return types.Bool }
var intType = func(t types.Type) types.Type { return types.Int }
var stringType = func(t types.Type) types.Type { return types.String }
var stringArrayType = func(t types.Type) types.Type { return types.Array(types.String) }
var dictType = func(t types.Type) types.Type { return types.Dict }
var blockType = func(t types.Type) types.Type { return types.Block }
var dictArrayType = func(t types.Type) types.Type { return types.Array(types.Dict) }

var builtinFunctions map[types.Type]map[string]compileHandler

func init() {
	builtinFunctions = map[types.Type]map[string]compileHandler{
		types.String: {
			"contains": {compile: compileStringContains, typ: boolType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.String}}},
			"find":     {typ: stringArrayType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.Regex}}},
			"length":   {typ: intType, signature: FunctionSignature{}},
			"downcase": {typ: stringType, signature: FunctionSignature{}},
			"lines":    {typ: stringArrayType, signature: FunctionSignature{}},
			"split":    {typ: stringArrayType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.String}}},
			"trim":     {typ: stringType, signature: FunctionSignature{Required: 0, Args: []types.Type{types.String}}},
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
			"{}": {typ: blockType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			// string-ish
			"contains": {compile: compileStringContains, typ: boolType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.String}}},
			"find":     {typ: stringArrayType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.Regex}}},
			"length":   {typ: intType, signature: FunctionSignature{}},
			"downcase": {typ: stringType, signature: FunctionSignature{}},
			"lines":    {typ: stringArrayType, signature: FunctionSignature{}},
			"split":    {typ: stringArrayType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.String}}},
			"trim":     {typ: stringType, signature: FunctionSignature{Required: 0, Args: []types.Type{types.String}}},
			// array-ish
			"where": {compile: compileWhere, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"all":   {compile: compileArrayAll, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"any":   {compile: compileArrayAny, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"one":   {compile: compileArrayOne, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"none":  {compile: compileArrayNone, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			// map-ish
			"keys":   {typ: stringArrayType, signature: FunctionSignature{}},
			"values": {typ: dictArrayType, signature: FunctionSignature{}},
		},
		types.ArrayLike: {
			"[]":         {typ: childType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.Int}}},
			"{}":         {typ: arrayBlockType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"length":     {typ: intType, signature: FunctionSignature{}},
			"where":      {compile: compileWhere, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"duplicates": {compile: compileArrayDuplicates, signature: FunctionSignature{Required: 0}},
			"unique":     {compile: compileArrayUnique, signature: FunctionSignature{Required: 0}},
			"contains":   {compile: compileArrayContains, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"all":        {compile: compileArrayAll, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"any":        {compile: compileArrayAny, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"one":        {compile: compileArrayOne, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"none":       {compile: compileArrayNone, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
		},
		types.MapLike: {
			"[]":     {typ: childType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.String}}},
			"{}":     {typ: blockType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"length": {typ: intType, signature: FunctionSignature{}},
			"keys":   {typ: stringArrayType, signature: FunctionSignature{}},
			"values": {typ: dictArrayType, signature: FunctionSignature{}},
		},
		types.ResourceLike: {
			// "":       compileHandler{compile: compileResourceDefault},
			"length":   {compile: compileResourceLength, signature: FunctionSignature{}},
			"where":    {compile: compileResourceWhere, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"contains": {compile: compileResourceContains, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"all":      {compile: compileResourceAll, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"any":      {compile: compileResourceAny, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"one":      {compile: compileResourceOne, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"none":     {compile: compileResourceNone, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
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
		return nil, errors.New("cannot find any functions for type '" + typ.Label() + "' during compile")
	}

	return nil, errors.New("cannot find function '" + id + "' for type '" + typ.Label() + "' during compile")
}

func fieldsInfo(resourceInfo *lumi.ResourceInfo) map[string]llx.Documentation {
	res := map[string]llx.Documentation{}
	for k, v := range resourceInfo.Fields {
		res[k] = llx.Documentation{
			Field: k,
			Title: v.Title,
			Desc:  v.Desc,
		}
	}

	return res
}

func availableFields(c *compiler, typ types.Type) map[string]llx.Documentation {
	var res map[string]llx.Documentation

	// resources maintain their own fields and may be list resources
	if typ.IsResource() {
		resourceInfo := c.Schema.Resources[typ.ResourceName()]
		res = fieldsInfo(resourceInfo)

		_, err := listResource(c, typ)
		if err == nil {
			m := builtinFunctions[typ.Underlying()]
			for k := range m {
				res[k] = llx.Documentation{
					Field: k,
				}
			}
		}

	}

	// We first try to auto-complete the full type. This is important for
	// more complex types, like resource types (eg `parse`).
	builtins := builtinFunctions[typ]
	if builtins == nil && res == nil {
		// Only if we fail to find the full resource AND if we couldn't look
		// up the resource definition either, will we look for additional
		// methods. Otherwise we stick to the directly defined methods, not any
		// potentially "shared" methods (which aren't actually shared).
		builtins = builtinFunctions[typ.Underlying()]
		if builtins == nil {
			return res
		}
	}

	// the non-resource use-case:
	if res == nil {
		res = make(map[string]llx.Documentation, len(builtins))
	}

	for k := range builtins {
		res[k] = llx.Documentation{
			Field: k,
		}
	}

	return res
}
