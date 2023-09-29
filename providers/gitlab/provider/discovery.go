// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/xanzy/go-gitlab"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/providers/gitlab/connection"
	"golang.org/x/exp/slices"
	"google.golang.org/protobuf/proto"
)

func (s *Service) discover(root *inventory.Asset, conn *connection.GitLabConnection) (*inventory.Inventory, error) {
	if conn.Conf.Discover == nil {
		return nil, nil
	}

	client := conn.Client()
	if client == nil {
		return nil, nil
	}

	assets := []*inventory.Asset{}
	projects := []*gitlab.Project{}

	targets := conn.Conf.Discover.Targets

	// The following calls to discover Groups and Projects will always return
	// gitlab.Group and gitlab.Project objects, no matter if we connect to only
	// one system or many. This reduces code complexity.

	groupAssets, groups, err := s.discoverGroups(root, conn)
	if err != nil {
		return nil, err
	}
	if slices.Contains(targets, DiscoveryGroup) {
		assets = append(assets, groupAssets...)
	}

	projectAssets, projects, err := s.discoverProjects(root, conn, groups)
	if err != nil {
		return nil, err
	}
	if slices.Contains(targets, DiscoveryProject) {
		assets = append(assets, projectAssets...)
	}

	if slices.Contains(targets, DiscoveryTerraform) {
		repos, err := s.discoverTerraform(root, conn, projects)
		if err != nil {
			return nil, err
		}
		assets = append(assets, repos...)
	}

	if len(assets) == 0 {
		return nil, nil
	}
	return &inventory.Inventory{
		Spec: &inventory.InventorySpec{
			Assets: assets,
		},
	}, nil
}

func (s *Service) discoverGroups(root *inventory.Asset, conn *connection.GitLabConnection) ([]*inventory.Asset, []*gitlab.Group, error) {
	// If the root asset it a group, we are done because it's the returned
	// main asset. If the root is a project, we want to additionally detect
	// the group and return it.
	// TODO: discover groups for generic gitlab connection
	if conn.IsGroup() {
		group, err := conn.Group()
		return []*inventory.Asset{root}, []*gitlab.Group{group}, err
	}

	group, err := conn.Group()
	if err != nil {
		return nil, nil, err
	}

	conf := conn.Conf.Clone()
	conf.Type = GitlabGroupConnection
	conf.Options = map[string]string{
		"group":    group.Name,
		"group-id": strconv.Itoa(group.ID),
	}
	asset := &inventory.Asset{
		Connections: []*inventory.Config{conf},
	}

	s.detectAsGroup(asset, group)

	return []*inventory.Asset{asset}, []*gitlab.Group{group}, nil
}

func (s *Service) discoverProjects(root *inventory.Asset, conn *connection.GitLabConnection, groups []*gitlab.Group) ([]*inventory.Asset, []*gitlab.Project, error) {
	if conn.IsProject() {
		project, err := conn.Project()
		return []*inventory.Asset{root}, []*gitlab.Project{project}, err
	}

	var assets []*inventory.Asset
	var projects []*gitlab.Project

	for i := range groups {
		group := groups[i]
		groupProjects, err := discoverGroupProjects(conn, group.ID)
		if err != nil {
			return nil, nil, err
		}

		for j := range groupProjects {
			project := groupProjects[j]
			conf := conn.Conf.Clone()
			conf.Type = GitlabProjectConnection
			conf.Options = map[string]string{
				"group":      group.Name,
				"group-id":   strconv.Itoa(group.ID),
				"project":    project.Name,
				"project-id": strconv.Itoa(project.ID),
			}
			asset := &inventory.Asset{
				Name:        project.NameWithNamespace,
				Connections: []*inventory.Config{conf},
			}

			s.detectAsProject(asset, group, project)
			if err != nil {
				return nil, nil, err
			}

			assets = append(assets, asset)
			projects = append(projects, project)
		}
	}
	return assets, projects, nil
}

func discoverGroupProjects(conn *connection.GitLabConnection, gid interface{}) ([]*gitlab.Project, error) {
	perPage := 50
	page := 1
	total := 50
	projects := []*gitlab.Project{}
	for page*perPage <= total {
		projs, resp, err := conn.Client().Groups.ListGroupProjects(gid, &gitlab.ListGroupProjectsOptions{ListOptions: gitlab.ListOptions{Page: page, PerPage: perPage}})
		if err != nil {
			return nil, err
		}
		projects = append(projects, projs...)
		total = resp.TotalItems
		page += 1
	}

	return projects, nil
}

func (s *Service) discoverTerraform(root *inventory.Asset, conn *connection.GitLabConnection, projects []*gitlab.Project) ([]*inventory.Asset, error) {
	// For git clone we need to set the user to oauth2 to be usable with the token.
	creds := make([]*vault.Credential, len(conn.Conf.Credentials))
	for i := range conn.Conf.Credentials {
		cred := conn.Conf.Credentials[i]
		cc := proto.Clone(cred).(*vault.Credential)
		if cc.User == "" {
			cc.User = "oauth2"
		}
		creds[i] = cc
	}

	var res []*inventory.Asset
	for i := range projects {
		project := projects[i]
		files, err := discoverTerraformHcl(conn.Client(), project.ID)
		if err != nil {
			log.Error().Err(err).Msg("failed to discover terraform repo in gitlab")
		} else if len(files) != 0 {
			res = append(res, &inventory.Asset{
				Connections: []*inventory.Config{{
					Type: "terraform-hcl-git",
					Options: map[string]string{
						"ssh-url":  project.SSHURLToRepo,
						"http-url": project.HTTPURLToRepo,
					},
					Credentials: creds,
				}},
			})
		}
	}
	return res, nil
}

// discoverTerraformHcl will check if the repository contains terraform files and return the terraform asset
func discoverTerraformHcl(client *gitlab.Client, pid interface{}) ([]string, error) {
	opts := &gitlab.ListTreeOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
		},
		Recursive: gitlab.Bool(true),
	}

	nodes := []*gitlab.TreeNode{}
	for {
		data, resp, err := client.Repositories.ListTree(pid, opts)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, data...)

		// Exit the loop when we've seen all pages.
		if resp.NextPage == 0 {
			break
		}

		// Update the page number to get the next page.
		opts.Page = resp.NextPage
	}

	terraformFiles := []string{}
	for i := range nodes {
		node := nodes[i]
		if node.Type == "blob" && strings.HasSuffix(node.Path, ".tf") {
			terraformFiles = append(terraformFiles, node.Path)
		}
	}

	return terraformFiles, nil
}
