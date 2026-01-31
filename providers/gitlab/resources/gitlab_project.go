// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers/gitlab/connection"
	"go.mondoo.com/cnquery/v12/types"
)

func getGitlabProjectArgs(prj *gitlab.Project) map[string]*llx.RawData {
	return map[string]*llx.RawData{
		"allowMergeOnSkippedPipeline": llx.BoolData(prj.AllowMergeOnSkippedPipeline),
		"archived":                    llx.BoolData(prj.Archived),
		"autocloseReferencedIssues":   llx.BoolData(prj.AutocloseReferencedIssues),
		"autoDevopsEnabled":           llx.BoolData(prj.AutoDevopsEnabled),
		"containerRegistryEnabled":    llx.BoolData(prj.ContainerRegistryEnabled),
		"createdAt":                   llx.TimeDataPtr(prj.CreatedAt),
		"defaultBranch":               llx.StringData(prj.DefaultBranch),
		"description":                 llx.StringData(prj.Description),
		"emailsDisabled":              llx.BoolData(!prj.EmailsEnabled),
		"emptyRepo":                   llx.BoolData(prj.EmptyRepo),
		"fullName":                    llx.StringData(prj.NameWithNamespace),
		"groupRunnersEnabled":         llx.BoolData(prj.GroupRunnersEnabled),
		"id":                          llx.IntData(int64(prj.ID)),
		"issuesEnabled":               llx.BoolData(prj.IssuesEnabled),
		"jobsEnabled":                 llx.BoolData(prj.JobsEnabled),
		"lfsEnabled":                  llx.BoolData(prj.LFSEnabled),
		"mergeRequestsEnabled":        llx.BoolData(prj.MergeRequestsEnabled),
		"mirror":                      llx.BoolData(prj.Mirror),
		"name":                        llx.StringData(prj.Name),
		"onlyAllowMergeIfAllDiscussionsAreResolved": llx.BoolData(prj.OnlyAllowMergeIfAllDiscussionsAreResolved),
		"onlyAllowMergeIfPipelineSucceeds":          llx.BoolData(prj.OnlyAllowMergeIfPipelineSucceeds),
		"packagesEnabled":                           llx.BoolData(prj.PackagesEnabled),
		"path":                                      llx.StringData(prj.Path),
		"removeSourceBranchAfterMerge":              llx.BoolData(prj.RemoveSourceBranchAfterMerge),
		"requirementsEnabled":                       llx.BoolData(prj.RequirementsEnabled),
		"serviceDeskEnabled":                        llx.BoolData(prj.ServiceDeskEnabled),
		"sharedRunnersEnabled":                      llx.BoolData(prj.SharedRunnersEnabled),
		"snippetsEnabled":                           llx.BoolData(prj.SnippetsEnabled),
		"visibility":                                llx.StringData(string(prj.Visibility)),
		"webURL":                                    llx.StringData(prj.WebURL),
		"wikiEnabled":                               llx.BoolData(prj.WikiEnabled),
		"forksCount":                                llx.IntData(prj.ForksCount),
		"starCount":                                 llx.IntData(prj.StarCount),
		"lastActivityAt":                            llx.TimeDataPtr(prj.LastActivityAt),
	}
}

func (g *mqlGitlabProject) id() (string, error) {
	return "gitlab.project/" + strconv.FormatInt(g.Id.Data, 10), nil
}

// init initializes the gitlab project with the arguments
func initGitlabProject(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.GitLabConnection)
	project, err := conn.Project()
	if err != nil {
		return nil, nil, err
	}

	args = getGitlabProjectArgs(project)
	return args, nil, nil
}

// approvalSettings fetches project approval settings
func (p *mqlGitlabProject) approvalSettings() (*mqlGitlabProjectApprovalSetting, error) {
	conn := p.MqlRuntime.Connection.(*connection.GitLabConnection)

	projectID := int(p.Id.Data)
	approvalConfig, _, err := conn.Client().Projects.GetApprovalConfiguration(projectID)
	if err != nil {
		return nil, err
	}

	approvalSettings := map[string]*llx.RawData{
		"approvalsBeforeMerge":                      llx.IntData(int64(approvalConfig.ApprovalsBeforeMerge)),
		"resetApprovalsOnPush":                      llx.BoolData(approvalConfig.ResetApprovalsOnPush),
		"disableOverridingApproversPerMergeRequest": llx.BoolData(approvalConfig.DisableOverridingApproversPerMergeRequest),
		"mergeRequestsAuthorApproval":               llx.BoolData(approvalConfig.MergeRequestsAuthorApproval),
		"mergeRequestsDisableCommittersApproval":    llx.BoolData(approvalConfig.MergeRequestsDisableCommittersApproval),
		"requirePasswordToApprove":                  llx.BoolData(approvalConfig.RequirePasswordToApprove),
		"selectiveCodeOwnerRemovals":                llx.BoolData(approvalConfig.SelectiveCodeOwnerRemovals),
	}

	mqlApprovalSettings, err := CreateResource(p.MqlRuntime, "gitlab.project.approvalSetting", approvalSettings)
	if err != nil {
		return nil, err
	}

	return mqlApprovalSettings.(*mqlGitlabProjectApprovalSetting), nil
}

// approvalRules fetches project approval rules
func (p *mqlGitlabProject) approvalRules() ([]any, error) {
	conn := p.MqlRuntime.Connection.(*connection.GitLabConnection)

	projectID := int(p.Id.Data)
	approvals, _, err := conn.Client().Projects.GetProjectApprovalRules(projectID, nil, nil)
	if err != nil {
		return nil, err
	}

	var approvalRules []any
	for _, rule := range approvals {
		approvalRule := map[string]*llx.RawData{
			"id":                llx.IntData(int64(rule.ID)),
			"name":              llx.StringData(rule.Name),
			"approvalsRequired": llx.IntData(int64(rule.ApprovalsRequired)),
		}
		mqlApprovalRule, err := CreateResource(p.MqlRuntime, "gitlab.project.approvalRule", approvalRule)
		if err != nil {
			return nil, err
		}
		approvalRules = append(approvalRules, mqlApprovalRule)
	}

	return approvalRules, nil
}

// mergeMethod fetches the project merge method
func (p *mqlGitlabProject) mergeMethod() (string, error) {
	conn := p.MqlRuntime.Connection.(*connection.GitLabConnection)

	projectID := int(p.Id.Data)
	project, _, err := conn.Client().Projects.GetProject(projectID, nil)
	if err != nil {
		return "", err
	}

	var mergeMethodString string
	switch project.MergeMethod {
	case "ff":
		mergeMethodString = "fast-forward merge"
	case "rebase_merge":
		mergeMethodString = "semi-linear merge"
	default:
		mergeMethodString = string(project.MergeMethod)
	}

	return mergeMethodString, nil
}

// id function for gitlab.project.protectedBranch
func (g *mqlGitlabProjectProtectedBranch) id() (string, error) {
	return g.Name.Data, nil
}

// protectedBranches fetches protected branch settings
func (p *mqlGitlabProject) protectedBranches() ([]any, error) {
	conn := p.MqlRuntime.Connection.(*connection.GitLabConnection)

	projectID := int(p.Id.Data)
	project, _, err := conn.Client().Projects.GetProject(projectID, nil)
	if err != nil {
		return nil, err
	}

	defaultBranch := project.DefaultBranch

	protectedBranches, _, err := conn.Client().ProtectedBranches.ListProtectedBranches(projectID, nil)
	if err != nil {
		return nil, err
	}

	var mqlProtectedBranches []any
	for _, branch := range protectedBranches {
		isDefaultBranch := branch.Name == defaultBranch

		branchSettings := map[string]*llx.RawData{
			"name":              llx.StringData(branch.Name),
			"allowForcePush":    llx.BoolData(branch.AllowForcePush),
			"defaultBranch":     llx.BoolData(isDefaultBranch),
			"codeOwnerApproval": llx.BoolData(branch.CodeOwnerApprovalRequired),
		}

		mqlProtectedBranch, err := CreateResource(p.MqlRuntime, "gitlab.project.protectedBranch", branchSettings)
		if err != nil {
			return nil, err
		}

		mqlProtectedBranches = append(mqlProtectedBranches, mqlProtectedBranch)
	}

	return mqlProtectedBranches, nil
}

// projectMembers fetches the list of members in the project with their roles
func (p *mqlGitlabProject) projectMembers() ([]any, error) {
	conn := p.MqlRuntime.Connection.(*connection.GitLabConnection)

	projectID := int(p.Id.Data)

	members, _, err := conn.Client().ProjectMembers.ListAllProjectMembers(projectID, nil)
	if err != nil {
		return nil, err
	}

	var mqlMembers []any
	for _, member := range members {
		role := mapAccessLevelToRole(int(member.AccessLevel))

		mqlUser, err := CreateResource(p.MqlRuntime, "gitlab.user", map[string]*llx.RawData{
			"id":               llx.IntData(int64(member.ID)),
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
			"id":   llx.IntData(int64(member.ID)),
			"user": llx.ResourceData(mqlUser, "gitlab.user"),
			"role": llx.StringData(role),
		}

		mqlMember, err := CreateResource(p.MqlRuntime, "gitlab.member", memberInfo)
		if err != nil {
			return nil, err
		}

		mqlMembers = append(mqlMembers, mqlMember)
	}

	return mqlMembers, nil
}

// id function for gitlab.project.file
func (f *mqlGitlabProjectFile) id() (string, error) {
	return f.Path.Data, nil
}

// projectFiles fetches the list of files in the project repository and their contents
func (p *mqlGitlabProject) projectFiles() ([]any, error) {
	// Return empty array if repository is empty to avoid 404 errors
	if p.EmptyRepo.Data {
		return []any{}, nil
	}

	conn := p.MqlRuntime.Connection.(*connection.GitLabConnection)

	projectID := int(p.Id.Data)
	defaultBranch := p.DefaultBranch.Data

	ref := &defaultBranch
	recursive := true

	listFilesOptions := &gitlab.ListTreeOptions{
		Ref:       ref,
		Recursive: &recursive,
	}

	files, _, err := conn.Client().Repositories.ListTree(projectID, listFilesOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list files in repository: %w", err)
	}

	var mqlFiles []any
	for _, file := range files {
		// Only fetch file content for blobs (files) not directories
		if file.Type == "blob" {
			fileContent, _, err := conn.Client().RepositoryFiles.GetFile(projectID, file.Path, &gitlab.GetFileOptions{Ref: ref})
			if err != nil {
				return nil, err
			}

			// Decode base64 content
			contentBytes, err := base64.StdEncoding.DecodeString(fileContent.Content)
			if err != nil {
				return nil, err
			}

			fileInfo := map[string]*llx.RawData{
				"path":    llx.StringData(file.Path),
				"type":    llx.StringData(file.Type),
				"name":    llx.StringData(file.Name),
				"content": llx.StringData(string(contentBytes)),
			}

			mqlFile, err := CreateResource(p.MqlRuntime, "gitlab.project.file", fileInfo)
			if err != nil {
				return nil, err
			}

			mqlFiles = append(mqlFiles, mqlFile)
		}
	}

	return mqlFiles, nil
}

// id function for gitlab.project.webhook
func (g *mqlGitlabProjectWebhook) id() (string, error) {
	return g.Url.Data, nil
}

// webhooks fetches the webhooks for a project
func (p *mqlGitlabProject) webhooks() ([]any, error) {
	conn := p.MqlRuntime.Connection.(*connection.GitLabConnection)

	projectID := int(p.Id.Data)

	hooks, _, err := conn.Client().Projects.ListProjectHooks(projectID, nil)
	if err != nil {
		return nil, err
	}

	var mqlWebhooks []any
	for _, hook := range hooks {
		hookInfo := map[string]*llx.RawData{
			"url":             llx.StringData(hook.URL),
			"sslVerification": llx.BoolData(hook.EnableSSLVerification),
		}

		mqlWebhook, err := CreateResource(p.MqlRuntime, "gitlab.project.webhook", hookInfo)
		if err != nil {
			return nil, err
		}

		mqlWebhooks = append(mqlWebhooks, mqlWebhook)
	}

	return mqlWebhooks, nil
}

// id function for gitlab.project.mergeRequest
func (m *mqlGitlabProjectMergeRequest) id() (string, error) {
	return strconv.FormatInt(m.Id.Data, 10), nil
}

// mergeRequests fetches the list of merge requests for the project
func (p *mqlGitlabProject) mergeRequests() ([]any, error) {
	conn := p.MqlRuntime.Connection.(*connection.GitLabConnection)

	projectID := int(p.Id.Data)

	// Fetch all merge requests with pagination
	perPage := int64(50)
	page := int64(1)
	var allMergeRequests []*gitlab.BasicMergeRequest

	for {
		mergeRequests, resp, err := conn.Client().MergeRequests.ListProjectMergeRequests(projectID, &gitlab.ListProjectMergeRequestsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		})
		if err != nil {
			return nil, err
		}

		allMergeRequests = append(allMergeRequests, mergeRequests...)

		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	var mqlMergeRequests []any
	for _, mr := range allMergeRequests {
		authorName := ""
		if mr.Author != nil {
			authorName = mr.Author.Username
		}

		mrInfo := map[string]*llx.RawData{
			"id":           llx.IntData(int64(mr.ID)),
			"internalId":   llx.IntData(int64(mr.IID)),
			"title":        llx.StringData(mr.Title),
			"state":        llx.StringData(mr.State),
			"description":  llx.StringData(mr.Description),
			"sourceBranch": llx.StringData(mr.SourceBranch),
			"targetBranch": llx.StringData(mr.TargetBranch),
			"author":       llx.StringData(authorName),
			"createdAt":    llx.TimeDataPtr(mr.CreatedAt),
			"updatedAt":    llx.TimeDataPtr(mr.UpdatedAt),
			"mergedAt":     llx.TimeDataPtr(mr.MergedAt),
			"draft":        llx.BoolData(mr.Draft),
			"webURL":       llx.StringData(mr.WebURL),
			"labels":       llx.ArrayData(convert.SliceAnyToInterface([]string(mr.Labels)), types.String),
		}

		mqlMR, err := CreateResource(p.MqlRuntime, "gitlab.project.mergeRequest", mrInfo)
		if err != nil {
			return nil, err
		}

		mqlMergeRequests = append(mqlMergeRequests, mqlMR)
	}

	return mqlMergeRequests, nil
}

// id function for gitlab.project.issue
func (i *mqlGitlabProjectIssue) id() (string, error) {
	return strconv.FormatInt(i.Id.Data, 10), nil
}

// issues fetches the list of issues for the project
func (p *mqlGitlabProject) issues() ([]any, error) {
	conn := p.MqlRuntime.Connection.(*connection.GitLabConnection)

	projectID := int(p.Id.Data)

	// Fetch all issues with pagination
	perPage := int64(50)
	page := int64(1)
	var allIssues []*gitlab.Issue

	for {
		issues, resp, err := conn.Client().Issues.ListProjectIssues(projectID, &gitlab.ListProjectIssuesOptions{
			ListOptions: gitlab.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		})
		if err != nil {
			return nil, err
		}

		allIssues = append(allIssues, issues...)

		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	var mqlIssues []any
	for _, issue := range allIssues {
		authorName := ""
		if issue.Author != nil {
			authorName = issue.Author.Username
		}

		var dueDate *time.Time
		if issue.DueDate != nil {
			t := time.Time(*issue.DueDate)
			dueDate = &t
		}

		issueInfo := map[string]*llx.RawData{
			"id":           llx.IntData(int64(issue.ID)),
			"internalId":   llx.IntData(int64(issue.IID)),
			"title":        llx.StringData(issue.Title),
			"state":        llx.StringData(issue.State),
			"description":  llx.StringData(issue.Description),
			"author":       llx.StringData(authorName),
			"createdAt":    llx.TimeDataPtr(issue.CreatedAt),
			"updatedAt":    llx.TimeDataPtr(issue.UpdatedAt),
			"closedAt":     llx.TimeDataPtr(issue.ClosedAt),
			"dueDate":      llx.TimeDataPtr(dueDate),
			"confidential": llx.BoolData(issue.Confidential),
			"webURL":       llx.StringData(issue.WebURL),
			"labels":       llx.ArrayData(convert.SliceAnyToInterface([]string(issue.Labels)), types.String),
		}

		mqlIssue, err := CreateResource(p.MqlRuntime, "gitlab.project.issue", issueInfo)
		if err != nil {
			return nil, err
		}

		mqlIssues = append(mqlIssues, mqlIssue)
	}

	return mqlIssues, nil
}

// id function for gitlab.project.release
func (r *mqlGitlabProjectRelease) id() (string, error) {
	return r.TagName.Data, nil
}

// releases fetches the list of releases for the project
func (p *mqlGitlabProject) releases() ([]any, error) {
	conn := p.MqlRuntime.Connection.(*connection.GitLabConnection)

	projectID := int(p.Id.Data)

	// Fetch all releases with pagination
	perPage := int64(50)
	page := int64(1)
	var allReleases []*gitlab.Release

	for {
		releases, resp, err := conn.Client().Releases.ListReleases(projectID, &gitlab.ListReleasesOptions{
			ListOptions: gitlab.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		})
		if err != nil {
			return nil, err
		}

		allReleases = append(allReleases, releases...)

		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	var mqlReleases []any
	for _, release := range allReleases {
		releaseInfo := map[string]*llx.RawData{
			"tagName":     llx.StringData(release.TagName),
			"name":        llx.StringData(release.Name),
			"description": llx.StringData(release.Description),
			"createdAt":   llx.TimeDataPtr(release.CreatedAt),
			"releasedAt":  llx.TimeDataPtr(release.ReleasedAt),
			"author":      llx.StringData(release.Author.Username),
		}

		mqlRelease, err := CreateResource(p.MqlRuntime, "gitlab.project.release", releaseInfo)
		if err != nil {
			return nil, err
		}

		mqlReleases = append(mqlReleases, mqlRelease)
	}

	return mqlReleases, nil
}

// id function for gitlab.project.variable
func (v *mqlGitlabProjectVariable) id() (string, error) {
	return v.Key.Data + "/" + v.EnvironmentScope.Data, nil
}

// variables fetches the list of CI/CD variables for the project
func (p *mqlGitlabProject) variables() ([]any, error) {
	conn := p.MqlRuntime.Connection.(*connection.GitLabConnection)

	projectID := int(p.Id.Data)

	// Fetch all variables with pagination
	perPage := int64(50)
	page := int64(1)
	var allVariables []*gitlab.ProjectVariable

	for {
		variables, resp, err := conn.Client().ProjectVariables.ListVariables(projectID, &gitlab.ListProjectVariablesOptions{
			ListOptions: gitlab.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		})
		if err != nil {
			return nil, err
		}

		allVariables = append(allVariables, variables...)

		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	var mqlVariables []any
	for _, v := range allVariables {
		varInfo := map[string]*llx.RawData{
			"key":              llx.StringData(v.Key),
			"variableType":     llx.StringData(string(v.VariableType)),
			"protected":        llx.BoolData(v.Protected),
			"masked":           llx.BoolData(v.Masked),
			"hidden":           llx.BoolData(v.Hidden),
			"raw":              llx.BoolData(v.Raw),
			"environmentScope": llx.StringData(v.EnvironmentScope),
			"description":      llx.StringData(v.Description),
		}

		mqlVar, err := CreateResource(p.MqlRuntime, "gitlab.project.variable", varInfo)
		if err != nil {
			return nil, err
		}

		mqlVariables = append(mqlVariables, mqlVar)
	}

	return mqlVariables, nil
}

// id function for gitlab.project.milestone
func (m *mqlGitlabProjectMilestone) id() (string, error) {
	return strconv.FormatInt(m.Id.Data, 10), nil
}

// milestones fetches the list of milestones for the project
func (p *mqlGitlabProject) milestones() ([]any, error) {
	conn := p.MqlRuntime.Connection.(*connection.GitLabConnection)

	projectID := int(p.Id.Data)

	// Fetch all milestones with pagination
	perPage := int64(50)
	page := int64(1)
	var allMilestones []*gitlab.Milestone

	for {
		milestones, resp, err := conn.Client().Milestones.ListMilestones(projectID, &gitlab.ListMilestonesOptions{
			ListOptions: gitlab.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		})
		if err != nil {
			return nil, err
		}

		allMilestones = append(allMilestones, milestones...)

		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	var mqlMilestones []any
	for _, milestone := range allMilestones {
		milestoneInfo := map[string]*llx.RawData{
			"id":          llx.IntData(milestone.ID),
			"internalId":  llx.IntData(milestone.IID),
			"projectId":   llx.IntData(milestone.ProjectID),
			"title":       llx.StringData(milestone.Title),
			"description": llx.StringData(milestone.Description),
			"state":       llx.StringData(milestone.State),
			"updatedAt":   llx.TimeDataPtr(milestone.UpdatedAt),
			"createdAt":   llx.TimeDataPtr(milestone.CreatedAt),
		}

		// Convert ISOTime to time.Time for startDate
		if milestone.StartDate != nil {
			t := time.Time(*milestone.StartDate)
			milestoneInfo["startDate"] = llx.TimeDataPtr(&t)
		}

		// Convert ISOTime to time.Time for dueDate
		if milestone.DueDate != nil {
			t := time.Time(*milestone.DueDate)
			milestoneInfo["dueDate"] = llx.TimeDataPtr(&t)
		}

		// Handle expired field (pointer to bool)
		if milestone.Expired != nil {
			milestoneInfo["expired"] = llx.BoolData(*milestone.Expired)
		} else {
			milestoneInfo["expired"] = llx.BoolData(false)
		}

		mqlMilestone, err := CreateResource(p.MqlRuntime, "gitlab.project.milestone", milestoneInfo)
		if err != nil {
			return nil, err
		}

		mqlMilestones = append(mqlMilestones, mqlMilestone)
	}

	return mqlMilestones, nil
}

// id function for gitlab.project.label
func (l *mqlGitlabProjectLabel) id() (string, error) {
	return strconv.FormatInt(l.Id.Data, 10), nil
}

// labels fetches the list of labels for the project
func (p *mqlGitlabProject) labels() ([]any, error) {
	conn := p.MqlRuntime.Connection.(*connection.GitLabConnection)

	projectID := int(p.Id.Data)

	// Fetch all labels with pagination
	perPage := int64(50)
	page := int64(1)
	var allLabels []*gitlab.Label

	for {
		labels, resp, err := conn.Client().Labels.ListLabels(projectID, &gitlab.ListLabelsOptions{
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

		mqlLabel, err := CreateResource(p.MqlRuntime, "gitlab.project.label", labelInfo)
		if err != nil {
			return nil, err
		}

		mqlLabels = append(mqlLabels, mqlLabel)
	}

	return mqlLabels, nil
}
