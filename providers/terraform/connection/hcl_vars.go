// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

// HCLEvalFunctions returns the function table used when evaluating HCL
// expressions. It is the single source of truth shared by both variable/local
// resolution here and expression evaluation in the resource layer, so the two
// can never diverge.
func HCLEvalFunctions() map[string]function.Function {
	return map[string]function.Function{
		"jsondecode": stdlib.JSONDecodeFunc,
		"jsonencode": stdlib.JSONEncodeFunc,
	}
}

// VariableEvalContext lazily builds and caches an hcl.EvalContext that resolves
// var.* and local.* references for this connection.
//
// Variable values come from `variable` block defaults, overridden by values in
// .tfvars / .tfvars.json (Terraform precedence). Locals are then evaluated from
// those, using a bounded multi-pass fixpoint so that locals referencing other
// locals (or variables) resolve regardless of declaration order. References
// that cannot be resolved statically (data sources, resource attributes,
// circular locals, missing values) are simply left out of the context, so the
// caller falls back to surfacing them as reference strings.
//
// It returns nil for assets without an HCL parser (plan/state).
func (c *Connection) VariableEvalContext() *hcl.EvalContext {
	c.varCtxOnce.Do(func() {
		c.varCtx = c.buildVariableEvalContext()
	})
	return c.varCtx
}

func (c *Connection) buildVariableEvalContext() *hcl.EvalContext {
	if c.parsed == nil {
		return nil
	}

	// base context used to evaluate variable defaults and tfvars; these must not
	// reference other vars/locals, so an empty (functions-only) context is right.
	baseCtx := &hcl.EvalContext{Functions: HCLEvalFunctions()}

	vars := map[string]cty.Value{}
	localExprs := map[string]hcl.Expression{}

	for _, file := range c.parsed.Files() {
		body, ok := file.Body.(*hclsyntax.Body)
		if !ok {
			// .tf.json and other non-native bodies are not walked here; their
			// vars/locals stay unresolved (surfaced as reference strings).
			continue
		}
		for _, block := range body.Blocks {
			switch block.Type {
			case "variable":
				if len(block.Labels) == 0 {
					continue
				}
				if attr, ok := block.Body.Attributes["default"]; ok {
					if v, diags := attr.Expr.Value(baseCtx); !diags.HasErrors() {
						vars[block.Labels[0]] = v
					}
				}
			case "locals":
				for name, attr := range block.Body.Attributes {
					localExprs[name] = attr.Expr
				}
			}
		}
	}

	// .tfvars values override variable defaults.
	for name, attr := range c.tfVars {
		if v, diags := attr.Expr.Value(baseCtx); !diags.HasErrors() {
			vars[name] = v
		}
	}

	ctx := &hcl.EvalContext{
		Functions: HCLEvalFunctions(),
		Variables: map[string]cty.Value{},
	}
	if len(vars) > 0 {
		ctx.Variables["var"] = cty.ObjectVal(vars)
	}

	// Resolve locals with a bounded fixpoint: each pass evaluates the
	// not-yet-resolved locals against the context built so far, adding any that
	// now succeed. We stop early once a pass makes no progress. The pass cap
	// (number of locals) bounds even a fully chained set of locals.
	locals := map[string]cty.Value{}
	for pass := 0; pass < len(localExprs); pass++ {
		progress := false
		for name, expr := range localExprs {
			if _, done := locals[name]; done {
				continue
			}
			if v, diags := expr.Value(ctx); !diags.HasErrors() {
				locals[name] = v
				progress = true
			}
		}
		if len(locals) > 0 {
			ctx.Variables["local"] = cty.ObjectVal(locals)
		}
		if !progress {
			break
		}
	}

	return ctx
}
