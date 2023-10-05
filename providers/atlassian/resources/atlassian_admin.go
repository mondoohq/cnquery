package resources

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers/atlassian/connection"
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

func (a *mqlAtlassianAdminOrganization) users() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AtlassianConnection)
	admin := conn.Admin()
	orgId := a.Id.Data
	users, response, err := admin.Organization.Users(context.Background(), orgId, "")
	if err != nil {
		log.Fatal().Err(err)
	}
	if response.Status != "200 OK" {
		log.Fatal().Msgf("Received response: %s\n", response.Status)
	}
	res := []interface{}{}
	for _, user := range users.Data {
		mqlAtlassianAdminUser, err := CreateResource(a.MqlRuntime, "atlassian.admin.organization.user",
			map[string]*llx.RawData{
				"id":   llx.StringData(user.AccountID),
				"name": llx.StringData(user.Name),
			})
		if err != nil {
			log.Fatal().Err(err)
		}
		res = append(res, mqlAtlassianAdminUser)
	}
	return res, nil
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
				"id":   llx.StringData(policy.ID),
				"type": llx.StringData(policy.Type),
				"name": llx.StringData(policy.Attributes.Name),
			})
		if err != nil {
			log.Fatal().Err(err)
		}
		res = append(res, mqlAtlassianAdminPolicy)
	}
	return res, nil
}
