// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/github/connection"
	"go.mondoo.com/cnquery/v10/utils/stringx"
)

func Discover(runtime *plugin.Runtime, opts map[string]string) (*inventory.Inventory, error) {
	conn := runtime.Connection.(*connection.GithubConnection)

	in := &inventory.Inventory{Spec: &inventory.InventorySpec{
		Assets: []*inventory.Asset{},
	}}

	targets := handleTargets(conn.Asset().Connections[0].Discover.Targets)
	list, err := discover(runtime, targets)
	if err != nil {
		return in, err
	}

	in.Spec.Assets = list
	return in, nil
}

func handleTargets(targets []string) []string {
	if stringx.Contains(targets, connection.DiscoveryAll) {
		return []string{connection.DiscoveryRepos, connection.DiscoveryUsers}
	}
	return targets
}

func discover(runtime *plugin.Runtime, targets []string) ([]*inventory.Asset, error) {
	conn := runtime.Connection.(*connection.GithubConnection)
	conf := conn.Asset().Connections[0]
	assetList := []*inventory.Asset{}
	if orgName := conf.Options["organization"]; orgName != "" {
		orgAssets, err := org(runtime, orgName, conn, targets)
		if err != nil {
			return nil, err
		}
		assetList = append(assetList, orgAssets...)
	}

	repoName := conf.Options["repository"]
	var owner string
	repoId := conf.Options["repository"]
	if repoId != "" {
		owner = conf.Options["owner"]
		if owner == "" {
			owner = conf.Options["organization"]
		}
		if owner == "" {
			owner = conf.Options["user"]
		}
	}
	if repoName != "" && owner != "" {
		repoAssets, err := repo(runtime, repoName, owner, conn, targets)
		if err != nil {
			return nil, err
		}
		assetList = append(assetList, repoAssets...)
	}

	userId := conf.Options["user"]
	if userId == "" {
		userId = conf.Options["owner"]
	}
	if conf.Options["user"] != "" {
		userAssets, err := user(runtime, userId, conn)
		if err != nil {
			return nil, err
		}
		assetList = append(assetList, userAssets...)
	}

	return assetList, nil
}

func org(runtime *plugin.Runtime, orgName string, conn *connection.GithubConnection, targets []string) ([]*inventory.Asset, error) {
	conf := conn.Asset().Connections[0]
	assetList := []*inventory.Asset{}
	org, err := getMqlGithubOrg(runtime, orgName)
	if err != nil {
		return nil, err
	}
	assetList = append(assetList, &inventory.Asset{
		PlatformIds: []string{connection.NewGithubOrgIdentifier(org.Login.Data)},
		Name:        org.Name.Data,
		Platform:    connection.GithubOrgPlatform,
		Labels:      map[string]string{},
		Connections: []*inventory.Config{conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.ID()))},
	})
	if stringx.Contains(targets, connection.DiscoveryRepos) || stringx.Contains(targets, connection.DiscoveryRepository) || stringx.Contains(targets, connection.DiscoveryAll) || stringx.Contains(targets, connection.DiscoveryAuto) {
		if stringx.Contains(targets, connection.DiscoveryRepos) || stringx.Contains(targets, connection.DiscoveryRepository) {
			assetList = []*inventory.Asset{}
		}
		for i := range org.GetRepositories().Data {
			repo := org.GetRepositories().Data[i].(*mqlGithubRepository)
			cfg := conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.ID()))
			cfg.Options["repository"] = repo.Name.Data
			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{connection.NewGitHubRepoIdentifier(org.Login.Data, repo.Name.Data)},
				Name:        org.Login.Data + "/" + repo.Name.Data,
				Platform:    connection.GithubRepoPlatform,
				Labels:      make(map[string]string),
				Connections: []*inventory.Config{cfg},
			})
		}
	}
	if stringx.Contains(targets, connection.DiscoveryUsers) || stringx.Contains(targets, connection.DiscoveryUser) {
		assetList = []*inventory.Asset{}
		for i := range org.GetMembers().Data {
			user := org.GetMembers().Data[i].(*mqlGithubUser)
			if user.Name.Data == "" {
				continue
			}
			cfg := conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.ID()))
			cfg.Options["user"] = user.Login.Data
			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{connection.NewGithubUserIdentifier(user.Login.Data)},
				Name:        user.Name.Data,
				Platform:    connection.GithubUserPlatform,
				Labels:      make(map[string]string),
				Connections: []*inventory.Config{cfg},
			})
		}
	}
	return assetList, nil
}

func getMqlGithubOrg(runtime *plugin.Runtime, orgName string) (*mqlGithubOrganization, error) {
	res, err := NewResource(runtime, "github.organization", map[string]*llx.RawData{"name": llx.StringData(orgName)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGithubOrganization), nil
}

func repo(runtime *plugin.Runtime, repoName string, owner string, conn *connection.GithubConnection, targets []string) ([]*inventory.Asset, error) {
	conf := conn.Asset().Connections[0]
	assetList := []*inventory.Asset{}

	repo, err := getMqlGithubRepo(runtime, repoName)
	if err != nil {
		return nil, err
	}
	cfg := conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.ID()))
	cfg.Options["repository"] = repo.Name.Data
	assetList = append(assetList, &inventory.Asset{
		PlatformIds: []string{connection.NewGitHubRepoIdentifier(owner, repo.Name.Data)},
		Name:        owner + "/" + repo.Name.Data,
		Platform:    connection.GithubRepoPlatform,
		Labels:      make(map[string]string),
		Connections: []*inventory.Config{cfg},
	})

	return assetList, nil
}

func getMqlGithubRepo(runtime *plugin.Runtime, repoName string) (*mqlGithubRepository, error) {
	res, err := NewResource(runtime, "github.repository", map[string]*llx.RawData{"name": llx.StringData(repoName)})
	if err != nil {
		return nil, err
	}

	return res.(*mqlGithubRepository), nil
}

func user(runtime *plugin.Runtime, userName string, conn *connection.GithubConnection) ([]*inventory.Asset, error) {
	conf := conn.Asset().Connections[0]
	assetList := []*inventory.Asset{}

	user, err := getMqlGithubUser(runtime, userName)
	if err != nil {
		return nil, err
	}
	cfg := conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.ID()))
	cfg.Options["user"] = user.Login.Data
	assetList = append(assetList, &inventory.Asset{
		PlatformIds: []string{connection.NewGithubUserIdentifier(user.Login.Data)},
		Name:        user.Name.Data,
		Platform:    connection.GithubUserPlatform,
		Labels:      make(map[string]string),
		Connections: []*inventory.Config{cfg},
	})
	return assetList, nil
}

func getMqlGithubUser(runtime *plugin.Runtime, userName string) (*mqlGithubUser, error) {
	res, err := NewResource(runtime, "github.user", map[string]*llx.RawData{"name": llx.StringData(userName)})
	if err != nil {
		return nil, err
	}

	return res.(*mqlGithubUser), nil
}
