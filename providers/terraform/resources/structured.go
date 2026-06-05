// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// These accessors turn data that previously had to be walked by hand into
// structured fields: nested blocks keyed by type, a normalized value view
// that unwraps jsonencode policy documents, flattened lifecycle
// meta-arguments, and a reverse reference index.

// --- nested blocks keyed by type ---

func (b *mqlTerraformBlock) nestedBlocks() (map[string]any, error) {
	children, err := b.blocks()
	if err != nil {
		return nil, err
	}
	res := map[string]any{}
	for i := range children {
		cb := children[i].(*mqlTerraformBlock)
		typ := cb.Type.Data
		lst, _ := res[typ].([]any)
		res[typ] = append(lst, cb)
	}
	return res, nil
}

func (c *mqlTerraformResource) nestedBlocks() (map[string]any, error) {
	if c.tfBlock == nil {
		return map[string]any{}, nil
	}
	return c.tfBlock.nestedBlocks()
}

// --- normalized value view ---

func (b *mqlTerraformBlock) config() (any, error) {
	hclBlock := blockHcl(b)
	if hclBlock == nil {
		return map[string]any{}, nil
	}
	attrs, _ := hclBlock.Body.JustAttributes()
	return hclConfigAttributesToDict(attrs), nil
}

func (c *mqlTerraformResource) config() (any, error) {
	return c.tfBlock.config()
}

func (c *mqlTerraformDatasource) config() (any, error) {
	return c.tfBlock.config()
}

// hclConfigAttributesToDict resolves arguments like hclResolvedAttributesToDict,
// but unwraps the single-element list that getCtyValue produces for a function
// call (e.g. `jsonencode({...})`) so the decoded object is returned directly.
func hclConfigAttributesToDict(attributes map[string]*hcl.Attribute) map[string]any {
	ctx := &hcl.EvalContext{Functions: hclFunctions()}
	dict := map[string]any{}
	for k := range attributes {
		v := getCtyValue(attributes[k].Expr, ctx)
		if _, isFunc := attributes[k].Expr.(*hclsyntax.FunctionCallExpr); isFunc {
			if lst, ok := v.([]any); ok && len(lst) == 1 {
				v = lst[0]
			}
		}
		dict[k] = v
	}
	return dict
}

// --- lifecycle meta-arguments ---

func (c *mqlTerraformResource) lifecycleBlock() *mqlTerraformBlock {
	if c.tfBlock == nil {
		return nil
	}
	children, err := c.tfBlock.blocks()
	if err != nil {
		return nil
	}
	for i := range children {
		cb := children[i].(*mqlTerraformBlock)
		if cb.Type.Data == "lifecycle" {
			return cb
		}
	}
	return nil
}

func (c *mqlTerraformResource) preventDestroy() (bool, error) {
	lb := c.lifecycleBlock()
	if lb == nil {
		return false, nil
	}
	return blockBoolArgument(lb, "prevent_destroy", false), nil
}

func (c *mqlTerraformResource) createBeforeDestroy() (bool, error) {
	lb := c.lifecycleBlock()
	if lb == nil {
		return false, nil
	}
	return blockBoolArgument(lb, "create_before_destroy", false), nil
}

func (c *mqlTerraformResource) ignoreChanges() ([]any, error) {
	lb := c.lifecycleBlock()
	if lb == nil {
		return []any{}, nil
	}
	v, err := blockArgument(lb, "ignore_changes")
	if err != nil {
		return nil, err
	}
	return toStringList(v), nil
}

// toStringList normalizes an ignore_changes value, which is either a list of
// attribute names or the bare keyword `all`, into a string list.
func toStringList(v any) []any {
	switch x := v.(type) {
	case string:
		return []any{x}
	case []any:
		out := make([]any, 0, len(x))
		for _, e := range x {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return []any{}
	}
}

// --- reverse reference index ---

func (c *mqlTerraformResource) referencedBy() ([]any, error) {
	return referencedByBlock(c.tf, c.tfBlock)
}

func (c *mqlTerraformDatasource) referencedBy() ([]any, error) {
	return referencedByBlock(c.tf, c.tfBlock)
}

func referencedByBlock(t *mqlTerraform, target *mqlTerraformBlock) ([]any, error) {
	if t == nil || target == nil {
		return []any{}, nil
	}
	idx, err := t.reverseRefIndex()
	if err != nil {
		return nil, err
	}
	tid, _ := target.id()
	return idx[tid], nil
}

// reverseRefIndex maps a block's id to the blocks that reference it anywhere in
// their bodies (including nested blocks). It is computed once and cached.
func (t *mqlTerraform) reverseRefIndex() (map[string][]any, error) {
	if err := t.refreshCache(nil); err != nil {
		return nil, err
	}

	// Gather all top-level blocks before locking; GetBlocks is already
	// resolved by refreshCache, so this does not re-enter the lock.
	blocksRaw := t.GetBlocks()
	if blocksRaw.Error != nil {
		return nil, blocksRaw.Error
	}
	allBlocks := blocksRaw.Data

	t.lock.Lock()
	defer t.lock.Unlock()
	if t.refIndex != nil {
		return t.refIndex, nil
	}

	idx := map[string][]any{}
	for i := range allBlocks {
		container := allBlocks[i].(*mqlTerraformBlock)
		cid, _ := container.id()

		targets := map[string]*mqlTerraformBlock{}
		collectResolvedTargets(t, blockHcl(container), targets)

		for tid := range targets {
			if tid == cid {
				continue
			}
			idx[tid] = append(idx[tid], container)
		}
	}

	t.refIndex = idx
	return idx, nil
}

// collectResolvedTargets walks a block body — attributes and nested blocks —
// and records every reference that resolves to a block, keyed by target id.
func collectResolvedTargets(t *mqlTerraform, hclBlock *hcl.Block, out map[string]*mqlTerraformBlock) {
	if hclBlock == nil {
		return
	}

	body, ok := hclBlock.Body.(*hclsyntax.Body)
	if !ok {
		attrs, _ := hclBlock.Body.JustAttributes()
		for _, attr := range attrs {
			collectTargetsFromExpr(t, attr.Expr, out)
		}
		return
	}

	for _, attr := range body.Attributes {
		collectTargetsFromExpr(t, attr.Expr, out)
	}
	for _, nb := range body.Blocks {
		collectResolvedTargets(t, nb.AsHCLBlock(), out)
	}
}

func collectTargetsFromExpr(t *mqlTerraform, expr hcl.Expression, out map[string]*mqlTerraformBlock) {
	for _, tr := range expr.Variables() {
		ref := classifyReference(traversalParts(tr))
		if ref == nil {
			continue
		}
		tb := t.resolveReference(ref)
		if tb == nil {
			continue
		}
		id, _ := tb.id()
		out[id] = tb
	}
}
