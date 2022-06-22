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
	"go.mondoo.io/mondoo/motor/transports"
	gh_transport "go.mondoo.io/mondoo/motor/transports/github"
)

func githubtransport(t transports.Transport) (*gh_transport.Transport, error) {
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

func (g *lumiGithubOrganization) init(args *lumi.Args) (*lumi.Args, GithubOrganization, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	gt, err := githubtransport(g.Runtime.Motor.Transport)
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
	gt, err := githubtransport(g.Runtime.Motor.Transport)
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

		r, err := g.Runtime.CreateResource("github.user",
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
	gt, err := githubtransport(g.Runtime.Motor.Transport)
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

		r, err := g.Runtime.CreateResource("github.user",
			"id", id,
			"login", toString(member.Login),
			"name", toString(member.Name),
			"email", toString(member.Email),
			"bio", toString(member.Bio),
			"createdAt", githubTimestamp(member.CreatedAt),
			"updatedAt", githubTimestamp(member.UpdatedAt),
			"suspendedAt", githubTimestamp(member.SuspendedAt),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (g *lumiGithubOrganization) GetRepositories() ([]interface{}, error) {
	gt, err := githubtransport(g.Runtime.Motor.Transport)
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

		r, err := g.Runtime.CreateResource("github.repository",
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
		)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (g *lumiGithubOrganization) GetInstallations() ([]interface{}, error) {
	gt, err := githubtransport(g.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	orgLogin, err := g.Login()
	if err != nil {
		return nil, err
	}

	apps, _, err := gt.Client().Organizations.ListInstallations(context.Background(), orgLogin, &github.ListOptions{})
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range apps.Installations {
		app := apps.Installations[i]

		var id int64
		if app.ID != nil {
			id = *app.ID
		}

		r, err := g.Runtime.CreateResource("github.installation",
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
	gt, err := githubtransport(g.Runtime.Motor.Transport)
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
	pulls, _, err := gt.Client().PullRequests.List(context.TODO(), orgName, repoName, &github.PullRequestListOptions{State: "open"})
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
		owner, err := g.Runtime.CreateResource("github.user",
			"id", toInt64(pr.User.ID),
			"login", toString(pr.User.Login),
		)
		if err != nil {
			return nil, err
		}

		assigneesRes := []interface{}{}
		for i := range pr.Assignees {
			assignee, err := g.Runtime.CreateResource("github.user",
				"id", toInt64(pr.Assignees[i].ID),
				"login", toString(pr.Assignees[i].Login),
			)
			if err != nil {
				return nil, err
			}
			assigneesRes = append(assigneesRes, assignee)
		}

		r, err := g.Runtime.CreateResource("github.mergeRequest",
			"id", toInt64(pr.ID),
			"state", toString(pr.State),
			"labels", labels,
			"createdAt", pr.CreatedAt,
			"title", toString(pr.Title),
			"owner", owner,
			"assignees", assigneesRes,
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

func (g *lumiGithubUser) init(args *lumi.Args) (*lumi.Args, GithubUser, error) {
	if len(*args) > 3 {
		return args, nil, nil
	}

	if (*args)["login"] == nil {
		return nil, nil, errors.New("login required to fetch github user")
	}
	userLogin := (*args)["login"].(string)

	gt, err := githubtransport(g.Runtime.Motor.Transport)
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
	gt, err := githubtransport(g.Runtime.Motor.Transport)
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
	branches, _, err := gt.Client().Repositories.ListBranches(context.TODO(), orgName, repoName, &github.BranchListOptions{})
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
		lumiCommit, err := g.Runtime.CreateResource("github.commit",
			"sha", toString(branch.Commit.SHA),
		)
		if err != nil {
			return nil, err
		}
		lumiBranch, err := g.Runtime.CreateResource("github.branch",
			"name", toString(branch.Name),
			"protected", toBool(branch.Protected),
			"commit", lumiCommit,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiBranch)
	}
	return res, nil
}

func (g *lumiGithubBranch) id() (string, error) {
	return g.Name()
}

func (g *lumiGithubCommit) id() (string, error) {
	return g.Sha()
}

func (g *lumiGithubRepository) GetCommits() ([]interface{}, error) {
	gt, err := githubtransport(g.Runtime.Motor.Transport)
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
	commits, _, err := gt.Client().Repositories.ListCommits(context.TODO(), orgName, repoName, &github.CommitsListOptions{})
	if err != nil {
		log.Error().Err(err).Msg("unable to get commits list")
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}
	res := []interface{}{}
	for i := range commits {
		lumiCommit, err := g.Runtime.CreateResource("github.commit",
			"sha", toString(commits[i].Commit.SHA),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiCommit)
	}
	return res, nil
}

func (g *lumiGithubRepository) GetContributors() ([]interface{}, error) {
	gt, err := githubtransport(g.Runtime.Motor.Transport)
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
	contributors, _, err := gt.Client().Repositories.ListContributors(context.TODO(), orgName, repoName, &github.ListContributorsOptions{})
	if err != nil {
		log.Error().Err(err).Msg("unable to get contributors list")
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}
	res := []interface{}{}
	for i := range contributors {
		lumiUser, err := g.Runtime.CreateResource("github.user",
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
func (g *lumiGithubRepository) GetFiles() ([]interface{}, error) {
	gt, err := githubtransport(g.Runtime.Motor.Transport)
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
	_, dirContent, _, err := gt.Client().Repositories.GetContents(context.TODO(), orgName, repoName, "", &github.RepositoryContentGetOptions{})
	if err != nil {
		log.Error().Err(err).Msg("unable to get contents list")
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}
	res := []interface{}{}
	for i := range dirContent {
		lumiFile, err := g.Runtime.CreateResource("github.file",
			"path", toString(dirContent[i].Path),
			"type", toString(dirContent[i].Type),
			"sha", toString(dirContent[i].SHA),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiFile)

	}
	return res, nil
}

func (g *lumiGithubFile) id() (string, error) {
	return g.Sha()
}
