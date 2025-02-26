// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/go-github/v69/github"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/internal/workerpool"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/logger"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/github/connection"
	"go.mondoo.com/cnquery/v11/types"
)

type mqlGithubOrganizationInternal struct {
	repoCacheMap map[string]*mqlGithubRepository
}

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
	defer logger.FuncDur(time.Now(), "provider.github.initGithubOrganization")

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

	org, err := getOrg(conn.Context(), runtime, conn, name)
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
	args["followers"] = llx.IntDataDefault(org.Followers, 0)
	args["following"] = llx.IntDataDefault(org.Following, 0)
	args["description"] = llx.StringDataPtr(org.Description)
	args["createdAt"] = llx.TimeDataPtr(githubTimestamp(org.CreatedAt))
	args["updatedAt"] = llx.TimeDataPtr(githubTimestamp(org.UpdatedAt))
	args["totalPrivateRepos"] = llx.IntDataPtr(org.TotalPrivateRepos)
	args["totalPublicRepos"] = llx.IntDataPtr(org.PublicRepos)
	args["ownedPrivateRepos"] = llx.IntDataPtr(org.OwnedPrivateRepos)
	args["privateGists"] = llx.IntDataDefault(org.PrivateGists, 0)
	args["diskUsage"] = llx.IntDataDefault(org.DiskUsage, 0)
	args["collaborators"] = llx.IntDataDefault(org.Collaborators, 0)
	args["billingEmail"] = llx.StringDataPtr(org.BillingEmail)

	plan, _ := convert.JsonToDict(org.Plan)
	args["plan"] = llx.MapData(plan, types.Any)

	args["twoFactorRequirementEnabled"] = llx.BoolData(convert.ToBool(org.TwoFactorRequirementEnabled))
	args["isVerified"] = llx.BoolData(convert.ToBool(org.IsVerified))

	args["hasOrganizationProjects"] = llx.BoolData(convert.ToBool(org.HasOrganizationProjects))
	args["hasRepositoryProjects"] = llx.BoolData(convert.ToBool(org.HasRepositoryProjects))

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

func (g *mqlGithubOrganizationCustomProperty) id() (string, error) {
	return "github.organization.customProperty/" + g.Name.Data, nil
}

func (g *mqlGithubOrganization) customProperties() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)
	if g.Login.Error != nil {
		return nil, g.Login.Error
	}
	orgLogin := g.Login.Data

	// API doesn't have pagination:
	//
	// https://docs.github.com/en/rest/orgs/custom-properties?apiVersion=2022-11-28#get-all-custom-properties-for-an-organization--parameters
	customProperties, _, err := conn.Client().Organizations.GetAllCustomProperties(conn.Context(), orgLogin)
	if err != nil {
		return nil, err
	}

	resources := []interface{}{}
	for _, property := range customProperties {
		r, err := CreateResource(g.MqlRuntime, "github.organization.customProperty", map[string]*llx.RawData{
			"name":             llx.StringDataPtr(property.PropertyName),
			"description":      llx.StringDataPtr(property.Description),
			"sourceType":       llx.StringDataPtr(property.SourceType),
			"valueType":        llx.StringData(property.ValueType),
			"required":         llx.BoolDataPtr(property.Required),
			"defaultValue":     llx.StringDataPtr(property.DefaultValue),
			"allowedValues":    llx.ArrayData(convert.SliceAnyToInterface[string](property.AllowedValues), types.String),
			"valuesEditableBy": llx.StringDataPtr(property.ValuesEditableBy),
		})
		if err != nil {
			return nil, err
		}
		resources = append(resources, r)
	}

	return resources, nil
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
		members, resp, err := conn.Client().Organizations.ListMembers(conn.Context(), orgLogin, listOpts)
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
			"id":    llx.IntDataDefault(member.ID, 0),
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
		members, resp, err := conn.Client().Organizations.ListMembers(conn.Context(), orgLogin, listOpts)
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
		teams, resp, err := conn.Client().Teams.ListTeams(conn.Context(), orgLogin, listOpts)
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
	defer logger.FuncDur(time.Now(), "provider.github.repositories")

	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)

	if g.Login.Error != nil {
		return nil, g.Login.Error
	}
	orgLogin := g.Login.Data
	listOpts := github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{
			PerPage: paginationPerPage,
			Page:    1,
		},
		Type: "all",
	}

	repoCount := g.TotalPrivateRepos.Data + g.TotalPublicRepos.Data
	workerPool := workerpool.New[[]*github.Repository](workers)
	workerPool.Start()
	defer workerPool.Close()

	log.Debug().
		Int("workers", workers).
		Int64("total_repos", repoCount).
		Str("organization", g.Name.Data).
		Msg("list repositories")

	for {
		// exit as soon as we collect all repositories
		reposLen := len(slices.Concat(workerPool.GetValues()...))
		if reposLen >= int(repoCount) {
			break
		}

		// send requests to workers
		opts := listOpts
		workerPool.Submit(func() ([]*github.Repository, error) {
			repos, _, err := conn.Client().Repositories.ListByOrg(conn.Context(), orgLogin, &opts)
			return repos, err
		})

		// next page
		listOpts.Page++

		// check if any request failed
		if errs := workerPool.GetErrors(); len(errs) != 0 {
			if err := errors.Join(errs...); err != nil {
				if strings.Contains(err.Error(), "404") {
					return nil, nil
				}
				return nil, err
			}
		}
	}

	if g.repoCacheMap == nil {
		g.repoCacheMap = make(map[string]*mqlGithubRepository)
	}

	res := []interface{}{}
	for _, repos := range workerPool.GetValues() {
		for i := range repos {
			repo := repos[i]

			r, err := newMqlGithubRepository(g.MqlRuntime, repo)
			if err != nil {
				return nil, err
			}
			res = append(res, r)
			g.repoCacheMap[repo.GetName()] = r
		}
	}

	return res, nil
}

func (g *mqlGithubOrganization) webhooks() ([]interface{}, error) {
	defer logger.FuncDur(time.Now(), "provider.github.webhooks")

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
		hooks, resp, err := conn.Client().Organizations.ListHooks(conn.Context(), ownerLogin, listOpts)
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
			packages, resp, err := conn.Client().Organizations.ListPackages(conn.Context(), ownerLogin, listOpts)
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

	repo, _, err := conn.Client().Repositories.Get(conn.Context(), ownerLogin, repoName)
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
		orgInstallations, resp, err := conn.Client().Organizations.ListInstallations(conn.Context(), orgLogin, listOpts)
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
