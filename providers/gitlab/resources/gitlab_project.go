// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gitlab/connection"
	"go.mondoo.com/mql/v13/types"
)

// mqlGitlabProjectInternal caches the full project payload from
// `GET /projects/:id` so multiple lazy fields (mergeMethod,
// containerExpirationPolicy, …) share one API call rather than each issuing
// its own GetProject for a different scalar.
type mqlGitlabProjectInternal struct {
	detailsOnce       sync.Once
	details           *gitlab.Project
	detailsErr        error
	detailsStatusCode int
}

// projectDetails returns the full *gitlab.Project payload, fetching it on
// first call and reusing it thereafter. Used by lazy accessors that need
// fields not seeded by getGitlabProjectArgs (MergeMethod,
// ContainerExpirationPolicy, etc.). detailsStatusCode is set whenever the
// SDK gave us a response, so callers can distinguish access-denied/missing
// (silent null) from real transport errors (propagate).
func (p *mqlGitlabProject) projectDetails(conn *connection.GitLabConnection) (*gitlab.Project, error) {
	p.detailsOnce.Do(func() {
		projectID := int(p.Id.Data)
		project, resp, err := conn.Client().Projects.GetProject(projectID, nil)
		if resp != nil {
			p.detailsStatusCode = resp.StatusCode
		}
		if err != nil {
			p.detailsErr = err
			return
		}
		p.details = project
	})
	return p.details, p.detailsErr
}

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
		"fullPath":                    llx.StringData(prj.PathWithNamespace),
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

	// If only id is provided, fetch the project by id - used by typed refs
	// (e.g. webhook.project(), deployKey.project(), auditEvent.entityProject()).
	// 403/404 yields a bare resource so the typed back-ref doesn't fail the
	// whole resource graph on insufficient perms.
	if idArg, ok := args["id"]; ok && idArg != nil && idArg.Error == nil {
		pid := int(idArg.Value.(int64))
		project, resp, err := conn.Client().Projects.GetProject(pid, nil)
		if err != nil {
			if resp != nil && (resp.StatusCode == 403 || resp.StatusCode == 404) {
				return args, nil, nil
			}
			return nil, nil, err
		}
		args = getGitlabProjectArgs(project)
		return args, nil, nil
	}

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
	perPage := int64(50)
	page := int64(1)
	var approvals []*gitlab.ProjectApprovalRule
	for {
		rules, resp, err := conn.Client().Projects.GetProjectApprovalRules(projectID, &gitlab.GetProjectApprovalRulesListsOptions{
			ListOptions: gitlab.ListOptions{Page: page, PerPage: perPage},
		})
		if err != nil {
			return nil, err
		}
		approvals = append(approvals, rules...)
		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	defaultBranchName := p.DefaultBranch.Data

	approvalRules := make([]any, 0, len(approvals))
	for _, rule := range approvals {
		users, err := basicUsersToMqlUsers(p.MqlRuntime, rule.Users)
		if err != nil {
			return nil, err
		}
		eligible, err := basicUsersToMqlUsers(p.MqlRuntime, rule.EligibleApprovers)
		if err != nil {
			return nil, err
		}
		branches, err := approvalRuleProtectedBranches(p.MqlRuntime, rule.ProtectedBranches, defaultBranchName)
		if err != nil {
			return nil, err
		}

		approvalRule := map[string]*llx.RawData{
			"id":                            llx.IntData(rule.ID),
			"name":                          llx.StringData(rule.Name),
			"approvalsRequired":             llx.IntData(rule.ApprovalsRequired),
			"ruleType":                      llx.StringData(rule.RuleType),
			"reportType":                    llx.StringData(rule.ReportType),
			"appliesToAllProtectedBranches": llx.BoolData(rule.AppliesToAllProtectedBranches),
			"containsHiddenGroups":          llx.BoolData(rule.ContainsHiddenGroups),
			"users":                         llx.ArrayData(users, types.Resource("gitlab.user")),
			"eligibleApprovers":             llx.ArrayData(eligible, types.Resource("gitlab.user")),
			"groups":                        llx.ArrayData(groupsToDicts(rule.Groups), types.Dict),
			"protectedBranches":             llx.ArrayData(branches, types.Resource("gitlab.project.protectedBranch")),
		}
		mqlApprovalRule, err := CreateResource(p.MqlRuntime, "gitlab.project.approvalRule", approvalRule)
		if err != nil {
			return nil, err
		}
		approvalRules = append(approvalRules, mqlApprovalRule)
	}

	return approvalRules, nil
}

// basicUsersToMqlUsers converts SDK BasicUser entries into gitlab.user
// resources via NewResource. BasicUser only carries a subset of the fields
// gitlab.user declares (no email, 2FA, bot, job/org/location, etc.), so
// instead of seeding the runtime cache with hardcoded zero values — which
// would poison the cache for any later `gitlab.users` query that returns
// real data on the same id — we hand only the id to NewResource and let
// `initGitlabUser` lazily fetch the full record on first access.
func basicUsersToMqlUsers(runtime *plugin.Runtime, users []*gitlab.BasicUser) ([]any, error) {
	out := make([]any, 0, len(users))
	for _, u := range users {
		if u == nil {
			continue
		}
		mqlUser, err := NewResource(runtime, "gitlab.user", map[string]*llx.RawData{
			"id": llx.IntData(u.ID),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, mqlUser)
	}
	return out, nil
}

// groupsToDicts flattens the linked-group summaries the API returns alongside
// an approval rule. A typed gitlab.group resource carries far more fields than
// the approval-rule payload provides, so we expose the subset as a dict.
func groupsToDicts(groups []*gitlab.Group) []any {
	out := make([]any, 0, len(groups))
	for _, g := range groups {
		if g == nil {
			continue
		}
		out = append(out, map[string]any{
			"id":          g.ID,
			"name":        g.Name,
			"fullPath":    g.FullPath,
			"visibility":  string(g.Visibility),
			"description": g.Description,
		})
	}
	return out
}

// approvalRuleProtectedBranches constructs the per-branch sub-resources
// associated with an approval rule. defaultBranchName comes from the parent
// project so each branch's `defaultBranch` flag stays consistent with the
// values surfaced by gitlab.project.protectedBranches().
//
// Both this path and gitlab.project.protectedBranches() populate the same four
// fields (name, allowForcePush, defaultBranch, codeOwnerApproval) from the
// same SDK type (*gitlab.ProtectedBranch). The GitLab `/approval_rules`
// response embeds the full ProtectedBranch object — not a name-only summary —
// so caching by name produces identical data regardless of which producer
// touches the cache first.
func approvalRuleProtectedBranches(runtime *plugin.Runtime, branches []*gitlab.ProtectedBranch, defaultBranchName string) ([]any, error) {
	out := make([]any, 0, len(branches))
	for _, b := range branches {
		if b == nil {
			continue
		}
		mqlBranch, err := CreateResource(runtime, "gitlab.project.protectedBranch", map[string]*llx.RawData{
			"name":              llx.StringData(b.Name),
			"allowForcePush":    llx.BoolData(b.AllowForcePush),
			"defaultBranch":     llx.BoolData(b.Name == defaultBranchName),
			"codeOwnerApproval": llx.BoolData(b.CodeOwnerApprovalRequired),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, mqlBranch)
	}
	return out, nil
}

// mergeMethod fetches the project merge method
func (p *mqlGitlabProject) mergeMethod() (string, error) {
	conn := p.MqlRuntime.Connection.(*connection.GitLabConnection)

	project, err := p.projectDetails(conn)
	if err != nil {
		// Stay consistent with containerExpirationPolicy and initGitlabProject:
		// 403/404 on the shared GetProject call shouldn't fail a broader query
		// just because the token can't read this scalar.
		if p.detailsStatusCode == 403 || p.detailsStatusCode == 404 {
			return "", nil
		}
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
	defaultBranch := p.DefaultBranch.Data

	perPage := int64(50)
	page := int64(1)
	var protectedBranches []*gitlab.ProtectedBranch
	for {
		branches, resp, err := conn.Client().ProtectedBranches.ListProtectedBranches(projectID, &gitlab.ListProtectedBranchesOptions{ListOptions: gitlab.ListOptions{Page: page, PerPage: perPage}})
		if err != nil {
			return nil, err
		}
		protectedBranches = append(protectedBranches, branches...)
		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
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

	perPage := int64(50)
	page := int64(1)
	var allMembers []*gitlab.ProjectMember

	for {
		members, resp, err := conn.Client().ProjectMembers.ListAllProjectMembers(projectID, &gitlab.ListProjectMembersOptions{
			ListOptions: gitlab.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		})
		if err != nil {
			return nil, err
		}

		allMembers = append(allMembers, members...)

		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	var mqlMembers []any
	for _, member := range allMembers {
		role := mapAccessLevelToRole(int(member.AccessLevel))

		// Seed only the id and let initGitlabUser lazily fetch the full record.
		// Hardcoding zero values for fields the member payload doesn't carry
		// (twoFactorEnabled, locked, bot, jobTitle, organization, location)
		// would poison the runtime cache for any later gitlab.user lookup on
		// the same id — e.g. reporting twoFactorEnabled=false for a user who
		// actually has it enabled.
		mqlUser, err := NewResource(p.MqlRuntime, "gitlab.user", map[string]*llx.RawData{
			"id": llx.IntData(int64(member.ID)),
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

// mqlGitlabProjectFileInternal carries the parent project context needed to
// lazily fetch file content. We don't expose these as schema fields because
// they only exist to satisfy the per-file GetFile lookup and would clutter
// the resource for users.
type mqlGitlabProjectFileInternal struct {
	projectID int
	ref       string
}

// content lazily fetches the file's raw content via the repository files
// endpoint. Eager fetching here would force an HTTP call per blob even for
// queries that never read `content` (e.g. `projectFiles { path }`); doing
// it on demand keeps the cost proportional to what the query asks for.
func (f *mqlGitlabProjectFile) content() (string, error) {
	if f.projectID == 0 || f.ref == "" || f.Path.Error != nil {
		return "", errors.New("gitlab.project.file: missing project context for content fetch")
	}
	conn := f.MqlRuntime.Connection.(*connection.GitLabConnection)
	fileContent, _, err := conn.Client().RepositoryFiles.GetFile(f.projectID, f.Path.Data, &gitlab.GetFileOptions{Ref: &f.ref})
	if err != nil {
		return "", err
	}
	contentBytes, err := base64.StdEncoding.DecodeString(fileContent.Content)
	if err != nil {
		return "", err
	}
	return string(contentBytes), nil
}

// projectFiles fetches the list of files in the project repository. File
// content is fetched lazily on access (see gitlab.project.file.content) so
// that queries reading only path/name/type don't pay for an HTTP round-trip
// per blob.
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
		ListOptions: gitlab.ListOptions{PerPage: 100},
		Ref:         ref,
		Recursive:   &recursive,
	}

	var files []*gitlab.TreeNode
	for {
		batch, resp, err := conn.Client().Repositories.ListTree(projectID, listFilesOptions)
		if err != nil {
			if resp != nil && resp.StatusCode == 404 {
				// new project with no commits/files
				break
			}
			return nil, fmt.Errorf("failed to list files in repository: %w", err)
		}
		files = append(files, batch...)
		if resp.NextPage == 0 {
			break
		}
		listFilesOptions.Page = resp.NextPage
	}

	var mqlFiles []any
	for _, file := range files {
		if file.Type != "blob" {
			continue
		}
		fileInfo := map[string]*llx.RawData{
			"path": llx.StringData(file.Path),
			"type": llx.StringData(file.Type),
			"name": llx.StringData(file.Name),
		}

		mqlFile, err := CreateResource(p.MqlRuntime, "gitlab.project.file", fileInfo)
		if err != nil {
			return nil, err
		}
		mf := mqlFile.(*mqlGitlabProjectFile)
		mf.projectID = projectID
		mf.ref = defaultBranch

		mqlFiles = append(mqlFiles, mqlFile)
	}

	return mqlFiles, nil
}

// id function for gitlab.project.webhook - hook ID is unique per project, but
// hooks are scoped to a project so we include the project ID for global uniqueness.
func (g *mqlGitlabProjectWebhook) id() (string, error) {
	return "gitlab.project.webhook/" + strconv.FormatInt(g.Id.Data, 10), nil
}

// mqlGitlabProjectWebhookInternal stores parent project context for typed refs.
type mqlGitlabProjectWebhookInternal struct {
	projectID int64
}

// project returns a typed reference to the parent project this webhook is registered on.
func (h *mqlGitlabProjectWebhook) project() (*mqlGitlabProject, error) {
	if h.projectID == 0 {
		h.Project.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlProject, err := NewResource(h.MqlRuntime, "gitlab.project", map[string]*llx.RawData{
		"id": llx.IntData(h.projectID),
	})
	if err != nil {
		return nil, err
	}
	return mqlProject.(*mqlGitlabProject), nil
}

// webhooks fetches the webhooks for a project. The list/get hook responses
// from GitLab never include the configured secret token (write-only field), so
// we cannot expose token presence or value here - sslVerification + the per-event
// trigger flags are the auditable surface.
func (p *mqlGitlabProject) webhooks() ([]any, error) {
	conn := p.MqlRuntime.Connection.(*connection.GitLabConnection)

	projectID := int(p.Id.Data)

	perPage := int64(50)
	page := int64(1)
	var allHooks []*gitlab.ProjectHook

	for {
		hooks, resp, err := conn.Client().Projects.ListProjectHooks(projectID, &gitlab.ListProjectHooksOptions{
			ListOptions: gitlab.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		})
		if err != nil {
			return nil, err
		}

		allHooks = append(allHooks, hooks...)

		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	var mqlWebhooks []any
	for _, hook := range allHooks {
		hookInfo := map[string]*llx.RawData{
			"id":                        llx.IntData(hook.ID),
			"url":                       llx.StringData(hook.URL),
			"name":                      llx.StringData(hook.Name),
			"description":               llx.StringData(hook.Description),
			"sslVerification":           llx.BoolData(hook.EnableSSLVerification),
			"pushEvents":                llx.BoolData(hook.PushEvents),
			"pushEventsBranchFilter":    llx.StringData(hook.PushEventsBranchFilter),
			"issuesEvents":              llx.BoolData(hook.IssuesEvents),
			"confidentialIssuesEvents":  llx.BoolData(hook.ConfidentialIssuesEvents),
			"mergeRequestsEvents":       llx.BoolData(hook.MergeRequestsEvents),
			"tagPushEvents":             llx.BoolData(hook.TagPushEvents),
			"noteEvents":                llx.BoolData(hook.NoteEvents),
			"confidentialNoteEvents":    llx.BoolData(hook.ConfidentialNoteEvents),
			"jobEvents":                 llx.BoolData(hook.JobEvents),
			"pipelineEvents":            llx.BoolData(hook.PipelineEvents),
			"wikiPageEvents":            llx.BoolData(hook.WikiPageEvents),
			"deploymentEvents":          llx.BoolData(hook.DeploymentEvents),
			"releasesEvents":            llx.BoolData(hook.ReleasesEvents),
			"resourceAccessTokenEvents": llx.BoolData(hook.ResourceAccessTokenEvents),
			"vulnerabilityEvents":       llx.BoolData(hook.VulnerabilityEvents),
			"featureFlagEvents":         llx.BoolData(hook.FeatureFlagEvents),
			"milestoneEvents":           llx.BoolData(hook.MilestoneEvents),
			"emojiEvents":               llx.BoolData(hook.EmojiEvents),
			"repositoryUpdateEvents":    llx.BoolData(hook.RepositoryUpdateEvents),
			"branchFilterStrategy":      llx.StringData(hook.BranchFilterStrategy),
			"customWebhookTemplate":     llx.StringData(hook.CustomWebhookTemplate),
			"createdAt":                 llx.TimeDataPtr(hook.CreatedAt),
			"disabledUntil":             llx.TimeDataPtr(hook.DisabledUntil),
			"alertStatus":               llx.StringData(hook.AlertStatus),
		}

		mqlWebhook, err := CreateResource(p.MqlRuntime, "gitlab.project.webhook", hookInfo)
		if err != nil {
			return nil, err
		}

		mqlWebhook.(*mqlGitlabProjectWebhook).projectID = p.Id.Data
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

// milestone fetches the milestone for a merge request. The milestone is
// populated eagerly when the merge request is materialized; this fallback
// covers MRs without an attached milestone.
func (m *mqlGitlabProjectMergeRequest) milestone() (*mqlGitlabProjectMilestone, error) {
	m.Milestone.State = plugin.StateIsSet | plugin.StateIsNull
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

// milestone fetches the milestone for an issue. The milestone is populated
// eagerly when the issue is materialized; this fallback covers issues without
// an attached milestone.
func (i *mqlGitlabProjectIssue) milestone() (*mqlGitlabProjectMilestone, error) {
	i.Milestone.State = plugin.StateIsSet | plugin.StateIsNull
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
			p.PushRules.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil // no push rules configured
		}
		return nil, err
	}
	if rules == nil {
		p.PushRules.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
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

// mqlGitlabProjectDeployKeyInternal stores parent project context for typed refs.
type mqlGitlabProjectDeployKeyInternal struct {
	projectID int64
}

// daysOld returns the age of the deploy key in days. Returns -1 when createdAt
// isn't set so callers can distinguish "missing data" from "fresh key".
func (k *mqlGitlabProjectDeployKey) daysOld() (int64, error) {
	if !k.CreatedAt.IsSet() || k.CreatedAt.Data == nil || k.CreatedAt.Data.IsZero() {
		return -1, nil
	}
	return int64(time.Since(*k.CreatedAt.Data).Hours() / 24), nil
}

// project returns a typed reference to the parent project the deploy key is registered against.
func (k *mqlGitlabProjectDeployKey) project() (*mqlGitlabProject, error) {
	if k.projectID == 0 {
		k.Project.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlProject, err := NewResource(k.MqlRuntime, "gitlab.project", map[string]*llx.RawData{
		"id": llx.IntData(k.projectID),
	})
	if err != nil {
		return nil, err
	}
	return mqlProject.(*mqlGitlabProject), nil
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

		mqlKey.(*mqlGitlabProjectDeployKey).projectID = p.Id.Data
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
			p.SecuritySettings.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil // not available on this GitLab tier
		}
		return nil, err
	}
	if settings == nil {
		p.SecuritySettings.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
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
