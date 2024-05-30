// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strings"

	"github.com/gobwas/glob"
	"github.com/google/go-github/v61/github"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/v11/providers/github/connection"
	"go.mondoo.com/cnquery/v11/utils/stringx"
	"google.golang.org/protobuf/proto"
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
		return []string{
			connection.DiscoveryRepos,
			connection.DiscoveryUsers,
			connection.DiscoveryTerraform,
			connection.DiscoveryK8sManifests,
		}
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
	reposFilter := NewReposFilter(conf)
	assetList := []*inventory.Asset{}
	org, err := getMqlGithubOrg(runtime, orgName)
	if err != nil {
		return nil, err
	}
	assetList = append(assetList, &inventory.Asset{
		PlatformIds: []string{connection.NewGithubOrgIdentifier(org.Login.Data)},
		Name:        org.Name.Data,
		Platform:    connection.NewGithubOrgPlatform(org.Login.Data),
		Labels:      map[string]string{},
		Connections: []*inventory.Config{conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.ID()))},
	})
	if stringx.ContainsAnyOf(targets, connection.DiscoveryRepos, connection.DiscoveryRepository, connection.DiscoveryAll, connection.DiscoveryAuto) {
		for i := range org.GetRepositories().Data {
			repo := org.GetRepositories().Data[i].(*mqlGithubRepository)
			if reposFilter.skipRepo(repo.Name.Data) {
				continue
			}
			cfg := conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.ID()))
			cfg.Options["repository"] = repo.Name.Data
			assetList = append(assetList, &inventory.Asset{
				PlatformIds: []string{connection.NewGitHubRepoIdentifier(org.Login.Data, repo.Name.Data)},
				Name:        org.Login.Data + "/" + repo.Name.Data,
				Platform:    connection.NewGitHubRepoPlatform(org.Login.Data, repo.Name.Data),
				Labels:      make(map[string]string),
				Connections: []*inventory.Config{cfg},
			})

			if stringx.ContainsAnyOf(targets, connection.DiscoveryAll, connection.DiscoveryTerraform) {
				terraformAssets, err := discoverTerraform(conn, repo)
				if err != nil {
					return nil, err
				}
				assetList = append(assetList, terraformAssets...)
			}
			if stringx.ContainsAnyOf(targets, connection.DiscoveryAll, connection.DiscoveryK8sManifests) {
				k8sAssets, err := discoverK8sManifests(conn, repo)
				if err != nil {
					return nil, err
				}
				assetList = append(assetList, k8sAssets...)
			}
		}
	}
	if stringx.ContainsAnyOf(targets, connection.DiscoveryUsers, connection.DiscoveryUser) {
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
				Platform:    connection.NewGithubUserPlatform(user.Login.Data),
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
		Platform:    connection.NewGitHubRepoPlatform(owner, repo.Name.Data),
		Labels:      make(map[string]string),
		Connections: []*inventory.Config{cfg},
	})

	if stringx.ContainsAnyOf(targets, connection.DiscoveryAll, connection.DiscoveryTerraform) {
		terraformAssets, err := discoverTerraform(conn, repo)
		if err != nil {
			return nil, err
		}
		assetList = append(assetList, terraformAssets...)
	}
	if stringx.ContainsAnyOf(targets, connection.DiscoveryAll, connection.DiscoveryK8sManifests) {
		k8sAssets, err := discoverK8sManifests(conn, repo)
		if err != nil {
			return nil, err
		}
		assetList = append(assetList, k8sAssets...)
	}

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
		Platform:    connection.NewGithubUserPlatform(user.Login.Data),
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

type ReposFilter struct {
	include []string
	exclude []string
}

func NewReposFilter(cfg *inventory.Config) ReposFilter {
	nsFilter := ReposFilter{}
	if include, ok := cfg.Options[connection.OPTION_REPOS]; ok && len(include) > 0 {
		nsFilter.include = strings.Split(include, ",")
	}

	if exclude, ok := cfg.Options[connection.OPTION_REPOS_EXCLUDE]; ok && len(exclude) > 0 {
		nsFilter.exclude = strings.Split(exclude, ",")
	}
	return nsFilter
}

func (f *ReposFilter) skipRepo(namespace string) bool {
	// anything explicitly specified in the list of includes means accept only from that list
	if len(f.include) > 0 {
		for _, ns := range f.include {
			g, err := glob.Compile(ns)
			if err != nil {
				log.Error().Err(err).Msg("failed to compile repos glob")
				return false
			}
			if g.Match(namespace) {
				// stop looking, we found our match
				return false
			}
		}

		// didn't find it, so it must be skipped
		return true
	}

	// if nothing explicitly meant to be included, then check whether
	// it should be excluded
	for _, ns := range f.exclude {
		g, err := glob.Compile(ns)
		if err != nil {
			log.Error().Err(err).Msg("failed to compile repos exclude glob")
			return false
		}
		if g.Match(namespace) {
			return true
		}
	}

	return false
}

func discoverTerraform(conn *connection.GithubConnection, repo *mqlGithubRepository) ([]*inventory.Asset, error) {
	// For git clone we need to set the user to oauth2 to be usable with the token.
	conf := conn.Asset().Connections[0]
	creds := make([]*vault.Credential, len(conf.Credentials))
	for i := range conf.Credentials {
		cred := conf.Credentials[i]
		cc := proto.Clone(cred).(*vault.Credential)
		if cc.User == "" {
			cc.User = "oauth2"
		}
		creds[i] = cc
	}

	var res []*inventory.Asset
	hasTf, err := hasTerraformHcl(conn.Client(), repo)
	if err != nil {
		log.Error().Err(err).Str("project", repo.FullName.Data).Msg("failed to discover terraform repo")
	} else if hasTf {
		res = append(res, &inventory.Asset{
			Connections: []*inventory.Config{{
				Type: "terraform-hcl-git",
				Options: map[string]string{
					"ssh-url":  repo.SshUrl.Data,
					"http-url": repo.CloneUrl.Data,
				},
				Credentials: creds,
			}},
		})
	}
	return res, nil
}

// hasTerraformHcl will check if the repository contains terraform files
func hasTerraformHcl(client *github.Client, repo *mqlGithubRepository) (bool, error) {
	query := "repo:" + repo.FullName.Data + " extension:tf"
	res, _, err := client.Search.Code(context.Background(), query, &github.SearchOptions{})
	if err != nil {
		return false, err
	}

	// Ignore tf files that are hidden or are in a hidden folder
	nonHiddenTf := 0
	for _, code := range res.CodeResults {
		fragments := strings.Split(code.GetPath(), "/")
		// skip hidden files
		isHidden := false
		for _, fragment := range fragments {
			if strings.HasPrefix(fragment, ".") {
				isHidden = true
				break
			}
		}

		if !isHidden {
			nonHiddenTf++
		}
	}

	return nonHiddenTf > 0, nil
}

func discoverK8sManifests(conn *connection.GithubConnection, repo *mqlGithubRepository) ([]*inventory.Asset, error) {
	// For git clone we need to set the user to oauth2 to be usable with the token.
	conf := conn.Asset().Connections[0]
	creds := make([]*vault.Credential, len(conf.Credentials))
	for i := range conf.Credentials {
		cred := conf.Credentials[i]
		cc := proto.Clone(cred).(*vault.Credential)
		if cc.User == "" {
			cc.User = "oauth2"
		}
		creds[i] = cc
	}

	var res []*inventory.Asset
	hasTf, err := hasYaml(conn.Client(), repo)
	if err != nil {
		log.Error().Err(err).Str("project", repo.FullName.Data).Msg("failed to discover k8s manifests repo")
	} else if hasTf {
		res = append(res, &inventory.Asset{
			Connections: []*inventory.Config{{
				Type: "k8s",
				Options: map[string]string{
					"ssh-url":  repo.SshUrl.Data,
					"http-url": repo.CloneUrl.Data,
				},
				Credentials: creds,
				Discover:    &inventory.Discovery{Targets: []string{"auto"}},
			}},
		})
	}
	return res, nil
}

// hasYaml will check if the repository contains YAML files
func hasYaml(client *github.Client, repo *mqlGithubRepository) (bool, error) {
	query := "repo:" + repo.FullName.Data + " extension:yaml OR extension:yml"
	res, _, err := client.Search.Code(context.Background(), query, &github.SearchOptions{})
	if err != nil {
		return false, err
	}

	// Ignore YAML files that are hidden or are in a hidden folder
	nonHiddenYaml := 0
	for _, code := range res.CodeResults {
		path := code.GetPath()

		// Skip MQL files
		if strings.HasSuffix(path, "mql.yaml") || strings.HasSuffix(path, "mql.yml") {
			continue
		}

		fragments := strings.Split(code.GetPath(), "/")
		// skip hidden files
		isHidden := false
		for _, fragment := range fragments {
			if strings.HasPrefix(fragment, ".") {
				isHidden = true
				break
			}
		}

		if !isHidden {
			nonHiddenYaml++
		}
	}
	return nonHiddenYaml > 0, nil
}
