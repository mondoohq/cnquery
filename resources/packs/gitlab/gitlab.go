package gitlab

import (
	"errors"
	"strconv"

	"go.mondoo.com/cnquery/motor/providers"
	provider "go.mondoo.com/cnquery/motor/providers/gitlab"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/gitlab/info"
)

var Registry = info.Registry

func init() {
	Init(Registry)
}

func gitlabProvider(t providers.Instance) (*provider.Provider, error) {
	gt, ok := t.(*provider.Provider)
	if !ok {
		return nil, errors.New("gitlab resource is not supported on this transport")
	}
	return gt, nil
}

func (g *mqlGitlabGroup) id() (string, error) {
	id, _ := g.Id()
	return "gitlab.group/" + strconv.FormatInt(id, 10), nil
}

// init initializes the gitlab group with the arguments
// see https://docs.gitlab.com/ee/api/groups.html#new-group
func (g *mqlGitlabGroup) init(args *resources.Args) (*resources.Args, GitlabGroup, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	gt, err := gitlabProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	grp, _, err := gt.Client().Groups.GetGroup(gt.GroupPath, nil)
	if err != nil {
		return nil, nil, err
	}

	(*args)["id"] = int64(grp.ID)
	(*args)["name"] = grp.Name
	(*args)["path"] = grp.Path
	(*args)["description"] = grp.Description
	(*args)["visibility"] = string(grp.Visibility)
	(*args)["requireTwoFactorAuthentication"] = grp.RequireTwoFactorAuth

	return args, nil, nil
}

// GetProjects list all projects that belong to a group
// see https://docs.gitlab.com/ee/api/projects.html
func (g *mqlGitlabGroup) GetProjects() ([]interface{}, error) {
	gt, err := gitlabProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	path, err := g.Path()
	if err != nil {
		return nil, err
	}

	grp, _, err := gt.Client().Groups.GetGroup(path, nil)
	if err != nil {
		return nil, err
	}

	var mqlProjects []interface{}
	for i := range grp.Projects {
		prj := grp.Projects[i]

		mqlProject, err := g.MotorRuntime.CreateResource("gitlab.project",
			"id", int64(prj.ID),
			"name", prj.Name,
			"path", prj.Path,
			"description", prj.Description,
			"visibility", string(prj.Visibility),
		)
		if err != nil {
			return nil, err
		}
		mqlProjects = append(mqlProjects, mqlProject)
	}

	return mqlProjects, nil
}

func (g *mqlGitlabProject) id() (string, error) {
	id, _ := g.Id()
	return "gitlab.project/" + strconv.FormatInt(id, 10), nil
}
