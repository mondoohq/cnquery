// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package github

import (
	"context"
	"strconv"
	"strings"

	"github.com/google/go-github/v49/github"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (g *mqlGithubTeam) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return "github.team/" + strconv.FormatInt(id, 10), nil
}

func (g *mqlGithubTeam) GetRepositories() ([]interface{}, error) {
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	teamID, err := g.Id()
	if err != nil {
		return nil, err
	}

	org, err := g.Organization()
	if err != nil {
		return nil, err
	}

	orgID, err := org.Id()
	if err != nil {
		return nil, err
	}

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

		r, err := newMqlGithubRepository(g.MotorRuntime, repo)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (g *mqlGithubTeam) GetMembers() ([]interface{}, error) {
	gt, err := githubProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	teamID, err := g.Id()
	if err != nil {
		return nil, err
	}

	org, err := g.Organization()
	if err != nil {
		return nil, err
	}

	orgID, err := org.Id()
	if err != nil {
		return nil, err
	}

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

		r, err := g.MotorRuntime.CreateResource("github.user",
			"id", core.ToInt64(member.ID),
			"login", core.ToString(member.Login),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}
