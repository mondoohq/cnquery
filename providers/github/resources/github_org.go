// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/google/go-github/v59/github"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/github/connection"
	"go.mondoo.com/cnquery/v10/types"
)

func (g *mqlGithubOrganization) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	return "github.organization/" + strconv.FormatInt(g.Id.Data, 10), nil
}

func initGithubOrganization(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.GithubConnection)

	orgId, err := conn.Organization()
	if err != nil {
		return nil, nil, err
	}

	name := orgId.Name
	if name == "" {
		if x, ok := args["name"]; ok {
			name = x.Value.(string)
		}
	}

	org, _, err := conn.Client().Organizations.Get(context.Background(), name)
	if err != nil {
		return args, nil, err
	}

	args["id"] = llx.IntDataPtr(org.ID)
	args["name"] = llx.StringData(name)
	args["login"] = llx.StringDataPtr(org.Login)
	args["nodeId"] = llx.StringDataPtr(org.NodeID)
	args["company"] = llx.StringDataPtr(org.Company)
	args["blog"] = llx.StringDataPtr(org.Blog)
	args["location"] = llx.StringDataPtr(org.Location)
	args["email"] = llx.StringDataPtr(org.Email)
	args["twitterUsername"] = llx.StringDataPtr(org.TwitterUsername)
	args["avatarUrl"] = llx.StringDataPtr(org.AvatarURL)
	args["followers"] = llx.IntData(convert.ToInt64FromInt(org.Followers))
	args["following"] = llx.IntData(convert.ToInt64FromInt(org.Following))
	args["description"] = llx.StringDataPtr(org.Description)
	args["createdAt"] = llx.TimeDataPtr(githubTimestamp(org.CreatedAt))
	args["updatedAt"] = llx.TimeDataPtr(githubTimestamp(org.UpdatedAt))
	args["totalPrivateRepos"] = llx.IntDataPtr(org.TotalPrivateRepos)
	args["ownedPrivateRepos"] = llx.IntDataPtr(org.OwnedPrivateRepos)
	args["privateGists"] = llx.IntData(convert.ToInt64FromInt(org.PrivateGists))
	args["diskUsage"] = llx.IntData(convert.ToInt64FromInt(org.DiskUsage))
	args["collaborators"] = llx.IntData(convert.ToInt64FromInt(org.Collaborators))
	args["billingEmail"] = llx.StringDataPtr(org.BillingEmail)

	plan, _ := convert.JsonToDict(org.Plan)
	args["plan"] = llx.MapData(plan, types.Any)

	args["twoFactorRequirementEnabled"] = llx.BoolData(convert.ToBool(org.TwoFactorRequirementEnabled))
	args["isVerified"] = llx.BoolData(convert.ToBool(org.IsVerified))

	args["defaultRepositoryPermission"] = llx.StringDataPtr(org.DefaultRepoPermission)
	args["membersCanCreateRepositories"] = llx.BoolData(convert.ToBool(org.MembersCanCreateRepos))
	args["membersCanCreatePublicRepositories"] = llx.BoolData(convert.ToBool(org.MembersCanCreatePublicRepos))
	args["membersCanCreatePrivateRepositories"] = llx.BoolData(convert.ToBool(org.MembersCanCreatePrivateRepos))
	args["membersCanCreateInternalRepositories"] = llx.BoolData(convert.ToBool(org.MembersCanCreateInternalRepos))
	args["membersCanCreatePages"] = llx.BoolData(convert.ToBool(org.MembersCanCreatePages))
	args["membersCanCreatePublicPages"] = llx.BoolData(convert.ToBool(org.MembersCanCreatePublicPages))
	args["membersCanCreatePrivatePages"] = llx.BoolData(convert.ToBool(org.MembersCanCreatePrivateRepos))
	args["membersCanForkPrivateRepos"] = llx.BoolData(convert.ToBool(org.MembersCanForkPrivateRepos))

	return args, nil, nil
}

func (g *mqlGithubOrganization) members() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)
	if g.Login.Error != nil {
		return nil, g.Login.Error
	}
	orgLogin := g.Login.Data

	listOpts := &github.ListMembersOptions{
		ListOptions: github.ListOptions{PerPage: paginationPerPage},
	}
	var allMembers []*github.User
	for {
		members, resp, err := conn.Client().Organizations.ListMembers(context.Background(), orgLogin, listOpts)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				return nil, nil
			}
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

		r, err := NewResource(g.MqlRuntime, "github.user", map[string]*llx.RawData{
			"id":    llx.IntData(convert.ToInt64(member.ID)),
			"login": llx.StringDataPtr(member.Login),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (g *mqlGithubOrganization) owners() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)

	if g.Login.Error != nil {
		return nil, g.Login.Error
	}
	orgLogin := g.Login.Data

	listOpts := &github.ListMembersOptions{
		ListOptions: github.ListOptions{PerPage: paginationPerPage},
		Role:        "admin",
	}
	var allMembers []*github.User
	for {
		members, resp, err := conn.Client().Organizations.ListMembers(context.Background(), orgLogin, listOpts)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				return nil, nil
			}
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

		var id int64
		if member.ID != nil {
			id = *member.ID
		}

		r, err := CreateResource(g.MqlRuntime, "github.user", map[string]*llx.RawData{
			"id":              llx.IntData(id),
			"login":           llx.StringDataPtr(member.Login),
			"name":            llx.StringDataPtr(member.Name),
			"email":           llx.StringDataPtr(member.Email),
			"blog":            llx.StringData(member.GetBlog()),
			"location":        llx.StringData(member.GetLocation()),
			"avatarUrl":       llx.StringData(member.GetAvatarURL()),
			"followers":       llx.IntData(int64(member.GetFollowers())),
			"following":       llx.IntData(int64(member.GetFollowing())),
			"twitterUsername": llx.StringData(member.GetTwitterUsername()),
			"bio":             llx.StringDataPtr(member.Bio),
			"createdAt":       llx.TimeDataPtr(githubTimestamp(member.CreatedAt)),
			"updatedAt":       llx.TimeDataPtr(githubTimestamp(member.UpdatedAt)),
			"suspendedAt":     llx.TimeDataPtr(githubTimestamp(member.SuspendedAt)),
			"company":         llx.StringDataPtr(member.Company),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (g *mqlGithubOrganization) teams() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)

	if g.Login.Error != nil {
		return nil, g.Login.Error
	}
	orgLogin := g.Login.Data

	listOpts := &github.ListOptions{
		PerPage: paginationPerPage,
	}
	var allTeams []*github.Team
	for {
		teams, resp, err := conn.Client().Teams.ListTeams(context.Background(), orgLogin, listOpts)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				return nil, nil
			}
			return nil, err
		}
		allTeams = append(allTeams, teams...)
		if resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}

	res := []interface{}{}
	for i := range allTeams {
		team := allTeams[i]
		r, err := CreateResource(g.MqlRuntime, "github.team", map[string]*llx.RawData{
			"id":                llx.IntDataPtr(team.ID),
			"name":              llx.StringDataPtr(team.Name),
			"description":       llx.StringDataPtr(team.Description),
			"slug":              llx.StringDataPtr(team.Slug),
			"privacy":           llx.StringDataPtr(team.Privacy),
			"defaultPermission": llx.StringDataPtr(team.Permission),
			"organization":      llx.ResourceData(g, g.MqlName()),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func (g *mqlGithubOrganization) repositories() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)

	if g.Login.Error != nil {
		return nil, g.Login.Error
	}
	orgLogin := g.Login.Data

	listOpts := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: paginationPerPage},
		Type:        "all",
	}

	var allRepos []*github.Repository
	for {
		repos, resp, err := conn.Client().Repositories.ListByOrg(context.Background(), orgLogin, listOpts)
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

func (g *mqlGithubOrganization) webhooks() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)

	if g.Login.Error != nil {
		return nil, g.Login.Error
	}
	ownerLogin := g.Login.Data

	listOpts := &github.ListOptions{
		PerPage: paginationPerPage,
	}
	var allHooks []*github.Hook
	for {
		hooks, resp, err := conn.Client().Organizations.ListHooks(context.TODO(), ownerLogin, listOpts)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				return nil, nil
			}
			return nil, err
		}
		allHooks = append(allHooks, hooks...)
		if resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}

	res := []interface{}{}
	for i := range allHooks {
		h := allHooks[i]
		config, err := convert.JsonToDict(h.Config)
		if err != nil {
			return nil, err
		}

		mqlUser, err := CreateResource(g.MqlRuntime, "github.webhook", map[string]*llx.RawData{
			"id":     llx.IntDataPtr(h.ID),
			"name":   llx.StringDataPtr(h.Name),
			"events": llx.ArrayData(convert.SliceAnyToInterface[string](h.Events), types.String),
			"config": llx.MapData(config, types.Any),
			"url":    llx.StringDataPtr(h.URL),
			"active": llx.BoolDataPtr(h.Active),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlUser)
	}

	return res, nil
}

type mqlGithubPackageInternal struct {
	packageRepository string
	parentResource    *mqlGithubOrganization
}

func (g *mqlGithubOrganization) packages() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)

	if g.Login.Error != nil {
		return nil, g.Login.Error
	}
	ownerLogin := g.Login.Data

	pkgTypes := []string{"npm", "maven", "rubygems", "docker", "nuget", "container"}
	res := []interface{}{}
	for i := range pkgTypes {
		listOpts := &github.PackageListOptions{
			ListOptions: github.ListOptions{PerPage: paginationPerPage},
			PackageType: github.String(pkgTypes[i]),
		}

		var allPackages []*github.Package
		for {
			packages, resp, err := conn.Client().Organizations.ListPackages(context.Background(), ownerLogin, listOpts)
			if err != nil {
				if strings.Contains(err.Error(), "404") {
					return nil, nil
				}
				return nil, err
			}
			allPackages = append(allPackages, packages...)
			if resp.NextPage == 0 {
				break
			}
			listOpts.Page = resp.NextPage
		}

		for i := range allPackages {
			p := allPackages[i]

			owner, err := NewResource(g.MqlRuntime, "github.user", map[string]*llx.RawData{
				"id":    llx.IntData(p.GetOwner().GetID()),
				"login": llx.StringData(p.GetOwner().GetLogin()),
			})
			if err != nil {
				return nil, err
			}

			mqlGhPackage, err := CreateResource(g.MqlRuntime, "github.package", map[string]*llx.RawData{
				"id":           llx.IntDataPtr(p.ID),
				"name":         llx.StringDataPtr(p.Name),
				"packageType":  llx.StringDataPtr(p.PackageType),
				"owner":        llx.ResourceData(owner, owner.MqlName()),
				"createdAt":    llx.TimeDataPtr(githubTimestamp(p.CreatedAt)),
				"updatedAt":    llx.TimeDataPtr(githubTimestamp(p.UpdatedAt)),
				"versionCount": llx.IntDataPtr(p.VersionCount),
				"visibility":   llx.StringDataPtr(p.Visibility),
			})
			if err != nil {
				return nil, err
			}
			pkg := mqlGhPackage.(*mqlGithubPackage)

			// NOTE: we need to fetch repo separately because the Github repo object is not complete, instead of
			// call the repo fetching all the time, we make this lazy loading
			if p.Repository != nil && p.Repository.Name != nil {
				pkg.packageRepository = convert.ToString(p.Repository.Name)
			}
			res = append(res, pkg)
		}
	}

	return res, nil
}

func (g *mqlGithubPackage) repository() (*mqlGithubRepository, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)

	if g.packageRepository == "" {
		return nil, errors.New("could not load the repository")
	}

	repoName := g.packageRepository

	if g.Owner.Error != nil {
		return nil, g.Owner.Error
	}
	owner := g.Owner.Data

	if owner.Login.Error != nil {
		return nil, owner.Login.Error
	}
	ownerLogin := owner.Login.Data

	repo, _, err := conn.Client().Repositories.Get(context.Background(), ownerLogin, repoName)
	if err != nil {
		return nil, err
	}
	return newMqlGithubRepository(g.MqlRuntime, repo)
}

func (g *mqlGithubOrganization) installations() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)

	if g.Login.Error != nil {
		return nil, g.Login.Error
	}
	orgLogin := g.Login.Data

	listOpts := &github.ListOptions{
		PerPage: paginationPerPage,
	}
	var allOrgInstallations []*github.Installation
	for {
		orgInstallations, resp, err := conn.Client().Organizations.ListInstallations(context.Background(), orgLogin, listOpts)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				return nil, nil
			}
			return nil, err
		}
		allOrgInstallations = append(allOrgInstallations, orgInstallations.Installations...)
		if resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}

	res := []interface{}{}
	for i := range allOrgInstallations {
		app := allOrgInstallations[i]

		var id int64
		if app.ID != nil {
			id = *app.ID
		}

		r, err := CreateResource(g.MqlRuntime, "github.installation", map[string]*llx.RawData{
			"id":        llx.IntData(id),
			"appId":     llx.IntDataPtr(app.AppID),
			"appSlug":   llx.StringDataPtr(app.AppSlug),
			"createdAt": llx.TimeDataPtr(githubTimestamp(app.CreatedAt)),
			"updatedAt": llx.TimeDataPtr(githubTimestamp(app.UpdatedAt)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}
