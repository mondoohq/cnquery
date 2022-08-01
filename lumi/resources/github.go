package resources

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/go-github/v43/github"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/motor/providers"
	gh_transport "go.mondoo.io/mondoo/motor/providers/github"
)

func githubtransport(t providers.Transport) (*gh_transport.Transport, error) {
	gt, ok := t.(*gh_transport.Transport)
	if !ok {
		return nil, errors.New("github resource is not supported on this transport")
	}
	return gt, nil
}

func githubTimestamp(ts *github.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	return &ts.Time
}

func (g *lumiGithubOrganization) id() (string, error) {
	return "github.organization", nil
}

func (g *lumiGithub) id() (string, error) {
	return "github", nil
}

func (g *lumiGithubOrganization) init(args *lumi.Args) (*lumi.Args, GithubOrganization, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	gt, err := githubtransport(g.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, nil, err
	}

	org, err := gt.Organization()
	if err != nil {
		return nil, nil, err
	}

	(*args)["id"] = toInt64(org.ID)
	(*args)["name"] = toString(org.Name)
	(*args)["login"] = toString(org.Login)
	(*args)["node_id"] = toString(org.NodeID)
	(*args)["company"] = toString(org.Company)
	(*args)["blog"] = toString(org.Blog)
	(*args)["location"] = toString(org.Location)
	(*args)["email"] = toString(org.Email)
	(*args)["twitter_username"] = toString(org.TwitterUsername)
	(*args)["description"] = toString(org.Description)
	(*args)["created_at"] = org.CreatedAt
	(*args)["updated_at"] = org.UpdatedAt
	(*args)["total_private_repos"] = toInt(org.TotalPrivateRepos)
	(*args)["owned_private_repos"] = toInt(org.OwnedPrivateRepos)
	(*args)["private_gists"] = toInt(org.PrivateGists)
	(*args)["disk_usage"] = toInt(org.DiskUsage)
	(*args)["collaborators"] = toInt(org.Collaborators)
	(*args)["billing_email"] = toString(org.BillingEmail)

	plan, _ := jsonToDict(org.Plan)
	(*args)["plan"] = plan

	(*args)["two_factor_requirement_enabled"] = toBool(org.TwoFactorRequirementEnabled)
	(*args)["is_verified"] = toBool(org.IsVerified)

	(*args)["default_repository_permission"] = toString(org.DefaultRepoPermission)
	(*args)["members_can_create_repositories"] = toBool(org.MembersCanCreateRepos)
	(*args)["members_can_create_public_repositories"] = toBool(org.MembersCanCreatePublicRepos)
	(*args)["members_can_create_private_repositories"] = toBool(org.MembersCanCreatePrivateRepos)
	(*args)["members_can_create_internal_repositories"] = toBool(org.MembersCanCreateInternalRepos)
	(*args)["members_can_create_pages"] = toBool(org.MembersCanCreatePages)
	(*args)["members_can_create_public_pages"] = toBool(org.MembersCanCreatePublicPages)
	(*args)["members_can_create_private_pages"] = toBool(org.MembersCanCreatePrivateRepos)

	return args, nil, nil
}

func (g *lumiGithubOrganization) GetMembers() ([]interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	orgLogin, err := g.Login()
	if err != nil {
		return nil, err
	}
	members, _, err := gt.Client().Organizations.ListMembers(context.Background(), orgLogin, nil)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range members {
		member := members[i]

		r, err := g.MotorRuntime.CreateResource("github.user",
			"id", toInt64(member.ID),
			"login", toString(member.Login),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (g *lumiGithubOrganization) GetOwners() ([]interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	orgLogin, err := g.Login()
	if err != nil {
		return nil, err
	}

	members, _, err := gt.Client().Organizations.ListMembers(context.Background(), orgLogin, &github.ListMembersOptions{
		Role: "admin",
	})
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range members {
		member := members[i]

		var id int64
		if member.ID != nil {
			id = *member.ID
		}

		r, err := g.MotorRuntime.CreateResource("github.user",
			"id", id,
			"login", toString(member.Login),
			"name", toString(member.Name),
			"email", toString(member.Email),
			"bio", toString(member.Bio),
			"createdAt", githubTimestamp(member.CreatedAt),
			"updatedAt", githubTimestamp(member.UpdatedAt),
			"suspendedAt", githubTimestamp(member.SuspendedAt),
			"company", toString(member.Company),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (g *lumiGithub) GetRepositories() ([]interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	user, err := gt.User()
	if err != nil {
		return nil, err
	}

	repos, _, err := gt.Client().Repositories.List(context.Background(), user.GetLogin(), &github.RepositoryListOptions{})
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range repos {
		repo := repos[i]

		var id int64
		if repo.ID != nil {
			id = *repo.ID
		}

		owner, err := g.MotorRuntime.CreateResource("github.user",
			"id", repo.GetOwner().GetID(),
			"login", repo.GetOwner().GetLogin(),
		)
		if err != nil {
			return nil, err
		}

		r, err := g.MotorRuntime.CreateResource("github.repository",
			"id", id,
			"name", toString(repo.Name),
			"fullName", toString(repo.FullName),
			"description", toString(repo.Description),
			"homepage", toString(repo.Homepage),
			"createdAt", githubTimestamp(repo.CreatedAt),
			"updatedAt", githubTimestamp(repo.UpdatedAt),
			"archived", toBool(repo.Archived),
			"disabled", toBool(repo.Disabled),
			"private", toBool(repo.Private),
			"visibility", toString(repo.Visibility),
			"allowAutoMerge", toBool(repo.AllowAutoMerge),
			"allowForking", toBool(repo.AllowForking),
			"allowMergeCommit", toBool(repo.AllowMergeCommit),
			"allowRebaseMerge", toBool(repo.AllowRebaseMerge),
			"allowSquashMerge", toBool(repo.AllowSquashMerge),
			"hasIssues", toBool(repo.HasIssues),
			"organizationName", "",
			"defaultBranchName", toString(repo.DefaultBranch),
			"owner", owner,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (g *lumiGithub) GetUser() (interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	user, err := gt.User()
	if err != nil {
		return nil, err
	}
	var x interface{}
	x = user
	return x, nil
}

func (g *lumiGithubOrganization) GetRepositories() ([]interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	orgLogin, err := g.Login()
	if err != nil {
		return nil, err
	}

	repos, _, err := gt.Client().Repositories.ListByOrg(context.Background(), orgLogin, &github.RepositoryListByOrgOptions{Type: "all"})
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range repos {
		repo := repos[i]

		var id int64
		if repo.ID != nil {
			id = *repo.ID
		}

		owner, err := g.MotorRuntime.CreateResource("github.user",
			"id", repo.GetOwner().GetID(),
			"login", repo.GetOwner().GetLogin(),
		)
		if err != nil {
			return nil, err
		}

		r, err := g.MotorRuntime.CreateResource("github.repository",
			"id", id,
			"name", toString(repo.Name),
			"fullName", toString(repo.FullName),
			"description", toString(repo.Description),
			"homepage", toString(repo.Homepage),
			"createdAt", githubTimestamp(repo.CreatedAt),
			"updatedAt", githubTimestamp(repo.UpdatedAt),
			"archived", toBool(repo.Archived),
			"disabled", toBool(repo.Disabled),
			"private", toBool(repo.Private),
			"visibility", toString(repo.Visibility),
			"allowAutoMerge", toBool(repo.AllowAutoMerge),
			"allowForking", toBool(repo.AllowForking),
			"allowMergeCommit", toBool(repo.AllowMergeCommit),
			"allowRebaseMerge", toBool(repo.AllowRebaseMerge),
			"allowSquashMerge", toBool(repo.AllowSquashMerge),
			"hasIssues", toBool(repo.HasIssues),
			"organizationName", orgLogin,
			"defaultBranchName", toString(repo.DefaultBranch),
			"owner", owner,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (g *lumiGithubOrganization) GetInstallations() ([]interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	orgLogin, err := g.Login()
	if err != nil {
		return nil, err
	}

	apps, _, err := gt.Client().Organizations.ListInstallations(context.Background(), orgLogin, &github.ListOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}

	res := []interface{}{}
	for i := range apps.Installations {
		app := apps.Installations[i]

		var id int64
		if app.ID != nil {
			id = *app.ID
		}

		r, err := g.MotorRuntime.CreateResource("github.installation",
			"id", id,
			"appId", toInt64(app.AppID),
			"appSlug", toString(app.AppSlug),
			"createdAt", githubTimestamp(app.CreatedAt),
			"updatedAt", githubTimestamp(app.UpdatedAt),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (g *lumiGithubUser) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(id, 10), nil
}

func (g *lumiGithubRepository) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(id, 10), nil
}

func (g *lumiGithubInstallation) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(id, 10), nil
}

func (g *lumiGithubRepository) GetOpenMergeRequests() ([]interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	repoName, err := g.Name()
	if err != nil {
		return nil, err
	}
	orgName, err := g.OrganizationName()
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
	pulls, _, err := gt.Client().PullRequests.List(context.TODO(), ownerLogin, repoName, &github.PullRequestListOptions{State: "open"})
	if err != nil {
		log.Error().Err(err).Msg("unable to pull merge requests list")
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}

	res := []interface{}{}
	for i := range pulls {
		pr := pulls[i]

		labels, err := jsonToDictSlice(pr.Labels)
		if err != nil {
			return nil, err
		}
		owner, err := g.MotorRuntime.CreateResource("github.user",
			"id", toInt64(pr.User.ID),
			"login", toString(pr.User.Login),
		)
		if err != nil {
			return nil, err
		}

		assigneesRes := []interface{}{}
		for i := range pr.Assignees {
			assignee, err := g.MotorRuntime.CreateResource("github.user",
				"id", toInt64(pr.Assignees[i].ID),
				"login", toString(pr.Assignees[i].Login),
			)
			if err != nil {
				return nil, err
			}
			assigneesRes = append(assigneesRes, assignee)
		}

		r, err := g.MotorRuntime.CreateResource("github.mergeRequest",
			"id", toInt64(pr.ID),
			"number", toInt(pr.Number),
			"state", toString(pr.State),
			"labels", labels,
			"createdAt", pr.CreatedAt,
			"title", toString(pr.Title),
			"owner", owner,
			"assignees", assigneesRes,
			"organizationName", orgName,
			"repoName", repoName,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (g *lumiGithubMergeRequest) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(id, 10), nil
}

func (g *lumiGithubRepository) init(args *lumi.Args) (*lumi.Args, GithubRepository, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}
	gt, err := githubtransport(g.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, nil, err
	}

	user, err := gt.User()
	if err != nil {
		return nil, nil, err
	}

	userLogin := user.GetLogin()

	if x, ok := (*args)["name"]; ok {
		path, ok := x.(string)
		if !ok {
			return nil, nil, errors.New("Wrong type for 'path' in github.repository initialization, it must be a string")
		}
		paths := strings.Split(path, "/")
		if len(paths) == 1 {
			path = paths[0]
		} else if len(paths) == 2 {
			userLogin = paths[0]
			path = paths[1]
		} else {
			return nil, nil, errors.New("unexpected value for path. should be owner/reponame or reponame")
		}
		repo, _, err := gt.Client().Repositories.Get(context.Background(), userLogin, path)
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

		(*args)["id"] = toInt64(repo.ID)
		(*args)["name"] = toString(repo.Name)
		(*args)["fullName"] = toString(repo.FullName)
		(*args)["description"] = toString(repo.Description)
		(*args)["homepage"] = toString(repo.Homepage)
		(*args)["createdAt"] = githubTimestamp(repo.CreatedAt)
		(*args)["updatedAt"] = githubTimestamp(repo.UpdatedAt)
		(*args)["archived"] = toBool(repo.Archived)
		(*args)["disabled"] = toBool(repo.Disabled)
		(*args)["private"] = toBool(repo.Private)
		(*args)["visibility"] = toString(repo.Visibility)
		(*args)["allowAutoMerge"] = toBool(repo.AllowAutoMerge)
		(*args)["allowForking"] = toBool(repo.AllowForking)
		(*args)["allowMergeCommit"] = toBool(repo.AllowMergeCommit)
		(*args)["allowRebaseMerge"] = toBool(repo.AllowRebaseMerge)
		(*args)["allowSquashMerge"] = toBool(repo.AllowSquashMerge)
		(*args)["hasIssues"] = toBool(repo.HasIssues)
		(*args)["organizationName"] = ""
		(*args)["defaultBranchName"] = toString(repo.DefaultBranch)
		(*args)["owner"] = owner
	}

	return args, nil, nil
}

func (g *lumiGithubUser) init(args *lumi.Args) (*lumi.Args, GithubUser, error) {
	if len(*args) > 3 {
		return args, nil, nil
	}

	if (*args)["login"] == nil {
		return nil, nil, errors.New("login required to fetch github user")
	}
	userLogin := (*args)["login"].(string)

	gt, err := githubtransport(g.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, nil, err
	}

	user, _, err := gt.Client().Users.Get(context.Background(), userLogin)
	if err != nil {
		return nil, nil, err
	}

	(*args)["id"] = (*args)["id"]
	(*args)["login"] = toString(user.Login)
	(*args)["name"] = toString(user.Name)
	(*args)["email"] = toString(user.Email)
	(*args)["bio"] = toString(user.Bio)
	createdAt := &time.Time{}
	if user.CreatedAt != nil {
		createdAt = &user.CreatedAt.Time
	}
	(*args)["createdAt"] = createdAt
	updatedAt := &time.Time{}
	if user.UpdatedAt != nil {
		updatedAt = &user.UpdatedAt.Time
	}
	(*args)["updatedAt"] = updatedAt
	suspendedAt := &time.Time{}
	if user.SuspendedAt != nil {
		suspendedAt = &user.SuspendedAt.Time
	}
	(*args)["suspendedAt"] = suspendedAt
	(*args)["company"] = toString(user.Company)
	return args, nil, nil
}

func (g *lumiGithubRepository) GetBranches() ([]interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	repoName, err := g.Name()
	if err != nil {
		return nil, err
	}
	orgName, err := g.OrganizationName()
	if err != nil {
		return nil, err
	}
	ownerName, err := g.Owner()
	if err != nil {
		return nil, err
	}
	repoDefaultBranchName, err := g.DefaultBranchName()
	if err != nil {
		return nil, err
	}
	ownerLogin, err := ownerName.Login()
	if err != nil {
		return nil, err
	}
	branches, _, err := gt.Client().Repositories.ListBranches(context.TODO(), ownerLogin, repoName, &github.BranchListOptions{})
	if err != nil {
		log.Error().Err(err).Msg("unable to pull branches list")
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}
	res := []interface{}{}
	for i := range branches {
		branch := branches[i]
		rc := branch.Commit
		var author interface{}
		if rc.Author != nil {
			author, err = g.MotorRuntime.CreateResource("github.user", "id", toInt64(rc.Author.ID), "login", toString(rc.Author.Login))
			if err != nil {
				return nil, err
			}
		}
		var committer interface{}
		if rc.Committer != nil {
			committer, err = g.MotorRuntime.CreateResource("github.user", "id", toInt64(rc.Committer.ID), "login", toString(rc.Author.Login))
			if err != nil {
				return nil, err
			}
		}
		var lumiCommit interface{}
		c := rc.Commit
		if c != nil {
			signatureDict, err := jsonToDict(c.GetVerification())
			if err != nil {
				return nil, err
			}
			lumiCommit, err = g.MotorRuntime.CreateResource("github.commit",
				"url", c.GetURL(),
				"author", author,
				"committer", committer,
				"message", c.GetMessage(),
				"signatureVerification", signatureDict,
				"organizationName", orgName,
				"repoName", repoName,
			)
			if err != nil {
				return nil, err
			}
		}

		defaultBranch := false
		if repoDefaultBranchName == toString(branch.Name) {
			defaultBranch = true
		}
		lumiBranch, err := g.MotorRuntime.CreateResource("github.branch",
			"name", toString(branch.Name),
			"protected", toBool(branch.Protected),
			"headCommit", lumiCommit,
			"organizationName", orgName,
			"repoName", repoName,
			"owner", ownerName,
			"isDefault", defaultBranch,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiBranch)
	}
	return res, nil
}

func (g *lumiGithubBranch) GetProtectionRules() (interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Transport)
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
	ownerName, err := g.Name()
	if err != nil {
		return nil, err
	}
	branchProtection, _, err := gt.Client().Repositories.GetBranchProtection(context.TODO(), ownerName, repoName, branchName)
	if err != nil {
		log.Debug().Err(err).Msg("note: branch protection can only be accessed by admin users")
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}
	rsc, err := jsonToDict(branchProtection.RequiredStatusChecks)
	if err != nil {
		return nil, err
	}
	rprr, err := jsonToDict(branchProtection.RequiredPullRequestReviews)
	if err != nil {
		return nil, err
	}
	ea, err := jsonToDict(branchProtection.EnforceAdmins)
	if err != nil {
		return nil, err
	}
	r, err := jsonToDict(branchProtection.Restrictions)
	if err != nil {
		return nil, err
	}
	rlh, err := jsonToDict(branchProtection.RequireLinearHistory)
	if err != nil {
		return nil, err
	}
	afp, err := jsonToDict(branchProtection.AllowForcePushes)
	if err != nil {
		return nil, err
	}
	ad, err := jsonToDict(branchProtection.AllowDeletions)
	if err != nil {
		return nil, err
	}
	rcr, err := jsonToDict(branchProtection.RequiredConversationResolution)
	if err != nil {
		return nil, err
	}
	lumiBranchProtection, err := g.MotorRuntime.CreateResource("github.branchprotection",
		"requiredStatusChecks", rsc,
		"requiredPullRequestReviews", rprr,
		"enforceAdmins", ea,
		"restrictions", r,
		"requireLinearHistory", rlh,
		"allowForcePushes", afp,
		"allowDeletions", ad,
		"requiredConversationResolution", rcr,
		"id", repoName+"/"+branchName,
	)
	if err != nil {
		return nil, err
	}
	return lumiBranchProtection, nil
}

func (g *lumiGithubBranchprotection) id() (string, error) {
	return g.Id()
}

func (g *lumiGithubBranch) id() (string, error) {
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

func (g *lumiGithubCommit) id() (string, error) {
	// the url is unique, e.g. "https://api.github.com/repos/vjeffrey/victoria-website/git/commits/7730d2707fdb6422f335fddc944ab169d45f3aa5"
	return g.Url()
}

func (g *lumiGithubRepository) GetCommits() ([]interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	repoName, err := g.Name()
	if err != nil {
		return nil, err
	}
	orgName, err := g.OrganizationName()
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
	commits, _, err := gt.Client().Repositories.ListCommits(context.TODO(), ownerLogin, repoName, &github.CommitsListOptions{})
	if err != nil {
		log.Error().Err(err).Msg("unable to get commits list")
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}
	res := []interface{}{}
	for i := range commits {
		rc := commits[i]
		var author interface{}
		if rc.Author != nil {
			author, err = g.MotorRuntime.CreateResource("github.user", "id", toInt64(rc.Author.ID), "login", toString(rc.Author.Login))
			if err != nil {
				return nil, err
			}
		}
		var committer interface{}
		if rc.Committer != nil {
			committer, err = g.MotorRuntime.CreateResource("github.user", "id", toInt64(rc.Committer.ID), "login", toString(rc.Committer.Login))
			if err != nil {
				return nil, err
			}
		}
		c := rc.Commit
		signatureDict, err := jsonToDict(c.GetVerification())
		if err != nil {
			return nil, err
		}
		lumiCommit, err := g.MotorRuntime.CreateResource("github.commit",
			"url", toString(c.URL),
			"author", author,
			"committer", committer,
			"message", toString(c.Message),
			"signatureVerification", signatureDict,
			"organizationName", orgName,
			"repoName", repoName,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiCommit)
	}
	return res, nil
}

func (g *lumiGithubMergeRequest) GetReviews() ([]interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	repoName, err := g.RepoName()
	if err != nil {
		return nil, err
	}
	orgName, err := g.OrganizationName()
	if err != nil {
		return nil, err
	}
	prID, err := g.Number()
	if err != nil {
		return nil, err
	}
	reviews, _, err := gt.Client().PullRequests.ListReviews(context.TODO(), orgName, repoName, int(prID), &github.ListOptions{})
	if err != nil {
		log.Error().Err(err).Msg("unable to get reviews list")
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}

	res := []interface{}{}
	for i := range reviews {
		r := reviews[i]
		var user interface{}
		if r.User != nil {
			user, err = g.MotorRuntime.CreateResource("github.user", "id", toInt64(r.User.ID), "login", toString(r.User.Login))
			if err != nil {
				return nil, err
			}
		}
		lumiReview, err := g.MotorRuntime.CreateResource("github.review",
			"url", toString(r.HTMLURL),
			"state", toString(r.State),
			"authorAssociation", toString(r.AuthorAssociation),
			"user", user,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiReview)
	}

	return res, nil
}

func (g *lumiGithubMergeRequest) GetCommits() ([]interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	repoName, err := g.RepoName()
	if err != nil {
		return nil, err
	}
	orgName, err := g.OrganizationName()
	if err != nil {
		return nil, err
	}
	prID, err := g.Number()
	if err != nil {
		return nil, err
	}
	commits, _, err := gt.Client().PullRequests.ListCommits(context.TODO(), orgName, repoName, int(prID), &github.ListOptions{})
	if err != nil {
		log.Error().Err(err).Msg("unable to get commits list")
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}
	res := []interface{}{}
	for i := range commits {
		rc := commits[i]
		var author interface{}
		if rc.Author != nil {
			author, err = g.MotorRuntime.CreateResource("github.user", "id", toInt64(rc.Author.ID), "login", toString(rc.Author.Login))
			if err != nil {
				return nil, err
			}
		}
		var committer interface{}
		if rc.Committer != nil {
			committer, err = g.MotorRuntime.CreateResource("github.user", "id", toInt64(rc.Committer.ID), "login", toString(rc.Committer.Login))
			if err != nil {
				return nil, err
			}
		}
		c := rc.Commit
		signatureDict, err := jsonToDict(c.GetVerification())
		if err != nil {
			return nil, err
		}

		lumiCommit, err := g.MotorRuntime.CreateResource("github.commit",
			"url", toString(c.URL),
			"author", author,
			"committer", committer,
			"message", toString(c.Message),
			"signatureVerification", signatureDict,
			"organizationName", orgName,
			"repoName", repoName,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiCommit)
	}
	return res, nil
}

func (g *lumiGithubRepository) GetContributors() ([]interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Transport)
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
	contributors, _, err := gt.Client().Repositories.ListContributors(context.TODO(), ownerLogin, repoName, &github.ListContributorsOptions{})
	if err != nil {
		log.Error().Err(err).Msg("unable to get contributors list")
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}
	res := []interface{}{}
	for i := range contributors {
		lumiUser, err := g.MotorRuntime.CreateResource("github.user",
			"id", toInt64(contributors[i].ID),
			"login", toString(contributors[i].Login),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiUser)
	}
	return res, nil
}

func (g *lumiGithubRepository) GetReleases() ([]interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Transport)
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
	releases, _, err := gt.Client().Repositories.ListReleases(context.TODO(), ownerLogin, repoName, &github.ListOptions{})
	if err != nil {
		log.Error().Err(err).Msg("unable to get releases list")
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}
	res := []interface{}{}
	for i := range releases {
		r := releases[i]
		lumiUser, err := g.MotorRuntime.CreateResource("github.release",
			"url", toString(r.HTMLURL),
			"name", toString(r.Name),
			"tagName", toString(r.TagName),
			"preRelease", toBool(r.Prerelease),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiUser)
	}

	return res, nil
}

func (g *lumiGithubReview) id() (string, error) {
	return g.Url()
}

func (g *lumiGithubRelease) id() (string, error) {
	return g.Url()
}

func (g *lumiGithubRepository) GetFiles() ([]interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	repoName, err := g.Name()
	if err != nil {
		return nil, err
	}
	orgName, err := g.OrganizationName()
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
		isBinary := false
		if toString(dirContent[i].Type) == "file" {
			file := strings.Split(toString(dirContent[i].Path), ".")
			if len(file) == 2 {
				isBinary = binaryFileTypes[file[1]]
			}
		}
		lumiFile, err := g.MotorRuntime.CreateResource("github.file",
			"path", toString(dirContent[i].Path),
			"type", toString(dirContent[i].Type),
			"sha", toString(dirContent[i].SHA),
			"isBinary", isBinary,
			"organizationName", orgName,
			"repoName", repoName,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiFile)

	}
	return res, nil
}

func (g *lumiGithubFile) GetFiles() ([]interface{}, error) {
	fileType, err := g.Type()
	if err != nil {
		return nil, err
	}
	if fileType != "dir" {
		return nil, nil
	}
	gt, err := githubtransport(g.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	repoName, err := g.RepoName()
	if err != nil {
		return nil, err
	}
	orgName, err := g.OrganizationName()
	if err != nil {
		return nil, err
	}
	path, err := g.Path()
	if err != nil {
		return nil, err
	}
	_, dirContent, _, err := gt.Client().Repositories.GetContents(context.TODO(), orgName, repoName, path, &github.RepositoryContentGetOptions{})
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
		if toString(dirContent[i].Type) == "file" {
			file := strings.Split(toString(dirContent[i].Path), ".")
			if len(file) == 2 {
				isBinary = binaryFileTypes[file[1]]
			}
		}
		lumiFile, err := g.MotorRuntime.CreateResource("github.file",
			"path", toString(dirContent[i].Path),
			"type", toString(dirContent[i].Type),
			"sha", toString(dirContent[i].SHA),
			"isBinary", isBinary,
			"organizationName", orgName,
			"repoName", repoName,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiFile)

	}
	return res, nil
}

func (g *lumiGithubFile) GetContent() (string, error) {
	fileType, err := g.Type()
	if err != nil {
		return "", err
	}
	if fileType == "dir" {
		return "", nil
	}
	gt, err := githubtransport(g.MotorRuntime.Motor.Transport)
	if err != nil {
		return "", err
	}
	repoName, err := g.RepoName()
	if err != nil {
		return "", err
	}
	orgName, err := g.OrganizationName()
	if err != nil {
		return "", err
	}
	path, err := g.Path()
	if err != nil {
		return "", err
	}
	fileContent, _, _, err := gt.Client().Repositories.GetContents(context.TODO(), orgName, repoName, path, &github.RepositoryContentGetOptions{})
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

func (g *lumiGithubFile) id() (string, error) {
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
