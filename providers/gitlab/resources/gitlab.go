// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
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
func (p *mqlGitlabProject) approvalSettings() (*mqlGitlabProjectApprovalSettings, error) {
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
	}

	mqlApprovalSettings, err := CreateResource(p.MqlRuntime, "gitlab.project.approvalSettings", approvalSettings)
	if err != nil {
		return nil, err
	}

	return mqlApprovalSettings.(*mqlGitlabProjectApprovalSettings), nil
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

// Initialize the gitlab project approval rule
func initGitlabProjectApprovalRule(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return args, nil, nil
}
