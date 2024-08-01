// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"strconv"
	"strings"

	"github.com/google/go-github/v62/github"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/github/connection"
	"sigs.k8s.io/yaml"
)

func (g *mqlGithubWorkflow) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return "github.workflow/" + strconv.FormatInt(id, 10), nil
}

func (g *mqlGithubWorkflow) configuration() (interface{}, error) {
	// TODO: to leverage the runtime, get the file resource, how to define the dependency
	if g.File.Error != nil {
		return nil, g.File.Error
	}
	file := g.File.Data
	if file.Content.Error != nil {
		return nil, file.Content.Error
	}
	content := file.Content.Data

	data := map[string]interface{}{}
	err := yaml.Unmarshal([]byte(content), &data)
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(data)
}

func (g *mqlGithubWorkflow) file() (*mqlGithubFile, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)

	if g.Path.Error != nil {
		return nil, g.Path.Error
	}
	filePath := g.Path.Data

	if g.repositoryFullName == "" {
		return nil, errors.New("repositoryFullName is not set")
	}
	fullName := g.repositoryFullName
	fullNameSplit := strings.Split(fullName, "/")
	ownerLogin := fullNameSplit[0]
	repoName := fullNameSplit[1]

	// TODO: no branch support yet
	// if we workflow is running for a branch only, we do not see from the response the branch name
	fileContent, _, _, err := conn.Client().Repositories.GetContents(conn.Context(), ownerLogin, repoName, filePath, &github.RepositoryContentGetOptions{})
	if err != nil {
		// TODO: should this be an error
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}
	return newMqlGithubFile(g.MqlRuntime, ownerLogin, repoName, fileContent)
}
