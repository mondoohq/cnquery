// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"strconv"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/gitlab/connection"
)

func gitlabProvider(c plugin.Connection) (*connection.GitLabConnection, error) {
	gt, ok := c.(*connection.GitLabConnection)
	if !ok {
		return nil, errors.New("gitlab resource is not supported on this provider")
	}
	return gt, nil
}

func (g *mqlGitlabGroup) id() (string, error) {
	return "gitlab.group/" + strconv.FormatInt(g.Id.Data, 10), nil
}

// init initializes the gitlab group with the arguments
// see https://docs.gitlab.com/ee/api/groups.html#new-group
func initGitlabGroup(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	gt, err := gitlabProvider(runtime.Connection)
	if err != nil {
		return nil, nil, err
	}

	grp, _, err := gt.Client().Groups.GetGroup(gt.GroupPath, nil)
	if err != nil {
		return nil, nil, err
	}

	args["id"] = llx.IntData(int64(grp.ID))
	args["name"] = llx.StringData(grp.Name)
	args["path"] = llx.StringData(grp.Path)
	args["description"] = llx.StringData(grp.Description)
	args["visibility"] = llx.StringData(string(grp.Visibility))
	args["requireTwoFactorAuthentication"] = llx.BoolData(grp.RequireTwoFactorAuth)

	return args, nil, nil
}

// GetProjects list all projects that belong to a group
// see https://docs.gitlab.com/ee/api/projects.html
func (g *mqlGitlabGroup) projects() ([]interface{}, error) {
	gt, err := gitlabProvider(g.MqlRuntime.Connection)
	if err != nil {
		return nil, err
	}

	if g.Path.Error != nil {
		return nil, g.Path.Error
	}
	path := g.Path.Data

	grp, _, err := gt.Client().Groups.GetGroup(path, nil)
	if err != nil {
		return nil, err
	}

	var mqlProjects []interface{}
	for i := range grp.Projects {
		prj := grp.Projects[i]

		mqlProject, err := CreateResource(g.MqlRuntime, "gitlab.project", map[string]*llx.RawData{
			"id":          llx.IntData(int64(prj.ID)),
			"name":        llx.StringData(prj.Name),
			"path":        llx.StringData(prj.Path),
			"description": llx.StringData(prj.Description),
			"visibility":  llx.StringData(string(prj.Visibility)),
		})
		if err != nil {
			return nil, err
		}
		mqlProjects = append(mqlProjects, mqlProject)
	}

	return mqlProjects, nil
}

func (g *mqlGitlabProject) id() (string, error) {
	return "gitlab.project/" + strconv.FormatInt(g.Id.Data, 10), nil
}
