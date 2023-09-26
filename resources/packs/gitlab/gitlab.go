package gitlab

import (
	"errors"
	"strconv"

	"github.com/rs/zerolog/log"
	"github.com/xanzy/go-gitlab"
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
		return nil, errors.New("gitlab resource is not supported on this provider")
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
	resArgs := []interface{}{
		"id", int64(grp.ID),
		"name", grp.Name,
		"path", grp.Path,
		"description", grp.Description,
		"visibility", string(grp.Visibility),
		"requireTwoFactorAuthentication", grp.RequireTwoFactorAuth,
	}
	projects, err := g.createProjectResources(grp)
	if err != nil {
		return nil, nil, err
	}
	resArgs = append(resArgs, "projects", projects)
	mqlGroup, err := g.MotorRuntime.CreateResource("gitlab.group", resArgs...)
	if err != nil {
		return nil, nil, err
	}
	return args, mqlGroup.(*mqlGitlabGroup), nil
}

func (g *mqlGitlabGroup) createProjectResources(grp *gitlab.Group) ([]interface{}, error) {
	var mqlProjects []interface{}
	for i := range grp.Projects {
		prj := grp.Projects[i]

		mqlProject, err := g.MotorRuntime.CreateResource("gitlab.project",
			"id", int64(prj.ID),
			"name", prj.Name,
			"path", prj.Path,
			"namespace", prj.Namespace.Name,
			"description", prj.Description,
			"visibility", string(prj.Visibility),
		)
		if err != nil {
			// log the err. we're seeing weird behavior with these apis. lets log if we have
			// issues here
			log.Error().Err(err).Str("path", prj.Path).Msg("cannot create gitlab project asset")
		} else {
			mqlProjects = append(mqlProjects, mqlProject)
		}
	}
	return mqlProjects, nil
}

// GetProjects list all projects that belong to a group
// see https://docs.gitlab.com/ee/api/projects.html
func (g *mqlGitlabGroup) GetProjects() ([]interface{}, error) {
	return g.Projects()
}

func (g *mqlGitlabProject) id() (string, error) {
	id, _ := g.Id()
	return "gitlab.project/" + strconv.FormatInt(id, 10), nil
}

// init initializes the gitlab project with the arguments
func (g *mqlGitlabProject) init(args *resources.Args) (*resources.Args, GitlabProject, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	gt, err := gitlabProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}
	obj, err := g.MotorRuntime.CreateResource("gitlab.group")
	if err != nil {
		return nil, nil, err
	}
	gr := obj.(*mqlGitlabGroup)

	rawResources, err := gr.Projects()
	if err != nil {
		return nil, nil, err
	}
	matcher := gt.ProjectPath
	for i := range rawResources {
		proj := rawResources[i].(*mqlGitlabProject)
		mqlPath, err := proj.Path()
		if err != nil {
			log.Error().Err(err).Msg("project is not initialized")
			return nil, nil, err
		}
		if mqlPath == matcher {
			return args, proj, nil
		}
	}

	return nil, nil, errors.New("project not found")
}
