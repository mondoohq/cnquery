package resources

import (
	"context"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/providers/atlassian/connection"
)

func (a *mqlAtlassianAdmin) id() (string, error) {
	return "wip", nil
}

func (a *mqlAtlassianAdmin) organizations() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AtlassianConnection)
	admin := conn.Admin()
	organizations, _, err := admin.Organization.Gets(context.Background(), "")
	if err != nil {
		log.Fatal().Err(err)
	}
	res := []interface{}{}
	for _, org := range organizations.Data {
		res = append(res, org)
	}
	return res, nil
}

func (a *mqlAtlassianAdminOrganization) users() ([]interface{}, error) {
	//conn := a.MqlRuntime.Connection.(*connection.AtlassianConnection)
	//admin := conn.Admin()

	res := []interface{}{}
	return res, nil
}
