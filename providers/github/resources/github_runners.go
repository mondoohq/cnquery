// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"
	"strings"

	"github.com/google/go-github/v84/github"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/github/connection"
	"go.mondoo.com/mql/v13/types"
)

func (g *mqlGithubRunner) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	return "github.runner/" + strconv.FormatInt(g.Id.Data, 10), nil
}

func (g *mqlGithubRunnerLabel) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	return "github.runnerLabel/" + strconv.FormatInt(g.Id.Data, 10), nil
}

// runners returns the self-hosted runners for an organization.
func (g *mqlGithubOrganization) runners() ([]any, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)

	if g.Login.Error != nil {
		return nil, g.Login.Error
	}
	orgLogin := g.Login.Data

	opts := &github.ListRunnersOptions{
		ListOptions: github.ListOptions{PerPage: paginationPerPage},
	}

	var allRunners []*github.Runner
	for {
		runners, resp, err := conn.Client().Actions.ListOrganizationRunners(conn.Context(), orgLogin, opts)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				return nil, nil
			}
			if strings.Contains(err.Error(), "403") {
				log.Debug().Msg("Self-hosted runners are not accessible for this organization")
				return nil, nil
			}
			return nil, err
		}
		allRunners = append(allRunners, runners.Runners...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return runnersToMql(g.MqlRuntime, allRunners)
}

// runners returns the self-hosted runners for a repository.
func (g *mqlGithubRepository) runners() ([]any, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)

	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	repoName := g.Name.Data

	if g.Owner.Error != nil {
		return nil, g.Owner.Error
	}
	owner := g.Owner.Data

	if owner.Login.Error != nil {
		return nil, owner.Login.Error
	}
	ownerLogin := owner.Login.Data

	opts := &github.ListRunnersOptions{
		ListOptions: github.ListOptions{PerPage: paginationPerPage},
	}

	var allRunners []*github.Runner
	for {
		runners, resp, err := conn.Client().Actions.ListRunners(conn.Context(), ownerLogin, repoName, opts)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				return nil, nil
			}
			if strings.Contains(err.Error(), "403") {
				log.Debug().Msg("Self-hosted runners are not accessible for this repository")
				return nil, nil
			}
			return nil, err
		}
		allRunners = append(allRunners, runners.Runners...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return runnersToMql(g.MqlRuntime, allRunners)
}

// runnersToMql converts a list of GitHub runners to MQL resources.
func runnersToMql(runtime *plugin.Runtime, runners []*github.Runner) ([]any, error) {
	res := []any{}
	for _, runner := range runners {
		labels := []any{}
		for _, label := range runner.Labels {
			labelRes, err := CreateResource(runtime, "github.runnerLabel", map[string]*llx.RawData{
				"id":   llx.IntDataDefault(label.ID, 0),
				"name": llx.StringDataPtr(label.Name),
				"type": llx.StringDataPtr(label.Type),
			})
			if err != nil {
				return nil, err
			}
			labels = append(labels, labelRes)
		}

		r, err := CreateResource(runtime, "github.runner", map[string]*llx.RawData{
			"id":     llx.IntDataDefault(runner.ID, 0),
			"name":   llx.StringDataPtr(runner.Name),
			"os":     llx.StringDataPtr(runner.OS),
			"status": llx.StringDataPtr(runner.Status),
			"busy":   llx.BoolDataPtr(runner.Busy),
			"labels": llx.ArrayData(labels, types.Resource("github.runnerLabel")),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}
