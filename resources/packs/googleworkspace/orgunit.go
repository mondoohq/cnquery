package googleworkspace

import (
	"go.mondoo.com/cnquery/resources"
	directory "google.golang.org/api/admin/directory/v1"
)

func (g *mqlGoogleworkspace) GetOrgUnits() ([]interface{}, error) {
	provider, directoryService, err := directoryService(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}

	orgUnits, err := directoryService.Orgunits.List(provider.GetCustomerID()).Do()
	if err != nil {
		return nil, err
	}

	for i := range orgUnits.OrganizationUnits {
		r, err := newMqlGoogleWorkspaceOrgUnit(g.MotorRuntime, orgUnits.OrganizationUnits[i])
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func newMqlGoogleWorkspaceOrgUnit(runtime *resources.Runtime, entry *directory.OrgUnit) (interface{}, error) {
	return runtime.CreateResource("googleworkspace.orgUnit",
		"id", entry.OrgUnitId,
		"name", entry.Name,
		"description", entry.Description,
	)
}

func (g *mqlGoogleworkspaceOrgUnit) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return "googleworkspace.orgUnit/" + id, nil
}
