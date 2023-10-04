package resources

import (
	"context"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers/atlassian/connection"
)

func (a *mqlAtlassianJira) id() (string, error) {
	return "wip", nil
}

func (a *mqlAtlassianJira) users() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AtlassianConnection)
	jira := conn.Jira()
	users, response, err := jira.User.Search.Do(context.Background(), "", " ", 0, 1000)
	if err != nil {
		log.Fatal().Err(err)
	}
	if response.Status != "200 OK" {
		log.Fatal().Msgf("Received response: %s\n", response.Status)
	}

	res := []interface{}{}
	for _, user := range users {
		mqlAtlassianJiraUser, err := CreateResource(a.MqlRuntime, "atlassian.jira.user",
			map[string]*llx.RawData{
				"id":   llx.StringData(user.AccountID),
				"name": llx.StringData(user.DisplayName),
				"type": llx.StringData(user.AccountType),
			})
		if err != nil {
			log.Fatal().Err(err)
		}
		res = append(res, mqlAtlassianJiraUser)
	}
	return res, nil
}

func (a *mqlAtlassianJiraUser) id() (string, error) {
	return a.Id.Data, nil
}
