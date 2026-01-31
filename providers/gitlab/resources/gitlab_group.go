// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/gitlab/connection"
)

func (g *mqlGitlabGroup) id() (string, error) {
	return "gitlab.group/" + strconv.FormatInt(g.Id.Data, 10), nil
}

func (u *mqlGitlabUser) id() (string, error) {
	return "gitlab.user/" + strconv.FormatInt(u.Id.Data, 10), nil
}

func (m *mqlGitlabMember) id() (string, error) {
	return "gitlab.member/" + strconv.FormatInt(m.Id.Data, 10), nil
}

// init initializes the gitlab group with the arguments
// see https://docs.gitlab.com/ee/api/groups.html#new-group
func initGitlabGroup(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.GitLabConnection)
	grp, err := conn.Group()
	if err != nil {
		return nil, nil, err
	}

	args["id"] = llx.IntData(int64(grp.ID))
	args["name"] = llx.StringData(grp.Name)
	args["path"] = llx.StringData(grp.Path)
	args["fullName"] = llx.StringData(grp.FullName)
	args["fullPath"] = llx.StringData(grp.FullPath)
	args["description"] = llx.StringData(grp.Description)
	args["createdAt"] = llx.TimeDataPtr(grp.CreatedAt)
	args["webURL"] = llx.StringData(string(grp.WebURL))
	args["visibility"] = llx.StringData(string(grp.Visibility))
	args["requireTwoFactorAuthentication"] = llx.BoolData(grp.RequireTwoFactorAuth)
	args["preventForkingOutsideGroup"] = llx.BoolData(grp.PreventForkingOutsideGroup)
	args["mentionsDisabled"] = llx.BoolData(grp.MentionsDisabled)
	args["emailsDisabled"] = llx.BoolData(!grp.EmailsEnabled)
	args["requestAccessEnabled"] = llx.BoolData(grp.RequestAccessEnabled)
	// Convert ISOTime to time.Time
	if grp.MarkedForDeletionOn != nil {
		t := time.Time(*grp.MarkedForDeletionOn)
		args["markedForDeletionOn"] = llx.TimeDataPtr(&t)
	}
	args["allowedEmailDomainsList"] = llx.StringData(grp.AllowedEmailDomainsList)
	args["lfsEnabled"] = llx.BoolData(grp.LFSEnabled)

	return args, nil, nil
}

// projects lists all projects that belong to a group
// see https://docs.gitlab.com/ee/api/projects.html
func (g *mqlGitlabGroup) projects() ([]any, error) {
	conn := g.MqlRuntime.Connection.(*connection.GitLabConnection)

	if g.Path.Error != nil {
		return nil, g.Path.Error
	}
	gid := int(g.Id.Data)

	// Fetch all projects with pagination
	perPage := int64(50)
	page := int64(1)
	var allProjects []*gitlab.Project

	for {
		projects, resp, err := conn.Client().Groups.ListGroupProjects(gid, &gitlab.ListGroupProjectsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		})
		if err != nil {
			return nil, err
		}

		allProjects = append(allProjects, projects...)

		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	var mqlProjects []any
	for _, prj := range allProjects {
		mqlProject, err := CreateResource(g.MqlRuntime, "gitlab.project", getGitlabProjectArgs(prj))
		if err != nil {
			return nil, err
		}
		mqlProjects = append(mqlProjects, mqlProject)
	}

	return mqlProjects, nil
}

// members fetches the list of members in the group with their roles
func (g *mqlGitlabGroup) members() ([]any, error) {
	conn := g.MqlRuntime.Connection.(*connection.GitLabConnection)

	groupID := int(g.Id.Data)

	members, _, err := conn.Client().Groups.ListAllGroupMembers(groupID, nil)
	if err != nil {
		return nil, err
	}

	var mqlMembers []any
	for _, member := range members {
		role := mapAccessLevelToRole(int(member.AccessLevel))

		mqlUser, err := CreateResource(g.MqlRuntime, "gitlab.user", map[string]*llx.RawData{
			"id":               llx.IntData(member.ID),
			"username":         llx.StringData(member.Username),
			"name":             llx.StringData(member.Name),
			"state":            llx.StringData(member.State),
			"email":            llx.StringData(member.Email),
			"webURL":           llx.StringData(member.WebURL),
			"avatarURL":        llx.StringData(member.AvatarURL),
			"createdAt":        llx.TimeDataPtr(member.CreatedAt),
			"jobTitle":         llx.StringData(""),
			"organization":     llx.StringData(""),
			"location":         llx.StringData(""),
			"locked":           llx.BoolData(false),
			"bot":              llx.BoolData(false),
			"twoFactorEnabled": llx.BoolData(false),
		})
		if err != nil {
			return nil, err
		}

		memberInfo := map[string]*llx.RawData{
			"id":   llx.IntData(member.ID),
			"user": llx.ResourceData(mqlUser, "gitlab.user"),
			"role": llx.StringData(role),
		}

		mqlMember, err := CreateResource(g.MqlRuntime, "gitlab.member", memberInfo)
		if err != nil {
			return nil, err
		}

		mqlMembers = append(mqlMembers, mqlMember)
	}

	return mqlMembers, nil
}

// subgroups fetches the list of subgroups that belong to this group
func (g *mqlGitlabGroup) subgroups() ([]any, error) {
	conn := g.MqlRuntime.Connection.(*connection.GitLabConnection)

	groupID := int(g.Id.Data)

	// Fetch all subgroups with pagination
	perPage := int64(50)
	page := int64(1)
	var allSubgroups []*gitlab.Group

	for {
		subgroups, resp, err := conn.Client().Groups.ListSubGroups(groupID, &gitlab.ListSubGroupsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		})
		if err != nil {
			return nil, err
		}

		allSubgroups = append(allSubgroups, subgroups...)

		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	var mqlSubgroups []any
	for _, subgroup := range allSubgroups {
		subgroupArgs := map[string]*llx.RawData{
			"id":                             llx.IntData(int64(subgroup.ID)),
			"name":                           llx.StringData(subgroup.Name),
			"path":                           llx.StringData(subgroup.Path),
			"fullName":                       llx.StringData(subgroup.FullName),
			"fullPath":                       llx.StringData(subgroup.FullPath),
			"description":                    llx.StringData(subgroup.Description),
			"createdAt":                      llx.TimeDataPtr(subgroup.CreatedAt),
			"webURL":                         llx.StringData(string(subgroup.WebURL)),
			"visibility":                     llx.StringData(string(subgroup.Visibility)),
			"requireTwoFactorAuthentication": llx.BoolData(subgroup.RequireTwoFactorAuth),
			"preventForkingOutsideGroup":     llx.BoolData(subgroup.PreventForkingOutsideGroup),
			"mentionsDisabled":               llx.BoolData(subgroup.MentionsDisabled),
			"emailsDisabled":                 llx.BoolData(!subgroup.EmailsEnabled),
			"requestAccessEnabled":           llx.BoolData(subgroup.RequestAccessEnabled),
			"allowedEmailDomainsList":        llx.StringData(subgroup.AllowedEmailDomainsList),
			"lfsEnabled":                     llx.BoolData(subgroup.LFSEnabled),
		}

		// Convert ISOTime to time.Time for markedForDeletionOn
		if subgroup.MarkedForDeletionOn != nil {
			t := time.Time(*subgroup.MarkedForDeletionOn)
			subgroupArgs["markedForDeletionOn"] = llx.TimeDataPtr(&t)
		}

		mqlSubgroup, err := CreateResource(g.MqlRuntime, "gitlab.group", subgroupArgs)
		if err != nil {
			return nil, err
		}

		mqlSubgroups = append(mqlSubgroups, mqlSubgroup)
	}

	return mqlSubgroups, nil
}

// id function for gitlab.group.label
func (l *mqlGitlabGroupLabel) id() (string, error) {
	return strconv.FormatInt(l.Id.Data, 10), nil
}

// labels fetches the list of labels for the group
func (g *mqlGitlabGroup) labels() ([]any, error) {
	conn := g.MqlRuntime.Connection.(*connection.GitLabConnection)

	groupID := int(g.Id.Data)

	// Fetch all labels with pagination
	perPage := int64(50)
	page := int64(1)
	var allLabels []*gitlab.GroupLabel

	for {
		labels, resp, err := conn.Client().GroupLabels.ListGroupLabels(groupID, &gitlab.ListGroupLabelsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		})
		if err != nil {
			return nil, err
		}

		allLabels = append(allLabels, labels...)

		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	var mqlLabels []any
	for _, label := range allLabels {
		labelInfo := map[string]*llx.RawData{
			"id":                     llx.IntData(label.ID),
			"name":                   llx.StringData(label.Name),
			"color":                  llx.StringData(label.Color),
			"textColor":              llx.StringData(label.TextColor),
			"description":            llx.StringData(label.Description),
			"descriptionHtml":        llx.StringData(""), // Not in API response
			"openIssuesCount":        llx.IntData(label.OpenIssuesCount),
			"closedIssuesCount":      llx.IntData(label.ClosedIssuesCount),
			"openMergeRequestsCount": llx.IntData(label.OpenMergeRequestsCount),
			"subscribed":             llx.BoolData(label.Subscribed),
			"priority":               llx.IntData(label.Priority),
			"isProjectLabel":         llx.BoolData(label.IsProjectLabel),
		}

		mqlLabel, err := CreateResource(g.MqlRuntime, "gitlab.group.label", labelInfo)
		if err != nil {
			return nil, err
		}

		mqlLabels = append(mqlLabels, mqlLabel)
	}

	return mqlLabels, nil
}
