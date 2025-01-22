// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/terraform/connection"
	"go.mondoo.com/cnquery/v11/types"
)

func (t *mqlTerraformPlan) id() (string, error) {
	return "terraform.plan", nil
}

func initTerraformPlan(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) != 0 {
		return args, nil, nil
	}
	conn := runtime.Connection.(*connection.Connection)

	plan, err := conn.Plan()
	if err != nil {
		return nil, nil, err
	}

	// TODO: This only creates compatibility with v8. Please revisit this section
	// after https://github.com/mondoohq/cnquery/issues/1943 is clarified.
	if plan == nil {
		return map[string]*llx.RawData{
			"formatVersion":    llx.StringData(""),
			"terraformVersion": llx.StringData(""),
			"resourceChanges":  llx.ArrayData([]interface{}{}, types.Resource("terraform.plan.resourceChange")),
		}, nil, nil
	}

	args["formatVersion"] = llx.StringData(plan.FormatVersion)
	args["terraformVersion"] = llx.StringData(plan.TerraformVersion)
	args["applyable"] = llx.BoolData(plan.Applyable)
	args["errored"] = llx.BoolData(plan.Errored)
	args["variables"] = llx.ArrayData(
		variablesToArrayInterface(runtime, plan.Variables),
		types.Resource("terraform.plan.variables"),
	)

	return args, nil, nil
}

func (t *mqlTerraformPlan) resourceChanges() ([]interface{}, error) {
	conn := t.MqlRuntime.Connection.(*connection.Connection)

	plan, err := conn.Plan()
	if err != nil {
		return nil, err
	}

	if plan.ResourceChanges == nil {
		return nil, nil
	}

	var list []interface{}
	for i := range plan.ResourceChanges {

		rc := plan.ResourceChanges[i]

		// TODO: temporarily ignore errors until dicts can be of type interface{}
		var before map[string]interface{}
		if rc.Change.Before != nil {
			if err := json.Unmarshal(rc.Change.Before, &before); err != nil {
				// return nil, err
			}
		}

		var after map[string]interface{}
		if rc.Change.After != nil {
			if err := json.Unmarshal(rc.Change.After, &after); err != nil {
				// return nil, err
			}
		}

		var afterUnknown map[string]interface{}
		if rc.Change.AfterUnknown != nil {
			if err := json.Unmarshal(rc.Change.AfterUnknown, &afterUnknown); err != nil {
				// return nil, err
			}
		}

		var beforeSensitive map[string]interface{}
		if rc.Change.BeforeSensitive != nil {
			if err := json.Unmarshal(rc.Change.BeforeSensitive, &beforeSensitive); err != nil {
				// return nil, err
			}
		}

		var afterSensitive map[string]interface{}
		if rc.Change.AfterSensitive != nil {
			if err := json.Unmarshal(rc.Change.AfterSensitive, &afterSensitive); err != nil {
				// return nil, err
			}
		}

		var replacePaths []interface{}
		if rc.Change.ReplacePaths != nil {
			if err := json.Unmarshal(rc.Change.ReplacePaths, &replacePaths); err != nil {
				return nil, err
			}
		}

		lumiChange, err := CreateResource(t.MqlRuntime, "terraform.plan.proposedChange", map[string]*llx.RawData{
			"address":         llx.StringData(rc.Address),
			"actions":         llx.ArrayData(convert.SliceAnyToInterface[string](rc.Change.Actions), types.String),
			"before":          llx.MapData(before, types.Any),
			"after":           llx.MapData(after, types.Any),
			"afterUnknown":    llx.MapData(afterUnknown, types.Any),
			"beforeSensitive": llx.MapData(beforeSensitive, types.Any),
			"afterSensitive":  llx.MapData(afterSensitive, types.Any),
			"replacePaths":    llx.ArrayData(replacePaths, types.Any),
		})
		if err != nil {
			return nil, err
		}

		r, err := CreateResource(t.MqlRuntime, "terraform.plan.resourceChange", map[string]*llx.RawData{
			"address":         llx.StringData(rc.Address),
			"previousAddress": llx.StringData(rc.PreviousAddress),
			"moduleAddress":   llx.StringData(rc.ModuleAddress),
			"mode":            llx.StringData(rc.Mode),
			"type":            llx.StringData(rc.Type),
			"name":            llx.StringData(rc.Name),
			"providerName":    llx.StringData(rc.ProviderName),
			"deposed":         llx.StringData(rc.Deposed),
			"actionReason":    llx.StringData(rc.ActionReason),
			"change":          llx.ResourceData(lumiChange, lumiChange.MqlName()),
		})
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}

	return list, nil
}

func (t *mqlTerraformPlanResourceChange) id() (string, error) {
	id := t.Address
	return "terraform.plan.resourceChange/address/" + id.Data, nil
}

func (t *mqlTerraformPlanProposedChange) id() (string, error) {
	id := t.Address
	return "terraform.plan.resourceChange/address/" + id.Data, nil
}

func (t *mqlTerraformPlanConfiguration) id() (string, error) {
	return "terraform.plan.configuration", nil
}

func (t *mqlTerraformPlanVariable) id() (string, error) {
	id := t.Name
	return "terraform.plan.variable/name/" + id.Data, nil
}

type PlanConfiguration struct {
	ProviderConfig map[string]json.RawMessage `json:"provider_config"`
	RootModule     struct {
		Resources []json.RawMessage `json:"resources"`
	} `json:"root_module"`
}

func (t *mqlTerraformPlanConfiguration) providerConfig() ([]interface{}, error) {
	conn := t.MqlRuntime.Connection.(*connection.Connection)
	plan, err := conn.Plan()
	if err != nil {
		return nil, err
	}

	// TODO: This only creates compatibility with v8. Please revisit this section
	// after https://github.com/mondoohq/cnquery/issues/1943 is clarified.
	if plan == nil {
		return []interface{}{}, nil
	}

	if plan.Configuration == nil {
		return nil, nil
	}

	pc := PlanConfiguration{}
	err = json.Unmarshal(plan.Configuration, &pc)

	res := []interface{}{}
	for i := range pc.ProviderConfig {
		config := pc.ProviderConfig[i]
		var entry interface{}
		if err := json.Unmarshal([]byte(config), &entry); err != nil {
			return nil, err
		}
		res = append(res, entry)
	}
	return res, nil
}

func (t *mqlTerraformPlanConfiguration) resources() ([]interface{}, error) {
	conn := t.MqlRuntime.Connection.(*connection.Connection)
	plan, err := conn.Plan()
	if err != nil {
		return nil, err
	}

	// TODO: This only creates compatibility with v8. Please revisit this section
	// after https://github.com/mondoohq/cnquery/issues/1943 is clarified.
	if plan == nil {
		return []interface{}{}, nil
	}

	if plan.Configuration == nil {
		return nil, nil
	}

	pc := PlanConfiguration{}
	err = json.Unmarshal(plan.Configuration, &pc)

	res := []interface{}{}
	for i := range pc.RootModule.Resources {
		config := pc.RootModule.Resources[i]
		var entry interface{}
		if err := json.Unmarshal([]byte(config), &entry); err != nil {
			return nil, err
		}
		res = append(res, entry)
	}
	return res, nil
}

func variablesToArrayInterface(runtime *plugin.Runtime, variables connection.Variables) []interface{} {
	var list []interface{}
	for k, v := range variables {
		var value interface{}
		err := json.Unmarshal(v.Value, &value)
		if err != nil {
			continue
		}
		variable, err := CreateResource(runtime, "terraform.plan.variable", map[string]*llx.RawData{
			"name":  llx.StringData(k),
			"value": llx.AnyData(value),
		})
		if err != nil {
			continue
		}

		list = append(list, variable)
	}

	return list
}
