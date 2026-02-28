// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gitlab/connection"
	"go.mondoo.com/mql/v13/types"
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
	var markedForDeletionOn *time.Time
	if grp.MarkedForDeletionOn != nil {
		t := time.Time(*grp.MarkedForDeletionOn)
		markedForDeletionOn = &t
	}
	args["markedForDeletionOn"] = llx.TimeDataPtr(markedForDeletionOn)
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
		// Convert ISOTime to time.Time for markedForDeletionOn
		var markedForDeletionOn *time.Time
		if subgroup.MarkedForDeletionOn != nil {
			t := time.Time(*subgroup.MarkedForDeletionOn)
			markedForDeletionOn = &t
		}

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
			"markedForDeletionOn":            llx.TimeDataPtr(markedForDeletionOn),
			"allowedEmailDomainsList":        llx.StringData(subgroup.AllowedEmailDomainsList),
			"lfsEnabled":                     llx.BoolData(subgroup.LFSEnabled),
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

// id function for gitlab.group.pushRule
func (r *mqlGitlabGroupPushRule) id() (string, error) {
	return "gitlab.group.pushRule/" + strconv.FormatInt(r.Id.Data, 10), nil
}

// pushRules fetches push rules for the group
func (g *mqlGitlabGroup) pushRules() (*mqlGitlabGroupPushRule, error) {
	conn := g.MqlRuntime.Connection.(*connection.GitLabConnection)

	groupID := int(g.Id.Data)
	rules, _, err := conn.Client().Groups.GetGroupPushRules(groupID)
	if err != nil {
		return nil, err
	}

	ruleInfo := map[string]*llx.RawData{
		"id":                         llx.IntData(rules.ID),
		"commitMessageRegex":         llx.StringData(rules.CommitMessageRegex),
		"commitMessageNegativeRegex": llx.StringData(rules.CommitMessageNegativeRegex),
		"branchNameRegex":            llx.StringData(rules.BranchNameRegex),
		"denyDeleteTag":              llx.BoolData(rules.DenyDeleteTag),
		"memberCheck":                llx.BoolData(rules.MemberCheck),
		"preventSecrets":             llx.BoolData(rules.PreventSecrets),
		"authorEmailRegex":           llx.StringData(rules.AuthorEmailRegex),
		"fileNameRegex":              llx.StringData(rules.FileNameRegex),
		"maxFileSize":                llx.IntData(rules.MaxFileSize),
		"commitCommitterCheck":       llx.BoolData(rules.CommitCommitterCheck),
		"commitCommitterNameCheck":   llx.BoolData(rules.CommitCommitterNameCheck),
		"rejectUnsignedCommits":      llx.BoolData(rules.RejectUnsignedCommits),
		"rejectNonDCOCommits":        llx.BoolData(rules.RejectNonDCOCommits),
		"createdAt":                  llx.TimeDataPtr(rules.CreatedAt),
	}

	mqlRule, err := CreateResource(g.MqlRuntime, "gitlab.group.pushRule", ruleInfo)
	if err != nil {
		return nil, err
	}

	return mqlRule.(*mqlGitlabGroupPushRule), nil
}

// id function for gitlab.group.accessToken
func (t *mqlGitlabGroupAccessToken) id() (string, error) {
	return "gitlab.group.accessToken/" + strconv.FormatInt(t.Id.Data, 10), nil
}

// accessTokens fetches the list of access tokens for the group
func (g *mqlGitlabGroup) accessTokens() ([]any, error) {
	conn := g.MqlRuntime.Connection.(*connection.GitLabConnection)

	groupID := int(g.Id.Data)

	perPage := int64(50)
	page := int64(1)
	var allTokens []*gitlab.GroupAccessToken

	for {
		tokens, resp, err := conn.Client().GroupAccessTokens.ListGroupAccessTokens(groupID, &gitlab.ListGroupAccessTokensOptions{
			ListOptions: gitlab.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		})
		if err != nil {
			return nil, err
		}

		allTokens = append(allTokens, tokens...)

		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	var mqlTokens []any
	for _, token := range allTokens {
		var expiresAt *time.Time
		if token.ExpiresAt != nil {
			t := time.Time(*token.ExpiresAt)
			expiresAt = &t
		}

		tokenInfo := map[string]*llx.RawData{
			"id":          llx.IntData(token.ID),
			"name":        llx.StringData(token.Name),
			"revoked":     llx.BoolData(token.Revoked),
			"active":      llx.BoolData(token.Active),
			"scopes":      llx.ArrayData(convert.SliceAnyToInterface(token.Scopes), types.String),
			"createdAt":   llx.TimeDataPtr(token.CreatedAt),
			"expiresAt":   llx.TimeDataPtr(expiresAt),
			"lastUsedAt":  llx.TimeDataPtr(token.LastUsedAt),
			"accessLevel": llx.IntData(int64(token.AccessLevel)),
		}

		mqlToken, err := CreateResource(g.MqlRuntime, "gitlab.group.accessToken", tokenInfo)
		if err != nil {
			return nil, err
		}

		mqlTokens = append(mqlTokens, mqlToken)
	}

	return mqlTokens, nil
}

// id function for gitlab.group.deployToken
func (t *mqlGitlabGroupDeployToken) id() (string, error) {
	return "gitlab.group.deployToken/" + strconv.FormatInt(t.Id.Data, 10), nil
}

// deployTokens fetches the list of deploy tokens for the group
func (g *mqlGitlabGroup) deployTokens() ([]any, error) {
	conn := g.MqlRuntime.Connection.(*connection.GitLabConnection)

	groupID := int(g.Id.Data)

	perPage := int64(50)
	page := int64(1)
	var allTokens []*gitlab.DeployToken

	for {
		tokens, resp, err := conn.Client().DeployTokens.ListGroupDeployTokens(groupID, &gitlab.ListGroupDeployTokensOptions{
			ListOptions: gitlab.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		})
		if err != nil {
			return nil, err
		}

		allTokens = append(allTokens, tokens...)

		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	var mqlTokens []any
	for _, token := range allTokens {
		tokenInfo := map[string]*llx.RawData{
			"id":        llx.IntData(token.ID),
			"name":      llx.StringData(token.Name),
			"username":  llx.StringData(token.Username),
			"expiresAt": llx.TimeDataPtr(token.ExpiresAt),
			"revoked":   llx.BoolData(token.Revoked),
			"expired":   llx.BoolData(token.Expired),
			"scopes":    llx.ArrayData(convert.SliceAnyToInterface(token.Scopes), types.String),
		}

		mqlToken, err := CreateResource(g.MqlRuntime, "gitlab.group.deployToken", tokenInfo)
		if err != nil {
			return nil, err
		}

		mqlTokens = append(mqlTokens, mqlToken)
	}

	return mqlTokens, nil
}

// id function for gitlab.group.protectedBranch
func (b *mqlGitlabGroupProtectedBranch) id() (string, error) {
	return "gitlab.group.protectedBranch/" + strconv.FormatInt(b.Id.Data, 10), nil
}

// protectedBranches fetches the list of protected branches for the group
func (g *mqlGitlabGroup) protectedBranches() ([]any, error) {
	conn := g.MqlRuntime.Connection.(*connection.GitLabConnection)

	groupID := int(g.Id.Data)

	perPage := int64(50)
	page := int64(1)
	var allBranches []*gitlab.GroupProtectedBranch

	for {
		branches, resp, err := conn.Client().GroupProtectedBranches.ListProtectedBranches(groupID, &gitlab.ListGroupProtectedBranchesOptions{
			ListOptions: gitlab.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		})
		if err != nil {
			return nil, err
		}

		allBranches = append(allBranches, branches...)

		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	var mqlBranches []any
	for _, branch := range allBranches {
		branchInfo := map[string]*llx.RawData{
			"id":                        llx.IntData(branch.ID),
			"name":                      llx.StringData(branch.Name),
			"allowForcePush":            llx.BoolData(branch.AllowForcePush),
			"codeOwnerApprovalRequired": llx.BoolData(branch.CodeOwnerApprovalRequired),
		}

		mqlBranch, err := CreateResource(g.MqlRuntime, "gitlab.group.protectedBranch", branchInfo)
		if err != nil {
			return nil, err
		}

		mqlBranches = append(mqlBranches, mqlBranch)
	}

	return mqlBranches, nil
}
