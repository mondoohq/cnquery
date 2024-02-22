// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"io"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/google/go-github/v59/github"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/github/connection"
	"go.mondoo.com/cnquery/v10/types"
	"go.mondoo.com/cnquery/v10/utils/stringx"
	"go.mondoo.com/ranger-rpc"
)

func (g *mqlGithubUser) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return "github.user/" + strconv.FormatInt(id, 10), nil
}

func initGithubUser(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 3 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.GithubConnection)

	userLogin := ""
	if args["login"] != nil {
		userLogin = args["login"].Value.(string)
	} else {
		userId, err := conn.User()
		if err != nil {
			return nil, nil, errors.New("login required to fetch github user")
		}
		userLogin = userId.Name
	}

	// finally grab the user from github
	githubUser, err := getUser(context.Background(), runtime, conn, userLogin)
	if err != nil {
		return nil, nil, err
	}

	name := githubUser.GetName()
	if name == "" {
		if x, ok := args["name"]; ok {
			name = x.Value.(string)
		}
	}
	args["id"] = llx.IntData(githubUser.GetID())
	args["login"] = llx.StringData(githubUser.GetLogin())
	args["name"] = llx.StringData(name)
	args["email"] = llx.StringData(githubUser.GetEmail())
	args["blog"] = llx.StringData(githubUser.GetBlog())
	args["location"] = llx.StringData(githubUser.GetLocation())
	args["avatarUrl"] = llx.StringData(githubUser.GetAvatarURL())
	args["followers"] = llx.IntData(int64(githubUser.GetFollowers()))
	args["following"] = llx.IntData(int64(githubUser.GetFollowing()))
	args["twitterUsername"] = llx.StringData(githubUser.GetTwitterUsername())
	args["bio"] = llx.StringData(githubUser.GetBio())

	args["createdAt"] = llx.TimeDataPtr(githubTimestamp(githubUser.CreatedAt))
	args["updatedAt"] = llx.TimeDataPtr(githubTimestamp(githubUser.UpdatedAt))
	args["suspendedAt"] = llx.TimeDataPtr(githubTimestamp(githubUser.SuspendedAt))
	args["company"] = llx.StringData(githubUser.GetCompany())
	return args, nil, nil
}

func (g *mqlGithubCollaborator) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return strconv.FormatInt(id, 10), nil
}

func (g *mqlGithubInstallation) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return strconv.FormatInt(id, 10), nil
}

func (g *mqlGithubUser) repositories() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)

	if g.Login.Error != nil {
		return nil, g.Login.Error
	}
	githubLogin := g.Login.Data

	listOpts := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{PerPage: paginationPerPage},
	}

	var allRepos []*github.Repository
	for {
		repos, resp, err := conn.Client().Repositories.List(context.Background(), githubLogin, listOpts)
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
		r, err := newMqlGithubRepository(g.MqlRuntime, repo)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (g *mqlGithubGist) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return "github.gist/" + id, nil
}

func (g *mqlGithubUser) gists() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)

	if g.Login.Error != nil {
		return nil, g.Login.Error
	}
	userLogin := g.Login.Data

	listOpts := &github.GistListOptions{
		ListOptions: github.ListOptions{PerPage: paginationPerPage},
	}

	var allGists []*github.Gist
	for {
		gists, resp, err := conn.Client().Gists.List(context.Background(), userLogin, listOpts)
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

			gistFile, err := CreateResource(g.MqlRuntime, "github.gistfile", map[string]*llx.RawData{
				"gistId":   llx.StringDataPtr(gist.ID),
				"filename": llx.StringData(f.GetFilename()),
				"type":     llx.StringData(f.GetType()),
				"language": llx.StringData(f.GetLanguage()),
				"rawUrl":   llx.StringData(f.GetRawURL()),
				"size":     llx.IntData(int64(f.GetSize())),
			})
			if err != nil {
				return nil, err
			}
			files = append(files, gistFile)
		}

		r, err := CreateResource(g.MqlRuntime, "github.gist", map[string]*llx.RawData{
			"id":          llx.StringDataPtr(gist.ID),
			"description": llx.StringDataPtr(gist.Description),
			"createdAt":   llx.TimeDataPtr(githubTimestamp(gist.CreatedAt)),
			"updatedAt":   llx.TimeDataPtr(githubTimestamp(gist.UpdatedAt)),
			"public":      llx.BoolDataPtr(gist.Public),
			"owner":       llx.ResourceData(g, g.MqlName()),
			"files":       llx.ArrayData(files, types.Any),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (g *mqlGithubGistfile) id() (string, error) {
	if g.GistId.Error != nil {
		return "", g.GistId.Error
	}
	id := g.GistId.Data
	if g.Filename.Error != nil {
		return "", g.Filename.Error
	}
	filename := g.Filename.Data
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

func (g *mqlGithubGistfile) content() (string, error) {
	if g.RawUrl.Error != nil {
		return "", g.RawUrl.Error
	}
	rawUrl := g.RawUrl.Data

	if g.Type.Error != nil {
		return "", g.Type.Error
	}
	filetyp := g.Type.Data

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
