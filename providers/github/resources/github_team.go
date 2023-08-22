// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strconv"
	"strings"

	"github.com/google/go-github/v49/github"
	"go.mondoo.com/cnquery/llx"
)

func (g *mqlGithubTeam) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return "github.team/" + strconv.FormatInt(id, 10), nil
}

func (g *mqlGithubTeam) repositories() ([]interface{}, error) {
	gt, err := githubProvider(g.MqlRuntime.Connection)
	if err != nil {
		return nil, err
	}

	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	teamID := g.Id.Data

	if g.Organization.Error != nil {
		return nil, g.Organization.Error
	}
	org := g.Organization.Data

	if org.Id.Error != nil {
		return nil, org.Id.Error
	}
	orgID := org.Id.Data

	listOpts := &github.ListOptions{}
	var allRepos []*github.Repository
	for {
		repos, resp, err := gt.Client().Teams.ListTeamReposByID(context.Background(), orgID, teamID, listOpts)
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

func (g *mqlGithubTeam) members() ([]interface{}, error) {
	gt, err := githubProvider(g.MqlRuntime.Connection)
	if err != nil {
		return nil, err
	}

	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	teamID := g.Id.Data

	if g.Organization.Error != nil {
		return nil, g.Organization.Error
	}
	org := g.Organization.Data

	if org.Id.Error != nil {
		return nil, org.Id.Error
	}
	orgID := org.Id.Data

	listOpts := &github.TeamListTeamMembersOptions{
		ListOptions: github.ListOptions{PerPage: paginationPerPage},
	}
	var allMembers []*github.User
	for {
		members, resp, err := gt.Client().Teams.ListTeamMembersByID(context.Background(), orgID, teamID, listOpts)
		if err != nil {
			return nil, err
		}
		allMembers = append(allMembers, members...)
		if resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}

	res := []interface{}{}
	for i := range allMembers {
		member := allMembers[i]

		r, err := CreateResource(g.MqlRuntime, "github.user", map[string]*llx.RawData{
			"id":    llx.IntDataPtr(member.ID),
			"login": llx.StringDataPtr(member.Login),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}
