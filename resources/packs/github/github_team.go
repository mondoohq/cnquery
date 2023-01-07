package github

import (
	"context"
	"strconv"

	"github.com/google/go-github/v47/github"
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

	repos, _, err := gt.Client().Teams.ListTeamReposByID(context.Background(), orgID, teamID, &github.ListOptions{})
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range repos {
		repo := repos[i]

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

	members, _, err := gt.Client().Teams.ListTeamMembersByID(context.Background(), orgID, teamID, &github.TeamListTeamMembersOptions{})
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range members {
		member := members[i]

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
