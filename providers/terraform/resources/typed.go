// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// The typed Terraform resources (terraform.resource, terraform.datasource,
// terraform.variable, terraform.output, terraform.provider) wrap an
// underlying terraform.block and surface its meaning with first-class
// fields and typed cross-references. Each keeps a pointer to the block it
// wraps and to the parent terraform resource so it can resolve references
// against the rest of the parsed configuration.

type mqlTerraformResourceInternal struct {
	tf      *mqlTerraform
	tfBlock *mqlTerraformBlock
}

type mqlTerraformDatasourceInternal struct {
	tf      *mqlTerraform
	tfBlock *mqlTerraformBlock
}

type mqlTerraformVariableInternal struct {
	tf      *mqlTerraform
	tfBlock *mqlTerraformBlock
}

type mqlTerraformOutputInternal struct {
	tf      *mqlTerraform
	tfBlock *mqlTerraformBlock
}

type mqlTerraformProviderInternal struct {
	tf      *mqlTerraform
	tfBlock *mqlTerraformBlock
}

// --- collection accessors on the root terraform resource ---

func (t *mqlTerraform) managedResources() ([]any, error) {
	if err := t.refreshCache(nil); err != nil {
		return nil, err
	}
	blocks := t.mqlTerraformInternal.resources
	res := make([]any, 0, len(blocks))
	for _, b := range blocks {
		r, err := newTerraformResource(t, b)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func (t *mqlTerraform) dataResources() ([]any, error) {
	if err := t.refreshCache(nil); err != nil {
		return nil, err
	}
	res := make([]any, 0, len(t.Datasources.Data))
	for i := range t.Datasources.Data {
		r, err := newTerraformDatasource(t, t.Datasources.Data[i].(*mqlTerraformBlock))
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func (t *mqlTerraform) variableDefinitions() ([]any, error) {
	if err := t.refreshCache(nil); err != nil {
		return nil, err
	}
	res := make([]any, 0, len(t.Variables.Data))
	for i := range t.Variables.Data {
		r, err := newTerraformVariable(t, t.Variables.Data[i].(*mqlTerraformBlock))
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func (t *mqlTerraform) outputValues() ([]any, error) {
	if err := t.refreshCache(nil); err != nil {
		return nil, err
	}
	res := make([]any, 0, len(t.Outputs.Data))
	for i := range t.Outputs.Data {
		r, err := newTerraformOutput(t, t.Outputs.Data[i].(*mqlTerraformBlock))
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func (t *mqlTerraform) providerConfigs() ([]any, error) {
	if err := t.refreshCache(nil); err != nil {
		return nil, err
	}
	res := make([]any, 0, len(t.Providers.Data))
	for i := range t.Providers.Data {
		r, err := newTerraformProvider(t, t.Providers.Data[i].(*mqlTerraformBlock))
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

// --- constructors ---

func newTerraformResource(t *mqlTerraform, b *mqlTerraformBlock) (*mqlTerraformResource, error) {
	bid, _ := b.id()
	r, err := CreateResource(t.MqlRuntime, "terraform.resource", map[string]*llx.RawData{
		"__id": llx.StringData("terraform.resource/" + bid),
		"type": llx.StringData(labelAt(b, 0)),
		"name": llx.StringData(labelAt(b, 1)),
	})
	if err != nil {
		return nil, err
	}
	res := r.(*mqlTerraformResource)
	res.tf = t
	res.tfBlock = b
	return res, nil
}

func newTerraformDatasource(t *mqlTerraform, b *mqlTerraformBlock) (*mqlTerraformDatasource, error) {
	bid, _ := b.id()
	r, err := CreateResource(t.MqlRuntime, "terraform.datasource", map[string]*llx.RawData{
		"__id": llx.StringData("terraform.datasource/" + bid),
		"type": llx.StringData(labelAt(b, 0)),
		"name": llx.StringData(labelAt(b, 1)),
	})
	if err != nil {
		return nil, err
	}
	res := r.(*mqlTerraformDatasource)
	res.tf = t
	res.tfBlock = b
	return res, nil
}

func newTerraformVariable(t *mqlTerraform, b *mqlTerraformBlock) (*mqlTerraformVariable, error) {
	bid, _ := b.id()
	r, err := CreateResource(t.MqlRuntime, "terraform.variable", map[string]*llx.RawData{
		"__id": llx.StringData("terraform.variable/" + bid),
		"name": llx.StringData(labelAt(b, 0)),
	})
	if err != nil {
		return nil, err
	}
	res := r.(*mqlTerraformVariable)
	res.tf = t
	res.tfBlock = b
	return res, nil
}

func newTerraformOutput(t *mqlTerraform, b *mqlTerraformBlock) (*mqlTerraformOutput, error) {
	bid, _ := b.id()
	r, err := CreateResource(t.MqlRuntime, "terraform.output", map[string]*llx.RawData{
		"__id": llx.StringData("terraform.output/" + bid),
		"name": llx.StringData(labelAt(b, 0)),
	})
	if err != nil {
		return nil, err
	}
	res := r.(*mqlTerraformOutput)
	res.tf = t
	res.tfBlock = b
	return res, nil
}

func newTerraformProvider(t *mqlTerraform, b *mqlTerraformBlock) (*mqlTerraformProvider, error) {
	bid, _ := b.id()
	r, err := CreateResource(t.MqlRuntime, "terraform.provider", map[string]*llx.RawData{
		"__id": llx.StringData("terraform.provider/" + bid),
		"name": llx.StringData(labelAt(b, 0)),
	})
	if err != nil {
		return nil, err
	}
	res := r.(*mqlTerraformProvider)
	res.tf = t
	res.tfBlock = b
	return res, nil
}

// labelAt returns the i-th label of a block, or "" when it does not exist.
func labelAt(b *mqlTerraformBlock, i int) string {
	if b == nil {
		return ""
	}
	labels := b.Labels.Data
	if i >= len(labels) {
		return ""
	}
	if s, ok := labels[i].(string); ok {
		return s
	}
	return ""
}

// --- terraform.resource fields ---

func (c *mqlTerraformResource) block() (*mqlTerraformBlock, error) {
	if c.tfBlock == nil {
		c.Block.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return c.tfBlock, nil
}

func (c *mqlTerraformResource) providerName() (string, error) {
	return blockProviderName(c.tfBlock), nil
}

func (c *mqlTerraformResource) provider() (*mqlTerraformProvider, error) {
	return resolveProvider(c.tf, c.tfBlock, &c.Provider)
}

func (c *mqlTerraformResource) arguments() (any, error) {
	if c.tfBlock == nil {
		return map[string]any{}, nil
	}
	return c.tfBlock.arguments()
}

func (c *mqlTerraformResource) tags() (any, error) {
	return blockArgument(c.tfBlock, "tags")
}

func (c *mqlTerraformResource) count() (any, error) {
	return blockArgument(c.tfBlock, "count")
}

func (c *mqlTerraformResource) forEach() (any, error) {
	return blockArgument(c.tfBlock, "for_each")
}

func (c *mqlTerraformResource) dependsOn() ([]any, error) {
	return blockDependsOn(c.tf, c.tfBlock)
}

func (c *mqlTerraformResource) references() ([]any, error) {
	return terraformReferences(c.MqlRuntime, c.tf, c.tfBlock)
}

// --- terraform.datasource fields ---

func (c *mqlTerraformDatasource) block() (*mqlTerraformBlock, error) {
	if c.tfBlock == nil {
		c.Block.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return c.tfBlock, nil
}

func (c *mqlTerraformDatasource) providerName() (string, error) {
	return blockProviderName(c.tfBlock), nil
}

func (c *mqlTerraformDatasource) provider() (*mqlTerraformProvider, error) {
	return resolveProvider(c.tf, c.tfBlock, &c.Provider)
}

func (c *mqlTerraformDatasource) arguments() (any, error) {
	if c.tfBlock == nil {
		return map[string]any{}, nil
	}
	return c.tfBlock.arguments()
}

func (c *mqlTerraformDatasource) references() ([]any, error) {
	return terraformReferences(c.MqlRuntime, c.tf, c.tfBlock)
}

// --- terraform.variable fields ---

func (c *mqlTerraformVariable) block() (*mqlTerraformBlock, error) {
	return c.tfBlock, nil
}

func (c *mqlTerraformVariable) description() (string, error) {
	return blockStringArgument(c.tfBlock, "description"), nil
}

func (c *mqlTerraformVariable) compute_type() (string, error) {
	return blockRawExpr(c.tfBlock, "type"), nil
}

func (c *mqlTerraformVariable) compute_default() (any, error) {
	return blockArgument(c.tfBlock, "default")
}

func (c *mqlTerraformVariable) sensitive() (bool, error) {
	return blockBoolArgument(c.tfBlock, "sensitive", false), nil
}

func (c *mqlTerraformVariable) nullable() (bool, error) {
	// Terraform variables are nullable by default.
	return blockBoolArgument(c.tfBlock, "nullable", true), nil
}

// --- terraform.output fields ---

func (c *mqlTerraformOutput) block() (*mqlTerraformBlock, error) {
	return c.tfBlock, nil
}

func (c *mqlTerraformOutput) description() (string, error) {
	return blockStringArgument(c.tfBlock, "description"), nil
}

func (c *mqlTerraformOutput) value() (any, error) {
	return blockArgument(c.tfBlock, "value")
}

func (c *mqlTerraformOutput) sensitive() (bool, error) {
	return blockBoolArgument(c.tfBlock, "sensitive", false), nil
}

func (c *mqlTerraformOutput) references() ([]any, error) {
	return terraformReferences(c.MqlRuntime, c.tf, c.tfBlock)
}

// --- terraform.provider fields ---

func (c *mqlTerraformProvider) block() (*mqlTerraformBlock, error) {
	return c.tfBlock, nil
}

func (c *mqlTerraformProvider) alias() (string, error) {
	return blockStringArgument(c.tfBlock, "alias"), nil
}

func (c *mqlTerraformProvider) requirement() (*mqlTerraformSettingsRequiredProvider, error) {
	name := labelAt(c.tfBlock, 0)

	// NewResource (not CreateResource) so initTerraformSettings runs and
	// populates the required_providers list.
	o, err := NewResource(c.MqlRuntime, "terraform.settings", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	settings := o.(*mqlTerraformSettings)
	reqs := settings.GetRequiredProviders()
	if reqs.Error != nil {
		return nil, reqs.Error
	}
	for i := range reqs.Data {
		rp := reqs.Data[i].(*mqlTerraformSettingsRequiredProvider)
		if rp.Name.Data == name {
			return rp, nil
		}
	}

	c.Requirement.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

func (c *mqlTerraformProvider) arguments() (any, error) {
	return c.tfBlock.arguments()
}

func (c *mqlTerraformProvider) references() ([]any, error) {
	return terraformReferences(c.MqlRuntime, c.tf, c.tfBlock)
}

// --- shared helpers ---

// blockProviderName returns the local provider name a resource/data source
// binds to: the explicit `provider` meta-argument when present, otherwise the
// type prefix before the first underscore (e.g. "aws" for "aws_instance").
func blockProviderName(b *mqlTerraformBlock) string {
	if name, _, ok := explicitProvider(b); ok {
		return name
	}
	typ := labelAt(b, 0)
	if idx := strings.Index(typ, "_"); idx > 0 {
		return typ[:idx]
	}
	return typ
}

// explicitProvider reads the `provider` meta-argument traversal (e.g.
// `provider = aws.west`) and returns its name and alias.
func explicitProvider(b *mqlTerraformBlock) (name string, alias string, ok bool) {
	hclBlock := blockHcl(b)
	if hclBlock == nil {
		return "", "", false
	}
	attrs, _ := hclBlock.Body.JustAttributes()
	attr, found := attrs["provider"]
	if !found {
		return "", "", false
	}
	vars := attr.Expr.Variables()
	if len(vars) == 0 {
		return "", "", false
	}
	parts := traversalParts(vars[0])
	if len(parts) == 0 {
		return "", "", false
	}
	name = parts[0]
	if len(parts) > 1 {
		alias = parts[1]
	}
	return name, alias, true
}

// resolveProvider finds the provider configuration a resource/data source is
// bound to and wraps it as a terraform.provider, or marks the field null.
func resolveProvider(t *mqlTerraform, b *mqlTerraformBlock, field *plugin.TValue[*mqlTerraformProvider]) (*mqlTerraformProvider, error) {
	if t == nil {
		field.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	name, alias, ok := explicitProvider(b)
	if !ok {
		name = blockProviderName(b)
		alias = ""
	}

	for i := range t.Providers.Data {
		pb := t.Providers.Data[i].(*mqlTerraformBlock)
		if labelAt(pb, 0) != name {
			continue
		}
		if blockStringArgument(pb, "alias") != alias {
			continue
		}
		return newTerraformProvider(t, pb)
	}

	field.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

// blockDependsOn resolves the addresses listed in a block's `depends_on`
// meta-argument to their blocks. Addresses that do not resolve within the
// parsed configuration are dropped here and surfaced via references instead.
func blockDependsOn(t *mqlTerraform, b *mqlTerraformBlock) ([]any, error) {
	hclBlock := blockHcl(b)
	if hclBlock == nil {
		return []any{}, nil
	}
	attrs, _ := hclBlock.Body.JustAttributes()
	attr, ok := attrs["depends_on"]
	if !ok {
		return []any{}, nil
	}

	res := []any{}
	seen := map[string]struct{}{}
	for _, tr := range attr.Expr.Variables() {
		ref := classifyReference(traversalParts(tr))
		if ref == nil {
			continue
		}
		target := t.resolveReference(ref)
		if target == nil {
			continue
		}
		bid, _ := target.id()
		if _, dup := seen[bid]; dup {
			continue
		}
		seen[bid] = struct{}{}
		res = append(res, target)
	}
	return res, nil
}

// blockArgument returns the resolved value of a single top-level argument, or
// nil when the argument is absent.
func blockArgument(b *mqlTerraformBlock, name string) (any, error) {
	if b == nil {
		return nil, nil
	}
	args, err := b.arguments()
	if err != nil {
		return nil, err
	}
	return args[name], nil
}

// blockStringArgument returns a string-valued argument, or "" when absent or
// not a string.
func blockStringArgument(b *mqlTerraformBlock, name string) string {
	v, err := blockArgument(b, name)
	if err != nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// blockBoolArgument returns a bool-valued argument, or fallback when absent or
// not a bool.
func blockBoolArgument(b *mqlTerraformBlock, name string, fallback bool) bool {
	v, err := blockArgument(b, name)
	if err != nil || v == nil {
		return fallback
	}
	if x, ok := v.(bool); ok {
		return x
	}
	return fallback
}

// blockRawExpr returns the raw source text of an argument's expression, e.g.
// the declared type expression `list(string)` of a variable.
func blockRawExpr(b *mqlTerraformBlock, name string) string {
	hclBlock := blockHcl(b)
	if hclBlock == nil {
		return ""
	}
	file := blockFile(b)
	if file == nil {
		return ""
	}
	attrs, _ := hclBlock.Body.JustAttributes()
	attr, ok := attrs[name]
	if !ok {
		return ""
	}
	r := attr.Expr.Range()
	if r.Start.Byte < 0 || r.End.Byte > len(file.Bytes) || r.Start.Byte > r.End.Byte {
		return ""
	}
	return strings.TrimSpace(string(file.Bytes[r.Start.Byte:r.End.Byte]))
}

func blockHcl(b *mqlTerraformBlock) *hcl.Block {
	if b == nil || b.block.State != plugin.StateIsSet {
		return nil
	}
	return b.block.Data
}

func blockFile(b *mqlTerraformBlock) *hcl.File {
	if b == nil || b.cachedFile.State != plugin.StateIsSet {
		return nil
	}
	return b.cachedFile.Data
}

// --- references ---

// tfRef is the classified form of a single traversal in an argument.
type tfRef struct {
	kind   string
	typ    string // resource/data source type, when applicable
	name   string
	target string
}

// terraformReferences builds the per-argument reference records for a block,
// resolving each reference to its block when it exists in the configuration.
func terraformReferences(runtime *plugin.Runtime, t *mqlTerraform, b *mqlTerraformBlock) ([]any, error) {
	hclBlock := blockHcl(b)
	if hclBlock == nil {
		return []any{}, nil
	}
	bid, _ := b.id()

	attrs, _ := hclBlock.Body.JustAttributes()
	names := make([]string, 0, len(attrs))
	for name := range attrs {
		names = append(names, name)
	}
	sort.Strings(names)

	res := []any{}
	for _, name := range names {
		for _, tr := range attrs[name].Expr.Variables() {
			ref := classifyReference(traversalParts(tr))
			if ref == nil {
				continue
			}

			// The `provider` meta-argument points at a provider
			// configuration (e.g. `provider = aws.west`), not a managed
			// resource. The bound provider is also available through the
			// typed `provider` accessor.
			if name == "provider" && ref.kind == "resource" {
				ref.kind = "provider"
			}

			var target *mqlTerraformBlock
			if t != nil {
				target = t.resolveReference(ref)
			}

			args := map[string]*llx.RawData{
				"__id":     llx.StringData(bid + "#" + name + "#" + ref.target),
				"argument": llx.StringData(name),
				"kind":     llx.StringData(ref.kind),
				"target":   llx.StringData(ref.target),
				"name":     llx.StringData(ref.name),
			}
			if target != nil {
				args["block"] = llx.ResourceData(target, "terraform.block")
			}

			r, err := CreateResource(runtime, "terraform.reference", args)
			if err != nil {
				return nil, err
			}
			if target == nil {
				r.(*mqlTerraformReference).Block.State = plugin.StateIsSet | plugin.StateIsNull
			}
			res = append(res, r)
		}
	}
	return res, nil
}

func (c *mqlTerraformReference) block() (*mqlTerraformBlock, error) {
	// block is set at creation when the target resolves; otherwise null.
	c.Block.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

// traversalParts flattens a traversal to its sequence of named steps,
// dropping index steps (e.g. `aws_subnet.ec2[0]` -> ["aws_subnet", "ec2"]).
func traversalParts(t hcl.Traversal) []string {
	parts := make([]string, 0, len(t))
	for _, step := range t {
		switch v := step.(type) {
		case hcl.TraverseRoot:
			parts = append(parts, v.Name)
		case hcl.TraverseAttr:
			parts = append(parts, v.Name)
		}
	}
	return parts
}

// classifyReference turns a traversal's named steps into a typed reference,
// or nil when there is nothing referable (e.g. a bare keyword).
func classifyReference(parts []string) *tfRef {
	if len(parts) == 0 {
		return nil
	}
	switch parts[0] {
	case "var", "local", "module":
		if len(parts) < 2 {
			return nil
		}
		return &tfRef{kind: parts[0], name: parts[1], target: parts[0] + "." + parts[1]}
	case "data":
		if len(parts) < 3 {
			return nil
		}
		return &tfRef{kind: "data", typ: parts[1], name: parts[2], target: "data." + parts[1] + "." + parts[2]}
	case "each", "count", "self", "path", "terraform":
		target := parts[0]
		if len(parts) > 1 {
			target += "." + parts[1]
		}
		return &tfRef{kind: parts[0], target: target}
	default:
		// managed resource reference: <type>.<name>
		if len(parts) < 2 {
			return nil
		}
		return &tfRef{kind: "resource", typ: parts[0], name: parts[1], target: parts[0] + "." + parts[1]}
	}
}

// resolveReference finds the block a reference points to, accounting for the
// separate namespaces Terraform uses (a managed resource and a data source can
// share a type and name).
func (t *mqlTerraform) resolveReference(ref *tfRef) *mqlTerraformBlock {
	switch ref.kind {
	case "resource":
		return findLabeledBlock(t.mqlTerraformInternal.resources, ref.typ, ref.name)
	case "data":
		return findLabeledBlockAny(t.Datasources.Data, ref.typ, ref.name)
	case "var":
		return findNamedBlockAny(t.Variables.Data, ref.name)
	case "output":
		return findNamedBlockAny(t.Outputs.Data, ref.name)
	case "module":
		if b, ok := t.blocksByName[ref.name]; ok && b.Type.Data == "module" {
			return b
		}
		return nil
	}
	return nil
}

func findLabeledBlock(blocks []*mqlTerraformBlock, typ, name string) *mqlTerraformBlock {
	for _, b := range blocks {
		if labelAt(b, 0) == typ && labelAt(b, 1) == name {
			return b
		}
	}
	return nil
}

func findLabeledBlockAny(blocks []any, typ, name string) *mqlTerraformBlock {
	for i := range blocks {
		b := blocks[i].(*mqlTerraformBlock)
		if labelAt(b, 0) == typ && labelAt(b, 1) == name {
			return b
		}
	}
	return nil
}

func findNamedBlockAny(blocks []any, name string) *mqlTerraformBlock {
	for i := range blocks {
		b := blocks[i].(*mqlTerraformBlock)
		if labelAt(b, 0) == name {
			return b
		}
	}
	return nil
}

// references on the underlying block: resolve the parent terraform resource so
// we can link references to their target blocks.
func (b *mqlTerraformBlock) references() ([]any, error) {
	o, err := CreateResource(b.MqlRuntime, "terraform", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	tf := o.(*mqlTerraform)
	if err := tf.refreshCache(nil); err != nil {
		return nil, err
	}
	return terraformReferences(b.MqlRuntime, tf, b)
}
