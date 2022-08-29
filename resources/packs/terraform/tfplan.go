package terraform

import (
	"encoding/json"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (t *mqlTerraformPlan) id() (string, error) {
	return "terraform.plan", nil
}

func (t *mqlTerraformPlan) init(args *resources.Args) (*resources.Args, TerraformPlan, error) {
	tfstateProvider, err := terraformProvider(t.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	plan, err := tfstateProvider.Plan()
	if err != nil {
		return nil, nil, err
	}

	(*args)["formatVersion"] = plan.FormatVersion
	(*args)["terraformVersion"] = plan.TerraformVersion

	return args, nil, nil
}

func (t *mqlTerraformPlan) GetResourceChanges() ([]interface{}, error) {
	provider, err := terraformProvider(t.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	plan, err := provider.Plan()
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

		var replacePaths map[string]interface{}
		if rc.Change.ReplacePaths != nil {
			if err := json.Unmarshal(rc.Change.ReplacePaths, &replacePaths); err != nil {
				return nil, err
			}
		}

		lumiChange, err := t.MotorRuntime.CreateResource("terraform.plan.proposedChange",
			"address", rc.Address,
			"actions", core.StrSliceToInterface(rc.Change.Actions),
			"before", before,
			"after", after,
			"afterUnknown", afterUnknown,
			"beforeSensitive", beforeSensitive,
			"afterSensitive", afterSensitive,
			"replacePaths", replacePaths,
		)
		if err != nil {
			return nil, err
		}

		r, err := t.MotorRuntime.CreateResource("terraform.plan.resourceChange",
			"address", rc.Address,
			"previousAddress", rc.PreviousAddress,
			"moduleAddress", rc.ModuleAddress,
			"mode", rc.Mode,
			"type", rc.Type,
			"name", rc.Name,
			"providerName", rc.ProviderName,
			"deposed", rc.Deposed,
			"actionReason", rc.ActionReason,
			"change", lumiChange,
		)
		if err != nil {
			return nil, err
		}
		// store output in cache
		r.MqlResource().Cache.Store("_change", &resources.CacheEntry{Data: rc})

		list = append(list, r)
	}

	return list, nil
}

func (t *mqlTerraformPlanResourceChange) id() (string, error) {
	id, err := t.Address()
	if err != nil {
		return "", err
	}
	return "terraform.plan.resourceChange/address/" + id, nil
}

func (t *mqlTerraformPlanProposedChange) id() (string, error) {
	id, err := t.Address()
	if err != nil {
		return "", err
	}
	return "terraform.plan.resourceChange/address/" + id, nil
}
