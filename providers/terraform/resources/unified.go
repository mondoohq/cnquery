// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/terraform/connection"
)

// The typed resource model is source-unified: terraform.managedResources and
// terraform.dataResources are populated from whichever source backs the
// connection — parsed HCL, a plan file, or a state file. Each resource exposes
// a single `values` view (resolved config / planned after-values / state
// values) so a policy written against `values` runs unchanged across sources.

func (t *mqlTerraform) managedResources() ([]any, error) {
	return t.unifiedResources("managed")
}

func (t *mqlTerraform) dataResources() ([]any, error) {
	return t.unifiedResources("data")
}

func (t *mqlTerraform) unifiedResources(mode string) ([]any, error) {
	conn, ok := t.MqlRuntime.Connection.(*connection.Connection)
	if !ok {
		return []any{}, nil
	}

	if conn.Parser() != nil {
		return t.hclResources(mode)
	}
	if st, _ := conn.State(); st != nil {
		return t.stateResources(st, mode)
	}
	if pl, _ := conn.Plan(); pl != nil {
		return t.planResources(pl, mode)
	}
	return []any{}, nil
}

func (t *mqlTerraform) hclResources(mode string) ([]any, error) {
	if err := t.refreshCache(nil); err != nil {
		return nil, err
	}
	if mode == "data" {
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

func (t *mqlTerraform) stateResources(st *connection.State, mode string) ([]any, error) {
	if st.Values == nil || st.Values.RootModule == nil {
		return []any{}, nil
	}

	res := []any{}
	var walk func(m *connection.Module) error
	walk = func(m *connection.Module) error {
		for i := range m.Resources {
			r := m.Resources[i]
			if r.Mode != mode {
				continue
			}
			rec, err := newResourceFromValues(t, mode, r.Address, r.Type, r.Name, r.ProviderName, r.AttributeValues)
			if err != nil {
				return err
			}
			res = append(res, rec)
		}
		for _, cm := range m.ChildModules {
			if err := walk(cm); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(st.Values.RootModule); err != nil {
		return nil, err
	}
	return res, nil
}

func (t *mqlTerraform) planResources(pl *connection.Plan, mode string) ([]any, error) {
	res := []any{}
	for i := range pl.ResourceChanges {
		rc := pl.ResourceChanges[i]
		if rc.Mode != mode {
			continue
		}
		var after map[string]any
		if len(rc.Change.After) > 0 {
			// Best effort: an unparseable after-value yields empty values.
			_ = json.Unmarshal(rc.Change.After, &after)
		}
		rec, err := newResourceFromValues(t, mode, rc.Address, rc.Type, rc.Name, rc.ProviderName, after)
		if err != nil {
			return nil, err
		}
		res = append(res, rec)
	}
	return res, nil
}

// newResourceFromValues builds a terraform.resource or terraform.datasource
// from already-resolved values (plan/state), with no backing HCL block.
func newResourceFromValues(t *mqlTerraform, mode, address, typ, name, provider string, values map[string]any) (plugin.Resource, error) {
	if values == nil {
		values = map[string]any{}
	}

	resourceName := "terraform.resource"
	if mode == "data" {
		resourceName = "terraform.datasource"
	}
	r, err := CreateResource(t.MqlRuntime, resourceName, map[string]*llx.RawData{
		"__id": llx.StringData(resourceName + "/" + address),
		"type": llx.StringData(typ),
		"name": llx.StringData(name),
	})
	if err != nil {
		return nil, err
	}

	if mode == "data" {
		ds := r.(*mqlTerraformDatasource)
		ds.tf = t
		ds.addr = address
		ds.cachedValues = values
		ds.valuesSet = true
		ds.cachedProvider = provider
		return ds, nil
	}
	res := r.(*mqlTerraformResource)
	res.tf = t
	res.addr = address
	res.cachedValues = values
	res.valuesSet = true
	res.cachedProvider = provider
	return res, nil
}

// --- source-unified accessors on terraform.resource ---

func (c *mqlTerraformResource) values() (any, error) {
	if c.valuesSet {
		return c.cachedValues, nil
	}
	return c.resolved()
}

func (c *mqlTerraformResource) address() (string, error) {
	if c.addr != "" {
		return c.addr, nil
	}
	return labelAt(c.tfBlock, 0) + "." + labelAt(c.tfBlock, 1), nil
}

func (c *mqlTerraformResource) mode() (string, error) {
	return "managed", nil
}

func (c *mqlTerraformResource) module() (*mqlTerraformModule, error) {
	return moduleForResource(c.MqlRuntime, c.tf, c.moduleCall, &c.Module)
}

// --- source-unified accessors on terraform.datasource ---

func (c *mqlTerraformDatasource) values() (any, error) {
	if c.valuesSet {
		return c.cachedValues, nil
	}
	return c.resolved()
}

func (c *mqlTerraformDatasource) address() (string, error) {
	if c.addr != "" {
		return c.addr, nil
	}
	return "data." + labelAt(c.tfBlock, 0) + "." + labelAt(c.tfBlock, 1), nil
}

func (c *mqlTerraformDatasource) mode() (string, error) {
	return "data", nil
}

func (c *mqlTerraformDatasource) module() (*mqlTerraformModule, error) {
	return moduleForResource(c.MqlRuntime, c.tf, c.moduleCall, &c.Module)
}

// moduleForResource wraps the module-call block this resource belongs to (set
// during HCL construction), or marks the field null for root-module and
// plan/state resources.
func moduleForResource(runtime *plugin.Runtime, t *mqlTerraform, moduleCall *mqlTerraformBlock, field *plugin.TValue[*mqlTerraformModule]) (*mqlTerraformModule, error) {
	if t == nil || moduleCall == nil {
		field.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := CreateResource(runtime, "terraform.module", map[string]*llx.RawData{
		"key":     llx.StringData(labelAt(moduleCall, 0)),
		"source":  llx.StringData(blockStringArgument(moduleCall, "source")),
		"version": llx.StringData(blockStringArgument(moduleCall, "version")),
		"dir":     llx.StringData(""),
	})
	if err != nil {
		return nil, err
	}
	m := r.(*mqlTerraformModule)
	m.tf = t
	m.tfBlock = moduleCall
	return m, nil
}
