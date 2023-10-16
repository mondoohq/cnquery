package resources

import (
	"context"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/providers/atlassian/connection/scim"
)

func (a *mqlAtlassianScim) id() (string, error) {
	return "scim", nil
}

func (a *mqlAtlassianScim) users() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*scim.ScimConnection)
	admin := conn.Client()
	directoryID := "9766e0d0-5319-494c-b216-f85c3882490f"
	scimUsers, response, err := admin.SCIM.User.Gets(context.Background(), directoryID, nil, 0, 1000)
	if err != nil {
		log.Fatal().Err(err)
	}
	if response.Status != "200 OK" {
		log.Fatal().Msgf("Received response: %s\n", response.Status)
	}
	res := []interface{}{}
	for _, scimUser := range scimUsers.Resources {
		mqlAtlassianAdminSCIMuser, err := CreateResource(a.MqlRuntime, "atlassian.scim.organization.scim.user",
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

func (a *mqlAtlassianScim) groups() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*scim.ScimConnection)
	admin := conn.Client()
	directoryID := "9766e0d0-5319-494c-b216-f85c3882490f"
	scimGroup, response, err := admin.SCIM.Group.Gets(context.Background(), directoryID, "", 0, 1000)
	if err != nil {
		log.Fatal().Err(err)
	}
	if response.Status != "200 OK" {
		log.Fatal().Msgf("Received response: %s\n", response.Status)
	}
	res := []interface{}{}
	for _, scimGroup := range scimGroup.Resources {
		mqlAtlassianAdminSCIMgroup, err := CreateResource(a.MqlRuntime, "atlassian.scim.organization.scim.group",
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

func (a *mqlAtlassianScimUser) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAtlassianScimGroup) id() (string, error) {
	return a.Id.Data, nil
}
