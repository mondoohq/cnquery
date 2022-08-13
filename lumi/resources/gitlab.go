package resources

import (
	"errors"
	"strconv"

	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/gitlab"
)

func gitlabtransport(t providers.Transport) (*gitlab.Provider, error) {
	gt, ok := t.(*gitlab.Provider)
	if !ok {
		return nil, errors.New("gitlab resource is not supported on this transport")
	}
	return gt, nil
}

func (g *lumiGitlabGroup) id() (string, error) {
	id, _ := g.Id()
	return "gitlab.group/" + strconv.FormatInt(id, 10), nil
}

// init initializes the gitlab group with the arguments
// see https://docs.gitlab.com/ee/api/groups.html#new-group
func (g *lumiGitlabGroup) init(args *lumi.Args) (*lumi.Args, GitlabGroup, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	gt, err := gitlabtransport(g.MotorRuntime.Motor.Transport)
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
func (g *lumiGitlabGroup) GetProjects() ([]interface{}, error) {
	gt, err := gitlabtransport(g.MotorRuntime.Motor.Transport)
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

	var lumiProjects []interface{}
	for i := range grp.Projects {
		prj := grp.Projects[i]

		lumiProject, err := g.MotorRuntime.CreateResource("gitlab.project",
			"id", int64(prj.ID),
			"name", prj.Name,
			"path", prj.Path,
			"description", prj.Description,
			"visibility", string(prj.Visibility),
		)
		if err != nil {
			return nil, err
		}
		lumiProjects = append(lumiProjects, lumiProject)
	}

	return lumiProjects, nil
}

func (g *lumiGitlabProject) id() (string, error) {
	id, _ := g.Id()
	return "gitlab.project/" + strconv.FormatInt(id, 10), nil
}
