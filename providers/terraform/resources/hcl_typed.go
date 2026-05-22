// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/terraform/connection"
	"go.mondoo.com/mql/v13/types"
)

// fileFor looks up the parsed *hcl.File for the given filename via the
// active connection's parser. Returns nil if the file isn't loaded.
func fileFor(runtime *plugin.Runtime, filename string) *hcl.File {
	if runtime == nil || filename == "" {
		return nil
	}
	conn, ok := runtime.Connection.(*connection.Connection)
	if !ok {
		return nil
	}
	parser := conn.Parser()
	if parser == nil {
		return nil
	}
	return parser.Files()[filename]
}

type mqlTerraformVariableInternal struct {
	hclBlock *hcl.Block
}

type mqlTerraformLocalInternal struct {
	expr hcl.Expression
}

type mqlTerraformOutputInternal struct {
	hclBlock *hcl.Block
}

type mqlTerraformProviderConfigInternal struct {
	hclBlock *hcl.Block
}

type mqlTerraformDataSourceInternal struct {
	hclBlock *hcl.Block
}

type mqlTerraformResourceInternal struct {
	hclBlock *hcl.Block
}

type mqlTerraformLifecycleInternal struct {
	hclBlock *hcl.Block
}

type mqlTerraformConditionInternal struct {
	hclBlock *hcl.Block
}

func positionID(b *mqlTerraformBlock) string {
	start := b.Start.Data
	if start == nil {
		return ""
	}
	return start.Path.Data + ":" + strconv.FormatInt(start.Line.Data, 10) + ":" + strconv.FormatInt(start.Column.Data, 10)
}

func (t *mqlTerraform) variableDecls() ([]any, error) {
	if err := t.refreshCache(nil); err != nil {
		return nil, err
	}
	out := make([]any, 0, len(t.Variables.Data))
	for _, raw := range t.Variables.Data {
		block, ok := raw.(*mqlTerraformBlock)
		if !ok {
			continue
		}
		labels := block.Labels.Data
		if len(labels) < 1 {
			continue
		}
		name, _ := labels[0].(string)
		typeSrc := readExprSource(block.block.Data, t.MqlRuntime, "type")

		res, err := CreateResource(t.MqlRuntime, "terraform.variable", map[string]*llx.RawData{
			"__id":  llx.StringData("variable." + name),
			"name":  llx.StringData(name),
			"type":  llx.StringData(typeSrc),
			"block": llx.ResourceData(block, "terraform.block"),
		})
		if err != nil {
			return nil, err
		}
		res.(*mqlTerraformVariable).hclBlock = block.block.Data
		out = append(out, res)
	}
	return out, nil
}

func (t *mqlTerraform) localDecls() ([]any, error) {
	if err := t.refreshCache(nil); err != nil {
		return nil, err
	}
	out := []any{}
	for _, block := range t.mqlTerraformInternal.localsBlocks {
		hb := block.block.Data
		attrs, _ := hb.Body.JustAttributes()
		keys := sortedAttrKeys(attrs)
		blockID := positionID(block)
		for _, name := range keys {
			attr := attrs[name]
			res, err := CreateResource(t.MqlRuntime, "terraform.local", map[string]*llx.RawData{
				"__id":  llx.StringData(blockID + "/" + name),
				"name":  llx.StringData(name),
				"block": llx.ResourceData(block, "terraform.block"),
			})
			if err != nil {
				return nil, err
			}
			res.(*mqlTerraformLocal).expr = attr.Expr
			out = append(out, res)
		}
	}
	return out, nil
}

func (t *mqlTerraform) outputDecls() ([]any, error) {
	if err := t.refreshCache(nil); err != nil {
		return nil, err
	}
	out := make([]any, 0, len(t.Outputs.Data))
	for _, raw := range t.Outputs.Data {
		block, ok := raw.(*mqlTerraformBlock)
		if !ok {
			continue
		}
		labels := block.Labels.Data
		if len(labels) < 1 {
			continue
		}
		name, _ := labels[0].(string)

		res, err := CreateResource(t.MqlRuntime, "terraform.output", map[string]*llx.RawData{
			"__id":  llx.StringData("output." + name),
			"name":  llx.StringData(name),
			"block": llx.ResourceData(block, "terraform.block"),
		})
		if err != nil {
			return nil, err
		}
		res.(*mqlTerraformOutput).hclBlock = block.block.Data
		out = append(out, res)
	}
	return out, nil
}

func (t *mqlTerraform) providerDecls() ([]any, error) {
	if err := t.refreshCache(nil); err != nil {
		return nil, err
	}
	out := make([]any, 0, len(t.Providers.Data))
	for _, raw := range t.Providers.Data {
		block, ok := raw.(*mqlTerraformBlock)
		if !ok {
			continue
		}
		labels := block.Labels.Data
		if len(labels) < 1 {
			continue
		}
		name, _ := labels[0].(string)

		res, err := CreateResource(t.MqlRuntime, "terraform.providerConfig", map[string]*llx.RawData{
			"__id":  llx.StringData("provider." + name + "@" + positionID(block)),
			"name":  llx.StringData(name),
			"block": llx.ResourceData(block, "terraform.block"),
		})
		if err != nil {
			return nil, err
		}
		res.(*mqlTerraformProviderConfig).hclBlock = block.block.Data
		out = append(out, res)
	}
	return out, nil
}

func (t *mqlTerraform) dataSourceDecls() ([]any, error) {
	if err := t.refreshCache(nil); err != nil {
		return nil, err
	}
	out := make([]any, 0, len(t.Datasources.Data))
	for _, raw := range t.Datasources.Data {
		block, ok := raw.(*mqlTerraformBlock)
		if !ok {
			continue
		}
		labels := block.Labels.Data
		if len(labels) < 2 {
			continue
		}
		typ, _ := labels[0].(string)
		name, _ := labels[1].(string)

		res, err := CreateResource(t.MqlRuntime, "terraform.dataSource", map[string]*llx.RawData{
			"__id":  llx.StringData("data." + typ + "." + name),
			"type":  llx.StringData(typ),
			"name":  llx.StringData(name),
			"block": llx.ResourceData(block, "terraform.block"),
		})
		if err != nil {
			return nil, err
		}
		res.(*mqlTerraformDataSource).hclBlock = block.block.Data
		out = append(out, res)
	}
	return out, nil
}

func (t *mqlTerraform) resourceDecls() ([]any, error) {
	if err := t.refreshCache(nil); err != nil {
		return nil, err
	}
	out := make([]any, 0, len(t.mqlTerraformInternal.resources))
	for _, block := range t.mqlTerraformInternal.resources {
		labels := block.Labels.Data
		if len(labels) < 2 {
			continue
		}
		typ, _ := labels[0].(string)
		name, _ := labels[1].(string)

		res, err := CreateResource(t.MqlRuntime, "terraform.resource", map[string]*llx.RawData{
			"__id":  llx.StringData("resource." + typ + "." + name),
			"type":  llx.StringData(typ),
			"name":  llx.StringData(name),
			"block": llx.ResourceData(block, "terraform.block"),
		})
		if err != nil {
			return nil, err
		}
		res.(*mqlTerraformResource).hclBlock = block.block.Data
		out = append(out, res)
	}
	return out, nil
}

// --- terraform.variable -----------------------------------------------------

func (v *mqlTerraformVariable) compute_default() (any, error) {
	attr := lookupAttr(v.hclBlock, "default")
	if attr == nil {
		v.Default.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return getCtyValue(attr.Expr, &hcl.EvalContext{Functions: hclFunctions()}), nil
}

func (v *mqlTerraformVariable) description() (string, error) {
	return readStringAttr(v.hclBlock, "description"), nil
}

func (v *mqlTerraformVariable) sensitive() (bool, error) {
	return readBoolAttr(v.hclBlock, "sensitive", false), nil
}

func (v *mqlTerraformVariable) nullable() (bool, error) {
	return readBoolAttr(v.hclBlock, "nullable", true), nil
}

// --- terraform.local --------------------------------------------------------

func (l *mqlTerraformLocal) value() (any, error) {
	if l.expr == nil {
		l.Value.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return getCtyValue(l.expr, &hcl.EvalContext{Functions: hclFunctions()}), nil
}

// --- terraform.output -------------------------------------------------------

func (o *mqlTerraformOutput) value() (any, error) {
	attr := lookupAttr(o.hclBlock, "value")
	if attr == nil {
		o.Value.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return getCtyValue(attr.Expr, &hcl.EvalContext{Functions: hclFunctions()}), nil
}

func (o *mqlTerraformOutput) description() (string, error) {
	return readStringAttr(o.hclBlock, "description"), nil
}

func (o *mqlTerraformOutput) sensitive() (bool, error) {
	return readBoolAttr(o.hclBlock, "sensitive", false), nil
}

// --- terraform.providerConfig ----------------------------------------------

func (p *mqlTerraformProviderConfig) alias() (string, error) {
	return readStringAttr(p.hclBlock, "alias"), nil
}

func (p *mqlTerraformProviderConfig) version() (string, error) {
	return readStringAttr(p.hclBlock, "version"), nil
}

func (p *mqlTerraformProviderConfig) config() (map[string]any, error) {
	return attributeDict(p.hclBlock, "alias", "version")
}

// --- terraform.dataSource ---------------------------------------------------

func (d *mqlTerraformDataSource) provider() (string, error) {
	return providerFromType(d.Type.Data), nil
}

func (d *mqlTerraformDataSource) arguments() (map[string]any, error) {
	return attributeDict(d.hclBlock)
}

// --- terraform.resource -----------------------------------------------------

var resourceMetaArgs = map[string]struct{}{
	"count":      {},
	"for_each":   {},
	"depends_on": {},
	"provider":   {},
}

func (r *mqlTerraformResource) provider() (string, error) {
	return providerFromType(r.Type.Data), nil
}

func (r *mqlTerraformResource) arguments() (map[string]any, error) {
	if r.hclBlock == nil {
		return map[string]any{}, nil
	}
	attrs, _ := r.hclBlock.Body.JustAttributes()
	dict := map[string]any{}
	for k, attr := range attrs {
		if _, isMeta := resourceMetaArgs[k]; isMeta {
			continue
		}
		dict[k] = getCtyValue(attr.Expr, &hcl.EvalContext{Functions: hclFunctions()})
	}
	return dict, nil
}

func (r *mqlTerraformResource) count() (any, error) {
	attr := lookupAttr(r.hclBlock, "count")
	if attr == nil {
		r.Count.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return getCtyValue(attr.Expr, &hcl.EvalContext{Functions: hclFunctions()}), nil
}

func (r *mqlTerraformResource) forEach() (any, error) {
	attr := lookupAttr(r.hclBlock, "for_each")
	if attr == nil {
		r.ForEach.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return getCtyValue(attr.Expr, &hcl.EvalContext{Functions: hclFunctions()}), nil
}

func (r *mqlTerraformResource) dependsOn() ([]any, error) {
	return tupleToRefStrings(lookupAttr(r.hclBlock, "depends_on")), nil
}

func (r *mqlTerraformResource) lifecycle() (*mqlTerraformLifecycle, error) {
	lc := findChildBlock(r.hclBlock, "lifecycle")
	if lc == nil {
		r.Lifecycle.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	wrapped, err := newMqlHclBlock(r.MqlRuntime, lc, nil)
	if err != nil {
		return nil, err
	}

	res, err := CreateResource(r.MqlRuntime, "terraform.lifecycle", map[string]*llx.RawData{
		"__id":                llx.StringData("lifecycle@" + hclRangeToID(lc.DefRange)),
		"createBeforeDestroy": llx.BoolData(readBoolAttr(lc, "create_before_destroy", false)),
		"preventDestroy":      llx.BoolData(readBoolAttr(lc, "prevent_destroy", false)),
		"ignoreChanges":       llx.ArrayData(parseIgnoreChanges(lc), types.String),
		"replaceTriggeredBy":  llx.ArrayData(tupleToRefStrings(lookupAttr(lc, "replace_triggered_by")), types.String),
		"block":               llx.ResourceData(wrapped, "terraform.block"),
	})
	if err != nil {
		return nil, err
	}
	res.(*mqlTerraformLifecycle).hclBlock = lc
	return res.(*mqlTerraformLifecycle), nil
}

// --- terraform.lifecycle ----------------------------------------------------

func (l *mqlTerraformLifecycle) preconditions() ([]any, error) {
	return l.conditionBlocks("precondition")
}

func (l *mqlTerraformLifecycle) postconditions() ([]any, error) {
	return l.conditionBlocks("postcondition")
}

func (l *mqlTerraformLifecycle) conditionBlocks(kind string) ([]any, error) {
	if l.hclBlock == nil {
		return []any{}, nil
	}
	body, ok := l.hclBlock.Body.(*hclsyntax.Body)
	if !ok {
		return []any{}, nil
	}
	out := []any{}
	for _, child := range body.Blocks {
		if child.Type != kind {
			continue
		}
		hb := syntaxToHclBlock(child)
		condSrc := readExprSource(hb, l.MqlRuntime, "condition")
		errMsg := readStringAttr(hb, "error_message")

		wrapped, err := newMqlHclBlock(l.MqlRuntime, hb, nil)
		if err != nil {
			return nil, err
		}
		res, err := CreateResource(l.MqlRuntime, "terraform.condition", map[string]*llx.RawData{
			"__id":         llx.StringData(kind + "@" + hclRangeToID(child.TypeRange)),
			"condition":    llx.StringData(condSrc),
			"errorMessage": llx.StringData(errMsg),
			"block":        llx.ResourceData(wrapped, "terraform.block"),
		})
		if err != nil {
			return nil, err
		}
		res.(*mqlTerraformCondition).hclBlock = hb
		out = append(out, res)
	}
	return out, nil
}

// --- helpers ----------------------------------------------------------------

func lookupAttr(b *hcl.Block, name string) *hcl.Attribute {
	if b == nil {
		return nil
	}
	attrs, _ := b.Body.JustAttributes()
	return attrs[name]
}

func readStringAttr(b *hcl.Block, name string) string {
	attr := lookupAttr(b, name)
	if attr == nil {
		return ""
	}
	val, diags := attr.Expr.Value(nil)
	if diags.HasErrors() || val.IsNull() || val.Type() != cty.String {
		return ""
	}
	return val.AsString()
}

func readBoolAttr(b *hcl.Block, name string, def bool) bool {
	attr := lookupAttr(b, name)
	if attr == nil {
		return def
	}
	val, diags := attr.Expr.Value(nil)
	if diags.HasErrors() || val.IsNull() || val.Type() != cty.Bool {
		return def
	}
	return val.True()
}

func attributeDict(b *hcl.Block, exclude ...string) (map[string]any, error) {
	if b == nil {
		return map[string]any{}, nil
	}
	excl := map[string]struct{}{}
	for _, k := range exclude {
		excl[k] = struct{}{}
	}
	attrs, _ := b.Body.JustAttributes()
	out := map[string]any{}
	for k, attr := range attrs {
		if _, skip := excl[k]; skip {
			continue
		}
		out[k] = getCtyValue(attr.Expr, &hcl.EvalContext{Functions: hclFunctions()})
	}
	return out, nil
}

func sortedAttrKeys(attrs map[string]*hcl.Attribute) []string {
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func providerFromType(typ string) string {
	if idx := strings.Index(typ, "_"); idx > 0 {
		return typ[:idx]
	}
	return ""
}

// readExprSource returns the raw HCL source for the named attribute's
// expression. Reads the file bytes via the connection's parser using the
// attribute's filename. Falls back to evaluating the expression when the
// source bytes aren't available (e.g. an evaluable literal in a context
// without a parser).
func readExprSource(b *hcl.Block, runtime *plugin.Runtime, name string) string {
	attr := lookupAttr(b, name)
	if attr == nil {
		return ""
	}
	rng := attr.Expr.Range()
	if file := fileFor(runtime, rng.Filename); file != nil && file.Bytes != nil {
		if rng.Start.Byte >= 0 && rng.End.Byte <= len(file.Bytes) && rng.Start.Byte < rng.End.Byte {
			return strings.TrimSpace(string(file.Bytes[rng.Start.Byte:rng.End.Byte]))
		}
	}
	val, diags := attr.Expr.Value(nil)
	if !diags.HasErrors() && !val.IsNull() && val.Type() == cty.String {
		return val.AsString()
	}
	return ""
}

func findChildBlock(parent *hcl.Block, typ string) *hcl.Block {
	if parent == nil {
		return nil
	}
	body, ok := parent.Body.(*hclsyntax.Body)
	if !ok {
		return nil
	}
	for _, child := range body.Blocks {
		if child.Type == typ {
			return syntaxToHclBlock(child)
		}
	}
	return nil
}

func syntaxToHclBlock(b *hclsyntax.Block) *hcl.Block {
	return &hcl.Block{
		Type:        b.Type,
		Labels:      b.Labels,
		Body:        b.Body,
		DefRange:    b.DefRange(),
		TypeRange:   b.TypeRange,
		LabelRanges: b.LabelRanges,
	}
}

func parseIgnoreChanges(lc *hcl.Block) []any {
	attr := lookupAttr(lc, "ignore_changes")
	if attr == nil {
		return []any{}
	}
	if trav, ok := attr.Expr.(*hclsyntax.ScopeTraversalExpr); ok {
		if len(trav.Traversal) == 1 {
			if root, ok := trav.Traversal[0].(hcl.TraverseRoot); ok {
				return []any{root.Name}
			}
		}
		if ref := traversalToString(trav); ref != "" {
			return []any{ref}
		}
		return []any{}
	}
	return tupleToRefStrings(attr)
}

func tupleToRefStrings(attr *hcl.Attribute) []any {
	if attr == nil {
		return []any{}
	}
	tuple, ok := attr.Expr.(*hclsyntax.TupleConsExpr)
	if !ok {
		return []any{}
	}
	out := make([]any, 0, len(tuple.Exprs))
	for _, expr := range tuple.Exprs {
		ref := traversalToString(expr)
		if ref != "" {
			out = append(out, ref)
		}
	}
	return out
}

func traversalToString(expr hcl.Expression) string {
	trav, ok := expr.(*hclsyntax.ScopeTraversalExpr)
	if !ok {
		return ""
	}
	parts := make([]string, 0, len(trav.Traversal))
	for _, step := range trav.Traversal {
		switch s := step.(type) {
		case hcl.TraverseRoot:
			parts = append(parts, s.Name)
		case hcl.TraverseAttr:
			parts = append(parts, s.Name)
		}
	}
	return strings.Join(parts, ".")
}

func hclRangeToID(r hcl.Range) string {
	return r.Filename + ":" + strconv.Itoa(r.Start.Line) + ":" + strconv.Itoa(r.Start.Column)
}
