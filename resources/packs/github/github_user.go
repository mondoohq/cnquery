package github

import (
	"context"
	"io"
	"strconv"
	"strings"
	"time"

	"errors"
	"github.com/google/go-github/v49/github"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"go.mondoo.com/cnquery/stringx"
	"go.mondoo.com/ranger-rpc"
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
	(*args)["blog"] = user.GetBlog()
	(*args)["location"] = user.GetLocation()
	(*args)["avatarUrl"] = user.GetAvatarURL()
	(*args)["followers"] = int64(user.GetFollowers())
	(*args)["following"] = int64(user.GetFollowing())
	(*args)["twitterUsername"] = user.GetTwitterUsername()
	(*args)["bio"] = user.GetBio()
	var createdAt *time.Time
	if user.CreatedAt != nil {
		createdAt = &user.CreatedAt.Time
	}
	(*args)["createdAt"] = createdAt
	var updatedAt *time.Time
	if user.UpdatedAt != nil {
		updatedAt = &user.UpdatedAt.Time
	}
	(*args)["updatedAt"] = updatedAt
	var suspendedAt *time.Time
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
		r, err := newMqlGithubRepository(g.MotorRuntime, repo)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (g *mqlGithubGist) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return "github.gist/" + id, nil
}

func (g *mqlGithubUser) GetGists() ([]interface{}, error) {
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	userLogin, err := g.Login()
	if err != nil {
		return nil, err
	}

	listOpts := &github.GistListOptions{
		ListOptions: github.ListOptions{PerPage: paginationPerPage},
	}

	var allGists []*github.Gist
	for {
		gists, resp, err := gt.Client().Gists.List(context.Background(), userLogin, listOpts)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				return nil, nil
			}
			return nil, err
		}
		allGists = append(allGists, gists...)
		if resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}

	res := []interface{}{}
	for i := range allGists {
		gist := allGists[i]

		files := []interface{}{}
		for k := range gist.Files {
			f := gist.Files[k]

			gistFile, err := g.MotorRuntime.CreateResource("github.gistfile",
				"gistId", core.ToString(gist.ID),
				"filename", f.GetFilename(),
				"type", f.GetType(),
				"language", f.GetLanguage(),
				"rawUrl", f.GetRawURL(),
				"size", int64(f.GetSize()),
			)
			if err != nil {
				return nil, err
			}
			files = append(files, gistFile)
		}

		r, err := g.MotorRuntime.CreateResource("github.gist",
			"id", core.ToString(gist.ID),
			"description", core.ToString(gist.Description),
			"createdAt", gist.CreatedAt,
			"updatedAt", gist.UpdatedAt,
			"public", core.ToBool(gist.Public),
			"owner", g,
			"files", files,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (g *mqlGithubGistfile) id() (string, error) {
	id, err := g.GistId()
	if err != nil {
		return "", err
	}
	filename, err := g.Filename()
	if err != nil {
		return "", err
	}
	return "github.gistfile/" + id + "/" + filename, nil
}

var supportedGistContentTypes = []string{
	"text/plain",
	"text/markdown",
	"text/html",
	"text/x-yaml",
	"application/json",
	"application/javascript",
	"application/x-ruby",
	"application/x-perl",
	"application/x-php",
	"application/x-python",
	"application/x-java",
	"application/x-csharp",
	"application/x-clojure",
	"application/x-sh",
	"application/x-powershell",
	"application/x-msdos-program",
}

func (g *mqlGithubGistfile) GetContent() (string, error) {
	rawUrl, err := g.RawUrl()
	if err != nil {
		return "", err
	}

	filetyp, err := g.Type()
	if err != nil {
		return "", err
	}

	// supported content types
	if !stringx.Contains(supportedGistContentTypes, filetyp) {
		return "", errors.New("unsupported content type: " + filetyp)
	}

	resp, err := ranger.DefaultHttpClient().Get(rawUrl)
	if err != nil {
		return "", err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
