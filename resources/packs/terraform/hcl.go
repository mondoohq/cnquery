package terraform

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
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/terraform"
	"go.mondoo.io/mondoo/resources"
	"go.mondoo.io/mondoo/resources/packs/core"
)

func terraformtransport(t providers.Instance) (*terraform.Provider, error) {
	gt, ok := t.(*terraform.Provider)
	if !ok {
		return nil, errors.New("terraform resource is not supported on this transport")
	}
	return gt, nil
}

func (g *mqlTerraform) id() (string, error) {
	return "terraform", nil
}

func (g *mqlTerraform) GetFiles() ([]interface{}, error) {
	t, err := terraformtransport(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	var mqlTerraformFiles []interface{}
	files := t.Parser().Files()
	for path := range files {
		mqlTerraformFile, err := g.MotorRuntime.CreateResource("terraform.file",
			"path", path,
		)
		if err != nil {
			return nil, err
		}
		mqlTerraformFiles = append(mqlTerraformFiles, mqlTerraformFile)
	}

	return mqlTerraformFiles, nil
}

func (g *mqlTerraform) GetTfvars() (interface{}, error) {
	t, err := terraformtransport(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	return hclAttributesToDict(t.TfVars())
}

func (g *mqlTerraform) GetModules() ([]interface{}, error) {
	t, err := terraformtransport(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	manifest := t.ModulesManifest()
	if manifest == nil {
		return nil, nil
	}

	var mqlModules []interface{}
	for i := range manifest.Records {
		record := manifest.Records[i]

		r, err := g.MotorRuntime.CreateResource("terraform.module",
			"key", record.Key,
			"source", record.SourceAddr,
			"version", record.Version,
			"dir", record.Dir,
		)
		if err != nil {
			return nil, err
		}
		mqlModules = append(mqlModules, r)
	}

	return mqlModules, nil
}

func (g *mqlTerraform) GetBlocks() ([]interface{}, error) {
	t, err := terraformtransport(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	files := t.Parser().Files()

	var mqlHclBlocks []interface{}
	for k := range files {
		f := files[k]
		blocks, err := listHclBlocks(g.MotorRuntime, f.Body, f)
		if err != nil {
			return nil, err
		}
		mqlHclBlocks = append(mqlHclBlocks, blocks...)
	}
	return mqlHclBlocks, nil
}

func filterBlockByType(runtime *resources.Runtime, filterType string) ([]interface{}, error) {
	t, err := terraformtransport(runtime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	files := t.Parser().Files()

	var mqlHclBlocks []interface{}
	for k := range files {
		f := files[k]
		blocks, err := listHclBlocks(runtime, f.Body, f)
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
				mqlHclBlocks = append(mqlHclBlocks, b)
			}
		}
	}
	return mqlHclBlocks, nil
}

func (g *mqlTerraform) GetProviders() ([]interface{}, error) {
	return filterBlockByType(g.MotorRuntime, "provider")
}

func (g *mqlTerraform) GetDatasources() ([]interface{}, error) {
	return filterBlockByType(g.MotorRuntime, "data")
}

func (g *mqlTerraform) GetResources() ([]interface{}, error) {
	return filterBlockByType(g.MotorRuntime, "resource")
}

func (g *mqlTerraform) GetVariables() ([]interface{}, error) {
	return filterBlockByType(g.MotorRuntime, "variable")
}

func (g *mqlTerraform) GetOutputs() ([]interface{}, error) {
	return filterBlockByType(g.MotorRuntime, "output")
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

func newMqlHclBlock(runtime *resources.Runtime, block *hcl.Block, file *hcl.File) (resources.ResourceType, error) {
	start, end, err := newFilePosRange(runtime, block.TypeRange)
	if err != nil {
		return nil, err
	}

	snippet := extractHclCodeSnippet(file, block.TypeRange)

	r, err := runtime.CreateResource("terraform.block",
		"type", block.Type,
		"labels", core.StrSliceToInterface(block.Labels),
		"start", start,
		"end", end,
		"snippet", snippet,
	)

	if err == nil {
		r.MqlResource().Cache.Store("_hclblock", &resources.CacheEntry{
			Data: block,
		})
		r.MqlResource().Cache.Store("_hclfile", &resources.CacheEntry{
			Data: file,
		})
	}

	return r, err
}

func (g *mqlTerraformBlock) id() (string, error) {
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

func (g *mqlTerraformBlock) GetNameLabel() (interface{}, error) {
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

func (g *mqlTerraformBlock) GetAttributes() (map[string]interface{}, error) {
	ce, ok := g.MqlResource().Cache.Load("_hclblock")
	if !ok {
		return nil, nil
	}

	hclBlock := ce.Data.(*hcl.Block)

	// do not handle diag information here, it also throws errors for blocks nearby
	attributes, _ := hclBlock.Body.JustAttributes()
	return hclAttributesToDict(attributes)
}

func (g *mqlTerraformBlock) GetArguments() (map[string]interface{}, error) {
	ce, ok := g.MqlResource().Cache.Load("_hclblock")
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

func (g *mqlTerraformBlock) GetBlocks() ([]interface{}, error) {
	ce, ok := g.MqlResource().Cache.Load("_hclblock")
	if !ok {
		return nil, nil
	}
	hclBlock := ce.Data.(*hcl.Block)

	hFile, ok := g.MqlResource().Cache.Load("_hclfile")
	if !ok {
		return nil, nil
	}
	hclFile := hFile.Data.(*hcl.File)

	return listHclBlocks(g.MotorRuntime, hclBlock.Body, hclFile)
}

func listHclBlocks(runtime *resources.Runtime, rawBody interface{}, file *hcl.File) ([]interface{}, error) {
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
		content, _, _ := body.PartialContent(terraform.TerraformSchema_0_12)
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

func newFilePosRange(runtime *resources.Runtime, r hcl.Range) (resources.ResourceType, resources.ResourceType, error) {
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

func (p *mqlTerraformFileposition) id() (string, error) {
	path, _ := p.Path()
	line, _ := p.Line()
	column, _ := p.Column()
	return "file.position/" + path + "/" + strconv.FormatInt(line, 10) + "/" + strconv.FormatInt(column, 10), nil
}

func (g *mqlTerraformFile) id() (string, error) {
	p, err := g.Path()
	if err != nil {
		return "", err
	}
	return "terraform.file/" + p, nil
}

func (g *mqlTerraformFile) GetBlocks() ([]interface{}, error) {
	t, err := terraformtransport(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	p, err := g.Path()
	if err != nil {
		return nil, err
	}

	files := t.Parser().Files()
	file := files[p]
	return listHclBlocks(g.MotorRuntime, file.Body, file)
}

func (g *mqlTerraformModule) id() (string, error) {
	k, _ := g.Key()
	v, _ := g.Version()
	return "terraform.module/key/" + k + "/version/" + v, nil
}

func (g *mqlTerraformSettings) id() (string, error) {
	return "terraform.settings", nil
}

func (s *mqlTerraformSettings) init(args *resources.Args) (*resources.Args, TerraformSettings, error) {
	blocks, err := filterBlockByType(s.MotorRuntime, "terraform")
	if err != nil {
		return nil, nil, err
	}

	if len(blocks) != 1 {
		return nil, nil, errors.New("no `terraform` settings block found")
	}

	settingsBlock := blocks[0].(TerraformBlock)
	(*args)["block"] = settingsBlock
	(*args)["requiredProviders"] = map[string]interface{}{}

	hclBlock, found := settingsBlock.MqlResource().Cache.Load("_hclblock")
	if found {
		hb := hclBlock.Data.(*hcl.Block)
		requireProviderBlock := getBlockByName(hb, "required_providers")
		if requireProviderBlock != nil {
			attributes, _ := requireProviderBlock.Body.JustAttributes()
			dict, err := hclResolvedAttributesToDict(attributes)
			if err != nil {
				return nil, nil, err
			}
			(*args)["requiredProviders"] = dict
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
		content, _, _ := body.PartialContent(terraform.TerraformSchema_0_12)
		for i := range content.Blocks {
			b := content.Blocks[i]
			if b.Type == name {
				return b
			}
		}
	}
	return nil
}
