package googleworkspace

import (
	"strconv"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	directory "google.golang.org/api/admin/directory/v1"
)

func (g *mqlGoogleworkspace) GetRoles() ([]interface{}, error) {
	provider, directoryService, err := directoryService(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	groups, err := directoryService.Roles.List(provider.GetCustomerID()).Do()
	if err != nil {
		return nil, err
	}
	for {
		for i := range groups.Items {
			r, err := newMqlGoogleWorkspaceRole(g.MotorRuntime, groups.Items[i])
			if err != nil {
				return nil, err
			}
			res = append(res, r)
		}

		if groups.NextPageToken == "" {
			break
		}

		groups, err = directoryService.Roles.List(provider.GetCustomerID()).PageToken(groups.NextPageToken).Do()
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func newMqlGoogleWorkspaceRole(runtime *resources.Runtime, entry *directory.Role) (interface{}, error) {
	privileges, err := core.JsonToDictSlice(entry.RolePrivileges)
	if err != nil {
		return nil, err
	}
	return runtime.CreateResource("googleworkspace.role",
		"id", entry.RoleId,
		"name", entry.RoleName,
		"description", entry.RoleDescription,
		"isSystemRole", entry.IsSystemRole,
		"isSuperAdminRole", entry.IsSuperAdminRole,
		"privileges", privileges,
	)
}

func (g *mqlGoogleworkspaceRole) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return "googleworkspace.role/" + strconv.FormatInt(id, 10), nil
}
