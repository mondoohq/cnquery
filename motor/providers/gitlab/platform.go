package gitlab

import (
	"errors"
	"net/url"
	"strconv"

	"github.com/xanzy/go-gitlab"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
)

var (
	GitLabProjectPlatform = &platform.Platform{
		Name:    "gitlab-project",
		Title:   "GitLab Project",
		Family:  []string{"gitlab"},
		Kind:    providers.Kind_KIND_API,
		Runtime: providers.RUNTIME_GITLAB,
	}
	GitLabGroupPlatform = &platform.Platform{
		Name:    "gitlab-group",
		Title:   "GitLab Group",
		Family:  []string{"gitlab"},
		Kind:    providers.Kind_KIND_API,
		Runtime: providers.RUNTIME_GITLAB,
	}
)

func NewGitLabGroupIdentifier(groupID string) string {
	return "//platformid.api.mondoo.app/runtime/gitlab/group/" + groupID
}

func NewGitLabProjectIdentifier(groupID string, projectID string) string {
	return "//platformid.api.mondoo.app/runtime/gitlab/group/" + groupID + "/project/" + projectID
}

func (t *Provider) Identifier() (string, error) {
	grp, err := t.Group()
	if err != nil {
		return "", err
	}

	if t.ProjectPath != "" {
		project, err := t.Project()
		if err != nil {
			return "", err
		}
		return NewGitLabProjectIdentifier(strconv.Itoa(grp.ID), strconv.Itoa(project.ID)), nil
	} else {
		return NewGitLabGroupIdentifier(strconv.Itoa(grp.ID)), nil
	}
}

func (t *Provider) Group() (*gitlab.Group, error) {
	grp, _, err := t.Client().Groups.GetGroup(t.GroupPath, nil)
	if err != nil {
		return nil, err
	}
	return grp, err
}

func (t *Provider) Project() (*gitlab.Project, error) {
	project, _, err := t.Client().Projects.GetProject(url.QueryEscape(t.GroupPath)+"/"+url.QueryEscape(t.ProjectPath), nil)
	if err != nil {
		return nil, err
	}
	return project, err
}

func (p *Provider) PlatformInfo() (*platform.Platform, error) {
	if projectName := p.opts["project"]; projectName != "" {
		return GitLabProjectPlatform, nil
	}

	if groupName := p.opts["group"]; groupName != "" {
		return GitLabGroupPlatform, nil
	}

	return nil, errors.New("could not detect GitLab asset type")
}
