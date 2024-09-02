// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers/atlassian/connection/jira"
)

const (
	JIRA_TIME_FORMAT        = "2006-01-02T15:04:05.999-0700"
	JIRA_SEARCH_MAX_RESULTS = 1000
)

func (a *mqlAtlassianJira) id() (string, error) {
	return "jira", nil
}

func (a *mqlAtlassianJira) users() ([]interface{}, error) {
	conn, ok := a.MqlRuntime.Connection.(*jira.JiraConnection)
	if !ok {
		return nil, errors.New("Current connection does not allow jira access")
	}
	jira := conn.Client()

	res := []any{}
	startAt := 0

	for {
		users, _, err := jira.User.Search.Do(context.Background(), "", " ", startAt, JIRA_SEARCH_MAX_RESULTS)
		if err != nil {
			return nil, err
		}
		if len(users) == 0 {
			break
		}

		for _, user := range users {
			mqlAtlassianJiraUser, err := CreateResource(a.MqlRuntime, "atlassian.jira.user",
				map[string]*llx.RawData{
					"id":      llx.StringData(user.AccountID),
					"name":    llx.StringData(user.DisplayName),
					"type":    llx.StringData(user.AccountType),
					"picture": llx.StringData(user.AvatarUrls.One6X16),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAtlassianJiraUser)
		}

		startAt += len(users)
	}
	return res, nil
}

func (a *mqlAtlassianJiraUser) applicationRoles() ([]interface{}, error) {
	conn, ok := a.MqlRuntime.Connection.(*jira.JiraConnection)
	if !ok {
		return nil, errors.New("Current connection does not allow jira access")
	}
	jira := conn.Client()
	expands := []string{"groups", "applicationRoles"}
	user, _, err := jira.User.Get(context.Background(), a.Id.Data, expands)
	if err != nil {
		return nil, err
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
			return nil, err
		}
		res = append(res, mqlAtlassianJiraUserRole)
	}
	return res, nil
}

func (a *mqlAtlassianJiraUser) groups() ([]interface{}, error) {
	conn, ok := a.MqlRuntime.Connection.(*jira.JiraConnection)
	if !ok {
		return nil, errors.New("Current connection does not allow jira access")
	}
	jira := conn.Client()
	groups, _, err := jira.Group.Bulk(context.Background(), nil, 0, 1000)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	for _, group := range groups.Values {
		mqlAtlassianJiraUserGroup, err := CreateResource(a.MqlRuntime, "atlassian.jira.group",
			map[string]*llx.RawData{
				"id":   llx.StringData(group.GroupID),
				"name": llx.StringData(group.Name),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAtlassianJiraUserGroup)
	}
	return res, nil
}

func (a *mqlAtlassianJira) groups() ([]interface{}, error) {
	conn, ok := a.MqlRuntime.Connection.(*jira.JiraConnection)
	if !ok {
		return nil, errors.New("Current connection does not allow jira access")
	}
	jira := conn.Client()
	groups, _, err := jira.Group.Bulk(context.Background(), nil, 0, 1000)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	for _, group := range groups.Values {
		mqlAtlassianJiraUserGroup, err := CreateResource(a.MqlRuntime, "atlassian.jira.group",
			map[string]*llx.RawData{
				"id":   llx.StringData(group.GroupID),
				"name": llx.StringData(group.Name),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAtlassianJiraUserGroup)
	}
	return res, nil
}

func (a *mqlAtlassianJira) serverInfos() (*mqlAtlassianJiraServerInfo, error) {
	conn, ok := a.MqlRuntime.Connection.(*jira.JiraConnection)
	if !ok {
		return nil, errors.New("Current connection does not allow jira access")
	}
	jira := conn.Client()
	info, _, err := jira.Server.Info(context.Background())
	if err != nil {
		return nil, err
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
	conn, ok := a.MqlRuntime.Connection.(*jira.JiraConnection)
	if !ok {
		return nil, errors.New("Current connection does not allow jira access")
	}
	jira := conn.Client()

	res := []any{}
	startAt := 0
	total := JIRA_SEARCH_MAX_RESULTS

	for startAt < total {
		projects, _, err := jira.Project.Search(context.Background(), nil, startAt, JIRA_SEARCH_MAX_RESULTS)
		if err != nil {
			return nil, err
		}

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
				return nil, err
			}
			res = append(res, mqlAtlassianJiraProject)
		}

		total = projects.Total
		startAt += len(projects.Values)
	}
	return res, nil
}

func (a *mqlAtlassianJira) issues() ([]interface{}, error) {
	conn, ok := a.MqlRuntime.Connection.(*jira.JiraConnection)
	if !ok {
		return nil, errors.New("Current connection does not allow jira access")
	}
	jira := conn.Client()
	validate := ""
	jql := "order by created DESC"
	fields := []string{"created", "creator", "status", "project", "description", "issuetype"}
	expands := []string{"changelog", "renderedFields", "names", "schema", "transitions", "operations", "editmeta"}

	res := []interface{}{}
	startAt := 0
	total := JIRA_SEARCH_MAX_RESULTS

	for startAt < total {
		issues, _, err := jira.Issue.Search.Get(context.Background(), jql, fields, expands, startAt, JIRA_SEARCH_MAX_RESULTS, validate)
		if err != nil {
			return nil, err
		}
		for _, issue := range issues.Issues {
			created, err := time.Parse(JIRA_TIME_FORMAT, issue.Fields.Created)
			if err != nil {
				return nil, err
			}

			creator := issue.Fields.Creator
			mqlAtlassianJiraUser, err := CreateResource(a.MqlRuntime, "atlassian.jira.user",
				map[string]*llx.RawData{
					"id":      llx.StringData(creator.AccountID),
					"name":    llx.StringData(creator.DisplayName),
					"type":    llx.StringData(creator.AccountType),
					"picture": llx.StringData(creator.AvatarUrls.One6X16),
				})
			if err != nil {
				return nil, err
			}

			mqlAtlassianJiraIssue, err := CreateResource(a.MqlRuntime, "atlassian.jira.issue",
				map[string]*llx.RawData{
					"id":          llx.StringData(issue.ID),
					"project":     llx.StringData(issue.Fields.Project.Name),
					"projectKey":  llx.StringData(issue.Fields.Project.Key),
					"status":      llx.StringData(issue.Fields.Status.Name),
					"description": llx.StringData(issue.Fields.Description),
					"createdAt":   llx.TimeData(created.UTC()),
					"creator":     llx.AnyData(mqlAtlassianJiraUser),
					"typeName":    llx.StringData(issue.Fields.IssueType.Name),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAtlassianJiraIssue)
		}

		total = issues.Total
		startAt += len(issues.Issues)
	}

	return res, nil
}

func (a *mqlAtlassianJiraIssue) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAtlassianJiraProject) properties() ([]interface{}, error) {
	conn, ok := a.MqlRuntime.Connection.(*jira.JiraConnection)
	if !ok {
		return nil, errors.New("Current connection does not allow jira access")
	}
	jira := conn.Client()
	properties, _, err := jira.Project.Property.Gets(context.Background(), a.Id.Data)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for _, property := range properties.Keys {
		fmt.Println(property.Key)
		mqlAtlassianJiraProjectProperty, err := CreateResource(a.MqlRuntime, "atlassian.jira.project.property",
			map[string]*llx.RawData{
				"id": llx.StringData(property.Key),
			})
		if err != nil {
			return nil, err
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
