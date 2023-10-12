package resources

import (
	"context"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/providers/atlassian/connection"
)

func (a *mqlAtlassianAdmin) id() (string, error) {
	return "wip", nil
}

func (a *mqlAtlassianAdmin) organizations() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AtlassianConnection)
	admin := conn.Admin()
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
	conn := a.MqlRuntime.Connection.(*connection.AtlassianConnection)
	admin := conn.Admin()
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

type atlassianUser struct {
	AccountID string
	Name      string
	Type      string
	OrgID     string
}

func (a *mqlAtlassianAdminOrganization) users() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AtlassianConnection)

	jira := conn.Jira()
	confluence := conn.Confluence()

	jiraUsers, response, err := jira.User.Search.Do(context.Background(), "", " ", 0, 1000)
	if err != nil {
		log.Fatal().Err(err)
	}
	if response.Status != "200 OK" {
		log.Fatal().Msgf("Received response: %s\n", response.Status)
	}

	cql := "type = user"
	confluenceUsers, response, err := confluence.Search.Users(context.Background(), cql, 0, 1000, nil)
	if err != nil {
		log.Fatal().Err(err)
	}
	if response.Status != "200 OK" {
		log.Fatal().Msgf("Received response: %s\n", response.Status)
	}
	var atlassianUsers []atlassianUser
	for _, jiraUser := range jiraUsers {
		user := atlassianUser{
			AccountID: jiraUser.AccountID,
			Name:      jiraUser.DisplayName,
			Type:      jiraUser.AccountType,
			OrgID:     a.Id.Data,
		}
		atlassianUsers = append(atlassianUsers, user)
	}
	for _, confluenceUser := range confluenceUsers.Results {
		user := atlassianUser{
			AccountID: confluenceUser.User.AccountID,
			Name:      confluenceUser.User.DisplayName,
			Type:      confluenceUser.User.AccountType,
			OrgID:     a.Id.Data,
		}
		atlassianUsers = append(atlassianUsers, user)
	}

	//TODO: is there a better way to get unique users?
	var uniqueAtlassianUsers []atlassianUser
loopMark:
	for _, v := range atlassianUsers {
		for i, u := range uniqueAtlassianUsers {
			if v.AccountID == u.AccountID {
				uniqueAtlassianUsers[i] = v
				continue loopMark
			}
		}
		uniqueAtlassianUsers = append(uniqueAtlassianUsers, v)
	}

	if err != nil {
		log.Fatal().Err(err)
	}
	if response.Status != "200 OK" {
		log.Fatal().Msgf("Received response: %s\n", response.Status)
	}
	res := []interface{}{}
	for _, user := range uniqueAtlassianUsers {
		mqlAtlassianAdminUser, err := CreateResource(a.MqlRuntime, "atlassian.admin.organization.user",
			map[string]*llx.RawData{
				"id":    llx.StringData(user.AccountID),
				"name":  llx.StringData(user.Name),
				"type":  llx.StringData(user.Type),
				"orgId": llx.StringData(user.OrgID),
			})
		if err != nil {
			log.Fatal().Err(err)
		}
		res = append(res, mqlAtlassianAdminUser)
	}
	return res, nil
}

func (a *mqlAtlassianAdminOrganizationUser) lastActive() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AtlassianConnection)
	admin := conn.Admin()
	accountId := a.Id.Data
	orgId := a.OrgId.Data
	lastActive, response, err := admin.Organization.Directory.Activity(context.Background(), orgId, accountId)
	if err != nil {
		log.Fatal().Err(err)
	}
	if response.Status != "200 OK" {
		log.Fatal().Msgf("Received response: %s\n", response.Status)
	}

	res := []interface{}{}
	for _, access := range lastActive.Data.ProductAccess {
		mqlAtlassianAdminUserLastActive, err := CreateResource(a.MqlRuntime, "atlassian.admin.organization.user.lastActive",
			map[string]*llx.RawData{
				"id":         llx.StringData(access.Id),
				"url":        llx.StringData(access.Url),
				"key":        llx.StringData(access.Key),
				"lastActive": llx.StringData(access.LastActive),
			})
		if err != nil {
			log.Fatal().Err(err)
		}
		res = append(res, mqlAtlassianAdminUserLastActive)
	}
	return res, nil

}

func (a *mqlAtlassianAdminOrganizationUserLastActive) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAtlassianAdminOrganizationUser) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAtlassianAdminOrganization) policies() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AtlassianConnection)
	admin := conn.Admin()
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
	conn := a.MqlRuntime.Connection.(*connection.AtlassianConnection)
	admin := conn.Admin()
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
	conn := a.MqlRuntime.Connection.(*connection.AtlassianConnection)
	admin := conn.Admin()
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
