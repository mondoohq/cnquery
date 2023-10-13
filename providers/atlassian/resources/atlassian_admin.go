package resources

import (
	"context"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/providers/atlassian/connection/admin"
)

func (a *mqlAtlassianAdmin) id() (string, error) {
	return "wip", nil
}

func (a *mqlAtlassianAdmin) organizations() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*admin.AdminConnection)
	admin := conn.Client()
	organizations, response, err := admin.Organization.Gets(context.Background(), "")
	if err != nil {
		log.Fatal().Err(err)
	}
	if response.Status != "200 OK" {
		log.Fatal().Msgf("Received response: %s\n", response.Status)
	}
	res := []interface{}{}
	for _, org := range organizations.Data {
		mqlAtlassianAdminOrg, err := CreateResource(a.MqlRuntime, "atlassian.admin.organization",
			map[string]*llx.RawData{
				"id":   llx.StringData(org.ID),
				"type": llx.StringData(org.Type),
			})
		if err != nil {
			log.Fatal().Err(err)
		}
		res = append(res, mqlAtlassianAdminOrg)
	}
	return res, nil
}

func (a *mqlAtlassianAdminOrganization) scim() (*mqlAtlassianAdminOrganizationScim, error) {
	mqlAtlassianAdminSCIM, err := CreateResource(a.MqlRuntime, "atlassian.admin.organization.scim",
		map[string]*llx.RawData{})
	if err != nil {
		log.Fatal().Err(err)
	}
	return mqlAtlassianAdminSCIM.(*mqlAtlassianAdminOrganizationScim), nil
}

func (a *mqlAtlassianAdminOrganizationScim) users() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*admin.AdminConnection)
	admin := conn.Client()
	scimUsers, response, err := admin.SCIM.User.Gets(context.Background(), "786d6a74-k7b3-14jk-7863-5b83a48k8c43", nil, 0, 1000)
	if err != nil {
		log.Fatal().Err(err)
	}
	if response.Status != "200 OK" {
		log.Fatal().Msgf("Received response: %s\n", response.Status)
	}
	res := []interface{}{}
	for _, scimUser := range scimUsers.Resources {
		mqlAtlassianAdminSCIMuser, err := CreateResource(a.MqlRuntime, "atlassian.admin.organization.scim.user",
			map[string]*llx.RawData{
				"id": llx.StringData(scimUser.ID),
			})
		if err != nil {
			log.Fatal().Err(err)
		}
		res = append(res, mqlAtlassianAdminSCIMuser)
	}
	return res, nil
}

func (a *mqlAtlassianAdminOrganizationScim) groups() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*admin.AdminConnection)
	admin := conn.Client()
	scimGroup, response, err := admin.SCIM.Group.Gets(context.Background(), "786d6a74-k7b3-14jk-7863-5b83a48k8c43", "", 0, 1000)
	if err != nil {
		log.Fatal().Err(err)
	}
	if response.Status != "200 OK" {
		log.Fatal().Msgf("Received response: %s\n", response.Status)
	}
	res := []interface{}{}
	for _, scimGroup := range scimGroup.Resources {
		mqlAtlassianAdminSCIMgroup, err := CreateResource(a.MqlRuntime, "atlassian.admin.organization.scim.group",
			map[string]*llx.RawData{
				"id": llx.StringData(scimGroup.ID),
			})
		if err != nil {
			log.Fatal().Err(err)
		}
		res = append(res, mqlAtlassianAdminSCIMgroup)
	}
	return res, nil
}

func (a *mqlAtlassianAdminOrganizationScimUser) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAtlassianAdminOrganizationScimGroup) id() (string, error) {
	return a.Id.Data, nil
}

type atlassianUser struct {
	AccountID string
	Name      string
	Type      string
	OrgID     string
}

func (a *mqlAtlassianAdminOrganization) managedUsers() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*admin.AdminConnection)

	admin := conn.Client()

	managedUsers, response, err := admin.Organization.Users(context.Background(), a.Id.Data, "")
	if err != nil {
		log.Fatal().Err(err)
	}
	if response.Status != "200 OK" {
		log.Fatal().Msgf("Received response: %s\n", response.Status)
	}
	res := []interface{}{}
	for _, user := range managedUsers.Data {
		mqlAtlassianAdminManagedUser, err := CreateResource(a.MqlRuntime, "atlassian.admin.organization.managedUser",
			map[string]*llx.RawData{
				"id":   llx.StringData(user.AccountID),
				"name": llx.StringData(user.Name),
				"type": llx.StringData(user.AccountType),
			})
		if err != nil {
			log.Fatal().Err(err)
		}
		res = append(res, mqlAtlassianAdminManagedUser)
	}
	return res, nil
}

func (a *mqlAtlassianAdminOrganization) policies() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*admin.AdminConnection)
	admin := conn.Client()
	orgId := a.Id.Data
	policies, response, err := admin.Organization.Policy.Gets(context.Background(), orgId, "", "")
	if err != nil {
		log.Fatal().Err(err)
	}
	if response.Status != "200 OK" {
		log.Fatal().Msgf("Received response: %s\n", response.Status)
	}
	res := []interface{}{}
	for _, policy := range policies.Data {
		mqlAtlassianAdminPolicy, err := CreateResource(a.MqlRuntime, "atlassian.admin.organization.policy",
			map[string]*llx.RawData{
				"id":         llx.StringData(policy.ID),
				"type":       llx.StringData(policy.Type),
				"name":       llx.StringData(policy.Attributes.Name),
				"status":     llx.StringData(policy.Attributes.Status),
				"policyType": llx.StringData(policy.Attributes.Type),
			})
		if err != nil {
			log.Fatal().Err(err)
		}
		res = append(res, mqlAtlassianAdminPolicy)
	}
	return res, nil
}

func (a *mqlAtlassianAdminOrganization) domains() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*admin.AdminConnection)
	admin := conn.Client()
	orgId := a.Id.Data
	domains, response, err := admin.Organization.Domains(context.Background(), orgId, "")
	if err != nil {
		log.Fatal().Err(err)
	}
	if response.Status != "200 OK" {
		log.Fatal().Msgf("Received response: %s\n", response.Status)
	}
	res := []interface{}{}
	for _, domain := range domains.Data {
		mqlAtlassianAdminDomain, err := CreateResource(a.MqlRuntime, "atlassian.admin.organization.domain",
			map[string]*llx.RawData{
				"id": llx.StringData(domain.ID),
			})
		if err != nil {
			log.Fatal().Err(err)
		}
		res = append(res, mqlAtlassianAdminDomain)
	}
	return res, nil
}

func (a *mqlAtlassianAdminOrganization) events() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*admin.AdminConnection)
	admin := conn.Client()
	orgId := a.Id.Data
	events, response, err := admin.Organization.Events(context.Background(), orgId, nil, "")
	if err != nil {
		log.Fatal().Err(err)
	}
	if response.Status != "200 OK" {
		log.Fatal().Msgf("Received response: %s\n", response.Status)
	}
	res := []interface{}{}
	for _, event := range events.Data {
		mqlAtlassianAdminDomain, err := CreateResource(a.MqlRuntime, "atlassian.admin.organization.event",
			map[string]*llx.RawData{
				"id": llx.StringData(event.ID),
			})
		if err != nil {
			log.Fatal().Err(err)
		}
		res = append(res, mqlAtlassianAdminDomain)
	}
	return res, nil
}

func (a *mqlAtlassianAdminOrganizationPolicy) id() (string, error) {
	return a.Id.Data, nil
}
