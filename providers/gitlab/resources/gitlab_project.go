// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gitlab/connection"
	"go.mondoo.com/mql/v13/types"
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

// Helper function to create a milestone resource from API data
func createMilestoneResource(runtime *plugin.Runtime, milestone *gitlab.Milestone) (*mqlGitlabProjectMilestone, error) {
	if milestone == nil {
		return nil, nil
	}

	milestoneInfo := map[string]*llx.RawData{
		"__id":        llx.StringData(strconv.FormatInt(milestone.ID, 10)),
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

	mqlMilestone, err := CreateResource(runtime, "gitlab.project.milestone", milestoneInfo)
	if err != nil {
		return nil, err
	}

	return mqlMilestone.(*mqlGitlabProjectMilestone), nil
}

// milestone fetches the milestone for a merge request
func (m *mqlGitlabProjectMergeRequest) milestone() (*mqlGitlabProjectMilestone, error) {
	// The milestone should already be set when the merge request is created
	// This method is only called as a fallback if it wasn't set
	// In that case, we would need to fetch the merge request details again
	// For now, return nil to indicate no milestone
	return nil, nil
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

		// Add milestone if present
		if mr.Milestone != nil {
			mqlMilestone, err := createMilestoneResource(p.MqlRuntime, mr.Milestone)
			if err != nil {
				return nil, err
			}
			if mqlMilestone != nil {
				mrInfo["milestone"] = llx.ResourceData(mqlMilestone, "gitlab.project.milestone")
			}
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

// milestone fetches the milestone for an issue
func (i *mqlGitlabProjectIssue) milestone() (*mqlGitlabProjectMilestone, error) {
	// The milestone should already be set when the issue is created
	// This method is only called as a fallback if it wasn't set
	// In that case, we would need to fetch the issue details again
	// For now, return nil to indicate no milestone
	return nil, nil
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

		// Add milestone if present
		if issue.Milestone != nil {
			mqlMilestone, err := createMilestoneResource(p.MqlRuntime, issue.Milestone)
			if err != nil {
				return nil, err
			}
			if mqlMilestone != nil {
				issueInfo["milestone"] = llx.ResourceData(mqlMilestone, "gitlab.project.milestone")
			}
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

// id function for gitlab.project.pipeline
func (p *mqlGitlabProjectPipeline) id() (string, error) {
	return strconv.FormatInt(p.Id.Data, 10), nil
}

// pipelines fetches the list of CI/CD pipelines for the project
func (p *mqlGitlabProject) pipelines() ([]any, error) {
	conn := p.MqlRuntime.Connection.(*connection.GitLabConnection)

	projectID := int(p.Id.Data)

	// Fetch all pipelines with pagination
	perPage := int64(50)
	page := int64(1)
	var allPipelines []*gitlab.PipelineInfo

	for {
		pipelines, resp, err := conn.Client().Pipelines.ListProjectPipelines(projectID, &gitlab.ListProjectPipelinesOptions{
			ListOptions: gitlab.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		})
		if err != nil {
			return nil, err
		}

		allPipelines = append(allPipelines, pipelines...)

		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	var mqlPipelines []any
	for _, pipeline := range allPipelines {
		pipelineInfo := map[string]*llx.RawData{
			"id":         llx.IntData(pipeline.ID),
			"internalId": llx.IntData(pipeline.IID),
			"projectId":  llx.IntData(pipeline.ProjectID),
			"status":     llx.StringData(pipeline.Status),
			"source":     llx.StringData(pipeline.Source),
			"ref":        llx.StringData(pipeline.Ref),
			"sha":        llx.StringData(pipeline.SHA),
			"name":       llx.StringData(pipeline.Name),
			"webURL":     llx.StringData(pipeline.WebURL),
			"createdAt":  llx.TimeDataPtr(pipeline.CreatedAt),
			"updatedAt":  llx.TimeDataPtr(pipeline.UpdatedAt),
		}

		mqlPipeline, err := CreateResource(p.MqlRuntime, "gitlab.project.pipeline", pipelineInfo)
		if err != nil {
			return nil, err
		}

		mqlPipelines = append(mqlPipelines, mqlPipeline)
	}

	return mqlPipelines, nil
}

// id function for gitlab.project.runner
func (r *mqlGitlabProjectRunner) id() (string, error) {
	return strconv.FormatInt(r.Id.Data, 10), nil
}

// runners fetches the list of runners available to the project
func (p *mqlGitlabProject) runners() ([]any, error) {
	conn := p.MqlRuntime.Connection.(*connection.GitLabConnection)

	projectID := int(p.Id.Data)

	// Fetch all runners with pagination
	perPage := int64(50)
	page := int64(1)
	var allRunners []*gitlab.Runner

	for {
		runners, resp, err := conn.Client().Runners.ListProjectRunners(projectID, &gitlab.ListProjectRunnersOptions{
			ListOptions: gitlab.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		})
		if err != nil {
			return nil, err
		}

		allRunners = append(allRunners, runners...)

		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	var mqlRunners []any
	for _, runner := range allRunners {
		runnerInfo := map[string]*llx.RawData{
			"id":          llx.IntData(runner.ID),
			"description": llx.StringData(runner.Description),
			"name":        llx.StringData(runner.Name),
			"runnerType":  llx.StringData(runner.RunnerType),
			"paused":      llx.BoolData(runner.Paused),
			"isShared":    llx.BoolData(runner.IsShared),
			"online":      llx.BoolData(runner.Online),
			"status":      llx.StringData(runner.Status),
		}

		mqlRunner, err := CreateResource(p.MqlRuntime, "gitlab.project.runner", runnerInfo)
		if err != nil {
			return nil, err
		}

		mqlRunners = append(mqlRunners, mqlRunner)
	}

	return mqlRunners, nil
}

// id function for gitlab.project.pushRule
func (r *mqlGitlabProjectPushRule) id() (string, error) {
	return "gitlab.project.pushRule/" + strconv.FormatInt(r.Id.Data, 10), nil
}

// pushRules fetches push rules for the project
func (p *mqlGitlabProject) pushRules() (*mqlGitlabProjectPushRule, error) {
	conn := p.MqlRuntime.Connection.(*connection.GitLabConnection)

	projectID := int(p.Id.Data)
	rules, resp, err := conn.Client().Projects.GetProjectPushRules(projectID)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return nil, nil // no push rules configured
		}
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

	mqlRule, err := CreateResource(p.MqlRuntime, "gitlab.project.pushRule", ruleInfo)
	if err != nil {
		return nil, err
	}

	return mqlRule.(*mqlGitlabProjectPushRule), nil
}

// id function for gitlab.project.accessToken
func (t *mqlGitlabProjectAccessToken) id() (string, error) {
	return "gitlab.project.accessToken/" + strconv.FormatInt(t.Id.Data, 10), nil
}

// accessTokens fetches the list of access tokens for the project
func (p *mqlGitlabProject) accessTokens() ([]any, error) {
	conn := p.MqlRuntime.Connection.(*connection.GitLabConnection)

	projectID := int(p.Id.Data)

	perPage := int64(50)
	page := int64(1)
	var allTokens []*gitlab.ProjectAccessToken

	for {
		tokens, resp, err := conn.Client().ProjectAccessTokens.ListProjectAccessTokens(projectID, &gitlab.ListProjectAccessTokensOptions{
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

		mqlToken, err := CreateResource(p.MqlRuntime, "gitlab.project.accessToken", tokenInfo)
		if err != nil {
			return nil, err
		}

		mqlTokens = append(mqlTokens, mqlToken)
	}

	return mqlTokens, nil
}

// id function for gitlab.project.deployKey
func (k *mqlGitlabProjectDeployKey) id() (string, error) {
	return "gitlab.project.deployKey/" + strconv.FormatInt(k.Id.Data, 10), nil
}

// deployKeys fetches the list of deploy keys for the project
func (p *mqlGitlabProject) deployKeys() ([]any, error) {
	conn := p.MqlRuntime.Connection.(*connection.GitLabConnection)

	projectID := int(p.Id.Data)

	perPage := int64(50)
	page := int64(1)
	var allKeys []*gitlab.ProjectDeployKey

	for {
		keys, resp, err := conn.Client().DeployKeys.ListProjectDeployKeys(projectID, &gitlab.ListProjectDeployKeysOptions{
			ListOptions: gitlab.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		})
		if err != nil {
			return nil, err
		}

		allKeys = append(allKeys, keys...)

		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	var mqlKeys []any
	for _, key := range allKeys {
		keyInfo := map[string]*llx.RawData{
			"id":                llx.IntData(key.ID),
			"title":             llx.StringData(key.Title),
			"key":               llx.StringData(key.Key),
			"fingerprint":       llx.StringData(key.Fingerprint),
			"fingerprintSHA256": llx.StringData(key.FingerprintSHA256),
			"createdAt":         llx.TimeDataPtr(key.CreatedAt),
			"expiresAt":         llx.TimeDataPtr(key.ExpiresAt),
			"canPush":           llx.BoolData(key.CanPush),
		}

		mqlKey, err := CreateResource(p.MqlRuntime, "gitlab.project.deployKey", keyInfo)
		if err != nil {
			return nil, err
		}

		mqlKeys = append(mqlKeys, mqlKey)
	}

	return mqlKeys, nil
}

// id function for gitlab.project.deployToken
func (t *mqlGitlabProjectDeployToken) id() (string, error) {
	return "gitlab.project.deployToken/" + strconv.FormatInt(t.Id.Data, 10), nil
}

// deployTokens fetches the list of deploy tokens for the project
func (p *mqlGitlabProject) deployTokens() ([]any, error) {
	conn := p.MqlRuntime.Connection.(*connection.GitLabConnection)

	projectID := int(p.Id.Data)

	perPage := int64(50)
	page := int64(1)
	var allTokens []*gitlab.DeployToken

	for {
		tokens, resp, err := conn.Client().DeployTokens.ListProjectDeployTokens(projectID, &gitlab.ListProjectDeployTokensOptions{
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

		mqlToken, err := CreateResource(p.MqlRuntime, "gitlab.project.deployToken", tokenInfo)
		if err != nil {
			return nil, err
		}

		mqlTokens = append(mqlTokens, mqlToken)
	}

	return mqlTokens, nil
}

// securitySettings fetches security settings for the project
func (p *mqlGitlabProject) securitySettings() (*mqlGitlabProjectSecuritySetting, error) {
	conn := p.MqlRuntime.Connection.(*connection.GitLabConnection)

	projectID := int(p.Id.Data)
	settings, resp, err := conn.Client().ProjectSecuritySettings.ListProjectSecuritySettings(projectID)
	if err != nil {
		if resp != nil && (resp.StatusCode == 403 || resp.StatusCode == 404) {
			return nil, nil // not available on this GitLab tier
		}
		return nil, err
	}

	settingInfo := map[string]*llx.RawData{
		"__id":                                llx.StringData("gitlab.project.securitySetting/" + strconv.Itoa(projectID)),
		"autoFixContainerScanning":            llx.BoolData(settings.AutoFixContainerScanning),
		"autoFixDAST":                         llx.BoolData(settings.AutoFixDAST),
		"autoFixDependencyScanning":           llx.BoolData(settings.AutoFixDependencyScanning),
		"autoFixSAST":                         llx.BoolData(settings.AutoFixSAST),
		"continuousVulnerabilityScansEnabled": llx.BoolData(settings.ContinuousVulnerabilityScansEnabled),
		"containerScanningForRegistryEnabled": llx.BoolData(settings.ContainerScanningForRegistryEnabled),
		"secretPushProtectionEnabled":         llx.BoolData(settings.SecretPushProtectionEnabled),
	}

	mqlSetting, err := CreateResource(p.MqlRuntime, "gitlab.project.securitySetting", settingInfo)
	if err != nil {
		return nil, err
	}

	return mqlSetting.(*mqlGitlabProjectSecuritySetting), nil
}
