package mqlc

import (
	"errors"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/mqlc/parser"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/types"
)

type compileHandler struct {
	typ       func(types.Type) types.Type
	signature FunctionSignature
	compile   func(*compiler, types.Type, uint64, string, *parser.Call) (types.Type, error)
}

var (
	childType       = func(t types.Type) types.Type { return t.Child() }
	arrayBlockType  = func(t types.Type) types.Type { return types.Array(types.Map(types.Int, types.Block)) }
	boolType        = func(t types.Type) types.Type { return types.Bool }
	intType         = func(t types.Type) types.Type { return types.Int }
	stringType      = func(t types.Type) types.Type { return types.String }
	stringArrayType = func(t types.Type) types.Type { return types.Array(types.String) }
	dictType        = func(t types.Type) types.Type { return types.Dict }
	blockType       = func(t types.Type) types.Type { return types.Block }
	dictArrayType   = func(t types.Type) types.Type { return types.Array(types.Dict) }
)

var builtinFunctions map[types.Type]map[string]compileHandler

func init() {
	builtinFunctions = map[types.Type]map[string]compileHandler{
		types.String: {
			"contains":  {compile: compileStringContains, typ: boolType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.String}}},
			"find":      {typ: stringArrayType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.Regex}}},
			"length":    {typ: intType, signature: FunctionSignature{}},
			"camelcase": {typ: stringType, signature: FunctionSignature{}},
			"downcase":  {typ: stringType, signature: FunctionSignature{}},
			"upcase":    {typ: stringType, signature: FunctionSignature{}},
			"lines":     {typ: stringArrayType, signature: FunctionSignature{}},
			"split":     {typ: stringArrayType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.String}}},
			"trim":      {typ: stringType, signature: FunctionSignature{Required: 0, Args: []types.Type{types.String}}},
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
			"contains":  {compile: compileStringContains, typ: boolType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.String}}},
			"find":      {typ: stringArrayType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.Regex}}},
			"length":    {typ: intType, signature: FunctionSignature{}},
			"camelcase": {typ: stringType, signature: FunctionSignature{}},
			"downcase":  {typ: stringType, signature: FunctionSignature{}},
			"upcase":    {typ: stringType, signature: FunctionSignature{}},
			"lines":     {typ: stringArrayType, signature: FunctionSignature{}},
			"split":     {typ: stringArrayType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.String}}},
			"trim":      {typ: stringType, signature: FunctionSignature{Required: 0, Args: []types.Type{types.String}}},
			// array-ish
			"where": {compile: compileWhere, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"all":   {compile: compileArrayAll, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"any":   {compile: compileArrayAny, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"one":   {compile: compileArrayOne, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"none":  {compile: compileArrayNone, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"map":   {compile: compileArrayMap, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			// map-ish
			"keys":   {typ: stringArrayType, signature: FunctionSignature{}},
			"values": {typ: dictArrayType, signature: FunctionSignature{}},
		},
		types.ArrayLike: {
			"[]":           {typ: childType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.Int}}},
			"first":        {typ: childType, signature: FunctionSignature{}},
			"last":         {typ: childType, signature: FunctionSignature{}},
			"{}":           {typ: arrayBlockType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"length":       {typ: intType, signature: FunctionSignature{}},
			"where":        {compile: compileWhere, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"duplicates":   {compile: compileArrayDuplicates, signature: FunctionSignature{Required: 0, Args: []types.Type{types.String}}},
			"unique":       {compile: compileArrayUnique, signature: FunctionSignature{Required: 0}},
			"contains":     {compile: compileArrayContains, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"containsOnly": {compile: compileArrayContainsOnly, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"containsNone": {compile: compileArrayContainsNone, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"all":          {compile: compileArrayAll, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"any":          {compile: compileArrayAny, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"one":          {compile: compileArrayOne, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"none":         {compile: compileArrayNone, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"map":          {compile: compileArrayMap, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
		},
		types.MapLike: {
			"[]":     {typ: childType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.String}}},
			"{}":     {typ: blockType, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
			"length": {typ: intType, signature: FunctionSignature{}},
			"where":  {compile: compileMapWhere, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
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
			"map":      {compile: compileResourceMap, signature: FunctionSignature{Required: 1, Args: []types.Type{types.FunctionLike}}},
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

// Compile calls to builtin type handlers, that aren't mapped via builtin
// functions above. Typically only used if we need to go deeper into the given
// type to figure out what to do. For example: list resources are just
// resource types, so we can't tell if there are builtin functions without
// detecting that we are looking at a list resource.
func (c *compiler) compileImplicitBuiltin(typ types.Type, id string) (*compileHandler, *variable, error) {
	if !typ.IsResource() {
		return nil, nil, nil
	}

	r := typ.ResourceName()
	resource := c.Schema.Resources[r]
	if resource == nil || resource.ListType == "" {
		return nil, nil, nil
	}

	ch, ok := builtinFunctions[types.ArrayLike][id]
	if !ok {
		return nil, nil, nil
	}

	resType := types.Array(types.Type(resource.ListType))
	c.addChunk(&llx.Chunk{
		Call: llx.Chunk_FUNCTION,
		Id:   "list",
		Function: &llx.Function{
			Type:    string(resType),
			Binding: c.tailRef(),
		},
	})
	return &ch, &variable{
		typ: resType,
		ref: c.tailRef(),
	}, nil
}

func publicFieldsInfo(c *compiler, resourceInfo *resources.ResourceInfo) map[string]llx.Documentation {
	res := map[string]llx.Documentation{}
	for k, v := range resourceInfo.Fields {
		if v.IsPrivate {
			continue
		}
		if v.IsEmbedded && !c.UseAssetContext {
			continue
		}

		if v.IsEmbedded && c.UseAssetContext {
			name := types.Type(v.Type).ResourceName()
			child, ok := c.Schema.Resources[name]
			if !ok {
				continue
			}
			childFields := publicFieldsInfo(c, child)
			for k, v := range childFields {
				res[k] = v
			}
			continue
		}

		if v.IsImplicitResource {
			name := types.Type(v.Type).ResourceName()
			child := c.Schema.Resources[name]
			if !child.HasEmptyInit() {
				continue
			}

			// implicit resources don't have their own metadata, so we grab it from
			// the resource itself
			res[k] = llx.Documentation{
				Field: k,
				Title: child.Title,
				Desc:  child.Desc,
			}
			continue
		}

		res[k] = llx.Documentation{
			Field: k,
			Title: v.Title,
			Desc:  v.Desc,
		}
	}

	return res
}

// Glob {*} all fields for a given type. Note, that this descends into
// list elements of array resources if permitted.
func availableGlobFields(c *compiler, typ types.Type, descend bool) map[string]llx.Documentation {
	var res map[string]llx.Documentation

	if !typ.IsResource() {
		return res
	}

	resourceInfo := c.Schema.Resources[typ.ResourceName()]
	if descend && resourceInfo.ListType != "" {
		base := types.Type(resourceInfo.ListType).ResourceName()
		if info, ok := c.Schema.Resources[base]; ok {
			resourceInfo = info
		}
	}

	return publicFieldsInfo(c, resourceInfo)
}

func availableFields(c *compiler, typ types.Type) map[string]llx.Documentation {
	var res map[string]llx.Documentation

	// resources maintain their own fields and may be list resources
	if typ.IsResource() {
		resourceInfo := c.Schema.Resources[typ.ResourceName()]
		res = publicFieldsInfo(c, resourceInfo)

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
