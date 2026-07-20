// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/gobwas/glob"
	"github.com/google/go-github/v89/github"
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
		// CloudFormation, Dockerfile, Bicep, Helm, and Kustomize discovery
		// each shallow-clone matched repos, so they stay opt-in via an
		// explicit --discover <type> and are deliberately left out of `all`.
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
	if userId != "" {
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

			iacAssets, err := discoverRepoIac(conn, repo, targets)
			if err != nil {
				return nil, err
			}
			assetList = append(assetList, iacAssets...)
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

	iacAssets, err := discoverRepoIac(conn, repo, targets)
	if err != nil {
		return nil, err
	}
	assetList = append(assetList, iacAssets...)

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

// discoverRepoIac runs every IaC discovery target for a single repository.
//
// Terraform (a language check) and CloudFormation (a content search for the
// AWSTemplateFormatVersion marker) each need their own API call. The remaining
// detectors — k8s manifests, Dockerfiles, Bicep, Helm, and Kustomize — are all
// filename/extension based, so they share a single recursive git-tree walk
// rather than spending one GitHub Code Search call per type. The Code Search
// API is limited to 30 requests/minute, so the previous per-type approach made
// `--discover all` start throttling at only a handful of repos per org.
func discoverRepoIac(conn *connection.GithubConnection, repo *mqlGithubRepository, targets []string) ([]*inventory.Asset, error) {
	var assetList []*inventory.Asset

	if stringx.ContainsAnyOf(targets, connection.DiscoveryAll, connection.DiscoveryTerraform) {
		terraformAssets, err := discoverTerraform(conn, repo)
		if err != nil {
			return nil, err
		}
		assetList = append(assetList, terraformAssets...)
	}
	if stringx.Contains(targets, connection.DiscoveryCloudformation) {
		cfAssets, err := discoverCloudformation(conn, repo)
		if err != nil {
			return nil, err
		}
		assetList = append(assetList, cfAssets...)
	}

	if !stringx.ContainsAnyOf(targets, connection.DiscoveryAll, connection.DiscoveryK8sManifests,
		connection.DiscoveryDockerfiles, connection.DiscoveryBicep, connection.DiscoveryHelm,
		connection.DiscoveryKustomize) {
		return assetList, nil
	}

	iac, err := repoIacFromTree(conn.Context(), conn.Client(), repo)
	if err != nil {
		// Tree discovery is best-effort; a missing/empty repo or permission
		// issue should not fail discovery of the rest of the org.
		log.Error().Err(err).Str("project", repo.FullName.Data).Msg("failed to walk repository tree for IaC discovery")
		return assetList, nil
	}

	creds := gitCredentials(conn.Asset().Connections[0])
	if iac.hasYaml && stringx.ContainsAnyOf(targets, connection.DiscoveryAll, connection.DiscoveryK8sManifests) {
		k8sAsset := gitAsset("k8s", repo, "", creds)
		k8sAsset.Connections[0].Discover = &inventory.Discovery{Targets: []string{"auto"}}
		assetList = append(assetList, k8sAsset)
	}
	if stringx.Contains(targets, connection.DiscoveryDockerfiles) {
		for _, path := range iac.dockerfiles {
			assetList = append(assetList, gitAsset("docker-file", repo, path, creds))
		}
	}
	if iac.hasBicep && stringx.Contains(targets, connection.DiscoveryBicep) {
		assetList = append(assetList, gitAsset("bicep", repo, "", creds))
	}
	if stringx.Contains(targets, connection.DiscoveryHelm) {
		for _, dir := range iac.helmChartDirs {
			assetList = append(assetList, gitAsset("helm", repo, dir, creds))
		}
	}
	if stringx.Contains(targets, connection.DiscoveryKustomize) {
		for _, dir := range iac.kustomizeDirs {
			assetList = append(assetList, gitAsset("kustomize", repo, dir, creds))
		}
	}
	return assetList, nil
}

// repoIac holds the IaC entry points found by a single recursive git-tree walk.
// CloudFormation is intentionally absent: it requires file-content inspection
// (the AWSTemplateFormatVersion marker), not a path match.
type repoIac struct {
	hasYaml       bool // k8s manifests
	hasBicep      bool
	dockerfiles   []string
	helmChartDirs []string
	kustomizeDirs []string
}

// repoIacFromTree fetches the repository's default-branch tree once (recursively)
// and classifies its files into the IaC entry points each detector needs. The
// git-tree endpoint is on the generous core rate limit rather than the Code
// Search limit, so this is one cheap call per repo instead of one Code Search
// call per IaC type. The switch ordering mirrors the GitLab provider so the two
// behave identically — notably, Chart.yaml and kustomization.yaml match their
// own cases and so do not also count as k8s manifests.
func repoIacFromTree(ctx context.Context, client *github.Client, repo *mqlGithubRepository) (*repoIac, error) {
	out := &repoIac{}

	ref := repo.DefaultBranchName.Data
	if ref == "" {
		// Empty repositories have no default branch and therefore no tree.
		return out, nil
	}

	tree, _, err := client.Git.GetTree(ctx, repo.Owner.Data.Login.Data, repo.Name.Data, ref, true)
	if err != nil {
		return nil, err
	}
	if tree.GetTruncated() {
		// The recursive tree endpoint truncates rather than paginating, so for
		// very large repositories some files may be missing from this walk.
		log.Warn().Str("project", repo.FullName.Data).Msg("github repository tree is truncated; some IaC files may not be discovered")
	}

	helmSeen := map[string]bool{}
	kustomizeSeen := map[string]bool{}
	for _, entry := range tree.Entries {
		if entry.GetType() != "blob" {
			continue
		}
		path := entry.GetPath()
		if isHiddenPath(path) {
			continue
		}

		base := filepath.Base(path)
		switch {
		case strings.HasSuffix(path, ".bicep"):
			out.hasBicep = true
		case base == "Chart.yaml":
			dir := iacDir(path)
			if !helmSeen[dir] {
				helmSeen[dir] = true
				out.helmChartDirs = append(out.helmChartDirs, dir)
			}
		case isKustomization(base):
			dir := iacDir(path)
			if !kustomizeSeen[dir] {
				kustomizeSeen[dir] = true
				out.kustomizeDirs = append(out.kustomizeDirs, dir)
			}
		case isDockerfile(base):
			out.dockerfiles = append(out.dockerfiles, path)
		case (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) &&
			!strings.HasSuffix(path, "mql.yaml") && !strings.HasSuffix(path, "mql.yml"):
			out.hasYaml = true
		}
	}
	return out, nil
}

// gitAsset builds a child asset that clones the repository and scans it with the
// given connection type. path is the repo-relative file or directory to scan,
// or "" when the connection scans the whole checkout.
func gitAsset(connType string, repo *mqlGithubRepository, path string, creds []*vault.Credential) *inventory.Asset {
	opts := map[string]string{
		"ssh-url":  repo.SshUrl.Data,
		"http-url": repo.CloneUrl.Data,
	}
	if path != "" {
		opts["path"] = path
	}
	return &inventory.Asset{
		Connections: []*inventory.Config{{
			Type:        connType,
			Options:     opts,
			Credentials: creds,
		}},
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

// isDockerfile reports whether a base file name follows a Dockerfile naming
// convention: `Dockerfile`, `Dockerfile.<suffix>` (e.g. Dockerfile.prod), or
// `<prefix>.Dockerfile`/`<prefix>.dockerfile` (e.g. app.Dockerfile). It rejects
// unrelated names that merely begin with "Dockerfile", like `DockerfileLint.md`.
func isDockerfile(base string) bool {
	return base == "Dockerfile" ||
		strings.HasPrefix(base, "Dockerfile.") ||
		strings.HasSuffix(base, ".Dockerfile") ||
		strings.HasSuffix(base, ".dockerfile")
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

// isKustomization reports whether a base file name is one of the recognized
// Kustomize entry-point file names.
func isKustomization(base string) bool {
	switch base {
	case "kustomization.yaml", "kustomization.yml", "Kustomization":
		return true
	}
	return false
}
