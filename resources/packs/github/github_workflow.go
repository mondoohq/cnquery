package github

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/google/go-github/v47/github"
	"go.mondoo.com/cnquery/resources/packs/core"
	"sigs.k8s.io/yaml"
)

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
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
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
