// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

type mqlTerraformModuleInternal struct {
	tf      *mqlTerraform
	tfBlock *mqlTerraformBlock
}

// moduleMetaArguments are the module-call arguments that are not module inputs.
var moduleMetaArguments = map[string]bool{
	"source":     true,
	"version":    true,
	"count":      true,
	"for_each":   true,
	"depends_on": true,
	"providers":  true,
}

// moduleCalls builds a terraform.module record from each `module` block in the
// source HCL, so module calls are queryable without `terraform init`.
func (t *mqlTerraform) moduleCalls() ([]any, error) {
	if err := t.refreshCache(nil); err != nil {
		return nil, err
	}
	blocksRaw := t.GetBlocks()
	if blocksRaw.Error != nil {
		return nil, blocksRaw.Error
	}

	res := []any{}
	for i := range blocksRaw.Data {
		b := blocksRaw.Data[i].(*mqlTerraformBlock)
		if b.Type.Data != "module" {
			continue
		}
		r, err := CreateResource(t.MqlRuntime, "terraform.module", map[string]*llx.RawData{
			"key":     llx.StringData(labelAt(b, 0)),
			"source":  llx.StringData(blockStringArgument(b, "source")),
			"version": llx.StringData(blockStringArgument(b, "version")),
			"dir":     llx.StringData(""),
		})
		if err != nil {
			return nil, err
		}
		m := r.(*mqlTerraformModule)
		m.tf = t
		m.tfBlock = b
		res = append(res, m)
	}
	return res, nil
}

// moduleBlock returns the HCL block backing this module — the one captured at
// construction for module calls, or the one located by the manifest path.
func (m *mqlTerraformModule) moduleBlock() *mqlTerraformBlock {
	if m.tfBlock != nil {
		return m.tfBlock
	}
	blk := m.GetBlock()
	if blk.Error != nil {
		return nil
	}
	return blk.Data
}

func (m *mqlTerraformModule) sourceType() (string, error) {
	return classifyModuleSource(m.Source.Data), nil
}

// classifyModuleSource maps a module source address to one of local, git,
// http, registry, or "" (unset), following Terraform's source-addressing.
func classifyModuleSource(s string) string {
	switch {
	case s == "":
		return ""
	case strings.HasPrefix(s, "./"), strings.HasPrefix(s, "../"), strings.HasPrefix(s, "/"):
		return "local"
	case strings.HasPrefix(s, "git::"), strings.HasPrefix(s, "git@"),
		strings.Contains(s, "github.com"), strings.Contains(s, "gitlab.com"),
		strings.Contains(s, "bitbucket.org"), strings.HasSuffix(s, ".git"):
		return "git"
	case strings.HasPrefix(s, "http://"), strings.HasPrefix(s, "https://"),
		strings.HasPrefix(s, "s3::"), strings.HasPrefix(s, "gcs::"):
		return "http"
	default:
		return "registry"
	}
}

func (m *mqlTerraformModule) inputs() (any, error) {
	b := m.moduleBlock()
	if b == nil {
		return map[string]any{}, nil
	}
	args, err := b.arguments()
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	for k := range args {
		if !moduleMetaArguments[k] {
			out[k] = args[k]
		}
	}
	return out, nil
}

func (m *mqlTerraformModule) references() ([]any, error) {
	b := m.moduleBlock()
	if b == nil {
		return []any{}, nil
	}
	if m.tf != nil {
		return terraformReferences(m.MqlRuntime, m.tf, b)
	}
	return b.references()
}

// --- typed selection by type / name ---

func initTerraformResource(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initTypedSelection(runtime, args, "resource")
}

func initTerraformDatasource(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initTypedSelection(runtime, args, "data")
}

func initTypedSelection(runtime *plugin.Runtime, args map[string]*llx.RawData, kind string) (map[string]*llx.RawData, plugin.Resource, error) {
	// No selection arguments means the resource is being constructed
	// internally (e.g. by managedResources); pass through unchanged.
	if args["type"] == nil && args["name"] == nil {
		return args, nil, nil
	}

	matchType, err := llx.StringOrRegexMatcher(args["type"])
	if err != nil {
		return nil, nil, err
	}
	matchName, err := llx.StringOrRegexMatcher(args["name"])
	if err != nil {
		return nil, nil, err
	}

	o, err := CreateResource(runtime, "terraform", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	tf := o.(*mqlTerraform)
	if err := tf.refreshCache(nil); err != nil {
		return nil, nil, err
	}

	var pool []*mqlTerraformBlock
	if kind == "resource" {
		pool = tf.mqlTerraformInternal.resources
	} else {
		for i := range tf.Datasources.Data {
			pool = append(pool, tf.Datasources.Data[i].(*mqlTerraformBlock))
		}
	}

	for _, b := range pool {
		if matchType != nil && !matchType(labelAt(b, 0)) {
			continue
		}
		if matchName != nil && !matchName(labelAt(b, 1)) {
			continue
		}
		if kind == "resource" {
			r, err := newTerraformResource(tf, b)
			return nil, r, err
		}
		r, err := newTerraformDatasource(tf, b)
		return nil, r, err
	}

	// No match: a bare record (its accessors return empty values).
	return args, nil, nil
}
