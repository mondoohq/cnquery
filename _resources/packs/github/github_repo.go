package github

import (
	"context"
	"strconv"
	"strings"

	"github.com/google/go-github/v49/github"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func newMqlGithubRepository(runtime *resources.Runtime, repo *github.Repository) (interface{}, error) {
	var id int64
	if repo.ID != nil {
		id = *repo.ID
	}

	owner, err := runtime.CreateResource("github.user",
		"id", repo.GetOwner().GetID(),
		"login", repo.GetOwner().GetLogin(),
	)
	if err != nil {
		return nil, err
	}

	return runtime.CreateResource("github.repository",
		"id", id,
		"name", core.ToString(repo.Name),
		"fullName", core.ToString(repo.FullName),
		"description", core.ToString(repo.Description),
		"homepage", core.ToString(repo.Homepage),
		"topics", core.StrSliceToInterface(repo.Topics),
		"language", repo.GetLanguage(),
		"createdAt", githubTimestamp(repo.CreatedAt),
		"updatedAt", githubTimestamp(repo.UpdatedAt),
		"pushedAt", githubTimestamp(repo.PushedAt),
		"archived", core.ToBool(repo.Archived),
		"disabled", core.ToBool(repo.Disabled),
		"private", core.ToBool(repo.Private),
		"isFork", repo.GetFork(),
		"watchersCount", int64(repo.GetWatchersCount()),
		"forksCount", int64(repo.GetForksCount()),
		"openIssuesCount", int64(repo.GetOpenIssues()),
		"stargazersCount", int64(repo.GetStargazersCount()),
		"visibility", core.ToString(repo.Visibility),
		"allowAutoMerge", core.ToBool(repo.AllowAutoMerge),
		"allowForking", core.ToBool(repo.AllowForking),
		"allowMergeCommit", core.ToBool(repo.AllowMergeCommit),
		"allowRebaseMerge", core.ToBool(repo.AllowRebaseMerge),
		"allowSquashMerge", core.ToBool(repo.AllowSquashMerge),
		"hasIssues", repo.GetHasIssues(),
		"hasProjects", repo.GetHasProjects(),
		"hasWiki", repo.GetHasWiki(),
		"hasPages", repo.GetHasPages(),
		"hasDownloads", repo.GetHasDownloads(),
		"defaultBranchName", core.ToString(repo.DefaultBranch),
		"cloneUrl", repo.GetCloneURL(),
		"sshUrl", repo.GetSSHURL(),
		"owner", owner,
	)
}

func (g *mqlGithubBranchprotection) id() (string, error) {
	return g.Id()
}

func (g *mqlGithubBranch) id() (string, error) {
	branchName, err := g.Name()
	if err != nil {
		return "", err
	}
	repoName, err := g.RepoName()
	if err != nil {
		return "", err
	}
	return repoName + "/" + branchName, nil
}

func (g *mqlGithubCommit) id() (string, error) {
	// the url is unique, e.g. "https://api.github.com/repos/vjeffrey/victoria-website/git/commits/7730d2707fdb6422f335fddc944ab169d45f3aa5"
	return g.Url()
}

func (g *mqlGithubReview) id() (string, error) {
	return g.Url()
}

func (g *mqlGithubRelease) id() (string, error) {
	return g.Url()
}

func (g *mqlGithubRepository) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(id, 10), nil
}

func (g *mqlGithubRepository) init(args *resources.Args) (*resources.Args, GithubRepository, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	// userLogin := user.GetLogin()
	var org *github.Organization
	var user *github.User
	org, err = gt.Organization()
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			log.Debug().Msg("could not find organization, trying to get user")
			user, err = gt.User()
			if err != nil {
				return nil, nil, err
			}
		} else {
			return nil, nil, err
		}
	}

	owner := ""
	if org != nil {
		owner = org.GetLogin()
	} else if user != nil {
		owner = user.GetLogin()
	}
	reponame := ""
	if x, ok := (*args)["name"]; ok {
		reponame = x.(string)
	} else {
		repo, err := gt.Repository()
		if err != nil {
			return nil, nil, err
		}
		reponame = *repo.Name
	}
	// return nil, nil, errors.New("Wrong type for 'path' in github.repository initialization, it must be a string")

	if owner != "" && reponame != "" {
		repo, _, err := gt.Client().Repositories.Get(context.Background(), owner, reponame)
		if err != nil {
			return nil, nil, err
		}

		owner, err := g.MotorRuntime.CreateResource("github.user",
			"id", repo.GetOwner().GetID(),
			"login", repo.GetOwner().GetLogin(),
		)
		if err != nil {
			return nil, nil, err
		}

		(*args)["id"] = core.ToInt64(repo.ID)
		(*args)["name"] = core.ToString(repo.Name)
		(*args)["fullName"] = core.ToString(repo.FullName)
		(*args)["description"] = core.ToString(repo.Description)
		(*args)["homepage"] = core.ToString(repo.Homepage)
		(*args)["topics"] = core.StrSliceToInterface(repo.Topics)
		(*args)["language"] = repo.GetLanguage()
		(*args)["watchersCount"] = int64(repo.GetWatchersCount())
		(*args)["forksCount"] = int64(repo.GetForksCount())
		(*args)["openIssuesCount"] = int64(repo.GetOpenIssues())
		(*args)["stargazersCount"] = int64(repo.GetStargazersCount())
		(*args)["createdAt"] = githubTimestamp(repo.CreatedAt)
		(*args)["updatedAt"] = githubTimestamp(repo.UpdatedAt)
		(*args)["pushedAt"] = githubTimestamp(repo.PushedAt)
		(*args)["archived"] = core.ToBool(repo.Archived)
		(*args)["disabled"] = core.ToBool(repo.Disabled)
		(*args)["private"] = core.ToBool(repo.Private)
		(*args)["isFork"] = repo.GetFork()
		(*args)["visibility"] = core.ToString(repo.Visibility)
		(*args)["allowAutoMerge"] = core.ToBool(repo.AllowAutoMerge)
		(*args)["allowForking"] = core.ToBool(repo.AllowForking)
		(*args)["allowMergeCommit"] = core.ToBool(repo.AllowMergeCommit)
		(*args)["allowRebaseMerge"] = core.ToBool(repo.AllowRebaseMerge)
		(*args)["allowSquashMerge"] = core.ToBool(repo.AllowSquashMerge)
		(*args)["hasIssues"] = repo.GetHasIssues()
		(*args)["hasProjects"] = repo.GetHasProjects()
		(*args)["hasWiki"] = repo.GetHasWiki()
		(*args)["hasPages"] = repo.GetHasPages()
		(*args)["hasDownloads"] = repo.GetHasDownloads()
		(*args)["defaultBranchName"] = core.ToString(repo.DefaultBranch)
		(*args)["cloneUrl"] = repo.GetCloneURL()
		(*args)["sshUrl"] = repo.GetSSHURL()
		(*args)["owner"] = owner
	}

	return args, nil, nil
}

func (g *mqlGithubLicense) id() (string, error) {
	id, err := g.SpdxId()
	if err != nil {
		return "", err
	}
	return "github.license/" + id, nil
}

func (g *mqlGithubRepository) GetLicense() (interface{}, error) {
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	repoName, err := g.Name()
	if err != nil {
		return nil, err
	}
	ownerName, err := g.Owner()
	if err != nil {
		return nil, err
	}
	ownerLogin, err := ownerName.Login()
	if err != nil {
		return nil, err
	}

	repoLicense, _, err := gt.Client().Repositories.License(context.Background(), ownerLogin, repoName)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}

	if repoLicense == nil || repoLicense.License == nil {
		return nil, nil
	}

	license := repoLicense.License
	return g.MotorRuntime.CreateResource("github.license",
		"key", license.GetKey(),
		"name", license.GetName(),
		"url", license.GetURL(),
		"spdxId", license.GetSPDXID(),
	)
}

func (g *mqlGithubRepository) GetOpenMergeRequests() ([]interface{}, error) {
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	repoName, err := g.Name()
	if err != nil {
		return nil, err
	}
	ownerName, err := g.Owner()
	if err != nil {
		return nil, err
	}
	ownerLogin, err := ownerName.Login()
	if err != nil {
		return nil, err
	}

	listOpts := &github.PullRequestListOptions{
		ListOptions: github.ListOptions{PerPage: paginationPerPage},
		State:       "open",
	}
	var allPulls []*github.PullRequest
	for {
		pulls, resp, err := gt.Client().PullRequests.List(context.TODO(), ownerLogin, repoName, listOpts)
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

		labels, err := core.JsonToDictSlice(pr.Labels)
		if err != nil {
			return nil, err
		}
		owner, err := g.MotorRuntime.CreateResource("github.user",
			"id", core.ToInt64(pr.User.ID),
			"login", core.ToString(pr.User.Login),
		)
		if err != nil {
			return nil, err
		}

		assigneesRes := []interface{}{}
		for i := range pr.Assignees {
			assignee, err := g.MotorRuntime.CreateResource("github.user",
				"id", core.ToInt64(pr.Assignees[i].ID),
				"login", core.ToString(pr.Assignees[i].Login),
			)
			if err != nil {
				return nil, err
			}
			assigneesRes = append(assigneesRes, assignee)
		}

		r, err := g.MotorRuntime.CreateResource("github.mergeRequest",
			"id", core.ToInt64(pr.ID),
			"number", core.ToInt(pr.Number),
			"state", core.ToString(pr.State),
			"labels", labels,
			"createdAt", pr.CreatedAt,
			"title", core.ToString(pr.Title),
			"owner", owner,
			"assignees", assigneesRes,
			"repoName", repoName,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (g *mqlGithubMergeRequest) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(id, 10), nil
}

func (g *mqlGithubRepository) GetBranches() ([]interface{}, error) {
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	repoName, err := g.Name()
	if err != nil {
		return nil, err
	}
	owner, err := g.Owner()
	if err != nil {
		return nil, err
	}

	ownerLogin, err := owner.Login()
	if err != nil {
		return nil, err
	}

	repoDefaultBranchName, err := g.DefaultBranchName()
	if err != nil {
		return nil, err
	}

	listOpts := &github.BranchListOptions{
		ListOptions: github.ListOptions{PerPage: paginationPerPage},
	}
	var allBranches []*github.Branch
	for {
		branches, resp, err := gt.Client().Repositories.ListBranches(context.TODO(), ownerLogin, repoName, listOpts)
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
		mqlCommit, err := newMqlGithubCommit(g.MotorRuntime, rc, ownerLogin, repoName)
		if err != nil {
			return nil, err
		}

		defaultBranch := false
		if repoDefaultBranchName == core.ToString(branch.Name) {
			defaultBranch = true
		}

		mqlBranch, err := g.MotorRuntime.CreateResource("github.branch",
			"name", branch.GetName(),
			// FIXME: DEPRECATED, remove in v10.0 vv
			"protected", branch.GetProtected(),
			// ^^ deprecated
			"isProtected", branch.GetProtected(),
			"headCommit", mqlCommit,
			"repoName", repoName,
			"owner", owner,
			"isDefault", defaultBranch,
		)
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

func (g *mqlGithubBranch) GetProtectionRules() (interface{}, error) {
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	repoName, err := g.RepoName()
	if err != nil {
		return nil, err
	}
	branchName, err := g.Name()
	if err != nil {
		return nil, err
	}
	owner, err := g.Owner()
	if err != nil {
		return nil, err
	}
	ownerName, err := owner.Login()
	if err != nil {
		log.Debug().Err(err).Msg("note: branch protection can only be accessed by admin users")
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}

	branchProtection, _, err := gt.Client().Repositories.GetBranchProtection(context.TODO(), ownerName, repoName, branchName)
	if err != nil {
		// NOTE it is possible that the branch does not have any protection rules, therefore we don't return an error
		// TODO: figure out if the client has the permission to fetch the protection rules
		return nil, nil
	}

	rsc, err := core.JsonToDict(branchProtection.RequiredStatusChecks)
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
		rprr, err = core.JsonToDict(githubRequiredPullRequestReviews{
			DismissStaleReviews:          branchProtection.RequiredPullRequestReviews.DismissStaleReviews,
			RequireCodeOwnerReviews:      branchProtection.RequiredPullRequestReviews.RequireCodeOwnerReviews,
			RequiredApprovingReviewCount: branchProtection.RequiredPullRequestReviews.RequiredApprovingReviewCount,
			DismissalRestrictions:        ghDismissalRestrictions,
		})
		if err != nil {
			return nil, err
		}
	}

	ea, err := core.JsonToDict(branchProtection.EnforceAdmins)
	if err != nil {
		return nil, err
	}
	r, err := core.JsonToDict(branchProtection.Restrictions)
	if err != nil {
		return nil, err
	}
	rlh, err := core.JsonToDict(branchProtection.RequireLinearHistory)
	if err != nil {
		return nil, err
	}
	afp, err := core.JsonToDict(branchProtection.AllowForcePushes)
	if err != nil {
		return nil, err
	}
	ad, err := core.JsonToDict(branchProtection.AllowDeletions)
	if err != nil {
		return nil, err
	}
	rcr, err := core.JsonToDict(branchProtection.RequiredConversationResolution)
	if err != nil {
		return nil, err
	}

	sc, _, err := gt.Client().Repositories.GetSignaturesProtectedBranch(context.TODO(), ownerName, repoName, branchName)
	if err != nil {
		log.Debug().Err(err).Msg("note: branch protection can only be accessed by admin users")
		return nil, err
	}

	mqlBranchProtection, err := g.MotorRuntime.CreateResource("github.branchprotection",
		"id", repoName+"/"+branchName,
		"requiredStatusChecks", rsc,
		"requiredPullRequestReviews", rprr,
		"enforceAdmins", ea,
		"restrictions", r,
		"requireLinearHistory", rlh,
		"allowForcePushes", afp,
		"allowDeletions", ad,
		"requiredConversationResolution", rcr,
		"requiredSignatures", core.ToBool(sc.Enabled),
	)
	if err != nil {
		return nil, err
	}
	return mqlBranchProtection, nil
}

func newMqlGithubCommit(runtime *resources.Runtime, rc *github.RepositoryCommit, owner string, repo string) (interface{}, error) {
	var githubAuthor interface{}
	var err error

	// if the github author is nil, we have to load the commit again
	if rc.Author == nil {
		gt, err := githubProvider(runtime.Motor.Provider)
		if err != nil {
			return nil, err
		}
		rc, _, err = gt.Client().Repositories.GetCommit(context.TODO(), owner, repo, rc.GetSHA(), nil)
		if err != nil {
			return nil, err
		}
	}

	if rc.Author != nil {
		githubAuthor, err = runtime.CreateResource("github.user", "id", core.ToInt64(rc.Author.ID), "login", core.ToString(rc.Author.Login))
		if err != nil {
			return nil, err
		}
	}
	var githubCommitter interface{}
	if rc.Committer != nil {
		githubCommitter, err = runtime.CreateResource("github.user", "id", core.ToInt64(rc.Committer.ID), "login", core.ToString(rc.Committer.Login))
		if err != nil {
			return nil, err
		}
	}

	sha := rc.GetSHA()

	stats, err := core.JsonToDict(rc.GetStats())
	if err != nil {
		return nil, err
	}

	mqlGitCommit, err := newMqlGitCommit(runtime, sha, rc.Commit)
	if err != nil {
		return nil, err
	}

	return runtime.CreateResource("github.commit",
		"url", rc.GetURL(),
		"sha", sha,
		"author", githubAuthor,
		"committer", githubCommitter,
		"owner", owner,
		"repository", repo,
		"commit", mqlGitCommit,
		"stats", stats,
	)
}

func (g *mqlGithubRepository) GetCommits() ([]interface{}, error) {
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	repoName, err := g.Name()
	if err != nil {
		return nil, err
	}
	ownerName, err := g.Owner()
	if err != nil {
		return nil, err
	}
	ownerLogin, err := ownerName.Login()
	if err != nil {
		return nil, err
	}

	listOpts := &github.CommitsListOptions{
		ListOptions: github.ListOptions{PerPage: paginationPerPage},
	}
	var allCommits []*github.RepositoryCommit
	for {
		commits, resp, err := gt.Client().Repositories.ListCommits(context.TODO(), ownerLogin, repoName, listOpts)
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
		mqlCommit, err := newMqlGithubCommit(g.MotorRuntime, rc, ownerLogin, repoName)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlCommit)
	}
	return res, nil
}

func (g *mqlGithubMergeRequest) GetReviews() ([]interface{}, error) {
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	repoName, err := g.RepoName()
	if err != nil {
		return nil, err
	}
	owner, err := g.Owner()
	if err != nil {
		return nil, err
	}
	ownerLogin, err := owner.Login()
	if err != nil {
		return nil, err
	}
	prID, err := g.Number()
	if err != nil {
		return nil, err
	}

	listOpts := &github.ListOptions{
		PerPage: paginationPerPage,
	}
	var allReviews []*github.PullRequestReview
	for {
		reviews, resp, err := gt.Client().PullRequests.ListReviews(context.TODO(), ownerLogin, repoName, int(prID), listOpts)
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
			user, err = g.MotorRuntime.CreateResource("github.user", "id", core.ToInt64(r.User.ID), "login", core.ToString(r.User.Login))
			if err != nil {
				return nil, err
			}
		}
		mqlReview, err := g.MotorRuntime.CreateResource("github.review",
			"url", core.ToString(r.HTMLURL),
			"state", core.ToString(r.State),
			"authorAssociation", core.ToString(r.AuthorAssociation),
			"user", user,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlReview)
	}

	return res, nil
}

func (g *mqlGithubMergeRequest) GetCommits() ([]interface{}, error) {
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	repoName, err := g.RepoName()
	if err != nil {
		return nil, err
	}
	owner, err := g.Owner()
	if err != nil {
		return nil, err
	}
	ownerLogin, err := owner.Login()
	if err != nil {
		return nil, err
	}
	prID, err := g.Number()
	if err != nil {
		return nil, err
	}

	listOpts := &github.ListOptions{
		PerPage: paginationPerPage,
	}
	var allCommits []*github.RepositoryCommit
	for {
		commits, resp, err := gt.Client().PullRequests.ListCommits(context.TODO(), ownerLogin, repoName, int(prID), listOpts)
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

		mqlCommit, err := newMqlGithubCommit(g.MotorRuntime, rc, ownerLogin, repoName)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlCommit)
	}
	return res, nil
}

func (g *mqlGithubRepository) GetContributors() ([]interface{}, error) {
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	repoName, err := g.Name()
	if err != nil {
		return nil, err
	}
	ownerName, err := g.Owner()
	if err != nil {
		return nil, err
	}
	ownerLogin, err := ownerName.Login()
	if err != nil {
		return nil, err
	}

	listOpts := &github.ListContributorsOptions{
		ListOptions: github.ListOptions{PerPage: paginationPerPage},
	}
	var allContributors []*github.Contributor
	for {
		contributors, resp, err := gt.Client().Repositories.ListContributors(context.TODO(), ownerLogin, repoName, listOpts)
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
		mqlUser, err := g.MotorRuntime.CreateResource("github.user",
			"id", core.ToInt64(allContributors[i].ID),
			"login", core.ToString(allContributors[i].Login),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlUser)
	}
	return res, nil
}

func (g *mqlGithubRepository) GetCollaborators() ([]interface{}, error) {
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	repoName, err := g.Name()
	if err != nil {
		return nil, err
	}
	ownerName, err := g.Owner()
	if err != nil {
		return nil, err
	}
	ownerLogin, err := ownerName.Login()
	if err != nil {
		return nil, err
	}

	listOpts := &github.ListCollaboratorsOptions{
		ListOptions: github.ListOptions{PerPage: paginationPerPage},
	}
	var allContributors []*github.User
	for {
		contributors, resp, err := gt.Client().Repositories.ListCollaborators(context.TODO(), ownerLogin, repoName, listOpts)
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
		mqlUser, err := g.MotorRuntime.CreateResource("github.user",
			"id", core.ToInt64(contributor.ID),
			"login", core.ToString(contributor.Login),
		)
		if err != nil {
			return nil, err
		}

		permissions := []string{}
		for k := range contributor.Permissions {
			permissions = append(permissions, k)
		}

		mqlContributor, err := g.MotorRuntime.CreateResource("github.collaborator",
			"id", core.ToInt64(contributor.ID),
			"user", mqlUser,
			"permissions", core.StrSliceToInterface(permissions),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlContributor)
	}
	return res, nil
}

func (g *mqlGithubRepository) GetReleases() ([]interface{}, error) {
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	repoName, err := g.Name()
	if err != nil {
		return nil, err
	}
	ownerName, err := g.Owner()
	if err != nil {
		return nil, err
	}
	ownerLogin, err := ownerName.Login()
	if err != nil {
		return nil, err
	}

	listOpts := &github.ListOptions{
		PerPage: paginationPerPage,
	}
	var allReleases []*github.RepositoryRelease
	for {
		releases, resp, err := gt.Client().Repositories.ListReleases(context.TODO(), ownerLogin, repoName, listOpts)
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
		mqlUser, err := g.MotorRuntime.CreateResource("github.release",
			"url", core.ToString(r.HTMLURL),
			"name", core.ToString(r.Name),
			"tagName", core.ToString(r.TagName),
			"preRelease", core.ToBool(r.Prerelease),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlUser)
	}

	return res, nil
}

func (g *mqlGithubWebhook) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return "github.webhook/" + strconv.FormatInt(id, 10), nil
}

func (g *mqlGithubRepository) GetWebhooks() ([]interface{}, error) {
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	repoName, err := g.Name()
	if err != nil {
		return nil, err
	}
	owner, err := g.Owner()
	if err != nil {
		return nil, err
	}
	ownerLogin, err := owner.Login()
	if err != nil {
		return nil, err
	}

	listOpts := &github.ListOptions{
		PerPage: paginationPerPage,
	}
	var allWebhooks []*github.Hook
	for {
		hooks, resp, err := gt.Client().Repositories.ListHooks(context.TODO(), ownerLogin, repoName, listOpts)
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
		config, err := core.JsonToDict(h.Config)
		if err != nil {
			return nil, err
		}

		mqlWebhook, err := g.MotorRuntime.CreateResource("github.webhook",
			"id", core.ToInt64(h.ID),
			"name", core.ToString(h.Name),
			"events", core.StrSliceToInterface(h.Events),
			"config", config,
			"url", core.ToString(h.URL),
			"name", core.ToString(h.Name),
			"active", core.ToBool(h.Active),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlWebhook)
	}

	return res, nil
}

func (g *mqlGithubRepository) GetWorkflows() ([]interface{}, error) {
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	repoName, err := g.Name()
	if err != nil {
		return nil, err
	}
	owner, err := g.Owner()
	if err != nil {
		return nil, err
	}
	ownerLogin, err := owner.Login()
	if err != nil {
		return nil, err
	}

	fullName, err := g.FullName()
	if err != nil {
		return nil, err
	}

	listOpts := &github.ListOptions{
		PerPage: paginationPerPage,
	}
	var allWorkflows []*github.Workflow
	for {
		workflows, resp, err := gt.Client().Actions.ListWorkflows(context.Background(), ownerLogin, repoName, listOpts)
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

		mqlWebhook, err := g.MotorRuntime.CreateResource("github.workflow",
			"id", core.ToInt64(w.ID),
			"name", core.ToString(w.Name),
			"path", core.ToString(w.Path),
			"state", core.ToString(w.State),
			"createdAt", githubTimestamp(w.CreatedAt),
			"updatedAt", githubTimestamp(w.UpdatedAt),
		)
		if err != nil {
			return nil, err
		}
		gw := mqlWebhook.(GithubWorkflow)
		gw.MqlResource().Cache.Store("_repositoryFullName", &resources.CacheEntry{
			Data: fullName,
		})
		res = append(res, gw)
	}
	return res, nil
}

func newMqlGithubFile(runtime *resources.Runtime, ownerName string, repoName string, content *github.RepositoryContent) (interface{}, error) {
	isBinary := false
	if core.ToString(content.Type) == "file" {
		file := strings.Split(core.ToString(content.Path), ".")
		if len(file) == 2 {
			isBinary = binaryFileTypes[file[1]]
		}
	}
	return runtime.CreateResource("github.file",
		"path", core.ToString(content.Path),
		"name", core.ToString(content.Name),
		"type", core.ToString(content.Type),
		"sha", core.ToString(content.SHA),
		"isBinary", isBinary,
		"ownerName", ownerName,
		"repoName", repoName,
	)
}

func (g *mqlGithubRepository) GetFiles() ([]interface{}, error) {
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	repoName, err := g.Name()
	if err != nil {
		return nil, err
	}

	owner, err := g.Owner()
	if err != nil {
		return nil, err
	}
	ownerLogin, err := owner.Login()
	if err != nil {
		return nil, err
	}
	_, dirContent, _, err := gt.Client().Repositories.GetContents(context.TODO(), ownerLogin, repoName, "", &github.RepositoryContentGetOptions{})
	if err != nil {
		log.Error().Err(err).Msg("unable to get contents list")
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}
	res := []interface{}{}
	for i := range dirContent {
		mqlFile, err := newMqlGithubFile(g.MotorRuntime, ownerLogin, repoName, dirContent[i])
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
	r, err := g.RepoName()
	if err != nil {
		return "", err
	}
	p, err := g.Path()
	if err != nil {
		return "", err
	}
	s, err := g.Sha()
	if err != nil {
		return "", err
	}
	return r + "/" + p + "/" + s, nil
}

func (g *mqlGithubFile) GetFiles() ([]interface{}, error) {
	fileType, err := g.Type()
	if err != nil {
		return nil, err
	}
	if fileType != "dir" {
		return nil, nil
	}
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	repoName, err := g.RepoName()
	if err != nil {
		return nil, err
	}
	ownerName, err := g.OwnerName()
	if err != nil {
		return nil, err
	}
	path, err := g.Path()
	if err != nil {
		return nil, err
	}
	_, dirContent, _, err := gt.Client().Repositories.GetContents(context.TODO(), ownerName, repoName, path, &github.RepositoryContentGetOptions{})
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
		if core.ToString(dirContent[i].Type) == "file" {
			file := strings.Split(core.ToString(dirContent[i].Path), ".")
			if len(file) == 2 {
				isBinary = binaryFileTypes[file[1]]
			}
		}
		mqlFile, err := g.MotorRuntime.CreateResource("github.file",
			"path", core.ToString(dirContent[i].Path),
			"name", core.ToString(dirContent[i].Name),
			"type", core.ToString(dirContent[i].Type),
			"sha", core.ToString(dirContent[i].SHA),
			"isBinary", isBinary,
			"ownerName", ownerName,
			"repoName", repoName,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlFile)

	}
	return res, nil
}

func (g *mqlGithubFile) GetContent() (string, error) {
	fileType, err := g.Type()
	if err != nil {
		return "", err
	}
	if fileType == "dir" {
		return "", nil
	}
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return "", err
	}
	repoName, err := g.RepoName()
	if err != nil {
		return "", err
	}
	ownerName, err := g.OwnerName()
	if err != nil {
		return "", err
	}
	path, err := g.Path()
	if err != nil {
		return "", err
	}
	fileContent, _, _, err := gt.Client().Repositories.GetContents(context.TODO(), ownerName, repoName, path, &github.RepositoryContentGetOptions{})
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

func (g *mqlGithubRepository) GetForks() ([]interface{}, error) {
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	repoName, err := g.Name()
	if err != nil {
		return nil, err
	}

	owner, err := g.Owner()
	if err != nil {
		return nil, err
	}
	ownerLogin, err := owner.Login()
	if err != nil {
		return nil, err
	}

	listOpts := &github.RepositoryListForksOptions{
		ListOptions: github.ListOptions{
			PerPage: paginationPerPage,
		},
	}
	var allForks []*github.Repository
	for {
		forks, resp, err := gt.Client().Repositories.ListForks(context.Background(), ownerLogin, repoName, listOpts)
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
		r, err := newMqlGithubRepository(g.MotorRuntime, repo)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func (g *mqlGithubRepository) GetStargazers() ([]interface{}, error) {
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	repoName, err := g.Name()
	if err != nil {
		return nil, err
	}

	owner, err := g.Owner()
	if err != nil {
		return nil, err
	}
	ownerLogin, err := owner.Login()
	if err != nil {
		return nil, err
	}

	listOpts := &github.ListOptions{
		PerPage: paginationPerPage,
	}
	var allStargazers []*github.Stargazer
	for {
		stargazers, resp, err := gt.Client().Activity.ListStargazers(context.Background(), ownerLogin, repoName, listOpts)
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
		r, err := g.MotorRuntime.CreateResource("github.user",
			"id", core.ToInt64(stargazer.User.ID),
			"login", core.ToString(stargazer.User.Login),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func (g *mqlGithubIssue) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}

	return "github.issue/" + strconv.FormatInt(id, 10), nil
}

func (g *mqlGithubRepository) GetOpenIssues() ([]interface{}, error) {
	return g.getIssues("open")
}

func (g *mqlGithubRepository) GetClosedIssues() ([]interface{}, error) {
	return g.getIssues("closed")
}

func (g *mqlGithubRepository) getIssues(state string) ([]interface{}, error) {
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	repoName, err := g.Name()
	if err != nil {
		return nil, err
	}

	owner, err := g.Owner()
	if err != nil {
		return nil, err
	}
	ownerLogin, err := owner.Login()
	if err != nil {
		return nil, err
	}

	listOpts := &github.IssueListByRepoOptions{
		State: state,
		ListOptions: github.ListOptions{
			PerPage: paginationPerPage,
		},
	}
	var allIssues []*github.Issue
	for {
		issues, resp, err := gt.Client().Issues.ListByRepo(context.Background(), ownerLogin, repoName, listOpts)
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
			r, err := g.MotorRuntime.CreateResource("github.user",
				"id", core.ToInt64(assignee.ID),
				"login", core.ToString(assignee.Login),
			)
			if err != nil {
				return nil, err
			}
			assignees = append(assignees, r)
		}

		var closedBy interface{}
		if issue.GetClosedBy() != nil {
			r, err := g.MotorRuntime.CreateResource("github.user",
				"id", core.ToInt64(issue.GetClosedBy().ID),
				"login", core.ToString(issue.GetClosedBy().Login),
			)
			if err != nil {
				return nil, err
			}
			closedBy = r
		}

		r, err := g.MotorRuntime.CreateResource("github.issue",
			"id", issue.GetID(),
			"number", int64(issue.GetNumber()),
			"title", issue.GetTitle(),
			"state", issue.GetState(),
			"body", issue.GetBody(),
			"url", issue.GetURL(),
			"createdAt", issue.CreatedAt,
			"updatedAt", issue.UpdatedAt,
			"closedAt", issue.ClosedAt,
			"assignees", assignees,
			"closedBy", closedBy,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}
