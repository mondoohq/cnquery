package gitlab

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	terraform_resolver "go.mondoo.com/cnquery/motor/discovery/terraform"

	"github.com/xanzy/go-gitlab"
	gitlab_lib "github.com/xanzy/go-gitlab"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/providers"
	gitlab_provider "go.mondoo.com/cnquery/motor/providers/gitlab"
	"go.mondoo.com/cnquery/motor/providers/resolver"
	"go.mondoo.com/cnquery/motor/vault"
)

const (
	DiscoveryGroup     = "group"
	DiscoveryProject   = "project"
	DiscoveryTerraform = "terraform"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Gitlab Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{common.DiscoveryAuto, common.DiscoveryAll, DiscoveryGroup, DiscoveryProject}
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, pCfg *providers.Config, credsResolver vault.Resolver, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	// establish connection to GitLab
	m, err := resolver.NewMotorConnection(ctx, pCfg, credsResolver)
	if err != nil {
		return nil, err
	}
	defer m.Close()

	p, ok := m.Provider.(*gitlab_provider.Provider)
	if !ok {
		return nil, errors.New("could not initialize gitlab transport")
	}

	pf, err := m.Platform()
	if err != nil {
		return nil, err
	}

	defaultName := root.Name
	list := []*asset.Asset{}

	switch pf.Name {
	case "gitlab-project":
		if pCfg.IncludesOneOfDiscoveryTarget(common.DiscoveryAuto, common.DiscoveryAll, DiscoveryProject) {
			name := defaultName
			project, _ := p.Project()
			grp, _ := p.Group()

			if name == "" {
				if project != nil {
					name = project.NameWithNamespace
				}
			}
			identifier, err := p.Identifier()
			if err != nil {
				return nil, err
			}
			projectAsset := &asset.Asset{
				PlatformIds: []string{identifier},
				Name:        name,
				Platform:    pf,
				Connections: []*providers.Config{pCfg}, // pass-in the current config
				State:       asset.State_STATE_ONLINE,
			}

			list = append(list, projectAsset)

			if pCfg.IncludesOneOfDiscoveryTarget(common.DiscoveryAuto, common.DiscoveryAll, DiscoveryTerraform) {
				terraformFiles, err := discoverTerraformHcl(ctx, p.Client(), grp.Path, project.Path)
				if err != nil {
					log.Error().Err(err).Msg("error discovering terraform")
				} else if len(terraformFiles) > 0 {
					terraformCfg := pCfg.Clone()
					terraformCfg.Backend = providers.ProviderType_TERRAFORM

					terraformCfg.Options = map[string]string{
						"asset-type": "hcl",
						"path":       "git+" + project.HTTPURLToRepo,
					}

					if pCfg.Credentials == nil {
						token := os.Getenv("GITLAB_TOKEN")
						terraformCfg.Credentials = []*vault.Credential{{
							Type:   vault.CredentialType_password,
							User:   "oauth2",
							Secret: []byte(token),
						}}
					} else {
						// add oauth2 user to the credentials
						for i := range pCfg.Credentials {
							cred := pCfg.Credentials[i]
							if cred.Type == vault.CredentialType_password {
								cred.User = "oauth2"
							}
						}
					}

					assets, err := (&terraform_resolver.Resolver{}).Resolve(ctx, projectAsset, terraformCfg, credsResolver, sfn, userIdDetectors...)
					if err == nil && len(assets) > 0 {
						list = append(list, assets...)
					}
				}
			}
		}
	case "gitlab-group":
		var grp *gitlab.Group
		if pCfg.IncludesOneOfDiscoveryTarget(common.DiscoveryAuto, common.DiscoveryAll, DiscoveryGroup) {
			name := root.Name
			if name == "" {
				grp, err = p.Group()
				if err != nil {
					return nil, err
				}
				if grp != nil {
					name = "GitLab Group " + grp.Name
				}
			}
			identifier, err := p.Identifier()
			if err != nil {
				return nil, err
			}
			list = append(list, &asset.Asset{
				PlatformIds: []string{identifier},
				Name:        name,
				Platform:    pf,
				Connections: []*providers.Config{pCfg}, // pass-in the current config
				State:       asset.State_STATE_ONLINE,
			})

			if pCfg.IncludesOneOfDiscoveryTarget(common.DiscoveryAuto, common.DiscoveryAll, DiscoveryProject) {
				p.Client().Projects.ListProjects(&gitlab.ListProjectsOptions{})

				for _, project := range grp.Projects {
					clonedConfig := pCfg.Clone()
					if clonedConfig.Options == nil {
						clonedConfig.Options = map[string]string{}
					}
					clonedConfig.Options["group"] = grp.Path
					clonedConfig.Options["project"] = project.Path

					id := gitlab_provider.NewGitLabProjectIdentifier(grp.Name, project.Name)
					projectAsset := &asset.Asset{
						PlatformIds: []string{id},
						Name:        project.NameWithNamespace,
						Platform:    gitlab_provider.GitLabProjectPlatform,
						Connections: []*providers.Config{clonedConfig}, // pass-in the current config
						State:       asset.State_STATE_ONLINE,
					}
					list = append(list, projectAsset)

					if pCfg.IncludesOneOfDiscoveryTarget(common.DiscoveryAuto, common.DiscoveryAll, DiscoveryTerraform) {
						terraformFiles, err := discoverTerraformHcl(ctx, p.Client(), grp.Path, project.Path)
						if err == nil && len(terraformFiles) > 0 {
							terraformCfg := pCfg.Clone()
							terraformCfg.Backend = providers.ProviderType_TERRAFORM
							// git+https://gitlab.com/mondoolabs/example-gitlab.git
							terraformCfg.Options["path"] = "git" + "+" + project.HTTPURLToRepo
							assets, err := (&terraform_resolver.Resolver{}).Resolve(ctx, projectAsset, terraformCfg, credsResolver, sfn)
							if err == nil && len(assets) > 0 {
								for i := range assets {
									if len(assets[i].PlatformIds) > 0 {
										assets[i].PlatformIds[0] = assets[i].PlatformIds[0] + "/" + project.Name
										list = append(list, assets[i])
									} else {
										log.Debug().Msg("missing platform id for asset")
										continue
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return list, nil
}

// discoverTerraformHcl will check if the repository contains terraform files and return the terraform asset
func discoverTerraformHcl(ctx context.Context, client *gitlab_lib.Client, group string, project string) ([]string, error) {
	opts := &gitlab_lib.ListTreeOptions{
		ListOptions: gitlab_lib.ListOptions{
			PerPage: 100,
		},
		Recursive: gitlab_lib.Bool(true),
	}

	nodes := []*gitlab_lib.TreeNode{}
	for {
		data, resp, err := client.Repositories.ListTree(group+"/"+project, opts)
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
