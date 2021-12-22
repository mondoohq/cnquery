package resources

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/typeexpr"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/rs/zerolog/log"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/terraform"
)

func terraformtransport(t transports.Transport) (*terraform.Transport, error) {
	gt, ok := t.(*terraform.Transport)
	if !ok {
		return nil, errors.New("terraform resource is not supported on this transport")
	}
	return gt, nil
}

func (g *lumiTerraform) id() (string, error) {
	return "terraform", nil
}

func (g *lumiTerraform) GetFiles() ([]interface{}, error) {
	t, err := terraformtransport(g.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	var lumiTerraformFiles []interface{}
	files := t.Parser().Files()
	for path := range files {
		lumiTerraformFile, err := g.Runtime.CreateResource("terraform.file",
			"path", path,
		)
		if err != nil {
			return nil, err
		}
		lumiTerraformFiles = append(lumiTerraformFiles, lumiTerraformFile)
	}

	return lumiTerraformFiles, nil
}

func (g *lumiTerraform) GetTfvars() (interface{}, error) {
	t, err := terraformtransport(g.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	return hclAttributesToDict(t.TfVars())
}

func (g *lumiTerraform) GetModules() ([]interface{}, error) {
	t, err := terraformtransport(g.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	manifest := t.ModulesManifest()
	if manifest == nil {
		return nil, nil
	}

	var lumiModules []interface{}
	for i := range manifest.Records {
		record := manifest.Records[i]

		r, err := g.Runtime.CreateResource("terraform.module",
			"key", record.Key,
			"source", record.SourceAddr,
			"version", record.Version,
			"dir", record.Dir,
		)
		if err != nil {
			return nil, err
		}
		lumiModules = append(lumiModules, r)
	}

	return lumiModules, nil
}

func (g *lumiTerraform) GetBlocks() ([]interface{}, error) {
	t, err := terraformtransport(g.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	files := t.Parser().Files()

	var lumiHclBlocks []interface{}
	for k := range files {
		f := files[k]
		blocks, err := listHclBlocks(g.Runtime, f.Body, f)
		if err != nil {
			return nil, err
		}
		lumiHclBlocks = append(lumiHclBlocks, blocks...)
	}
	return lumiHclBlocks, nil
}

func (g *lumiTerraform) filterBlockByType(filterType string) ([]interface{}, error) {
	t, err := terraformtransport(g.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	files := t.Parser().Files()

	var lumiHclBlocks []interface{}
	for k := range files {
		f := files[k]
		blocks, err := listHclBlocks(g.Runtime, f.Body, f)
		if err != nil {
			return nil, err
		}

		for i := range blocks {
			b := blocks[i].(TerraformBlock)
			blockType, err := b.Type()
			if err != nil {
				return nil, err
			}
			if blockType == filterType {
				lumiHclBlocks = append(lumiHclBlocks, b)
			}
		}
	}
	return lumiHclBlocks, nil
}

func (g *lumiTerraform) GetProviders() ([]interface{}, error) {
	return g.filterBlockByType("provider")
}

func (g *lumiTerraform) GetDatasources() ([]interface{}, error) {
	return g.filterBlockByType("data")
}

func (g *lumiTerraform) GetResources() ([]interface{}, error) {
	return g.filterBlockByType("resource")
}

func (g *lumiTerraform) GetVariables() ([]interface{}, error) {
	return g.filterBlockByType("variable")
}

func (g *lumiTerraform) GetOutputs() ([]interface{}, error) {
	return g.filterBlockByType("output")
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

func newLumiHclBlock(runtime *lumi.Runtime, block *hcl.Block, file *hcl.File) (lumi.ResourceType, error) {
	start, end, err := newFilePosRange(runtime, block.TypeRange)
	if err != nil {
		return nil, err
	}

	snippet := extractHclCodeSnippet(file, block.TypeRange)

	r, err := runtime.CreateResource("terraform.block",
		"type", block.Type,
		"labels", sliceInterface(block.Labels),
		"start", start,
		"end", end,
		"snippet", snippet,
	)

	if err == nil {
		r.LumiResource().Cache.Store("_hclblock", &lumi.CacheEntry{
			Data: block,
		})
		r.LumiResource().Cache.Store("_hclfile", &lumi.CacheEntry{
			Data: file,
		})
	}

	return r, err
}

func (g *lumiTerraformBlock) id() (string, error) {
	// NOTE: a hcl block is identified by its filename and position
	fp, err := g.Start()
	if err != nil {
		return "", err
	}
	file, _ := fp.Path()
	line, _ := fp.Line()
	column, _ := fp.Column()

	return "terraform.block/" + file + "/" + strconv.FormatInt(line, 10) + "/" + strconv.FormatInt(column, 10), nil
}

func (g *lumiTerraformBlock) GetNameLabel() (interface{}, error) {
	labels, err := g.Labels()
	if err != nil {
		return nil, err
	}

	// labels are string
	if len(labels) == 0 {
		return "", nil
	}

	return labels[0].(string), nil
}

func (g *lumiTerraformBlock) GetAttributes() (map[string]interface{}, error) {
	ce, ok := g.LumiResource().Cache.Load("_hclblock")
	if !ok {
		return nil, nil
	}

	hclBlock := ce.Data.(*hcl.Block)

	// do not handle diag information here, it also throws errors for blocks nearby
	attributes, _ := hclBlock.Body.JustAttributes()
	return hclAttributesToDict(attributes)
}

func (g *lumiTerraformBlock) GetArguments() (map[string]interface{}, error) {
	ce, ok := g.LumiResource().Cache.Load("_hclblock")
	if !ok {
		return nil, nil
	}

	hclBlock := ce.Data.(*hcl.Block)

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
			keyString := key.(string)
			result[keyString] = value
		}
		return result
	case *hclsyntax.ObjectConsKeyExpr:
		res := getCtyValue(t.Wrapped, ctx)
		return res
	default:
		log.Warn().Msgf("unknown type %T", t)
		return nil
	}
	return nil
}

func (g *lumiTerraformBlock) GetBlocks() ([]interface{}, error) {
	ce, ok := g.LumiResource().Cache.Load("_hclblock")
	if !ok {
		return nil, nil
	}
	hclBlock := ce.Data.(*hcl.Block)

	hFile, ok := g.LumiResource().Cache.Load("_hclfile")
	if !ok {
		return nil, nil
	}
	hclFile := hFile.Data.(*hcl.File)

	return listHclBlocks(g.Runtime, hclBlock.Body, hclFile)
}

func listHclBlocks(runtime *lumi.Runtime, rawBody interface{}, file *hcl.File) ([]interface{}, error) {
	var lumiHclBlocks []interface{}

	switch body := rawBody.(type) {
	case *hclsyntax.Body:
		for i := range body.Blocks {
			lumiBlock, err := newLumiHclBlock(runtime, body.Blocks[i].AsHCLBlock(), file)
			if err != nil {
				return nil, err
			}
			lumiHclBlocks = append(lumiHclBlocks, lumiBlock)
		}
	case hcl.Body:
		content, _, _ := body.PartialContent(terraform.TerraformSchema_0_12)
		for i := range content.Blocks {
			lumiBlock, err := newLumiHclBlock(runtime, content.Blocks[i], file)
			if err != nil {
				return nil, err
			}
			lumiHclBlocks = append(lumiHclBlocks, lumiBlock)
		}
	default:
		return nil, errors.New("unsupported hcl block type")
	}

	return lumiHclBlocks, nil
}

func newFilePosRange(runtime *lumi.Runtime, r hcl.Range) (lumi.ResourceType, lumi.ResourceType, error) {
	start, err := runtime.CreateResource("terraform.fileposition",
		"path", r.Filename,
		"line", int64(r.Start.Line),
		"column", int64(r.Start.Column),
		"byte", int64(r.Start.Byte),
	)
	if err != nil {
		return nil, nil, err
	}

	end, err := runtime.CreateResource("terraform.fileposition",
		"path", r.Filename,
		"line", int64(r.Start.Line),
		"column", int64(r.Start.Column),
		"byte", int64(r.Start.Byte),
	)
	if err != nil {
		return nil, nil, err
	}

	return start, end, nil
}

func (p *lumiTerraformFileposition) id() (string, error) {
	path, _ := p.Path()
	line, _ := p.Line()
	column, _ := p.Column()
	return "file.position/" + path + "/" + strconv.FormatInt(line, 10) + "/" + strconv.FormatInt(column, 10), nil
}

func (g *lumiTerraformFile) id() (string, error) {
	p, err := g.Path()
	if err != nil {
		return "", err
	}
	return "terraform.file/" + p, nil
}

func (g *lumiTerraformFile) GetBlocks() ([]interface{}, error) {
	t, err := terraformtransport(g.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	p, err := g.Path()
	if err != nil {
		return nil, err
	}

	files := t.Parser().Files()
	file := files[p]
	return listHclBlocks(g.Runtime, file.Body, file)
}

func (g *lumiTerraformModule) id() (string, error) {
	k, _ := g.Key()
	v, _ := g.Version()
	return "terraform.module/key/" + k + "/version/" + v, nil
}
