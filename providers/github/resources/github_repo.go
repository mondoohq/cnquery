// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-github/v61/github"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/github/connection"
	"go.mondoo.com/cnquery/v11/types"
)

func newMqlGithubRepository(runtime *plugin.Runtime, repo *github.Repository) (*mqlGithubRepository, error) {
	var id int64
	if repo.ID != nil {
		id = *repo.ID
	}

	owner, err := NewResource(runtime, "github.user", map[string]*llx.RawData{
		"id":    llx.IntData(repo.GetOwner().GetID()),
		"login": llx.StringData(repo.GetOwner().GetLogin()),
	})
	if err != nil {
		return nil, err
	}

	res, err := CreateResource(runtime, "github.repository", map[string]*llx.RawData{
		"id":                       llx.IntData(id),
		"name":                     llx.StringDataPtr(repo.Name),
		"fullName":                 llx.StringDataPtr(repo.FullName),
		"description":              llx.StringDataPtr(repo.Description),
		"homepage":                 llx.StringDataPtr(repo.Homepage),
		"topics":                   llx.ArrayData(convert.SliceAnyToInterface[string](repo.Topics), types.String),
		"language":                 llx.StringData(repo.GetLanguage()),
		"createdAt":                llx.TimeDataPtr(githubTimestamp(repo.CreatedAt)),
		"updatedAt":                llx.TimeDataPtr(githubTimestamp(repo.UpdatedAt)),
		"pushedAt":                 llx.TimeDataPtr(githubTimestamp(repo.PushedAt)),
		"archived":                 llx.BoolDataPtr(repo.Archived),
		"disabled":                 llx.BoolDataPtr(repo.Disabled),
		"private":                  llx.BoolDataPtr(repo.Private),
		"isFork":                   llx.BoolDataPtr(repo.Fork),
		"watchersCount":            llx.IntData(int64(repo.GetWatchersCount())),
		"forksCount":               llx.IntData(int64(repo.GetForksCount())),
		"openIssuesCount":          llx.IntData(int64(repo.GetOpenIssues())),
		"stargazersCount":          llx.IntData(int64(repo.GetStargazersCount())),
		"visibility":               llx.StringDataPtr(repo.Visibility),
		"allowAutoMerge":           llx.BoolData(convert.ToBool(repo.AllowAutoMerge)),
		"allowForking":             llx.BoolData(convert.ToBool(repo.AllowForking)),
		"allowMergeCommit":         llx.BoolData(convert.ToBool(repo.AllowMergeCommit)),
		"allowRebaseMerge":         llx.BoolData(convert.ToBool(repo.AllowRebaseMerge)),
		"allowSquashMerge":         llx.BoolData(convert.ToBool(repo.AllowSquashMerge)),
		"allowUpdateBranch":        llx.BoolData(convert.ToBool(repo.AllowUpdateBranch)),
		"webCommitSignoffRequired": llx.BoolData(convert.ToBool(repo.WebCommitSignoffRequired)),
		"deleteBranchOnMerge":      llx.BoolData(convert.ToBool(repo.DeleteBranchOnMerge)),
		"hasIssues":                llx.BoolData(repo.GetHasIssues()),
		"hasProjects":              llx.BoolData(repo.GetHasProjects()),
		"hasWiki":                  llx.BoolData(repo.GetHasWiki()),
		"hasPages":                 llx.BoolData(repo.GetHasPages()),
		"hasDownloads":             llx.BoolData(repo.GetHasDownloads()),
		"hasDiscussions":           llx.BoolData(repo.GetHasDiscussions()),
		"isTemplate":               llx.BoolData(repo.GetIsTemplate()),
		"defaultBranchName":        llx.StringDataPtr(repo.DefaultBranch),
		"cloneUrl":                 llx.StringData(repo.GetCloneURL()),
		"sshUrl":                   llx.StringData(repo.GetSSHURL()),
		"owner":                    llx.ResourceData(owner, owner.MqlName()),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGithubRepository), nil
}

func (g *mqlGithubBranchprotection) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	return g.Id.Data, nil
}

func (g *mqlGithubBranch) id() (string, error) {
	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	branchName := g.Name.Data
	if g.RepoName.Error != nil {
		return "", g.RepoName.Error
	}
	repoName := g.RepoName.Data
	return repoName + "/" + branchName, nil
}

func (g *mqlGithubCommit) id() (string, error) {
	// the url is unique, e.g. "https://api.github.com/repos/vjeffrey/victoria-website/git/commits/7730d2707fdb6422f335fddc944ab169d45f3aa5"
	if g.Url.Error != nil {
		return "", g.Url.Error
	}
	return g.Url.Data, nil
}

func (g *mqlGithubReview) id() (string, error) {
	if g.Url.Error != nil {
		return "", g.Url.Error
	}
	return g.Url.Data, nil
}

func (g *mqlGithubRelease) id() (string, error) {
	if g.Url.Error != nil {
		return "", g.Url.Error
	}
	return g.Url.Data, nil
}

func (g *mqlGithubRepository) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return strconv.FormatInt(id, 10), nil
}

func initGithubRepository(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	conn := runtime.Connection.(*connection.GithubConnection)

	var org *mqlGithubOrganization
	var user *mqlGithubUser
	var err error
	orgId, err := conn.Organization()
	if err == nil {
		obj, err := NewResource(runtime, "github.organization", map[string]*llx.RawData{
			"login": llx.StringData(orgId.Name),
		})
		if err != nil {
			// If the owner isn't an org, we try to find a user
			if strings.Contains(err.Error(), "404") {
				userId, err := conn.User()
				if err != nil {
					return nil, nil, err
				}
				obj, err = CreateResource(runtime, "github.user", map[string]*llx.RawData{
					"login": llx.StringData(userId.Name),
				})
				if err != nil {
					// If a user doesn't exist either, then we error out
					return nil, nil, err
				}
				user = obj.(*mqlGithubUser)
			}
			return nil, nil, err
		} else {
			org = obj.(*mqlGithubOrganization)
		}
	}

	reponame := ""
	if x, ok := args["name"]; ok {
		reponame = x.Value.(string)
	} else {
		repo, err := conn.Repository()
		if err != nil {
			return nil, nil, err
		}
		reponame = repo.Name
	}

	if reponame == "" {
		return nil, nil, errors.New("name must be set for github.repository initialization")
	}

	var repos *plugin.TValue[[]interface{}]
	if org != nil {
		repos = org.GetRepositories()
		if repos.Error != nil {
			return nil, nil, repos.Error
		}
	} else if user != nil {
		repos = user.GetRepositories()
		if repos.Error != nil {
			return nil, nil, repos.Error
		}
	} else {
		return nil, nil, errors.New("no user and no org specified")
	}

	for _, obj := range repos.Data {
		repo := obj.(*mqlGithubRepository)
		if repo.Name.Data == reponame {
			return args, repo, nil
		}
	}

	return args, nil, fmt.Errorf("could not find repository %q. Make sure the repository exists and the token has sufficient permissions to access it", reponame)
}

func (g *mqlGithubLicense) id() (string, error) {
	if g.SpdxId.Error != nil {
		return "", g.SpdxId.Error
	}
	id := g.SpdxId.Data
	return "github.license/" + id, nil
}

func (g *mqlGithubRepository) license() (*mqlGithubLicense, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	repoName := g.Name.Data
	if g.Owner.Error != nil {
		return nil, g.Owner.Error
	}
	ownerName := g.Owner.Data
	if ownerName.Login.Error != nil {
		return nil, ownerName.Login.Error
	}
	ownerLogin := ownerName.Login.Data

	repoLicense, _, err := conn.Client().Repositories.License(context.Background(), ownerLogin, repoName)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, errors.New("not found")
		}
		return nil, err
	}

	if repoLicense == nil || repoLicense.License == nil {
		return nil, nil
	}

	license := repoLicense.License
	res, err := CreateResource(g.MqlRuntime, "github.license", map[string]*llx.RawData{
		"key":    llx.StringData(license.GetKey()),
		"name":   llx.StringData(license.GetName()),
		"url":    llx.StringData(license.GetURL()),
		"spdxId": llx.StringData(license.GetSPDXID()),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGithubLicense), nil
}

func (g *mqlGithubRepository) getMergeRequests(state string) ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	repoName := g.Name.Data
	if g.Owner.Error != nil {
		return nil, g.Owner.Error
	}
	ownerName := g.Owner.Data
	if ownerName.Login.Error != nil {
		return nil, ownerName.Login.Error
	}
	ownerLogin := ownerName.Login.Data

	listOpts := &github.PullRequestListOptions{
		ListOptions: github.ListOptions{PerPage: paginationPerPage},
		State:       state,
	}
	var allPulls []*github.PullRequest
	for {
		pulls, resp, err := conn.Client().PullRequests.List(context.TODO(), ownerLogin, repoName, listOpts)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				return nil, nil
			}
			return nil, err
		}
		allPulls = append(allPulls, pulls...)
		if resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}

	res := []interface{}{}
	for i := range allPulls {
		pr := allPulls[i]

		labels, err := convert.JsonToDictSlice(pr.Labels)
		if err != nil {
			return nil, err
		}
		owner, err := NewResource(g.MqlRuntime, "github.user", map[string]*llx.RawData{
			"id":    llx.IntDataPtr(pr.User.ID),
			"login": llx.StringDataPtr(pr.User.Login),
		})
		if err != nil {
			return nil, err
		}

		assigneesRes := []interface{}{}
		for i := range pr.Assignees {
			assignee, err := NewResource(g.MqlRuntime, "github.user", map[string]*llx.RawData{
				"id":    llx.IntDataPtr(pr.Assignees[i].ID),
				"login": llx.StringDataPtr(pr.Assignees[i].Login),
			})
			if err != nil {
				return nil, err
			}
			assigneesRes = append(assigneesRes, assignee)
		}

		r, err := CreateResource(g.MqlRuntime, "github.mergeRequest", map[string]*llx.RawData{
			"id":        llx.IntDataPtr(pr.ID),
			"number":    llx.IntData(int64(*pr.Number)),
			"state":     llx.StringDataPtr(pr.State),
			"labels":    llx.ArrayData(labels, types.Any),
			"createdAt": llx.TimeDataPtr(githubTimestamp(pr.CreatedAt)),
			"title":     llx.StringDataPtr(pr.Title),
			"owner":     llx.ResourceData(owner, owner.MqlName()),
			"assignees": llx.ArrayData(assigneesRes, types.Any),
			"repoName":  llx.StringData(repoName),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (g *mqlGithubRepository) allMergeRequests() ([]interface{}, error) {
	res, err := g.getMergeRequests("all")
	if err != nil {
		return nil, err
	}
	return res, err
}

func (g *mqlGithubRepository) closedMergeRequests() ([]interface{}, error) {
	res, err := g.getMergeRequests("closed")
	if err != nil {
		return nil, err
	}
	return res, err
}

func (g *mqlGithubRepository) openMergeRequests() ([]interface{}, error) {
	res, err := g.getMergeRequests("open")
	if err != nil {
		return nil, err
	}
	return res, err
}

func (g *mqlGithubMergeRequest) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return strconv.FormatInt(id, 10), nil
}

func (g *mqlGithubRepository) branches() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	repoName := g.Name.Data
	if g.Owner.Error != nil {
		return nil, g.Owner.Error
	}
	owner := g.Owner.Data

	if owner.Login.Error != nil {
		return nil, owner.Login.Error
	}
	ownerLogin := owner.Login.Data

	if g.DefaultBranchName.Error != nil {
		return nil, g.DefaultBranchName.Error
	}
	repoDefaultBranchName := g.DefaultBranchName.Data

	listOpts := &github.BranchListOptions{
		ListOptions: github.ListOptions{PerPage: paginationPerPage},
	}
	var allBranches []*github.Branch
	for {
		branches, resp, err := conn.Client().Repositories.ListBranches(context.TODO(), ownerLogin, repoName, listOpts)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				return nil, nil
			}
			return nil, err
		}
		allBranches = append(allBranches, branches...)
		if resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}
	res := []interface{}{}
	for i := range allBranches {
		branch := allBranches[i]
		rc := branch.Commit
		mqlCommit, err := newMqlGithubCommit(g.MqlRuntime, rc, ownerLogin, repoName)
		if err != nil {
			return nil, err
		}

		defaultBranch := false
		if repoDefaultBranchName == *branch.Name {
			defaultBranch = true
		}

		mqlBranch, err := CreateResource(g.MqlRuntime, "github.branch", map[string]*llx.RawData{
			"name":        llx.StringData(branch.GetName()),
			"isProtected": llx.BoolData(branch.GetProtected()),
			"headCommit":  llx.AnyData(mqlCommit),
			"repoName":    llx.StringData(repoName),
			"owner":       llx.ResourceData(owner, owner.MqlName()),
			"isDefault":   llx.BoolData(defaultBranch),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlBranch)
	}
	return res, nil
}

type githubDismissalRestrictions struct {
	Users []string `json:"users"`
	Teams []string `json:"teams"`
}

type githubRequiredPullRequestReviews struct {
	DismissalRestrictions *githubDismissalRestrictions `json:"dismissalRestrictions"`
	// Specifies if approved reviews are dismissed automatically, when a new commit is pushed.
	DismissStaleReviews bool `json:"dismissStaleReviews"`
	// RequireCodeOwnerReviews specifies if an approved review is required in pull requests including files with a designated code owner.
	RequireCodeOwnerReviews bool `json:"requireCodeOwnerReviews"`
	// RequiredApprovingReviewCount specifies the number of approvals required before the pull request can be merged.
	// Valid values are 1-6.
	RequiredApprovingReviewCount int `json:"requiredApprovingReviewCount"`
}

func (g *mqlGithubBranch) protectionRules() (*mqlGithubBranchprotection, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)
	var err error

	if g.RepoName.Error != nil {
		return nil, g.RepoName.Error
	}
	repoName := g.RepoName.Data
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	branchName := g.Name.Data
	if g.Owner.Error != nil {
		return nil, g.Owner.Error
	}
	owner := g.Owner.Data
	if owner.Login.Error != nil {
		log.Debug().Err(err).Msg("note: branch protection can only be accessed by admin users")
		if strings.Contains(owner.Login.Error.Error(), "404") {
			g.ProtectionRules.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		}
		return nil, owner.Login.Error
	}
	ownerName := owner.Login.Data

	branchProtection, _, err := conn.Client().Repositories.GetBranchProtection(context.TODO(), ownerName, repoName, branchName)
	if err != nil {
		// NOTE it is possible that the branch does not have any protection rules, therefore we don't return an error
		if strings.Contains(err.Error(), "Not Found") {
			g.ProtectionRules.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		}
		// TODO: figure out if the client has the permission to fetch the protection rules
		return nil, err
	}

	rsc, err := convert.JsonToDict(branchProtection.RequiredStatusChecks)
	if err != nil {
		return nil, err
	}

	var ghDismissalRestrictions *githubDismissalRestrictions

	var rprr map[string]interface{}
	if branchProtection.RequiredPullRequestReviews != nil {

		if branchProtection.RequiredPullRequestReviews.DismissalRestrictions != nil {
			ghDismissalRestrictions = &githubDismissalRestrictions{
				Users: make([]string, 0, len(branchProtection.RequiredPullRequestReviews.DismissalRestrictions.Users)),
				Teams: make([]string, 0, len(branchProtection.RequiredPullRequestReviews.DismissalRestrictions.Teams)),
			}

			for i := range branchProtection.RequiredPullRequestReviews.DismissalRestrictions.Teams {
				ghDismissalRestrictions.Teams = append(ghDismissalRestrictions.Teams, branchProtection.RequiredPullRequestReviews.DismissalRestrictions.Teams[i].GetName())
			}
			for i := range branchProtection.RequiredPullRequestReviews.DismissalRestrictions.Users {
				ghDismissalRestrictions.Users = append(ghDismissalRestrictions.Users, branchProtection.RequiredPullRequestReviews.DismissalRestrictions.Users[i].GetLogin())
			}
		}

		// we use a separate struct to ensure that the output is proper camelCase
		rprr, err = convert.JsonToDict(githubRequiredPullRequestReviews{
			DismissStaleReviews:          branchProtection.RequiredPullRequestReviews.DismissStaleReviews,
			RequireCodeOwnerReviews:      branchProtection.RequiredPullRequestReviews.RequireCodeOwnerReviews,
			RequiredApprovingReviewCount: branchProtection.RequiredPullRequestReviews.RequiredApprovingReviewCount,
			DismissalRestrictions:        ghDismissalRestrictions,
		})
		if err != nil {
			return nil, err
		}
	}

	ea, err := convert.JsonToDict(branchProtection.EnforceAdmins)
	if err != nil {
		return nil, err
	}
	r, err := convert.JsonToDict(branchProtection.Restrictions)
	if err != nil {
		return nil, err
	}
	rlh, err := convert.JsonToDict(branchProtection.RequireLinearHistory)
	if err != nil {
		return nil, err
	}
	afp, err := convert.JsonToDict(branchProtection.AllowForcePushes)
	if err != nil {
		return nil, err
	}
	ad, err := convert.JsonToDict(branchProtection.AllowDeletions)
	if err != nil {
		return nil, err
	}
	rcr, err := convert.JsonToDict(branchProtection.RequiredConversationResolution)
	if err != nil {
		return nil, err
	}

	sc, _, err := conn.Client().Repositories.GetSignaturesProtectedBranch(context.TODO(), ownerName, repoName, branchName)
	if err != nil {
		log.Debug().Err(err).Msg("note: branch protection can only be accessed by admin users")
		return nil, err
	}

	res, err := CreateResource(g.MqlRuntime, "github.branchprotection", map[string]*llx.RawData{
		"id":                             llx.StringData(repoName + "/" + branchName),
		"requiredStatusChecks":           llx.MapData(rsc, types.Any),
		"requiredPullRequestReviews":     llx.MapData(rprr, types.Any),
		"enforceAdmins":                  llx.MapData(ea, types.Any),
		"restrictions":                   llx.MapData(r, types.Any),
		"requireLinearHistory":           llx.MapData(rlh, types.Any),
		"allowForcePushes":               llx.MapData(afp, types.Any),
		"allowDeletions":                 llx.MapData(ad, types.Any),
		"requiredConversationResolution": llx.MapData(rcr, types.Any),
		"requiredSignatures":             llx.BoolDataPtr(sc.Enabled),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGithubBranchprotection), nil
}

func newMqlGithubCommit(runtime *plugin.Runtime, rc *github.RepositoryCommit, owner string, repo string) (interface{}, error) {
	var githubAuthor interface{}
	var err error
	conn := runtime.Connection.(*connection.GithubConnection)

	// if the github author is nil, we have to load the commit again
	if rc.Author == nil {
		rc, _, err = conn.Client().Repositories.GetCommit(context.TODO(), owner, repo, rc.GetSHA(), nil)
		if err != nil {
			return nil, err
		}
	}

	if rc.Author != nil && rc.Author.ID != nil && rc.Author.Login != nil {
		githubAuthor, err = NewResource(runtime, "github.user", map[string]*llx.RawData{
			"id":    llx.IntDataPtr(rc.Author.ID),
			"login": llx.StringDataPtr(rc.Author.Login),
		})
		if err != nil {
			return nil, err
		}
	}
	var githubCommitter interface{}
	if rc.Committer != nil && rc.Committer.ID != nil && rc.Committer.Login != nil {
		githubCommitter, err = NewResource(runtime, "github.user", map[string]*llx.RawData{
			"id":    llx.IntDataPtr(rc.Committer.ID),
			"login": llx.StringDataPtr(rc.Committer.Login),
		})
		if err != nil {
			return nil, err
		}
	}

	sha := rc.GetSHA()

	stats, err := convert.JsonToDict(rc.GetStats())
	if err != nil {
		return nil, err
	}

	mqlGitCommit, err := newMqlGitCommit(runtime, sha, rc.Commit)
	if err != nil {
		return nil, err
	}

	return CreateResource(runtime, "github.commit", map[string]*llx.RawData{
		"url":        llx.StringData(rc.GetURL()),
		"sha":        llx.StringData(sha),
		"author":     llx.AnyData(githubAuthor),
		"committer":  llx.AnyData(githubCommitter),
		"owner":      llx.StringData(owner),
		"repository": llx.StringData(repo),
		"commit":     llx.AnyData(mqlGitCommit),
		"stats":      llx.MapData(stats, types.Any),
	})
}

func (g *mqlGithubRepository) commits() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)

	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	repoName := g.Name.Data
	if g.Owner.Error != nil {
		return nil, g.Owner.Error
	}
	ownerName := g.Owner.Data
	if ownerName.Login.Error != nil {
		return nil, ownerName.Login.Error
	}
	ownerLogin := ownerName.Login.Data

	listOpts := &github.CommitsListOptions{
		ListOptions: github.ListOptions{PerPage: paginationPerPage},
	}
	var allCommits []*github.RepositoryCommit
	for {
		commits, resp, err := conn.Client().Repositories.ListCommits(context.TODO(), ownerLogin, repoName, listOpts)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				return nil, nil
			}
			return nil, err
		}
		allCommits = append(allCommits, commits...)
		if resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage

	}
	res := []interface{}{}
	for i := range allCommits {
		rc := allCommits[i]
		mqlCommit, err := newMqlGithubCommit(g.MqlRuntime, rc, ownerLogin, repoName)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlCommit)
	}
	return res, nil
}

func (g *mqlGithubMergeRequest) reviews() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)
	var err error
	if g.RepoName.Error != nil {
		return nil, g.RepoName.Error
	}
	repoName := g.RepoName.Data
	if g.Owner.Error != nil {
		return nil, g.Owner.Error
	}
	owner := g.Owner.Data
	if owner.Login.Error != nil {
		return nil, owner.Login.Error
	}
	ownerLogin := owner.Login.Data
	if g.Number.Error != nil {
		return nil, g.Number.Error
	}
	prID := g.Number.Data

	listOpts := &github.ListOptions{
		PerPage: paginationPerPage,
	}
	var allReviews []*github.PullRequestReview
	for {
		reviews, resp, err := conn.Client().PullRequests.ListReviews(context.TODO(), ownerLogin, repoName, int(prID), listOpts)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				return nil, nil
			}
			return nil, err
		}
		allReviews = append(allReviews, reviews...)
		if resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}
	res := []interface{}{}
	for i := range allReviews {
		r := allReviews[i]
		var user interface{}
		if r.User != nil {
			user, err = NewResource(g.MqlRuntime, "github.user", map[string]*llx.RawData{
				"id":    llx.IntDataPtr(r.User.ID),
				"login": llx.StringDataPtr(r.User.Login),
			})
			if err != nil {
				return nil, err
			}
		}
		mqlReview, err := CreateResource(g.MqlRuntime, "github.review", map[string]*llx.RawData{
			"url":               llx.StringDataPtr(r.HTMLURL),
			"state":             llx.StringDataPtr(r.State),
			"authorAssociation": llx.StringDataPtr(r.AuthorAssociation),
			"user":              llx.AnyData(user),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlReview)
	}

	return res, nil
}

func (g *mqlGithubMergeRequest) commits() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)
	if g.RepoName.Error != nil {
		return nil, g.RepoName.Error
	}
	repoName := g.RepoName.Data
	if g.Owner.Error != nil {
		return nil, g.Owner.Error
	}
	owner := g.Owner.Data
	if owner.Login.Error != nil {
		return nil, owner.Login.Error
	}
	ownerLogin := owner.Login.Data
	if g.Number.Error != nil {
		return nil, g.Number.Error
	}
	prID := g.Number.Data

	listOpts := &github.ListOptions{
		PerPage: paginationPerPage,
	}
	var allCommits []*github.RepositoryCommit
	for {
		commits, resp, err := conn.Client().PullRequests.ListCommits(context.TODO(), ownerLogin, repoName, int(prID), listOpts)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				return nil, nil
			}
			return nil, err
		}
		allCommits = append(allCommits, commits...)
		if resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}
	res := []interface{}{}
	for i := range allCommits {
		rc := allCommits[i]

		mqlCommit, err := newMqlGithubCommit(g.MqlRuntime, rc, ownerLogin, repoName)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlCommit)
	}
	return res, nil
}

func (g *mqlGithubRepository) contributors() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)

	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	repoName := g.Name.Data
	if g.Owner.Error != nil {
		return nil, g.Owner.Error
	}
	ownerName := g.Owner.Data
	if ownerName.Login.Error != nil {
		return nil, ownerName.Login.Error
	}
	ownerLogin := ownerName.Login.Data

	listOpts := &github.ListContributorsOptions{
		ListOptions: github.ListOptions{PerPage: paginationPerPage},
	}
	var allContributors []*github.Contributor
	for {
		contributors, resp, err := conn.Client().Repositories.ListContributors(context.TODO(), ownerLogin, repoName, listOpts)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				return nil, nil
			}
			return nil, err
		}
		allContributors = append(allContributors, contributors...)
		if resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}
	res := []interface{}{}
	for i := range allContributors {
		mqlUser, err := NewResource(g.MqlRuntime, "github.user", map[string]*llx.RawData{
			"id":    llx.IntDataPtr(allContributors[i].ID),
			"login": llx.StringDataPtr(allContributors[i].Login),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlUser)
	}
	return res, nil
}

func (g *mqlGithubRepository) collaborators() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)

	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	repoName := g.Name.Data
	if g.Owner.Error != nil {
		return nil, g.Owner.Error
	}
	ownerName := g.Owner.Data
	if ownerName.Login.Error != nil {
		return nil, ownerName.Login.Error
	}
	ownerLogin := ownerName.Login.Data

	listOpts := &github.ListCollaboratorsOptions{
		ListOptions: github.ListOptions{PerPage: paginationPerPage},
	}
	var allContributors []*github.User
	for {
		contributors, resp, err := conn.Client().Repositories.ListCollaborators(context.TODO(), ownerLogin, repoName, listOpts)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				return nil, nil
			}
			return nil, err
		}
		allContributors = append(allContributors, contributors...)
		if resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}
	res := []interface{}{}
	for i := range allContributors {
		contributor := allContributors[i]
		mqlUser, err := NewResource(g.MqlRuntime, "github.user", map[string]*llx.RawData{
			"id":    llx.IntDataPtr(contributor.ID),
			"login": llx.StringDataPtr(contributor.Login),
		})
		if err != nil {
			return nil, err
		}

		permissions := []string{}
		for k := range contributor.Permissions {
			permissions = append(permissions, k)
		}

		mqlContributor, err := CreateResource(g.MqlRuntime, "github.collaborator", map[string]*llx.RawData{
			"id":          llx.IntDataPtr(contributor.ID),
			"user":        llx.ResourceData(mqlUser, mqlUser.MqlName()),
			"permissions": llx.ArrayData(convert.SliceAnyToInterface[string](permissions), types.String),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlContributor)
	}
	return res, nil
}

func (g *mqlGithubRepository) releases() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)

	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	repoName := g.Name.Data
	if g.Owner.Error != nil {
		return nil, g.Owner.Error
	}
	ownerName := g.Owner.Data
	if ownerName.Login.Error != nil {
		return nil, ownerName.Login.Error
	}
	ownerLogin := ownerName.Login.Data

	listOpts := &github.ListOptions{
		PerPage: paginationPerPage,
	}
	var allReleases []*github.RepositoryRelease
	for {
		releases, resp, err := conn.Client().Repositories.ListReleases(context.TODO(), ownerLogin, repoName, listOpts)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				return nil, nil
			}
			return nil, err
		}
		allReleases = append(allReleases, releases...)
		if resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}

	res := []interface{}{}
	for i := range allReleases {
		r := allReleases[i]
		mqlUser, err := CreateResource(g.MqlRuntime, "github.release", map[string]*llx.RawData{
			"url":        llx.StringDataPtr(r.HTMLURL),
			"name":       llx.StringDataPtr(r.Name),
			"tagName":    llx.StringDataPtr(r.TagName),
			"preRelease": llx.BoolDataPtr(r.Prerelease),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlUser)
	}

	return res, nil
}

func (g *mqlGithubWebhook) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return "github.webhook/" + strconv.FormatInt(id, 10), nil
}

func (g *mqlGithubRepository) webhooks() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)

	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	repoName := g.Name.Data
	if g.Owner.Error != nil {
		return nil, g.Owner.Error
	}
	owner := g.Owner.Data
	if owner.Login.Error != nil {
		return nil, owner.Login.Error
	}
	ownerLogin := owner.Login.Data

	listOpts := &github.ListOptions{
		PerPage: paginationPerPage,
	}
	var allWebhooks []*github.Hook
	for {
		hooks, resp, err := conn.Client().Repositories.ListHooks(context.TODO(), ownerLogin, repoName, listOpts)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				return nil, nil
			}
			return nil, err
		}
		allWebhooks = append(allWebhooks, hooks...)
		if resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}
	res := []interface{}{}
	for i := range allWebhooks {
		h := allWebhooks[i]
		config, err := convert.JsonToDict(h.Config)
		if err != nil {
			return nil, err
		}

		mqlWebhook, err := CreateResource(g.MqlRuntime, "github.webhook", map[string]*llx.RawData{
			"id":     llx.IntDataPtr(h.ID),
			"name":   llx.StringDataPtr(h.Name),
			"events": llx.ArrayData(convert.SliceAnyToInterface[string](h.Events), types.String),
			"config": llx.MapData(config, types.Any),
			"url":    llx.StringDataPtr(h.URL),
			"active": llx.BoolDataPtr(h.Active),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlWebhook)
	}

	return res, nil
}

type mqlGithubWorkflowInternal struct {
	repositoryFullName string
	parentResource     *mqlGithubRepository
}

func (g *mqlGithubRepository) workflows() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)

	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	repoName := g.Name.Data
	if g.Owner.Error != nil {
		return nil, g.Owner.Error
	}
	owner := g.Owner.Data
	if owner.Login.Error != nil {
		return nil, owner.Login.Error
	}
	ownerLogin := owner.Login.Data

	if g.FullName.Error != nil {
		return nil, g.FullName.Error
	}
	fullName := g.FullName.Data

	listOpts := &github.ListOptions{
		PerPage: paginationPerPage,
	}
	var allWorkflows []*github.Workflow
	for {
		workflows, resp, err := conn.Client().Actions.ListWorkflows(context.Background(), ownerLogin, repoName, listOpts)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				return nil, nil
			}
			return nil, err
		}
		allWorkflows = append(allWorkflows, workflows.Workflows...)
		if resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}

	res := []interface{}{}
	for i := range allWorkflows {
		w := allWorkflows[i]

		mqlWebhook, err := CreateResource(g.MqlRuntime, "github.workflow", map[string]*llx.RawData{
			"id":        llx.IntDataPtr(w.ID),
			"name":      llx.StringDataPtr(w.Name),
			"path":      llx.StringDataPtr(w.Path),
			"state":     llx.StringDataPtr(w.State),
			"createdAt": llx.TimeDataPtr(githubTimestamp(w.CreatedAt)),
			"updatedAt": llx.TimeDataPtr(githubTimestamp(w.UpdatedAt)),
		})
		if err != nil {
			return nil, err
		}

		gw := mqlWebhook.(*mqlGithubWorkflow)
		gw.repositoryFullName = fullName
		res = append(res, gw)
	}
	return res, nil
}

func newMqlGithubFile(runtime *plugin.Runtime, ownerName string, repoName string, content *github.RepositoryContent) (*mqlGithubFile, error) {
	isBinary := false
	if convert.ToString(content.Type) == "file" {
		file := strings.Split(convert.ToString(content.Path), ".")
		if len(file) == 2 {
			isBinary = binaryFileTypes[file[1]]
		}
	}
	res, err := CreateResource(runtime, "github.file", map[string]*llx.RawData{
		"path":      llx.StringDataPtr(content.Path),
		"name":      llx.StringDataPtr(content.Name),
		"type":      llx.StringDataPtr(content.Type),
		"sha":       llx.StringDataPtr(content.SHA),
		"isBinary":  llx.BoolData(isBinary),
		"ownerName": llx.StringData(ownerName),
		"repoName":  llx.StringData(repoName),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGithubFile), nil
}

func (g *mqlGithubRepository) files() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)

	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	repoName := g.Name.Data

	if g.Owner.Error != nil {
		return nil, g.Owner.Error
	}
	owner := g.Owner.Data
	if owner.Login.Error != nil {
		return nil, owner.Login.Error
	}
	ownerLogin := owner.Login.Data
	_, dirContent, _, err := conn.Client().Repositories.GetContents(context.TODO(), ownerLogin, repoName, "", &github.RepositoryContentGetOptions{})
	if err != nil {
		log.Error().Err(err).Msg("unable to get contents list")
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}
	res := []interface{}{}
	for i := range dirContent {
		mqlFile, err := newMqlGithubFile(g.MqlRuntime, ownerLogin, repoName, dirContent[i])
		if err != nil {
			return nil, err
		}
		res = append(res, mqlFile)

	}
	return res, nil
}

var binaryFileTypes = map[string]bool{
	"crx":    true,
	"deb":    true,
	"dex":    true,
	"dey":    true,
	"elf":    true,
	"o":      true,
	"so":     true,
	"iso":    true,
	"class":  true,
	"jar":    true,
	"bundle": true,
	"dylib":  true,
	"lib":    true,
	"msi":    true,
	"dll":    true,
	"drv":    true,
	"efi":    true,
	"exe":    true,
	"ocx":    true,
	"pyc":    true,
	"pyo":    true,
	"par":    true,
	"rpm":    true,
	"whl":    true,
}

func (g *mqlGithubFile) id() (string, error) {
	if g.RepoName.Error != nil {
		return "", g.RepoName.Error
	}
	r := g.RepoName.Data
	if g.Path.Error != nil {
		return "", g.Path.Error
	}
	p := g.Path.Data
	if g.Sha.Error != nil {
		return "", g.Sha.Error
	}
	s := g.Sha.Data
	return r + "/" + p + "/" + s, nil
}

func (g *mqlGithubFile) files() ([]interface{}, error) {
	if g.Type.Error != nil {
		return nil, g.Type.Error
	}
	fileType := g.Type.Data
	if fileType != "dir" {
		return nil, nil
	}
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)
	if g.RepoName.Error != nil {
		return nil, g.RepoName.Error
	}
	repoName := g.RepoName.Data
	if g.OwnerName.Error != nil {
		return nil, g.OwnerName.Error
	}
	ownerName := g.OwnerName.Data
	if g.Path.Error != nil {
		return nil, g.Path.Error
	}
	path := g.Path.Data
	_, dirContent, _, err := conn.Client().Repositories.GetContents(context.TODO(), ownerName, repoName, path, &github.RepositoryContentGetOptions{})
	if err != nil {
		log.Error().Err(err).Msg("unable to get contents list")
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}
	res := []interface{}{}
	for i := range dirContent {
		isBinary := false
		if convert.ToString(dirContent[i].Type) == "file" {
			file := strings.Split(convert.ToString(dirContent[i].Path), ".")
			if len(file) == 2 {
				isBinary = binaryFileTypes[file[1]]
			}
		}
		mqlFile, err := CreateResource(g.MqlRuntime, "github.file", map[string]*llx.RawData{
			"path":      llx.StringDataPtr(dirContent[i].Path),
			"name":      llx.StringDataPtr(dirContent[i].Name),
			"type":      llx.StringDataPtr(dirContent[i].Type),
			"sha":       llx.StringDataPtr(dirContent[i].SHA),
			"isBinary":  llx.BoolData(isBinary),
			"ownerName": llx.StringData(ownerName),
			"repoName":  llx.StringData(repoName),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlFile)

	}
	return res, nil
}

func (g *mqlGithubFile) content() (string, error) {
	if g.Type.Error != nil {
		return "", g.Type.Error
	}
	fileType := g.Type.Data
	if fileType == "dir" {
		return "", nil
	}
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)
	if g.RepoName.Error != nil {
		return "", g.RepoName.Error
	}
	repoName := g.RepoName.Data
	if g.OwnerName.Error != nil {
		return "", g.OwnerName.Error
	}
	ownerName := g.OwnerName.Data
	if g.Path.Error != nil {
		return "", g.Path.Error
	}
	path := g.Path.Data
	fileContent, _, _, err := conn.Client().Repositories.GetContents(context.TODO(), ownerName, repoName, path, &github.RepositoryContentGetOptions{})
	if err != nil {
		log.Error().Err(err).Msg("unable to get contents list")
		if strings.Contains(err.Error(), "404") {
			return "", nil
		}
		return "", err
	}

	content, err := fileContent.GetContent()
	if err != nil {
		if strings.Contains(err.Error(), "unsupported content encoding: none") {
			// TODO: i'm unclear why this error happens. the function checks for bas64 encoding and empty string encoding. if it's neither, it returns this error.
			// the error blocks the rest of the output, so we log it instead
			log.Error().Msgf("unable to get content for path %v", path)
			return "", nil
		}
	}
	return content, nil
}

func (g *mqlGithubRepository) forks() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	repoName := g.Name.Data

	if g.Owner.Error != nil {
		return nil, g.Owner.Error
	}
	owner := g.Owner.Data
	if owner.Login.Error != nil {
		return nil, owner.Login.Error
	}
	ownerLogin := owner.Login.Data

	listOpts := &github.RepositoryListForksOptions{
		ListOptions: github.ListOptions{
			PerPage: paginationPerPage,
		},
	}
	var allForks []*github.Repository
	for {
		forks, resp, err := conn.Client().Repositories.ListForks(context.Background(), ownerLogin, repoName, listOpts)
		if err != nil {
			log.Error().Err(err).Msg("unable to get contents list")
			if strings.Contains(err.Error(), "404") {
				return nil, nil
			}
			return nil, err
		}
		allForks = append(allForks, forks...)
		if resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}

	res := []interface{}{}
	for i := range allForks {
		repo := allForks[i]
		r, err := newMqlGithubRepository(g.MqlRuntime, repo)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func (g *mqlGithubRepository) stargazers() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	repoName := g.Name.Data

	if g.Owner.Error != nil {
		return nil, g.Owner.Error
	}
	owner := g.Owner.Data
	if owner.Login.Error != nil {
		return nil, owner.Login.Error
	}
	ownerLogin := owner.Login.Data

	listOpts := &github.ListOptions{
		PerPage: paginationPerPage,
	}
	var allStargazers []*github.Stargazer
	for {
		stargazers, resp, err := conn.Client().Activity.ListStargazers(context.Background(), ownerLogin, repoName, listOpts)
		if err != nil {
			log.Error().Err(err).Msg("unable to get contents list")
			if strings.Contains(err.Error(), "404") {
				return nil, nil
			}
			return nil, err
		}
		allStargazers = append(allStargazers, stargazers...)
		if resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}

	res := []interface{}{}
	for i := range allStargazers {
		stargazer := allStargazers[i]
		r, err := NewResource(g.MqlRuntime, "github.user", map[string]*llx.RawData{
			"id":    llx.IntDataPtr(stargazer.User.ID),
			"login": llx.StringDataPtr(stargazer.User.Login),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func (g *mqlGithubIssue) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data

	return "github.issue/" + strconv.FormatInt(id, 10), nil
}

func (g *mqlGithubRepository) openIssues() ([]interface{}, error) {
	return g.getIssues("open")
}

func (g *mqlGithubRepository) closedIssues() ([]interface{}, error) {
	return g.getIssues("closed")
}

func (g *mqlGithubRepository) getIssues(state string) ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	repoName := g.Name.Data

	if g.Owner.Error != nil {
		return nil, g.Owner.Error
	}
	owner := g.Owner.Data
	if owner.Login.Error != nil {
		return nil, owner.Login.Error
	}
	ownerLogin := owner.Login.Data

	listOpts := &github.IssueListByRepoOptions{
		State: state,
		ListOptions: github.ListOptions{
			PerPage: paginationPerPage,
		},
	}
	var allIssues []*github.Issue
	for {
		issues, resp, err := conn.Client().Issues.ListByRepo(context.Background(), ownerLogin, repoName, listOpts)
		if err != nil {
			log.Error().Err(err).Msg("unable to get contents list")
			if strings.Contains(err.Error(), "404") {
				return nil, nil
			}
			return nil, err
		}
		allIssues = append(allIssues, issues...)
		if resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}

	res := []interface{}{}
	for i := range allIssues {
		issue := allIssues[i]

		var assignees []interface{}
		for _, assignee := range issue.Assignees {
			r, err := NewResource(g.MqlRuntime, "github.user", map[string]*llx.RawData{
				"id":    llx.IntDataPtr(assignee.ID),
				"login": llx.StringDataPtr(assignee.Login),
			})
			if err != nil {
				return nil, err
			}
			assignees = append(assignees, r)
		}

		var closedBy interface{}
		if issue.GetClosedBy() != nil {
			r, err := NewResource(g.MqlRuntime, "github.user", map[string]*llx.RawData{
				"id":    llx.IntDataPtr(issue.GetClosedBy().ID),
				"login": llx.StringDataPtr(issue.GetClosedBy().Login),
			})
			if err != nil {
				return nil, err
			}
			closedBy = r
		}

		r, err := CreateResource(g.MqlRuntime, "github.issue", map[string]*llx.RawData{
			"id":        llx.IntData(issue.GetID()),
			"number":    llx.IntData(int64(issue.GetNumber())),
			"title":     llx.StringData(issue.GetTitle()),
			"state":     llx.StringData(issue.GetState()),
			"body":      llx.StringData(issue.GetBody()),
			"url":       llx.StringData(issue.GetURL()),
			"createdAt": llx.TimeDataPtr(githubTimestamp(issue.CreatedAt)),
			"updatedAt": llx.TimeDataPtr(githubTimestamp(issue.UpdatedAt)),
			"closedAt":  llx.TimeDataPtr(githubTimestamp(issue.ClosedAt)),
			"assignees": llx.ArrayData(assignees, types.Any),
			"closedBy":  llx.AnyData(closedBy),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}
