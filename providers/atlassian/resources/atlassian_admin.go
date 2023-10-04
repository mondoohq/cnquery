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
	fmt.Println("orgId: ", orgId)
	users, response, err := admin.Organization.Users(context.Background(), orgId, "")
	fmt.Println(response.Status)
	fmt.Println(response.Endpoint)
	fmt.Println(users.Data)
	if err != nil {
		log.Fatal().Err(err)
	}
	if response.Status != "200 OK" {
		log.Fatal().Msgf("Received response: %s\n", response.Status)
	}
	res := []interface{}{}
	fmt.Println("Num of users: ", len(users.Data))
	for _, user := range users.Data {
		fmt.Println("adding user: ", user.Name)
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
