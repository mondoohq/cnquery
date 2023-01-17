package github

import (
	"context"
	"strconv"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/go-github/v47/github"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

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

	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
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

func (g *mqlGithubUser) GetRepositories() ([]interface{}, error) {
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	githubLogin, err := g.Login()
	if err != nil {
		return nil, err
	}

	repos, _, err := gt.Client().Repositories.List(context.Background(), githubLogin, &github.RepositoryListOptions{})
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
