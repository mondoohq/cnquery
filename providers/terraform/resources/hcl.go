// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/typeexpr"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/rs/zerolog/log"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/terraform/connection"
	"go.mondoo.com/cnquery/types"
)

func (t *mqlTerraform) id() (string, error) {
	return "terraform", nil
}

func (t *mqlTerraform) files() ([]interface{}, error) {
	conn := t.MqlRuntime.Connection.(*connection.Connection)

	var mqlTerraformFiles []interface{}
	files := conn.Parser().Files()
	for path := range files {
		mqlTerraformFile, err := CreateResource(t.MqlRuntime, "terraform.file", map[string]*llx.RawData{
			"path": llx.StringData(path),
		})
		if err != nil {
			return nil, err
		}
		mqlTerraformFiles = append(mqlTerraformFiles, mqlTerraformFile)
	}

	return mqlTerraformFiles, nil
}

func (t *mqlTerraform) tfvars() (interface{}, error) {
	conn := t.MqlRuntime.Connection.(*connection.Connection)
	return hclAttributesToDict(conn.TfVars())
}

func (t *mqlTerraform) modules() ([]interface{}, error) {
	conn := t.MqlRuntime.Connection.(*connection.Connection)

	manifest := conn.ModulesManifest()
	if manifest == nil {
		return nil, nil
	}

	var mqlModules []interface{}
	for i := range manifest.Records {
		record := manifest.Records[i]

		r, err := CreateResource(t.MqlRuntime, "terraform.module", map[string]*llx.RawData{
			"key":     llx.StringData(record.Key),
			"source":  llx.StringData(record.SourceAddr),
			"version": llx.StringData(record.Version),
			"dir":     llx.StringData(record.Dir),
		})
		if err != nil {
			return nil, err
		}
		mqlModules = append(mqlModules, r)
	}

	return mqlModules, nil
}

func (t *mqlTerraform) blocks() ([]interface{}, error) {
	conn := t.MqlRuntime.Connection.(*connection.Connection)
	files := conn.Parser().Files()

	var mqlHclBlocks []interface{}
	for k := range files {
		f := files[k]
		blocks, err := listHclBlocks(t.MqlRuntime, f.Body, f)
		if err != nil {
			return nil, err
		}
		mqlHclBlocks = append(mqlHclBlocks, blocks...)
	}
	return mqlHclBlocks, nil
}

func filterBlockByType(runtime *plugin.Runtime, filterType string) ([]interface{}, error) {
	conn := runtime.Connection.(*connection.Connection)
	files := conn.Parser().Files()

	var mqlHclBlocks []interface{}
	for k := range files {
		f := files[k]
		blocks, err := listHclBlocks(runtime, f.Body, f)
		if err != nil {
			return nil, err
		}

		for i := range blocks {
			b := blocks[i].(*mqlTerraformBlock)
			blockType := b.Type.Data
			if blockType == filterType {
				mqlHclBlocks = append(mqlHclBlocks, b)
			}
		}
	}
	return mqlHclBlocks, nil
}

func (t *mqlTerraform) providers() ([]interface{}, error) {
	return filterBlockByType(t.MqlRuntime, "provider")
}

func (t *mqlTerraform) datasources() ([]interface{}, error) {
	return filterBlockByType(t.MqlRuntime, "data")
}

func (t *mqlTerraform) resources() ([]interface{}, error) {
	return filterBlockByType(t.MqlRuntime, "resource")
}

func (t *mqlTerraform) variables() ([]interface{}, error) {
	return filterBlockByType(t.MqlRuntime, "variable")
}

func (t *mqlTerraform) outputs() ([]interface{}, error) {
	return filterBlockByType(t.MqlRuntime, "output")
}

func extractHclCodeSnippet(file *hcl.File, fileRange hcl.Range) string {
	if file == nil {
		return ""
	}

	lines := append([]string{""}, strings.Split(string(file.Bytes), "\n")...)

	// determine few surrounding lines
	start := fileRange.Start.Line - 3
	if start <= 0 {
		start = 1
	}
	end := fileRange.End.Line + 3
	if end >= len(lines) {
		end = len(lines) - 1
	}

	// build the snippet
	sb := strings.Builder{}
	for lineNo := start; lineNo <= end; lineNo++ {
		sb.WriteString(fmt.Sprintf("% 6d | ", lineNo))
		sb.WriteString(fmt.Sprintf("%s", lines[lineNo]))
		sb.WriteString("\n")
	}

	return sb.String()
}

func newMqlHclBlock(runtime *plugin.Runtime, block *hcl.Block, file *hcl.File) (plugin.Resource, error) {
	start, end, err := newFilePosRange(runtime, block.TypeRange)
	if err != nil {
		return nil, err
	}

	snippet := extractHclCodeSnippet(file, block.TypeRange)

	res, err := CreateResource(runtime, "terraform.block", map[string]*llx.RawData{
		"type":    llx.StringData(block.Type),
		"labels":  llx.ArrayData(llx.TArr2Raw(block.Labels), types.String),
		"start":   llx.ResourceData(start, "terraform.fileposition"),
		"end":     llx.ResourceData(end, "terraform.fileposition"),
		"snippet": llx.StringData(snippet),
	})
	if err != nil {
		return nil, err
	}
	r := res.(*mqlTerraformBlock)
	r.block = plugin.TValue[*hcl.Block]{State: plugin.StateIsSet, Data: block}
	r.cachedFile = plugin.TValue[*hcl.File]{State: plugin.StateIsSet, Data: file}
	return r, err
}

type mqlTerraformBlockInternal struct {
	block      plugin.TValue[*hcl.Block]
	cachedFile plugin.TValue[*hcl.File]
}

func (t *mqlTerraformBlock) id() (string, error) {
	// NOTE: a hcl block is identified by its filename and position
	fp := t.Start

	file := fp.Data.Path.Data
	line := fp.Data.Line.Data
	column := fp.Data.Column.Data

	return "terraform.block/" + file + "/" + strconv.FormatInt(line, 10) + "/" + strconv.FormatInt(column, 10), nil
}

func (t *mqlTerraformBlock) nameLabel() (string, error) {
	labels := t.Labels.Data

	// labels are string
	if len(labels) == 0 {
		return "", nil
	}

	return labels[0].(string), nil
}

func (t *mqlTerraformBlock) attributes() (map[string]interface{}, error) {
	var hclBlock *hcl.Block
	if t.block.State == plugin.StateIsSet {
		hclBlock = t.block.Data
	} else {
		if t.block.Error != nil {
			return nil, t.block.Error
		}
		return nil, errors.New("cannot get hcl block")
	}

	// do not handle diag information here, it also throws errors for blocks nearby
	attributes, _ := hclBlock.Body.JustAttributes()
	return hclAttributesToDict(attributes)
}

func (t *mqlTerraformBlock) arguments() (map[string]interface{}, error) {
	var hclBlock *hcl.Block
	if t.block.State == plugin.StateIsSet {
		hclBlock = t.block.Data
	} else {
		if t.block.Error != nil {
			return nil, t.block.Error
		}
		return nil, errors.New("cannot get hcl block")
	}

	// do not handle diag information here, it also throws errors for blocks nearby
	attributes, _ := hclBlock.Body.JustAttributes()
	return hclResolvedAttributesToDict(attributes)
}

func hclResolvedAttributesToDict(attributes map[string]*hcl.Attribute) (map[string]interface{}, error) {
	dict := map[string]interface{}{}
	for k := range attributes {
		dict[k] = getCtyValue(attributes[k].Expr, &hcl.EvalContext{
			Functions: hclFunctions(),
		})
	}
	return dict, nil
}

func hclAttributesToDict(attributes map[string]*hcl.Attribute) (map[string]interface{}, error) {
	dict := map[string]interface{}{}
	for k := range attributes {
		val, _ := attributes[k].Expr.Value(nil)
		dict[k] = map[string]interface{}{
			"value": getCtyValue(attributes[k].Expr, &hcl.EvalContext{
				Functions: hclFunctions(),
			}),
			"type": typeexpr.TypeString(val.Type()),
		}
	}

	return dict, nil
}

func hclFunctions() map[string]function.Function {
	return map[string]function.Function{
		"jsondecode": stdlib.JSONDecodeFunc,
		"jsonencode": stdlib.JSONEncodeFunc,
	}
}

func getCtyValue(expr hcl.Expression, ctx *hcl.EvalContext) interface{} {
	switch t := expr.(type) {
	case *hclsyntax.TupleConsExpr:
		results := []interface{}{}
		for _, expr := range t.Exprs {
			res := getCtyValue(expr, ctx)
			switch v := res.(type) {
			case []interface{}:
				results = append(results, v...)
			default:
				results = append(results, v)
			}
		}
		return results
	case *hclsyntax.ScopeTraversalExpr:
		traversal := t.Variables()
		res := []string{}
		for i := range traversal {
			tr := traversal[i]
			for j := range tr {
				switch v := tr[j].(type) {
				case hcl.TraverseRoot:
					res = append(res, v.Name)
				case hcl.TraverseAttr:
					res = append(res, v.Name)
				}
			}
		}
		// TODO: are we sure we want to do this?
		return strings.Join(res, ".")
	case *hclsyntax.FunctionCallExpr:
		results := []interface{}{}
		subVal, err := t.Value(ctx)
		if err == nil && subVal.Type() == cty.String {
			if t.Name == "jsonencode" {
				res := map[string]interface{}{}
				err := json.Unmarshal([]byte(subVal.AsString()), &res)
				if err == nil {
					results = append(results, res)
				}
			} else {
				results = append(results, subVal.AsString())
			}
		}
		return results
	case *hclsyntax.ConditionalExpr:
		results := []interface{}{}
		subVal, err := t.Value(ctx)
		if err == nil && subVal.Type() == cty.String {
			results = append(results, subVal.AsString())
		}
		return results
	case *hclsyntax.LiteralValueExpr:
		switch t.Val.Type() {
		case cty.String:
			return t.Val.AsString()
		case cty.Bool:
			return t.Val.True()
		case cty.Number:
			f, _ := t.Val.AsBigFloat().Float64()
			return f
		default:
			log.Warn().Msgf("unknown literal type %s", t.Val.Type().GoString())
			return nil
		}
	case *hclsyntax.TemplateExpr:
		// walk the parts of the expression to ensure that it has a literal value

		if len(t.Parts) == 1 {
			return getCtyValue(t.Parts[0], ctx)
		}

		results := []interface{}{}
		for _, p := range t.Parts {
			res := getCtyValue(p, ctx)
			switch v := res.(type) {
			case []interface{}:
				results = append(results, v...)
			default:
				results = append(results, v)
			}
		}
		return results
	case *hclsyntax.TemplateWrapExpr:
		results := []interface{}{}
		res := getCtyValue(t.Wrapped, ctx)
		switch v := res.(type) {
		case []interface{}:
			results = append(results, v...)
		default:
			results = append(results, v)
		}
		return results
	case *hclsyntax.ObjectConsExpr:
		result := map[string]interface{}{}
		for _, o := range t.Items {
			key := getCtyValue(o.KeyExpr, ctx)
			value := getCtyValue(o.ValueExpr, ctx)
			keyString := GetKeyString(key)
			result[keyString] = value
		}
		return result
	case *hclsyntax.ObjectConsKeyExpr:
		res := getCtyValue(t.Wrapped, ctx)
		return res
	case *hclsyntax.ParenthesesExpr:
		v := getCtyValue(t.Expression, ctx)
		return v
	default:
		log.Warn().Msgf("unknown type %T", t)
		return nil
	}
}

func GetKeyString(key interface{}) string {
	switch v := key.(type) {
	case []string:
		return strings.Join(v, ",")
	case []interface{}:
		s := ""
		for i := range v {
			s = s + v[i].(string)
		}
		return s
	default:
		return key.(string)
	}
}

func (g *mqlTerraformBlock) blocks() ([]interface{}, error) {
	var hclBlock *hcl.Block
	if g.block.State == plugin.StateIsSet {
		hclBlock = g.block.Data
	} else {
		if g.block.Error != nil {
			return nil, g.block.Error
		}
		return nil, errors.New("cannot get hcl block")
	}

	var hclFile *hcl.File
	if g.cachedFile.State == plugin.StateIsSet {
		hclFile = g.cachedFile.Data
	}

	if hclFile == nil {
		return nil, errors.New("cannot get hcl file")
	}

	return listHclBlocks(g.MqlRuntime, hclBlock.Body, hclFile)
}

func listHclBlocks(runtime *plugin.Runtime, rawBody interface{}, file *hcl.File) ([]interface{}, error) {
	var mqlHclBlocks []interface{}

	switch body := rawBody.(type) {
	case *hclsyntax.Body:
		for i := range body.Blocks {
			mqlBlock, err := newMqlHclBlock(runtime, body.Blocks[i].AsHCLBlock(), file)
			if err != nil {
				return nil, err
			}
			mqlHclBlocks = append(mqlHclBlocks, mqlBlock)
		}
	case hcl.Body:
		content, _, _ := body.PartialContent(connection.TerraformSchema_0_12)
		for i := range content.Blocks {
			mqlBlock, err := newMqlHclBlock(runtime, content.Blocks[i], file)
			if err != nil {
				return nil, err
			}
			mqlHclBlocks = append(mqlHclBlocks, mqlBlock)
		}
	default:
		return nil, errors.New("unsupported hcl block type")
	}

	return mqlHclBlocks, nil
}

func newFilePosRange(runtime *plugin.Runtime, r hcl.Range) (plugin.Resource, plugin.Resource, error) {
	start, err := CreateResource(runtime, "terraform.fileposition", map[string]*llx.RawData{
		"path":   llx.StringData(r.Filename),
		"line":   llx.IntData(int64(r.Start.Line)),
		"column": llx.IntData(int64(r.Start.Column)),
		"byte":   llx.IntData(int64(r.Start.Byte)),
	})
	if err != nil {
		return nil, nil, err
	}

	end, err := CreateResource(runtime, "terraform.fileposition", map[string]*llx.RawData{
		"path":   llx.StringData(r.Filename),
		"line":   llx.IntData(int64(r.End.Line)),
		"column": llx.IntData(int64(r.End.Column)),
		"byte":   llx.IntData(int64(r.End.Byte)),
	})
	if err != nil {
		return nil, nil, err
	}

	return start, end, nil
}

func (t *mqlTerraformFileposition) id() (string, error) {
	path := t.Path.Data
	line := t.Line.Data
	column := t.Column.Data
	return "file.position/" + path + "/" + strconv.FormatInt(line, 10) + "/" + strconv.FormatInt(column, 10), nil
}

func (t *mqlTerraformFile) id() (string, error) {
	p := t.Path.Data
	return "terraform.file/" + p, nil
}

func (t *mqlTerraformFile) blocks() ([]interface{}, error) {
	conn := t.MqlRuntime.Connection.(*connection.Connection)
	p := t.Path.Data

	files := conn.Parser().Files()
	file := files[p]
	return listHclBlocks(t.MqlRuntime, file.Body, file)
}

func (t *mqlTerraformModule) id() (string, error) {
	// FIXME: Do we need to check .Error first?
	k := t.Key.Data
	v := t.Version.Data
	return "terraform.module/key/" + k + "/version/" + v, nil
}

func (t *mqlTerraformModule) block() (*mqlTerraformBlock, error) {
	key := t.Key.Data
	conn := t.MqlRuntime.Connection.(*connection.Connection)
	files := conn.Parser().Files()

	var mqlHclBlock *mqlTerraformBlock
	for k := range files {
		f := files[k]
		blocks, err := listHclBlocks(t.MqlRuntime, f.Body, f)
		if err != nil {
			return nil, err
		}

		for i := range blocks {
			b := blocks[i].(*mqlTerraformBlock)
			blockType := b.Type.Data
			namedlabel := b.NameLabel.Data

			if blockType == "module" && namedlabel == key {
				mqlHclBlock = b
			}
		}
	}

	return mqlHclBlock, nil
}

func initTerraformSettings(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	blocks, err := filterBlockByType(runtime, "terraform")
	if err != nil {
		return nil, nil, err
	}

	if len(blocks) != 1 {
		// no terraform settings block found, this is ok for terraform and not an error
		args["block"] = nil
		args["requiredProviders"] = llx.DictData(map[string]interface{}{})
		return args, nil, nil
	}

	settingsBlock := blocks[0].(*mqlTerraformBlock)
	args["block"] = llx.ResourceData(settingsBlock, "terraform.block")
	args["requiredProviders"] = llx.DictData(map[string]interface{}{})

	if settingsBlock.block.State == plugin.StateIsSet {
		hb := settingsBlock.block.Data
		requireProviderBlock := getBlockByName(hb, "required_providers")
		if requireProviderBlock != nil {
			attributes, _ := requireProviderBlock.Body.JustAttributes()
			dict, err := hclResolvedAttributesToDict(attributes)
			if err != nil {
				return nil, nil, err
			}
			args["requiredProviders"] = llx.DictData(dict)
		}
	}

	return args, nil, nil
}

func getBlockByName(hb *hcl.Block, name string) *hcl.Block {
	rawBody := hb.Body
	switch body := rawBody.(type) {
	case *hclsyntax.Body:
		for i := range body.Blocks {
			b := body.Blocks[i].AsHCLBlock()
			if b.Type == name {
				return b
			}
		}
	case hcl.Body:
		content, _, _ := body.PartialContent(connection.TerraformSchema_0_12)
		for i := range content.Blocks {
			b := content.Blocks[i]
			if b.Type == name {
				return b
			}
		}
	}
	return nil
}
