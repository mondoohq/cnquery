package services

import (
	"context"
	"strconv"
	"strings"
	"time"

	"sigs.k8s.io/yaml"

	"github.com/cockroachdb/errors"
	"github.com/google/go-github/v45/github"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/providers"
	gh_transport "go.mondoo.io/mondoo/motor/providers/github"
	"go.mondoo.io/mondoo/resources"
	"go.mondoo.io/mondoo/resources/packs/core"
)

func githubtransport(t providers.Transport) (*gh_transport.Provider, error) {
	gt, ok := t.(*gh_transport.Provider)
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

func (g *mqlGithub) id() (string, error) {
	return "github", nil
}

func (g *mqlGithub) GetUser() (interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
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

func (g *mqlGithub) GetRepositories() ([]interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
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
			"name", core.ToString(repo.Name),
			"fullName", core.ToString(repo.FullName),
			"description", core.ToString(repo.Description),
			"homepage", core.ToString(repo.Homepage),
			"createdAt", githubTimestamp(repo.CreatedAt),
			"updatedAt", githubTimestamp(repo.UpdatedAt),
			"archived", core.ToBool(repo.Archived),
			"disabled", core.ToBool(repo.Disabled),
			"private", core.ToBool(repo.Private),
			"visibility", core.ToString(repo.Visibility),
			"allowAutoMerge", core.ToBool(repo.AllowAutoMerge),
			"allowForking", core.ToBool(repo.AllowForking),
			"allowMergeCommit", core.ToBool(repo.AllowMergeCommit),
			"allowRebaseMerge", core.ToBool(repo.AllowRebaseMerge),
			"allowSquashMerge", core.ToBool(repo.AllowSquashMerge),
			"hasIssues", core.ToBool(repo.HasIssues),
			"organizationName", "",
			"defaultBranchName", core.ToString(repo.DefaultBranch),
			"owner", owner,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (g *mqlGithubOrganization) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return "github.organization/" + strconv.FormatInt(id, 10), nil
}

func (g *mqlGithubOrganization) init(args *resources.Args) (*resources.Args, GithubOrganization, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	org, err := gt.Organization()
	if err != nil {
		return nil, nil, err
	}

	(*args)["id"] = core.ToInt64(org.ID)
	(*args)["name"] = core.ToString(org.Name)
	(*args)["login"] = core.ToString(org.Login)
	(*args)["nodeId"] = core.ToString(org.NodeID)
	(*args)["company"] = core.ToString(org.Company)
	(*args)["blog"] = core.ToString(org.Blog)
	(*args)["location"] = core.ToString(org.Location)
	(*args)["email"] = core.ToString(org.Email)
	(*args)["twitterUsername"] = core.ToString(org.TwitterUsername)
	(*args)["description"] = core.ToString(org.Description)
	(*args)["createdAt"] = org.CreatedAt
	(*args)["updatedAt"] = org.UpdatedAt
	(*args)["totalPrivateRepos"] = core.ToInt(org.TotalPrivateRepos)
	(*args)["ownedPrivateRepos"] = core.ToInt(org.OwnedPrivateRepos)
	(*args)["privateGists"] = core.ToInt(org.PrivateGists)
	(*args)["diskUsage"] = core.ToInt(org.DiskUsage)
	(*args)["collaborators"] = core.ToInt(org.Collaborators)
	(*args)["billingEmail"] = core.ToString(org.BillingEmail)

	plan, _ := core.JsonToDict(org.Plan)
	(*args)["plan"] = plan

	(*args)["twoFactorRequirementEnabled"] = core.ToBool(org.TwoFactorRequirementEnabled)
	(*args)["isVerified"] = core.ToBool(org.IsVerified)

	(*args)["defaultRepositoryPermission"] = core.ToString(org.DefaultRepoPermission)
	(*args)["membersCanCreateRepositories"] = core.ToBool(org.MembersCanCreateRepos)
	(*args)["membersCanCreatePublicRepositories"] = core.ToBool(org.MembersCanCreatePublicRepos)
	(*args)["membersCanCreatePrivateRepositories"] = core.ToBool(org.MembersCanCreatePrivateRepos)
	(*args)["membersCanCreateInternalRepositories"] = core.ToBool(org.MembersCanCreateInternalRepos)
	(*args)["membersCanCreatePages"] = core.ToBool(org.MembersCanCreatePages)
	(*args)["membersCanCreatePublicPages"] = core.ToBool(org.MembersCanCreatePublicPages)
	(*args)["membersCanCreatePrivatePages"] = core.ToBool(org.MembersCanCreatePrivateRepos)

	return args, nil, nil
}

func (g *mqlGithubOrganization) GetMembers() ([]interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
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
			"id", core.ToInt64(member.ID),
			"login", core.ToString(member.Login),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (g *mqlGithubOrganization) GetOwners() ([]interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
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
			"login", core.ToString(member.Login),
			"name", core.ToString(member.Name),
			"email", core.ToString(member.Email),
			"bio", core.ToString(member.Bio),
			"createdAt", githubTimestamp(member.CreatedAt),
			"updatedAt", githubTimestamp(member.UpdatedAt),
			"suspendedAt", githubTimestamp(member.SuspendedAt),
			"company", core.ToString(member.Company),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (g *mqlGithubOrganization) GetTeams() ([]interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	orgLogin, err := g.Login()
	if err != nil {
		return nil, err
	}
	teams, _, err := gt.Client().Teams.ListTeams(context.Background(), orgLogin, nil)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range teams {
		team := teams[i]
		r, err := g.MotorRuntime.CreateResource("github.team",
			"id", core.ToInt64(team.ID),
			"name", core.ToString(team.Name),
			"description", core.ToString(team.Description),
			"slug", core.ToString(team.Slug),
			"privacy", core.ToString(team.Privacy),
			"defaultPermission", core.ToString(team.Permission),
			"organization", g,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (g *mqlGithubOrganization) GetRepositories() ([]interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
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
			"name", core.ToString(repo.Name),
			"fullName", core.ToString(repo.FullName),
			"description", core.ToString(repo.Description),
			"homepage", core.ToString(repo.Homepage),
			"createdAt", githubTimestamp(repo.CreatedAt),
			"updatedAt", githubTimestamp(repo.UpdatedAt),
			"archived", core.ToBool(repo.Archived),
			"disabled", core.ToBool(repo.Disabled),
			"private", core.ToBool(repo.Private),
			"visibility", core.ToString(repo.Visibility),
			"allowAutoMerge", core.ToBool(repo.AllowAutoMerge),
			"allowForking", core.ToBool(repo.AllowForking),
			"allowMergeCommit", core.ToBool(repo.AllowMergeCommit),
			"allowRebaseMerge", core.ToBool(repo.AllowRebaseMerge),
			"allowSquashMerge", core.ToBool(repo.AllowSquashMerge),
			"hasIssues", core.ToBool(repo.HasIssues),
			"organizationName", orgLogin,
			"defaultBranchName", core.ToString(repo.DefaultBranch),
			"owner", owner,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (g *mqlGithubOrganization) GetWebhooks() ([]interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ownerLogin, err := g.Login()
	if err != nil {
		return nil, err
	}

	hooks, _, err := gt.Client().Organizations.ListHooks(context.TODO(), ownerLogin, &github.ListOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}
	res := []interface{}{}
	for i := range hooks {
		h := hooks[i]
		config, err := core.JsonToDict(h.Config)
		if err != nil {
			return nil, err
		}

		mqlUser, err := g.MotorRuntime.CreateResource("github.webhook",
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
		res = append(res, mqlUser)
	}

	return res, nil
}

func (g *mqlGithubPackage) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return "github.package/" + strconv.FormatInt(id, 10), nil
}

func (g *mqlGithubOrganization) GetPackages() ([]interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ownerLogin, err := g.Login()
	if err != nil {
		return nil, err
	}

	pkgTypes := []string{"npm", "maven", "rubygems", "docker", "nuget", "container"}
	res := []interface{}{}
	for i := range pkgTypes {
		packages, _, err := gt.Client().Organizations.ListPackages(context.Background(), ownerLogin, &github.PackageListOptions{
			PackageType: github.String(pkgTypes[i]),
		})
		if err != nil {
			log.Error().Err(err).Msg("unable to get hooks list")
			if strings.Contains(err.Error(), "404") {
				return nil, nil
			}
			return nil, err
		}

		for i := range packages {
			p := packages[i]

			owner, err := g.MotorRuntime.CreateResource("github.user",
				"id", p.GetOwner().GetID(),
				"login", p.GetOwner().GetLogin(),
			)
			if err != nil {
				return nil, err
			}

			mqlGhPackage, err := g.MotorRuntime.CreateResource("github.package",
				"id", core.ToInt64(p.ID),
				"name", core.ToString(p.Name),
				"packageType", core.ToString(p.PackageType),
				"owner", owner,
				"createdAt", githubTimestamp(p.CreatedAt),
				"updatedAt", githubTimestamp(p.UpdatedAt),
				"versionCount", core.ToInt64(p.VersionCount),
				"visibility", core.ToString(p.Visibility),
			)
			if err != nil {
				return nil, err
			}
			pkg := mqlGhPackage.(GithubPackage)

			// NOTE: we need to fetch repo separately because the Github repo object is not complete, instead of
			// call the repo fetching all the time, we make this lazy loading
			if p.Repository != nil && p.Repository.Name != nil {
				pkg.MqlResource().Cache.Store("_repository", &resources.CacheEntry{Data: core.ToString(p.Repository.Name)})
			}
			res = append(res, pkg)
		}
	}

	return res, nil
}

func (g *mqlGithubPackage) GetRepository() (interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	entry, ok := g.Cache.Load("_repository")
	if !ok {
		return nil, errors.New("could not load the repository")
	}

	repoName := entry.Data.(string)

	owner, err := g.Owner()
	if err != nil {
		return nil, err
	}

	ownerLogin, err := owner.Login()
	if err != nil {
		return nil, err
	}

	repo, _, err := gt.Client().Repositories.Get(context.Background(), ownerLogin, repoName)
	if err != nil {
		return nil, err
	}
	return newMqlGithubRepository(g.MotorRuntime, repo)
}

func (g *mqlGithubOrganization) GetInstallations() ([]interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
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
			"appId", core.ToInt64(app.AppID),
			"appSlug", core.ToString(app.AppSlug),
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

func (g *mqlGithubTeam) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return "github.team/" + strconv.FormatInt(id, 10), nil
}

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
		"createdAt", githubTimestamp(repo.CreatedAt),
		"updatedAt", githubTimestamp(repo.UpdatedAt),
		"archived", core.ToBool(repo.Archived),
		"disabled", core.ToBool(repo.Disabled),
		"private", core.ToBool(repo.Private),
		"visibility", core.ToString(repo.Visibility),
		"allowAutoMerge", core.ToBool(repo.AllowAutoMerge),
		"allowForking", core.ToBool(repo.AllowForking),
		"allowMergeCommit", core.ToBool(repo.AllowMergeCommit),
		"allowRebaseMerge", core.ToBool(repo.AllowRebaseMerge),
		"allowSquashMerge", core.ToBool(repo.AllowSquashMerge),
		"hasIssues", core.ToBool(repo.HasIssues),
		"organizationName", "",
		"defaultBranchName", core.ToString(repo.DefaultBranch),
		"owner", owner,
	)
}

func (g *mqlGithubTeam) GetRepositories() ([]interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	teamID, err := g.Id()
	if err != nil {
		return nil, err
	}

	org, err := g.Organization()
	if err != nil {
		return nil, err
	}

	orgID, err := org.Id()
	if err != nil {
		return nil, err
	}

	repos, _, err := gt.Client().Teams.ListTeamReposByID(context.Background(), orgID, teamID, &github.ListOptions{})
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range repos {
		repo := repos[i]

		r, err := newMqlGithubRepository(g.MotorRuntime, repo)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (g *mqlGithubTeam) GetMembers() ([]interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	teamID, err := g.Id()
	if err != nil {
		return nil, err
	}

	org, err := g.Organization()
	if err != nil {
		return nil, err
	}

	orgID, err := org.Id()
	if err != nil {
		return nil, err
	}

	members, _, err := gt.Client().Teams.ListTeamMembersByID(context.Background(), orgID, teamID, &github.TeamListTeamMembersOptions{})
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range members {
		member := members[i]

		r, err := g.MotorRuntime.CreateResource("github.user",
			"id", core.ToInt64(member.ID),
			"login", core.ToString(member.Login),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (g *mqlGithubUser) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return "github.user/" + strconv.FormatInt(id, 10), nil
}

func (g *mqlGithubUser) init(args *resources.Args) (*resources.Args, GithubUser, error) {
	if len(*args) > 3 {
		return args, nil, nil
	}

	if (*args)["login"] == nil {
		return nil, nil, errors.New("login required to fetch github user")
	}
	userLogin := (*args)["login"].(string)

	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	user, _, err := gt.Client().Users.Get(context.Background(), userLogin)
	if err != nil {
		return nil, nil, err
	}

	(*args)["id"] = (*args)["id"]
	(*args)["login"] = core.ToString(user.Login)
	(*args)["name"] = core.ToString(user.Name)
	(*args)["email"] = core.ToString(user.Email)
	(*args)["bio"] = core.ToString(user.Bio)
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
	(*args)["company"] = core.ToString(user.Company)
	return args, nil, nil
}

func (g *mqlGithubCollaborator) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(id, 10), nil
}

func (g *mqlGithubInstallation) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(id, 10), nil
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
	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	// userLogin := user.GetLogin()
	org, err := gt.Organization()
	if err != nil {
		return nil, nil, err
	}

	owner := org.GetLogin()
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
		(*args)["createdAt"] = githubTimestamp(repo.CreatedAt)
		(*args)["updatedAt"] = githubTimestamp(repo.UpdatedAt)
		(*args)["archived"] = core.ToBool(repo.Archived)
		(*args)["disabled"] = core.ToBool(repo.Disabled)
		(*args)["private"] = core.ToBool(repo.Private)
		(*args)["visibility"] = core.ToString(repo.Visibility)
		(*args)["allowAutoMerge"] = core.ToBool(repo.AllowAutoMerge)
		(*args)["allowForking"] = core.ToBool(repo.AllowForking)
		(*args)["allowMergeCommit"] = core.ToBool(repo.AllowMergeCommit)
		(*args)["allowRebaseMerge"] = core.ToBool(repo.AllowRebaseMerge)
		(*args)["allowSquashMerge"] = core.ToBool(repo.AllowSquashMerge)
		(*args)["hasIssues"] = core.ToBool(repo.HasIssues)
		(*args)["organizationName"] = ""
		(*args)["defaultBranchName"] = core.ToString(repo.DefaultBranch)
		(*args)["owner"] = owner
	}

	return args, nil, nil
}

func (g *mqlGithubRepository) GetOpenMergeRequests() ([]interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
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

func (g *mqlGithubMergeRequest) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(id, 10), nil
}

func (g *mqlGithubRepository) GetBranches() ([]interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
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
			"protected", branch.GetProtected(),
			"headCommit", mqlCommit,
			"organizationName", orgName,
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
	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
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

	if branchProtection.RequiredPullRequestReviews.DismissalRestrictions != nil {
		ghDismissalRestrictions = &githubDismissalRestrictions{
			Users: []string{},
			Teams: []string{},
		}

		for i := range branchProtection.RequiredPullRequestReviews.DismissalRestrictions.Teams {
			ghDismissalRestrictions.Teams = append(ghDismissalRestrictions.Teams, branchProtection.RequiredPullRequestReviews.DismissalRestrictions.Teams[i].GetName())
		}
		for i := range branchProtection.RequiredPullRequestReviews.DismissalRestrictions.Users {
			ghDismissalRestrictions.Users = append(ghDismissalRestrictions.Users, branchProtection.RequiredPullRequestReviews.DismissalRestrictions.Users[i].GetLogin())
		}
	}

	// we use a separate struct to ensure that the output is proper camelCase
	rprr, err := core.JsonToDict(githubRequiredPullRequestReviews{
		DismissStaleReviews:          branchProtection.RequiredPullRequestReviews.DismissStaleReviews,
		RequireCodeOwnerReviews:      branchProtection.RequiredPullRequestReviews.RequireCodeOwnerReviews,
		RequiredApprovingReviewCount: branchProtection.RequiredPullRequestReviews.RequiredApprovingReviewCount,
		DismissalRestrictions:        ghDismissalRestrictions,
	})
	if err != nil {
		return nil, err
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
		gt, err := githubtransport(runtime.Motor.Provider)
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
		githubCommitter, err = runtime.CreateResource("github.user", "id", core.ToInt64(rc.Committer.ID), "login", core.ToString(rc.Author.Login))
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
	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
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
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "Git Repository is empty") {
			return nil, nil
		}
		return nil, err
	}
	res := []interface{}{}
	for i := range commits {
		rc := commits[i]
		mqlCommit, err := newMqlGithubCommit(g.MotorRuntime, rc, orgName, repoName)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlCommit)
	}
	return res, nil
}

func (g *mqlGithubMergeRequest) GetReviews() ([]interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
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
	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
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

		mqlCommit, err := newMqlGithubCommit(g.MotorRuntime, rc, orgName, repoName)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlCommit)
	}
	return res, nil
}

func (g *mqlGithubRepository) GetContributors() ([]interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
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
		mqlUser, err := g.MotorRuntime.CreateResource("github.user",
			"id", core.ToInt64(contributors[i].ID),
			"login", core.ToString(contributors[i].Login),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlUser)
	}
	return res, nil
}

func (g *mqlGithubRepository) GetCollaborators() ([]interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
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
	contributors, _, err := gt.Client().Repositories.ListCollaborators(context.TODO(), ownerLogin, repoName, &github.ListCollaboratorsOptions{})
	if err != nil {
		log.Error().Err(err).Msg("unable to get collaborator list")
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}
	res := []interface{}{}
	for i := range contributors {

		mqlUser, err := g.MotorRuntime.CreateResource("github.user",
			"id", core.ToInt64(contributors[i].ID),
			"login", core.ToString(contributors[i].Login),
		)
		if err != nil {
			return nil, err
		}

		permissions := []string{}
		for k := range contributors[i].Permissions {
			permissions = append(permissions, k)
		}

		mqlContributor, err := g.MotorRuntime.CreateResource("github.collaborator",
			"id", core.ToInt64(contributors[i].ID),
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
	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
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
	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
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

	hooks, _, err := gt.Client().Repositories.ListHooks(context.TODO(), ownerLogin, repoName, &github.ListOptions{})
	if err != nil {
		log.Error().Err(err).Msg("unable to get hooks list")
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}
	res := []interface{}{}
	for i := range hooks {
		h := hooks[i]
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

func (g *mqlGithubWorkflow) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return "github.workflow/" + strconv.FormatInt(id, 10), nil
}

func (g *mqlGithubWorkflow) GetConfiguration() (interface{}, error) {
	// TODO: to leverage the runtime, get the file resource, how to define the dependency
	file, err := g.File()
	if err != nil {
		return nil, err
	}
	content, err := file.Content()
	if err != nil {
		return nil, err
	}

	data := map[string]interface{}{}
	err = yaml.Unmarshal([]byte(content), &data)
	if err != nil {
		return nil, err
	}

	return core.JsonToDict(data)
}

func (g *mqlGithubWorkflow) GetFile() (interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	filePath, err := g.Path()
	if err != nil {
		return nil, err
	}

	entry, ok := g.MqlResource().Cache.Load("_repositoryFullName")
	if !ok {
		return nil, errors.New("unable to get repository name")
	}

	fullName := entry.Data.(string)
	fullNameSplit := strings.Split(fullName, "/")
	ownerLogin := fullNameSplit[0]
	repoName := fullNameSplit[1]

	// TODO: no branch support yet
	// if we workflow is running for a branch only, we do not see from the response the branch name
	fileContent, _, _, err := gt.Client().Repositories.GetContents(context.Background(), ownerLogin, repoName, filePath, &github.RepositoryContentGetOptions{})
	if err != nil {
		// TODO: should this be an error
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}
	return newMqlGithubFile(g.MotorRuntime, ownerLogin, repoName, fileContent)
}

func (g *mqlGithubRepository) GetWorkflows() ([]interface{}, error) {
	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
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

	workflows, _, err := gt.Client().Actions.ListWorkflows(context.Background(), ownerLogin, repoName, &github.ListOptions{})
	if err != nil {
		log.Error().Err(err).Msg("unable to get hooks list")
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}

	res := []interface{}{}
	for i := range workflows.Workflows {
		w := workflows.Workflows[i]

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
	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
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
	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
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
	gt, err := githubtransport(g.MotorRuntime.Motor.Provider)
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
