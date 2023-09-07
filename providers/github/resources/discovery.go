// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/github/connection"
	"go.mondoo.com/cnquery/utils/stringx"
)

func Discover(runtime *plugin.Runtime, opts map[string]string) (*inventory.Inventory, error) {
	conn := runtime.Connection.(*connection.GithubConnection)

	in := &inventory.Inventory{Spec: &inventory.InventorySpec{
		Assets: []*inventory.Asset{},
	}}

	targets := handleTargets(conn.Conf.Discover.Targets)
	for i := range targets {
		target := targets[i]
		list, err := discover(runtime, target)
		if err != nil {
			log.Error().Err(err).Msg("error during discovery")
			continue
		}
		in.Spec.Assets = append(in.Spec.Assets, list...)
	}

	return in, nil
}

func handleTargets(targets []string) []string {
	if stringx.Contains(targets, connection.DiscoveryAll) {
		return connection.All
	}
	return targets
}

func discover(runtime *plugin.Runtime, target string) ([]*inventory.Asset, error) {
	conn := runtime.Connection.(*connection.GithubConnection)
	assetList := []*inventory.Asset{}
	switch target {
	case connection.DiscoveryOrganization:
		orgName := conn.Conf.Options["organization"]
		orgAssets, err := org(runtime, orgName, conn)
		if err != nil {
			return nil, err
		}
		assetList = append(assetList, orgAssets...)

	case connection.DiscoveryRepository:
		repoName := conn.Conf.Options["repository"]
		var owner string
		repoId := conn.Conf.Options["repository"]
		if repoId != "" {
			owner = conn.Conf.Options["owner"]
			if owner == "" {
				owner = conn.Conf.Options["organization"]
			}
			if owner == "" {
				owner = conn.Conf.Options["user"]
			}
		}
		repoAssets, err := repo(runtime, repoName, owner, conn)
		if err != nil {
			return nil, err
		}
		assetList = append(assetList, repoAssets...)

	case connection.DiscoveryUser:
		userId := conn.Conf.Options["user"]
		if userId == "" {
			userId = conn.Conf.Options["owner"]
		}
		userAssets, err := user(runtime, userId, conn)
		if err != nil {
			return nil, err
		}
		assetList = append(assetList, userAssets...)
	}
	return assetList, nil
}

func cloneInventoryConf(invConf *inventory.Config) *inventory.Config {
	invConfClone := invConf.Clone()
	// We do not want to run discovery again for the already discovered assets
	invConfClone.Discover = &inventory.Discovery{}
	return invConfClone
}

func org(runtime *plugin.Runtime, orgName string, conn *connection.GithubConnection) ([]*inventory.Asset, error) {
	assetList := []*inventory.Asset{}
	org, err := getMqlGithubOrg(runtime, orgName)
	if err != nil {
		return nil, err
	}
	assetList = append(assetList, &inventory.Asset{
		PlatformIds: []string{connection.NewGithubOrgIdentifier(org.Name.Data)},
		Name:        org.Name.Data,
		Platform:    connection.GithubOrgPlatform,
		Labels:      map[string]string{},
		Connections: []*inventory.Config{cloneInventoryConf(conn.Conf)},
	})
	for i := range org.GetRepositories().Data {
		repo := org.GetRepositories().Data[i].(*mqlGithubRepository)
		assetList = append(assetList, &inventory.Asset{
			PlatformIds: []string{connection.NewGitHubRepoIdentifier(org.Name.Data, repo.Name.Data)},
			Name:        org.Name.Data + "/" + repo.Name.Data,
			Platform:    connection.GithubRepoPlatform,
			Labels:      make(map[string]string),
			Connections: []*inventory.Config{cloneInventoryConf(conn.Conf)},
		})
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

func repo(runtime *plugin.Runtime, repoName string, owner string, conn *connection.GithubConnection) ([]*inventory.Asset, error) {
	assetList := []*inventory.Asset{}

	repo, err := getMqlGithubRepo(runtime, repoName)
	if err != nil {
		return nil, err
	}

	assetList = append(assetList, &inventory.Asset{
		PlatformIds: []string{connection.NewGitHubRepoIdentifier(owner, repo.Name.Data)},
		Name:        owner + "/" + repo.Name.Data,
		Platform:    connection.GithubRepoPlatform,
		Labels:      make(map[string]string),
		Connections: []*inventory.Config{cloneInventoryConf(conn.Conf)},
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
	assetList := []*inventory.Asset{}

	user, err := getMqlGithubUser(runtime, userName)
	if err != nil {
		return nil, err
	}
	assetList = append(assetList, &inventory.Asset{
		PlatformIds: []string{connection.NewGithubUserIdentifier(user.Name.Data)},
		Name:        user.Name.Data,
		Platform:    connection.GithubUserPlatform,
		Labels:      make(map[string]string),
		Connections: []*inventory.Config{cloneInventoryConf(conn.Conf)},
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
