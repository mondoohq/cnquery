// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-github/v89/github"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/github/connection"
	"go.mondoo.com/mql/v13/types"
)

// ghRunnerExt mirrors the Runner JSON returned by GitHub plus fields that
// go-github v85 doesn't expose on its Runner struct (ephemeral, architecture).
type ghRunnerExt struct {
	ID           *int64                 `json:"id,omitempty"`
	Name         *string                `json:"name,omitempty"`
	OS           *string                `json:"os,omitempty"`
	Status       *string                `json:"status,omitempty"`
	Busy         *bool                  `json:"busy,omitempty"`
	Ephemeral    *bool                  `json:"ephemeral,omitempty"`
	Architecture *string                `json:"architecture,omitempty"`
	Labels       []*github.RunnerLabels `json:"labels,omitempty"`
}

type ghRunnersListResp struct {
	TotalCount int            `json:"total_count"`
	Runners    []*ghRunnerExt `json:"runners"`
}

// listRunnersRaw paginates through a self-hosted runners endpoint (org or repo)
// using the raw API so we can read fields go-github v85 doesn't expose.
func listRunnersRaw(ctx context.Context, client *github.Client, basePath string) ([]*ghRunnerExt, error) {
	var all []*ghRunnerExt
	page := 1
	for {
		var body ghRunnersListResp
		url := fmt.Sprintf("%s?per_page=%d&page=%d", basePath, paginationPerPage, page)
		resp, err := doRawJSON(ctx, client, url, &body)
		if err != nil {
			return nil, err
		}
		all = append(all, body.Runners...)
		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}
	return all, nil
}

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

	allRunners, err := listRunnersRaw(conn.Context(), conn.Client(), fmt.Sprintf("orgs/%s/actions/runners", orgLogin))
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

	allRunners, err := listRunnersRaw(conn.Context(), conn.Client(), fmt.Sprintf("repos/%s/%s/actions/runners", ownerLogin, repoName))
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

	return runnersToMql(g.MqlRuntime, allRunners)
}

// runnersToMql converts a list of GitHub runners to MQL resources.
func runnersToMql(runtime *plugin.Runtime, runners []*ghRunnerExt) ([]any, error) {
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
			"id":           llx.IntDataDefault(runner.ID, 0),
			"name":         llx.StringDataPtr(runner.Name),
			"os":           llx.StringDataPtr(runner.OS),
			"status":       llx.StringDataPtr(runner.Status),
			"busy":         llx.BoolDataPtr(runner.Busy),
			"ephemeral":    llx.BoolDataPtr(runner.Ephemeral),
			"architecture": llx.StringDataPtr(runner.Architecture),
			"labels":       llx.ArrayData(labels, types.Resource("github.runnerLabel")),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}
