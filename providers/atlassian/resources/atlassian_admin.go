package resources

import (
	"context"

	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/providers/atlassian/connection/admin"
)

func (a *mqlAtlassianAdmin) id() (string, error) {
	return "admin", nil
}

func (a *mqlAtlassianAdmin) organizations() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*admin.AdminConnection)
	admin := conn.Client()
	organizations, _, err := admin.Organization.Gets(context.Background(), "")
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	for _, org := range organizations.Data {
		mqlAtlassianAdminOrg, err := CreateResource(a.MqlRuntime, "atlassian.admin.organization",
			map[string]*llx.RawData{
				"id":   llx.StringData(org.ID),
				"type": llx.StringData(org.Type),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAtlassianAdminOrg)
	}
	return res, nil
}

func (a *mqlAtlassianAdminOrganization) managedUsers() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*admin.AdminConnection)

	admin := conn.Client()

	managedUsers, _, err := admin.Organization.Users(context.Background(), a.Id.Data, "")
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	for _, user := range managedUsers.Data {
		mqlAtlassianAdminManagedUser, err := CreateResource(a.MqlRuntime, "atlassian.admin.organization.managedUser",
			map[string]*llx.RawData{
				"id":         llx.StringData(user.AccountID),
				"name":       llx.StringData(user.Name),
				"type":       llx.StringData(user.AccountType),
				"email":      llx.StringData(user.Email),
				"lastActive": llx.StringData(user.LastActive),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAtlassianAdminManagedUser)
	}
	return res, nil
}

func (a *mqlAtlassianAdminOrganization) policies() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*admin.AdminConnection)
	admin := conn.Client()
	orgId := a.Id.Data
	policies, _, err := admin.Organization.Policy.Gets(context.Background(), orgId, "", "")
	if err != nil {
		return nil, err
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
			return nil, err
		}
		res = append(res, mqlAtlassianAdminPolicy)
	}
	return res, nil
}

func (a *mqlAtlassianAdminOrganization) domains() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*admin.AdminConnection)
	admin := conn.Client()
	orgId := a.Id.Data
	domains, _, err := admin.Organization.Domains(context.Background(), orgId, "")
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	for _, domain := range domains.Data {
		mqlAtlassianAdminDomain, err := CreateResource(a.MqlRuntime, "atlassian.admin.organization.domain",
			map[string]*llx.RawData{
				"id":   llx.StringData(domain.ID),
				"name": llx.StringData(domain.Attributes.Name),
				"type": llx.StringData(domain.Type),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAtlassianAdminDomain)
	}
	return res, nil
}

func (a *mqlAtlassianAdminOrganizationPolicy) id() (string, error) {
	return a.Id.Data, nil
}
