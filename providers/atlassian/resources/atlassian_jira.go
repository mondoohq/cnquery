// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/atlassian/connection/jira"
	"go.mondoo.com/mql/v13/types"
)

const (
	JIRA_TIME_FORMAT        = "2006-01-02T15:04:05.999-0700"
	JIRA_SEARCH_MAX_RESULTS = 1000
)

func (a *mqlAtlassianJira) id() (string, error) {
	return "jira", nil
}

func (a *mqlAtlassianJira) users() ([]any, error) {
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
					"picture": llx.StringData(user.AvatarURLs.One6X16),
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

func (a *mqlAtlassianJiraUser) applicationRoles() ([]any, error) {
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

	res := []any{}
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

func (a *mqlAtlassianJiraUser) groups() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*jira.JiraConnection)
	if !ok {
		return nil, errors.New("Current connection does not allow jira access")
	}
	jira := conn.Client()
	groups, _, err := jira.Group.Bulk(context.Background(), nil, 0, 1000)
	if err != nil {
		return nil, err
	}
	res := []any{}
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

func (a *mqlAtlassianJira) groups() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*jira.JiraConnection)
	if !ok {
		return nil, errors.New("Current connection does not allow jira access")
	}
	jira := conn.Client()
	groups, _, err := jira.Group.Bulk(context.Background(), nil, 0, 1000)
	if err != nil {
		return nil, err
	}
	res := []any{}
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

func (a *mqlAtlassianJira) projects() ([]any, error) {
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

func (a *mqlAtlassianJira) issues() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*jira.JiraConnection)
	if !ok {
		return nil, errors.New("Current connection does not allow jira access")
	}
	jira := conn.Client()
	validate := ""
	jql := "order by created DESC"
	fields := []string{"created", "creator", "status", "project", "description", "issuetype"}
	expands := []string{"changelog", "renderedFields", "names", "schema", "transitions", "operations", "editmeta"}

	res := []any{}
	startAt := 0
	total := JIRA_SEARCH_MAX_RESULTS

	for startAt < total {
		issues, _, err := jira.Issue.Search.Get(context.Background(), jql, fields, expands, startAt, JIRA_SEARCH_MAX_RESULTS, validate)
		if err != nil {
			return nil, err
		}
		for _, issue := range issues.Issues {
			var createdAt *time.Time
			if issue.Fields.Created != nil {
				t := time.Time(*issue.Fields.Created).UTC()
				createdAt = &t
			}

			creator := issue.Fields.Creator
			mqlAtlassianJiraUser, err := CreateResource(a.MqlRuntime, "atlassian.jira.user",
				map[string]*llx.RawData{
					"id":      llx.StringData(creator.AccountID),
					"name":    llx.StringData(creator.DisplayName),
					"type":    llx.StringData(creator.AccountType),
					"picture": llx.StringData(creator.AvatarURLs.One6X16),
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
					"createdAt":   llx.TimeDataPtr(createdAt),
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

func (a *mqlAtlassianJiraProject) properties() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*jira.JiraConnection)
	if !ok {
		return nil, errors.New("Current connection does not allow jira access")
	}
	jira := conn.Client()
	properties, _, err := jira.Project.Property.Gets(context.Background(), a.Id.Data)
	if err != nil {
		return nil, err
	}

	res := []any{}
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

// auditRecords fetches all Jira audit log records (most recent first), paginating
// through the Jira REST API in pages of up to 1000 records.
func (a *mqlAtlassianJira) auditRecords() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*jira.JiraConnection)
	if !ok {
		return nil, errors.New("Current connection does not allow jira access")
	}
	jiraClient := conn.Client()

	res := []any{}
	offset := 0
	// Safety bound to guard against misbehaving servers that never signal end-of-results.
	const maxOffset = 100000

	for offset < maxOffset {
		page, _, err := jiraClient.Audit.Get(context.Background(), nil, offset, JIRA_SEARCH_MAX_RESULTS)
		if err != nil {
			return nil, err
		}
		if page == nil {
			break
		}

		for _, record := range page.Records {
			if record == nil {
				continue
			}

			var createdAt *time.Time
			if record.Created != "" {
				// Audit records use RFC3339-ish timestamps; tolerate parse failures.
				if t, perr := time.Parse(time.RFC3339, record.Created); perr == nil {
					ut := t.UTC()
					createdAt = &ut
				} else if t, perr := time.Parse(JIRA_TIME_FORMAT, record.Created); perr == nil {
					ut := t.UTC()
					createdAt = &ut
				}
			}

			var objectItem any
			if record.ObjectItem != nil {
				objectItem = map[string]any{
					"id":         record.ObjectItem.ID,
					"name":       record.ObjectItem.Name,
					"typeName":   record.ObjectItem.TypeName,
					"parentId":   record.ObjectItem.ParentID,
					"parentName": record.ObjectItem.ParentName,
				}
			}

			changedValues := []any{}
			for _, cv := range record.ChangedValues {
				if cv == nil {
					continue
				}
				changedValues = append(changedValues, map[string]any{
					"fieldName":   cv.FieldName,
					"changedFrom": cv.ChangedFrom,
					"changedTo":   cv.ChangedTo,
				})
			}

			associatedItems := []any{}
			for _, ai := range record.AssociatedItems {
				if ai == nil {
					continue
				}
				associatedItems = append(associatedItems, map[string]any{
					"id":         ai.ID,
					"name":       ai.Name,
					"typeName":   ai.TypeName,
					"parentId":   ai.ParentID,
					"parentName": ai.ParentName,
				})
			}

			args := map[string]*llx.RawData{
				"id":              llx.IntData(int64(record.ID)),
				"summary":         llx.StringData(record.Summary),
				"category":        llx.StringData(record.Category),
				"eventSource":     llx.StringData(record.EventSource),
				"description":     llx.StringData(record.Description),
				"authorKey":       llx.StringData(record.AuthorKey),
				"remoteAddress":   llx.StringData(record.RemoteAddress),
				"createdAt":       llx.TimeDataPtr(createdAt),
				"objectItem":      llx.DictData(objectItem),
				"changedValues":   llx.ArrayData(changedValues, types.Dict),
				"associatedItems": llx.ArrayData(associatedItems, types.Dict),
			}

			mqlAuditRecord, err := CreateResource(a.MqlRuntime, "atlassian.jira.auditRecord", args)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAuditRecord)
		}

		offset += len(page.Records)
		if len(page.Records) < JIRA_SEARCH_MAX_RESULTS {
			break
		}
	}
	return res, nil
}

func (a *mqlAtlassianJiraAuditRecord) id() (string, error) {
	return "atlassian.jira.auditRecord/" + strconv.FormatInt(a.Id.Data, 10), nil
}

// mqlAtlassianJiraPermissionSchemeInternal caches state needed by lazy methods.
// projectKey is the parent project's key, used by grants() to call the
// project-scoped permission API (which can return a different scheme/ID than
// the global Permission.Scheme.Get endpoint).
type mqlAtlassianJiraPermissionSchemeInternal struct {
	cacheProjectKey string
}

// permissionScheme returns the permission scheme assigned to this Jira project.
func (a *mqlAtlassianJiraProject) permissionScheme() (*mqlAtlassianJiraPermissionScheme, error) {
	conn, ok := a.MqlRuntime.Connection.(*jira.JiraConnection)
	if !ok {
		return nil, errors.New("Current connection does not allow jira access")
	}
	jiraClient := conn.Client()

	projectKey := a.Key.Data
	if projectKey == "" {
		projectKey = a.Id.Data
	}
	if projectKey == "" {
		a.PermissionScheme.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	scheme, _, err := jiraClient.Project.Permission.Get(context.Background(), projectKey, []string{"permissions", "user", "group", "projectRole", "field", "all"})
	if err != nil {
		return nil, err
	}
	if scheme == nil {
		a.PermissionScheme.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	res, err := CreateResource(a.MqlRuntime, "atlassian.jira.permissionScheme",
		map[string]*llx.RawData{
			"id":          llx.IntData(int64(scheme.ID)),
			"name":        llx.StringData(scheme.Name),
			"description": llx.StringData(scheme.Description),
		})
	if err != nil {
		return nil, err
	}

	mqlScheme := res.(*mqlAtlassianJiraPermissionScheme)
	// Cache the project key so the grants() fallback can re-fetch via the
	// project-scoped endpoint instead of the global one.
	mqlScheme.cacheProjectKey = projectKey
	// Cache the grants we already have so grants() doesn't have to call out again.
	if scheme.Permissions != nil {
		grants := []any{}
		for _, grant := range scheme.Permissions {
			if grant == nil {
				continue
			}
			holderType := ""
			holderParam := ""
			if grant.Holder != nil {
				holderType = grant.Holder.Type
				holderParam = grant.Holder.Parameter
			}
			mqlGrant, err := CreateResource(a.MqlRuntime, "atlassian.jira.permissionScheme.grant",
				map[string]*llx.RawData{
					"id":              llx.StringData(strconv.Itoa(grant.ID)),
					"permission":      llx.StringData(grant.Permission),
					"holderType":      llx.StringData(holderType),
					"holderParameter": llx.StringData(holderParam),
				})
			if err != nil {
				return nil, err
			}
			grants = append(grants, mqlGrant)
		}
		mqlScheme.Grants = plugin.TValue[[]any]{Data: grants, State: plugin.StateIsSet}
	}

	return mqlScheme, nil
}

func (a *mqlAtlassianJiraPermissionScheme) id() (string, error) {
	return "atlassian.jira.permissionScheme/" + strconv.FormatInt(a.Id.Data, 10), nil
}

// grants is a fallback if grants weren't pre-populated by permissionScheme().
// In practice this rarely runs because the parent caches grants on creation.
// Uses the project-scoped endpoint (matching the parent fetch) so the grant
// list aligns with the scheme returned for the project — the global
// Permission.Scheme.Get endpoint can return a different scheme/grant set.
func (a *mqlAtlassianJiraPermissionScheme) grants() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*jira.JiraConnection)
	if !ok {
		return nil, errors.New("Current connection does not allow jira access")
	}
	jiraClient := conn.Client()

	if a.cacheProjectKey == "" {
		return []any{}, nil
	}

	scheme, _, err := jiraClient.Project.Permission.Get(context.Background(), a.cacheProjectKey, []string{"permissions", "user", "group", "projectRole", "field", "all"})
	if err != nil {
		return nil, err
	}
	res := []any{}
	if scheme == nil {
		return res, nil
	}
	for _, grant := range scheme.Permissions {
		if grant == nil {
			continue
		}
		holderType := ""
		holderParam := ""
		if grant.Holder != nil {
			holderType = grant.Holder.Type
			holderParam = grant.Holder.Parameter
		}
		mqlGrant, err := CreateResource(a.MqlRuntime, "atlassian.jira.permissionScheme.grant",
			map[string]*llx.RawData{
				"id":              llx.StringData(strconv.Itoa(grant.ID)),
				"permission":      llx.StringData(grant.Permission),
				"holderType":      llx.StringData(holderType),
				"holderParameter": llx.StringData(holderParam),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlGrant)
	}
	return res, nil
}

func (a *mqlAtlassianJiraPermissionSchemeGrant) id() (string, error) {
	return "atlassian.jira.permissionScheme.grant/" + a.Id.Data, nil
}
