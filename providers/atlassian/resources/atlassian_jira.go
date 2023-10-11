package resources

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/providers/atlassian/connection"
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
				"id":      llx.StringData(user.AccountID),
				"name":    llx.StringData(user.DisplayName),
				"type":    llx.StringData(user.AccountType),
				"picture": llx.StringData(user.AvatarUrls.One6X16),
			})
		if err != nil {
			log.Fatal().Err(err)
		}
		res = append(res, mqlAtlassianJiraUser)
	}
	return res, nil
}

func (a *mqlAtlassianJiraUser) applicationRoles() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AtlassianConnection)
	jira := conn.Jira()
	expands := []string{"groups", "applicationRoles"}
	user, response, err := jira.User.Get(context.Background(), a.Id.Data, expands)
	if err != nil {
		log.Fatal().Err(err)
	}
	if response.Status != "200 OK" {
		log.Fatal().Msgf("Received response: %s\n", response.Status)
	}
	roles := user.ApplicationRoles

	res := []interface{}{}
	for _, role := range roles.Items {
		mqlAtlassianJiraUserRole, err := CreateResource(a.MqlRuntime, "atlassian.jira.applicationRole",
			map[string]*llx.RawData{
				"id":   llx.StringData(role.Key),
				"name": llx.StringData(role.Name),
			})
		if err != nil {
			log.Fatal().Err(err)
		}
		res = append(res, mqlAtlassianJiraUserRole)
	}
	return res, nil
}

func (a *mqlAtlassianJiraUser) groups() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AtlassianConnection)
	jira := conn.Jira()
	groups, response, err := jira.Group.Bulk(context.Background(), nil, 0, 1000)
	if err != nil {
		log.Fatal().Err(err)
	}
	if response.Status != "200 OK" {
		log.Fatal().Msgf("Received response: %s\n", response.Status)
	}

	res := []interface{}{}
	for _, group := range groups.Values {
		mqlAtlassianJiraUserGroup, err := CreateResource(a.MqlRuntime, "atlassian.jira.group",
			map[string]*llx.RawData{
				"id":   llx.StringData(group.GroupID),
				"name": llx.StringData(group.Name),
			})
		if err != nil {
			log.Fatal().Err(err)
		}
		res = append(res, mqlAtlassianJiraUserGroup)
	}
	return res, nil
}

func (a *mqlAtlassianJira) groups() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AtlassianConnection)
	jira := conn.Jira()
	groups, response, err := jira.Group.Bulk(context.Background(), nil, 0, 1000)
	if err != nil {
		log.Fatal().Err(err)
	}
	if response.Status != "200 OK" {
		log.Fatal().Msgf("Received response: %s\n", response.Status)
	}

	res := []interface{}{}
	for _, group := range groups.Values {
		mqlAtlassianJiraUserGroup, err := CreateResource(a.MqlRuntime, "atlassian.jira.group",
			map[string]*llx.RawData{
				"id":   llx.StringData(group.GroupID),
				"name": llx.StringData(group.Name),
			})
		if err != nil {
			log.Fatal().Err(err)
		}
		res = append(res, mqlAtlassianJiraUserGroup)
	}
	return res, nil
}

func (a *mqlAtlassianJira) serverInfos() (*mqlAtlassianJiraServerInfo, error) {
	conn := a.MqlRuntime.Connection.(*connection.AtlassianConnection)
	jira := conn.Jira()
	info, response, err := jira.Server.Info(context.Background())
	if err != nil {
		log.Fatal().Err(err)
	}
	if response.Status != "200 OK" {
		log.Fatal().Msgf("Received response: %s\n", response.Status)
	}
	res, err := CreateResource(a.MqlRuntime, "atlassian.jira.serverInfo",
		map[string]*llx.RawData{
			"baseUrl":        llx.StringData(info.BaseURL),
			"serverTitle":    llx.StringData(info.ServerTitle),
			"buildNumber":    llx.IntData(int64(info.BuildNumber)),
			"deploymentType": llx.StringData(info.DeploymentType),
		})
	return res.(*mqlAtlassianJiraServerInfo), err
}

func (a *mqlAtlassianJira) projects() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AtlassianConnection)
	jira := conn.Jira()
	projects, response, err := jira.Project.Search(context.Background(), nil, 0, 1000)
	if err != nil {
		log.Fatal().Err(err)
	}
	if response.Status != "200 OK" {
		log.Fatal().Msgf("Received response: %s\n", response.Status)
	}

	res := []interface{}{}
	for _, project := range projects.Values {
		mqlAtlassianJiraProject, err := CreateResource(a.MqlRuntime, "atlassian.jira.project",
			map[string]*llx.RawData{
				"id":       llx.StringData(project.ID),
				"name":     llx.StringData(project.Name),
				"uuid":     llx.StringData(project.UUID),
				"key":      llx.StringData(project.Key),
				"url":      llx.StringData(project.URL),
				"email":    llx.StringData(project.Email),
				"private":  llx.BoolData(project.IsPrivate),
				"deleted":  llx.BoolData(project.Deleted),
				"archived": llx.BoolData(project.Archived),
			})
		if err != nil {
			log.Fatal().Err(err)
		}
		res = append(res, mqlAtlassianJiraProject)
	}
	return res, nil
}

func (a *mqlAtlassianJiraProject) properties() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AtlassianConnection)
	jira := conn.Jira()
	properties, response, err := jira.Project.Property.Gets(context.Background(), a.Id.Data)
	if err != nil {
		log.Fatal().Err(err)
	}
	if response.Status != "200 OK" {
		log.Fatal().Msgf("Received response: %s\n", response.Status)
	}

	res := []interface{}{}
	for _, property := range properties.Keys {
		fmt.Println(property.Key)
		mqlAtlassianJiraProjectProperty, err := CreateResource(a.MqlRuntime, "atlassian.jira.project.property",
			map[string]*llx.RawData{
				"id": llx.StringData(property.Key),
			})
		if err != nil {
			log.Fatal().Err(err)
		}
		res = append(res, mqlAtlassianJiraProjectProperty)
	}
	return res, nil
}

func (a *mqlAtlassianJiraProjectProperty) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAtlassianJiraUser) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAtlassianJiraGroup) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAtlassianJiraProject) id() (string, error) {
	return a.Id.Data, nil
}
