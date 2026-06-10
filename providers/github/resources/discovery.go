// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/gobwas/glob"
	"github.com/google/go-github/v88/github"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/logger"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
	"go.mondoo.com/mql/v13/providers/github/connection"
	"go.mondoo.com/mql/v13/utils/stringx"
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
			connection.DiscoveryCloudformation,
			connection.DiscoveryDockerfiles,
			connection.DiscoveryBicep,
			connection.DiscoveryHelm,
			connection.DiscoveryKustomize,
		}
	}
	return targets
}

func discover(runtime *plugin.Runtime, targets []string) ([]*inventory.Asset, error) {
	defer logger.FuncDur(time.Now(), "provider.github.discover")

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

	// only scan the org if the discover flag is provided, this allows you to scan all repos in an org with simply using
	// --discover repos. If users provide a repo filter, we also want to skip org scan.
	if stringx.ContainsAnyOf(targets, connection.DiscoveryOrganization, connection.DiscoveryAll, connection.DiscoveryAuto) && reposFilter.empty() {
		labels := map[string]string{}
		for j := range org.GetCustomProperties().Data {
			customProperty := org.GetCustomProperties().Data[j].(*mqlGithubOrganizationCustomProperty)
			value := ""
			if customProperty.DefaultValue.IsSet() {
				// if the default value of the org-level custom property is set, use it as the label value
				value = customProperty.DefaultValue.Data
			}
			labels[customProperty.Name.Data] = value
		}
		assetList = append(assetList, &inventory.Asset{
			PlatformIds: []string{connection.NewGithubOrgIdentifier(org.Login.Data)},
			Name:        org.Name.Data,
			Platform:    connection.NewGithubOrgPlatform(org.Login.Data),
			Labels:      labels,
			Connections: []*inventory.Config{conf.Clone(inventory.WithoutDiscovery(), inventory.WithParentConnectionId(conn.ID()))},
		})
	}

	if stringx.ContainsAnyOf(targets, connection.DiscoveryRepos, connection.DiscoveryAll, connection.DiscoveryAuto) {
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
				Labels:      convert.DictToTypedMap[string](repo.CustomProperties.Data),
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
			if stringx.ContainsAnyOf(targets, connection.DiscoveryAll, connection.DiscoveryCloudformation) {
				cfAssets, err := discoverCloudformation(conn, repo)
				if err != nil {
					return nil, err
				}
				assetList = append(assetList, cfAssets...)
			}
			if stringx.ContainsAnyOf(targets, connection.DiscoveryAll, connection.DiscoveryDockerfiles) {
				dockerfileAssets, err := discoverDockerfiles(conn, repo)
				if err != nil {
					return nil, err
				}
				assetList = append(assetList, dockerfileAssets...)
			}
			if stringx.ContainsAnyOf(targets, connection.DiscoveryAll, connection.DiscoveryBicep) {
				bicepAssets, err := discoverBicep(conn, repo)
				if err != nil {
					return nil, err
				}
				assetList = append(assetList, bicepAssets...)
			}
			if stringx.ContainsAnyOf(targets, connection.DiscoveryAll, connection.DiscoveryHelm) {
				helmAssets, err := discoverHelm(conn, repo)
				if err != nil {
					return nil, err
				}
				assetList = append(assetList, helmAssets...)
			}
			if stringx.ContainsAnyOf(targets, connection.DiscoveryAll, connection.DiscoveryKustomize) {
				kustomizeAssets, err := discoverKustomize(conn, repo)
				if err != nil {
					return nil, err
				}
				assetList = append(assetList, kustomizeAssets...)
			}
		}
	}
	if stringx.ContainsAnyOf(targets, connection.DiscoveryUsers) {
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
				Labels:      map[string]string{},
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
		Labels:      convert.DictToTypedMap[string](repo.CustomProperties.Data),
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
	if stringx.ContainsAnyOf(targets, connection.DiscoveryAll, connection.DiscoveryCloudformation) {
		cfAssets, err := discoverCloudformation(conn, repo)
		if err != nil {
			return nil, err
		}
		assetList = append(assetList, cfAssets...)
	}
	if stringx.ContainsAnyOf(targets, connection.DiscoveryAll, connection.DiscoveryDockerfiles) {
		dockerfileAssets, err := discoverDockerfiles(conn, repo)
		if err != nil {
			return nil, err
		}
		assetList = append(assetList, dockerfileAssets...)
	}
	if stringx.ContainsAnyOf(targets, connection.DiscoveryAll, connection.DiscoveryBicep) {
		bicepAssets, err := discoverBicep(conn, repo)
		if err != nil {
			return nil, err
		}
		assetList = append(assetList, bicepAssets...)
	}
	if stringx.ContainsAnyOf(targets, connection.DiscoveryAll, connection.DiscoveryHelm) {
		helmAssets, err := discoverHelm(conn, repo)
		if err != nil {
			return nil, err
		}
		assetList = append(assetList, helmAssets...)
	}
	if stringx.ContainsAnyOf(targets, connection.DiscoveryAll, connection.DiscoveryKustomize) {
		kustomizeAssets, err := discoverKustomize(conn, repo)
		if err != nil {
			return nil, err
		}
		assetList = append(assetList, kustomizeAssets...)
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
	res, err := NewResource(runtime, "github.user", map[string]*llx.RawData{"login": llx.StringData(userName)})
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

func (f *ReposFilter) empty() bool {
	return (len(f.exclude) + len(f.include)) == 0
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

// gitCredentials clones the parent GitHub connection credentials for use in a
// git clone, defaulting the user to "oauth2" so the token works over HTTPS.
func gitCredentials(conf *inventory.Config) []*vault.Credential {
	creds := make([]*vault.Credential, len(conf.Credentials))
	for i := range conf.Credentials {
		cc := conf.Credentials[i].CloneVT()
		if cc.User == "" {
			cc.User = "oauth2"
		}
		creds[i] = cc
	}
	return creds
}

// isHiddenPath reports whether any path segment is hidden (starts with a dot),
// e.g. files under .github/ or a top-level .drone.yml.
func isHiddenPath(p string) bool {
	for _, fragment := range strings.Split(p, "/") {
		if strings.HasPrefix(fragment, ".") {
			return true
		}
	}
	return false
}

// searchCode runs a GitHub code search and returns every matching result,
// following pagination. The code search API returns at most 100 items per page
// (30 by default), so a repo with many matching files would otherwise be
// silently truncated to the first page.
func searchCode(ctx context.Context, client *github.Client, query string) ([]*github.CodeResult, error) {
	opts := &github.SearchOptions{ListOptions: github.ListOptions{PerPage: 100}}
	var results []*github.CodeResult
	for {
		res, resp, err := client.Search.Code(ctx, query, opts)
		if err != nil {
			return nil, err
		}
		results = append(results, res.CodeResults...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return results, nil
}

// searchCodeExists reports whether any non-hidden file matches the query. It
// follows pagination but returns as soon as the first match is found, so an
// existence check on a repo with many matching files does not pull every page.
func searchCodeExists(ctx context.Context, client *github.Client, query string) (bool, error) {
	opts := &github.SearchOptions{ListOptions: github.ListOptions{PerPage: 100}}
	for {
		res, resp, err := client.Search.Code(ctx, query, opts)
		if err != nil {
			return false, err
		}
		for _, code := range res.CodeResults {
			if !isHiddenPath(code.GetPath()) {
				return true, nil
			}
		}
		if resp.NextPage == 0 {
			return false, nil
		}
		opts.Page = resp.NextPage
	}
}

func discoverTerraform(conn *connection.GithubConnection, repo *mqlGithubRepository) ([]*inventory.Asset, error) {
	creds := gitCredentials(conn.Asset().Connections[0])

	var res []*inventory.Asset
	hasTf, err := hasTerraformHcl(conn.Context(), conn.Client(), repo)
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
func hasTerraformHcl(ctx context.Context, client *github.Client, repo *mqlGithubRepository) (bool, error) {
	languages, _, err := client.Repositories.ListLanguages(ctx, repo.Owner.Data.Login.Data, repo.Name.Data)
	if err != nil {
		return false, err
	}
	if languages["HCL"] > 0 {
		return true, nil
	}
	return false, nil
}

func discoverK8sManifests(conn *connection.GithubConnection, repo *mqlGithubRepository) ([]*inventory.Asset, error) {
	creds := gitCredentials(conn.Asset().Connections[0])

	var res []*inventory.Asset
	hasTf, err := hasYaml(conn.Context(), conn.Client(), repo)
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
func hasYaml(ctx context.Context, client *github.Client, repo *mqlGithubRepository) (bool, error) {
	query := "repo:" + repo.FullName.Data + " extension:yaml OR extension:yml"
	res, _, err := client.Search.Code(ctx, query, &github.SearchOptions{})
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

// discoverCloudformation emits one asset per CloudFormation template found in
// the repository. The CloudFormation connection scans a single template file
// (not a directory), so each asset carries its own repo-relative path and
// performs its own shallow clone of the repo on connect. For a repo with many
// templates this means several clones of the same repo — an accepted trade-off
// that keeps the connection's single-template model and avoids a shared clone
// cache.
func discoverCloudformation(conn *connection.GithubConnection, repo *mqlGithubRepository) ([]*inventory.Asset, error) {
	creds := gitCredentials(conn.Asset().Connections[0])

	paths, err := cloudformationTemplatePaths(conn.Context(), conn.Client(), repo)
	if err != nil {
		log.Error().Err(err).Str("project", repo.FullName.Data).Msg("failed to discover cloudformation repo")
		return nil, nil
	}

	var res []*inventory.Asset
	for _, path := range paths {
		res = append(res, &inventory.Asset{
			Connections: []*inventory.Config{{
				Type: "cloudformation",
				Options: map[string]string{
					"ssh-url":  repo.SshUrl.Data,
					"http-url": repo.CloneUrl.Data,
					"path":     path,
				},
				Credentials: creds,
			}},
		})
	}
	return res, nil
}

// cloudformationTemplatePaths searches the repository for CloudFormation/SAM
// templates. Because CloudFormation templates share the .yaml/.json extensions
// with many other config files, we match on the template marker
// `AWSTemplateFormatVersion` rather than the extension alone.
func cloudformationTemplatePaths(ctx context.Context, client *github.Client, repo *mqlGithubRepository) ([]string, error) {
	query := "repo:" + repo.FullName.Data + " AWSTemplateFormatVersion"
	results, err := searchCode(ctx, client, query)
	if err != nil {
		return nil, err
	}

	var paths []string
	seen := map[string]bool{}
	for _, code := range results {
		path := code.GetPath()
		if isHiddenPath(path) || seen[path] {
			continue
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".yaml", ".yml", ".json", ".template":
			seen[path] = true
			paths = append(paths, path)
		}
	}
	return paths, nil
}

// discoverDockerfiles emits one asset per Dockerfile found in the repository.
// Like CloudFormation, each Dockerfile asset clones the repo independently on
// connect (the dockerfile connection targets a single file), so a repo with
// many Dockerfiles results in several clones — an accepted trade-off.
func discoverDockerfiles(conn *connection.GithubConnection, repo *mqlGithubRepository) ([]*inventory.Asset, error) {
	creds := gitCredentials(conn.Asset().Connections[0])

	paths, err := dockerfilePaths(conn.Context(), conn.Client(), repo)
	if err != nil {
		log.Error().Err(err).Str("project", repo.FullName.Data).Msg("failed to discover dockerfiles repo")
		return nil, nil
	}

	var res []*inventory.Asset
	for _, path := range paths {
		res = append(res, &inventory.Asset{
			Connections: []*inventory.Config{{
				Type: "docker-file",
				Options: map[string]string{
					"ssh-url":  repo.SshUrl.Data,
					"http-url": repo.CloneUrl.Data,
					"path":     path,
				},
				Credentials: creds,
			}},
		})
	}
	return res, nil
}

// dockerfilePaths searches the repository for files named `Dockerfile`.
func dockerfilePaths(ctx context.Context, client *github.Client, repo *mqlGithubRepository) ([]string, error) {
	query := "repo:" + repo.FullName.Data + " filename:Dockerfile"
	results, err := searchCode(ctx, client, query)
	if err != nil {
		return nil, err
	}

	var paths []string
	seen := map[string]bool{}
	for _, code := range results {
		path := code.GetPath()
		if isHiddenPath(path) || seen[path] || !isDockerfile(filepath.Base(path)) {
			continue
		}
		seen[path] = true
		paths = append(paths, path)
	}
	return paths, nil
}

// isDockerfile reports whether a base file name follows a Dockerfile naming
// convention: `Dockerfile`, `Dockerfile.<suffix>` (e.g. Dockerfile.prod), or
// `<prefix>.Dockerfile`/`<prefix>.dockerfile` (e.g. app.Dockerfile). The GitHub
// `filename:Dockerfile` qualifier is a prefix match, so it can also return
// unrelated files like `DockerfileLint.md` that this filter rejects.
func isDockerfile(base string) bool {
	return base == "Dockerfile" ||
		strings.HasPrefix(base, "Dockerfile.") ||
		strings.HasSuffix(base, ".Dockerfile") ||
		strings.HasSuffix(base, ".dockerfile")
}

func discoverBicep(conn *connection.GithubConnection, repo *mqlGithubRepository) ([]*inventory.Asset, error) {
	creds := gitCredentials(conn.Asset().Connections[0])

	has, err := hasBicep(conn.Context(), conn.Client(), repo)
	if err != nil {
		log.Error().Err(err).Str("project", repo.FullName.Data).Msg("failed to discover bicep repo")
		return nil, nil
	}

	var res []*inventory.Asset
	if has {
		res = append(res, &inventory.Asset{
			Connections: []*inventory.Config{{
				Type: "bicep",
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

// hasBicep will check if the repository contains Bicep files
func hasBicep(ctx context.Context, client *github.Client, repo *mqlGithubRepository) (bool, error) {
	query := "repo:" + repo.FullName.Data + " extension:bicep"
	return searchCodeExists(ctx, client, query)
}

// iacDir returns the repo-relative directory containing the given file, with a
// top-level file ("." from filepath.Dir) normalized to "" so it joins onto the
// clone root cleanly.
func iacDir(path string) string {
	if dir := filepath.Dir(path); dir != "." {
		return dir
	}
	return ""
}

// discoverHelm emits one asset per Helm chart found in the repository. A chart
// is a directory containing a `Chart.yaml`; the Helm connection scans that
// directory, so each chart carries its own repo-relative path and clones the
// repo independently on connect.
func discoverHelm(conn *connection.GithubConnection, repo *mqlGithubRepository) ([]*inventory.Asset, error) {
	creds := gitCredentials(conn.Asset().Connections[0])

	dirs, err := helmChartDirs(conn.Context(), conn.Client(), repo)
	if err != nil {
		log.Error().Err(err).Str("project", repo.FullName.Data).Msg("failed to discover helm repo")
		return nil, nil
	}

	var res []*inventory.Asset
	for _, dir := range dirs {
		res = append(res, &inventory.Asset{
			Connections: []*inventory.Config{{
				Type: "helm",
				Options: map[string]string{
					"ssh-url":  repo.SshUrl.Data,
					"http-url": repo.CloneUrl.Data,
					"path":     dir,
				},
				Credentials: creds,
			}},
		})
	}
	return res, nil
}

// helmChartDirs returns the repo-relative directories that contain a chart
// (identified by a `Chart.yaml` file).
func helmChartDirs(ctx context.Context, client *github.Client, repo *mqlGithubRepository) ([]string, error) {
	query := "repo:" + repo.FullName.Data + " filename:Chart.yaml"
	results, err := searchCode(ctx, client, query)
	if err != nil {
		return nil, err
	}

	var dirs []string
	seen := map[string]bool{}
	for _, code := range results {
		path := code.GetPath()
		if isHiddenPath(path) || filepath.Base(path) != "Chart.yaml" {
			continue
		}
		dir := iacDir(path)
		if seen[dir] {
			continue
		}
		seen[dir] = true
		dirs = append(dirs, dir)
	}
	return dirs, nil
}

// discoverKustomize emits one asset per Kustomize configuration found in the
// repository. A configuration is a directory containing a kustomization file;
// each carries its own repo-relative path and clones the repo independently on
// connect.
func discoverKustomize(conn *connection.GithubConnection, repo *mqlGithubRepository) ([]*inventory.Asset, error) {
	creds := gitCredentials(conn.Asset().Connections[0])

	dirs, err := kustomizeDirs(conn.Context(), conn.Client(), repo)
	if err != nil {
		log.Error().Err(err).Str("project", repo.FullName.Data).Msg("failed to discover kustomize repo")
		return nil, nil
	}

	var res []*inventory.Asset
	for _, dir := range dirs {
		res = append(res, &inventory.Asset{
			Connections: []*inventory.Config{{
				Type: "kustomize",
				Options: map[string]string{
					"ssh-url":  repo.SshUrl.Data,
					"http-url": repo.CloneUrl.Data,
					"path":     dir,
				},
				Credentials: creds,
			}},
		})
	}
	return res, nil
}

// kustomizeDirs returns the repo-relative directories that contain a Kustomize
// entry point (`kustomization.yaml`, `kustomization.yml`, or `Kustomization`).
func kustomizeDirs(ctx context.Context, client *github.Client, repo *mqlGithubRepository) ([]string, error) {
	query := "repo:" + repo.FullName.Data + " filename:kustomization"
	results, err := searchCode(ctx, client, query)
	if err != nil {
		return nil, err
	}

	var dirs []string
	seen := map[string]bool{}
	for _, code := range results {
		path := code.GetPath()
		if isHiddenPath(path) || !isKustomization(filepath.Base(path)) {
			continue
		}
		dir := iacDir(path)
		if seen[dir] {
			continue
		}
		seen[dir] = true
		dirs = append(dirs, dir)
	}
	return dirs, nil
}

// isKustomization reports whether a base file name is one of the recognized
// Kustomize entry-point file names.
func isKustomization(base string) bool {
	switch base {
	case "kustomization.yaml", "kustomization.yml", "Kustomization":
		return true
	}
	return false
}
