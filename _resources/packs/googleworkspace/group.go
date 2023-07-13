package googleworkspace

import (
	"context"
	"strings"

	"google.golang.org/api/groupssettings/v1"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	directory "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/cloudidentity/v1"
	"google.golang.org/api/option"
)

func (g *mqlGoogleworkspace) GetGroups() ([]interface{}, error) {
	provider, directoryService, err := directoryService(g.MotorRuntime.Motor.Provider, directory.AdminDirectoryGroupReadonlyScope)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	groups, err := directoryService.Groups.List().Customer(provider.GetCustomerID()).Do()
	if err != nil {
		return nil, err
	}

	for {
		for i := range groups.Groups {
			r, err := newMqlGoogleWorkspaceGroup(g.MotorRuntime, groups.Groups[i])
			if err != nil {
				return nil, err
			}
			res = append(res, r)
		}

		if groups.NextPageToken == "" {
			break
		}

		groups, err = directoryService.Groups.List().Customer(provider.GetCustomerID()).PageToken(groups.NextPageToken).Do()
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func newMqlGoogleWorkspaceGroup(runtime *resources.Runtime, entry *directory.Group) (interface{}, error) {
	return runtime.CreateResource("googleworkspace.group",
		"id", entry.Id,
		"name", entry.Name,
		"email", entry.Email,
		"description", entry.Description,
		"aliases", core.StrSliceToInterface(entry.Aliases),
		"directMembersCount", entry.DirectMembersCount,
		"adminCreated", entry.AdminCreated,
	)
}

func (g *mqlGoogleworkspaceGroup) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return "googleworkspace.group/" + id, nil
}

func (g *mqlGoogleworkspaceGroup) GetMembers() ([]interface{}, error) {
	provider, err := workspaceProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	client, err := provider.Client()
	if err != nil {
		return nil, err
	}

	directoryService, err := directory.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	id, err := g.Id()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}

	members, err := directoryService.Members.List(id).Do()
	if err != nil {
		return nil, err
	}

	for {
		for i := range members.Members {
			r, err := newMqlGoogleWorkspaceMember(g.MotorRuntime, members.Members[i])
			if err != nil {
				return nil, err
			}
			res = append(res, r)
		}

		if members.NextPageToken == "" {
			break
		}

		members, err = directoryService.Members.List(id).PageToken(members.NextPageToken).Do()
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func newMqlGoogleWorkspaceMember(runtime *resources.Runtime, entry *directory.Member) (interface{}, error) {
	return runtime.CreateResource("googleworkspace.member",
		"id", entry.Id,
		"email", entry.Email,
		"status", entry.Status,
		"type", entry.Type,
	)
}

func (g *mqlGoogleworkspaceMember) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return "googleworkspace.member/" + id, nil
}

func (g *mqlGoogleworkspaceMember) GetUser() (interface{}, error) {
	email, err := g.Email()
	if err != nil {
		return "", err
	}
	typ, err := g.Type()
	if err != nil {
		return "", err
	}

	if strings.ToLower(typ) != "user" {
		return nil, nil
	}

	obj, err := g.MotorRuntime.CreateResource("googleworkspace")
	if err != nil {
		return nil, err
	}
	gws := obj.(Googleworkspace)

	users, err := gws.Users()
	if err != nil {
		return nil, err
	}

	for i := range users {
		user := users[i].(*mqlGoogleworkspaceUser)
		primaryEmail, err := user.PrimaryEmail()
		if err != nil {
			return nil, err
		}
		if primaryEmail == email {
			return user, nil
		}
	}
	return nil, nil
}

func (g *mqlGoogleworkspaceGroup) GetSettings() (interface{}, error) {
	_, service, err := groupSettingsService(g.MotorRuntime.Motor.Provider, groupssettings.AppsGroupsSettingsScope)
	if err != nil {
		return nil, err
	}

	email, err := g.Email()
	if err != nil {
		return nil, err
	}

	settings, err := service.Groups.Get(email).Do()
	if err != nil {
		return nil, err
	}

	return core.JsonToDict(settings)
}

func (g *mqlGoogleworkspaceGroup) GetSecuritySettings() (interface{}, error) {
	_, service, err := cloudIdentityService(g.MotorRuntime.Motor.Provider, cloudidentity.CloudIdentityGroupsReadonlyScope)
	if err != nil {
		return nil, err
	}

	groupId, err := g.Id()
	if err != nil {
		return nil, err
	}

	securitySettings, err := service.Groups.GetSecuritySettings(`groups/` + groupId + `/securitySettings`).ReadMask("*").Do()
	if err != nil {
		return nil, err
	}

	return core.JsonToDict(securitySettings)
}
