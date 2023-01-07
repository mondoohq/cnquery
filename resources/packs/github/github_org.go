package github

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/google/go-github/v47/github"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

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

	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
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
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
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
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
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
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
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
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
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
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
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

func (g *mqlGithubOrganization) GetPackages() ([]interface{}, error) {
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
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
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
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
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
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
