// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/base64"
	"fmt"
	"strconv"

	"github.com/xanzy/go-gitlab"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/gitlab/connection"
)

func (g *mqlGitlabGroup) id() (string, error) {
	return "gitlab.group/" + strconv.FormatInt(g.Id.Data, 10), nil
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
	args["description"] = llx.StringData(grp.Description)
	args["createdAt"] = llx.TimeDataPtr(grp.CreatedAt)
	args["webURL"] = llx.StringData(string(grp.WebURL))
	args["visibility"] = llx.StringData(string(grp.Visibility))
	args["requireTwoFactorAuthentication"] = llx.BoolData(grp.RequireTwoFactorAuth)
	args["preventForkingOutsideGroup"] = llx.BoolData(grp.PreventForkingOutsideGroup)
	args["mentionsDisabled"] = llx.BoolData(grp.MentionsDisabled)
	args["emailsDisabled"] = llx.BoolData(grp.EmailsDisabled)

	return args, nil, nil
}

// GetProjects list all projects that belong to a group
// see https://docs.gitlab.com/ee/api/projects.html
func (g *mqlGitlabGroup) projects() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GitLabConnection)

	if g.Path.Error != nil {
		return nil, g.Path.Error
	}
	gid := g.Id.Data

	grp, _, err := conn.Client().Groups.GetGroup(int(gid), nil)
	if err != nil {
		return nil, err
	}

	var mqlProjects []interface{}
	for i := range grp.Projects {
		prj := grp.Projects[i]

		mqlProject, err := CreateResource(g.MqlRuntime, "gitlab.project", getGitlabProjectArgs(prj))
		if err != nil {
			return nil, err
		}
		mqlProjects = append(mqlProjects, mqlProject)
	}

	return mqlProjects, nil
}

func getGitlabProjectArgs(prj *gitlab.Project) map[string]*llx.RawData {
	return map[string]*llx.RawData{
		"id":                          llx.IntData(int64(prj.ID)),
		"name":                        llx.StringData(prj.Name),
		"fullName":                    llx.StringData(prj.NameWithNamespace),
		"allowMergeOnSkippedPipeline": llx.BoolData(prj.AllowMergeOnSkippedPipeline),
		"archived":                    llx.BoolData(prj.Archived),
		"autoDevopsEnabled":           llx.BoolData(prj.AutoDevopsEnabled),
		"containerRegistryEnabled":    llx.BoolData(prj.ContainerRegistryEnabled),
		"createdAt":                   llx.TimeDataPtr(prj.CreatedAt),
		"defaultBranch":               llx.StringData(prj.DefaultBranch),
		"description":                 llx.StringData(prj.Description),
		"emailsDisabled":              llx.BoolData(!prj.EmailsEnabled),
		"issuesEnabled":               llx.BoolData(prj.IssuesEnabled),
		"mergeRequestsEnabled":        llx.BoolData(prj.MergeRequestsEnabled),
		"mirror":                      llx.BoolData(prj.Mirror),
		"onlyAllowMergeIfAllDiscussionsAreResolved": llx.BoolData(prj.OnlyAllowMergeIfAllDiscussionsAreResolved),
		"onlyAllowMergeIfPipelineSucceeds":          llx.BoolData(prj.OnlyAllowMergeIfPipelineSucceeds),
		"packagesEnabled":                           llx.BoolData(prj.PackagesEnabled),
		"path":                                      llx.StringData(prj.Path),
		"requirementsEnabled":                       llx.BoolData(prj.RequirementsEnabled),
		"serviceDeskEnabled":                        llx.BoolData(prj.ServiceDeskEnabled),
		"snippetsEnabled":                           llx.BoolData(prj.SnippetsEnabled),
		"visibility":                                llx.StringData(string(prj.Visibility)),
		"webURL":                                    llx.StringData(prj.WebURL),
		"wikiEnabled":                               llx.BoolData(prj.WikiEnabled),
		"jobsEnabled":                               llx.BoolData(prj.JobsEnabled),
		"emptyRepo":                                 llx.BoolData(prj.EmptyRepo),
		"license":                                   llx.StringData(prj.License.Name),
		"sharedRunnersEnabled":                      llx.BoolData(prj.SharedRunnersEnabled),
		"groupRunnersEnabled":                       llx.BoolData(prj.GroupRunnersEnabled),
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

// New function to fetch project approval settings
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

// New function to fetch project approval rules
func (p *mqlGitlabProject) approvalRules() ([]interface{}, error) {
	conn := p.MqlRuntime.Connection.(*connection.GitLabConnection)

	projectID := int(p.Id.Data)
	approvals, _, err := conn.Client().Projects.GetProjectApprovalRules(projectID, nil, nil)
	if err != nil {
		return nil, err
	}

	var approvalRules []interface{}
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

// To fetch project merge method
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

// Define the id function for a unique identifier for a resource instance gitlab.project.repository.protectedBranch
// The struct name mqlGitlabProjectRepositoryProtectedBranch is derived from the resource path gitlab.project.repository.protectedBranch. This is a convention used to maintain consistency and clarity within the Mondoo framework by adding mql in the front, ensuring that each resource can be uniquely identified and managed.
func (g *mqlGitlabProjectProtectedBranch) id() (string, error) {
	return g.Name.Data, nil
}

// To fetch protected branch settings
func (p *mqlGitlabProject) protectedBranches() ([]interface{}, error) {
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

	var mqlProtectedBranches []interface{}
	for _, branch := range protectedBranches {
		// Declare and initialize isDefaultBranch variable
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

// id related to gitlab.project.member
func (g *mqlGitlabProjectMember) id() (string, error) {
	return strconv.FormatInt(g.Id.Data, 10), nil
}

// To fetch the list of members in the project with their roles
func (p *mqlGitlabProject) projectMembers() ([]interface{}, error) {
	conn := p.MqlRuntime.Connection.(*connection.GitLabConnection)

	projectID := int(p.Id.Data)

	// Fetch the list of members, keep it in mind it is different from ListProjectMembers
	members, _, err := conn.Client().ProjectMembers.ListAllProjectMembers(projectID, nil)
	if err != nil {
		return nil, err
	}

	// function to map access levels to roles, encapsulated inside projectMembers
	mapAccessLevelToRole := func(accessLevel int) string {
		switch accessLevel {
		case 10:
			return "Guest"
		case 20:
			return "Reporter"
		case 30:
			return "Developer"
		case 40:
			return "Maintainer"
		case 50:
			return "Owner"
		default:
			return "Unknown"
		}
	}

	var mqlMembers []interface{}
	for _, member := range members {
		role := mapAccessLevelToRole(int(member.AccessLevel))
		memberInfo := map[string]*llx.RawData{
			"id":       llx.IntData(int64(member.ID)),
			"username": llx.StringData(member.Username),
			"state":    llx.StringData(member.State),
			"email":    llx.StringData(member.Email),
			"name":     llx.StringData(member.Name),
			"role":     llx.StringData(role),
		}

		mqlMember, err := CreateResource(p.MqlRuntime, "gitlab.project.member", memberInfo)
		if err != nil {
			return nil, err
		}

		mqlMembers = append(mqlMembers, mqlMember)
	}

	return mqlMembers, nil
}

// Define the id function for a unique identifier for a file resource
func (f *mqlGitlabProjectFile) id() (string, error) {
	return f.Path.Data, nil
}

// To fetch the list of files in the project repository and their contents
func (p *mqlGitlabProject) projectFiles() ([]interface{}, error) {
	conn := p.MqlRuntime.Connection.(*connection.GitLabConnection)

	projectID := int(p.Id.Data)
	defaultBranch := p.DefaultBranch.Data

	ref := &defaultBranch
	recursive := true

	// ListTree function expects pointer to the struct
	listFilesOptions := &gitlab.ListTreeOptions{
		Ref:       ref,
		Recursive: &recursive,
	}

	// Fetch the list of files
	files, _, err := conn.Client().Repositories.ListTree(projectID, listFilesOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list files in repository: %w", err)
	}

	var mqlFiles []interface{}
	for _, file := range files {
		// Making sure we only fetch file content for blobs (files) not directories
		if file.Type == "blob" {
			// Fetch file content
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

// id function for a unique identifier for a resource instance gitlab.project.webhook
func (g *mqlGitlabProjectWebhook) id() (string, error) {
	return g.Url.Data, nil
}

// Function to fetch and check the webhooks for a project
func (p *mqlGitlabProject) webhooks() ([]interface{}, error) {
	conn := p.MqlRuntime.Connection.(*connection.GitLabConnection)

	projectID := int(p.Id.Data)

	hooks, _, err := conn.Client().Projects.ListProjectHooks(projectID, nil)
	if err != nil {
		return nil, err
	}

	var mqlWebhooks []interface{}
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
