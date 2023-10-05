package resources

import (
	"context"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers/atlassian/connection"
)

func (a *mqlAtlassianConfluence) id() (string, error) {
	return "wip", nil
}

func (a *mqlAtlassianConfluence) users() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AtlassianConnection)
	confluence := conn.Confluence()
	cql := "type = user"
	users, response, err := confluence.Search.Users(context.Background(), cql, 0, 1000, nil)
	if err != nil {
		log.Fatal().Err(err)
	}
	if response.Status != "200 OK" {
		log.Fatal().Msgf("Received response: %s\n", response.Status)
	}
	res := []interface{}{}
	for _, user := range users.Results {
		mqlAtlassianConfluenceUser, err := CreateResource(a.MqlRuntime, "atlassian.confluence.user",
			map[string]*llx.RawData{
				"id":   llx.StringData(user.User.AccountID),
				"name": llx.StringData(user.User.DisplayName),
			})
		if err != nil {
			log.Fatal().Err(err)
		}
		res = append(res, mqlAtlassianConfluenceUser)
	}
	return res, nil
}

func (a *mqlAtlassianConfluenceUser) id() (string, error) {
	return a.Id.Data, nil
}
