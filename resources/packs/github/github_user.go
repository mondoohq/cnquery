package github

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/go-github/v49/github"
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

	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	var user *github.User
	if (*args)["login"] == nil {
		user, err = gt.User()
		if err != nil {
			return nil, nil, errors.New("login required to fetch github user")
		}
	} else {
		userLogin := (*args)["login"].(string)
		user, _, err = gt.Client().Users.Get(context.Background(), userLogin)
		if err != nil {
			return nil, nil, err
		}
	}

	(*args)["id"] = user.GetID()
	(*args)["login"] = user.GetLogin()
	(*args)["name"] = user.GetName()
	(*args)["email"] = user.GetEmail()
	(*args)["bio"] = user.GetBio()
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
	(*args)["company"] = user.GetCompany()
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

	listOpts := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{PerPage: paginationPerPage},
	}

	var allRepos []*github.Repository
	for {
		repos, resp, err := gt.Client().Repositories.List(context.Background(), githubLogin, listOpts)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				return nil, nil
			}
			return nil, err
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}

	res := []interface{}{}
	for i := range allRepos {
		repo := allRepos[i]

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
