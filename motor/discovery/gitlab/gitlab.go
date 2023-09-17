package gitlab

import (
	"context"
	"errors"

	"github.com/xanzy/go-gitlab"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/providers"
	gitlab_provider "go.mondoo.com/cnquery/motor/providers/gitlab"
	"go.mondoo.com/cnquery/motor/providers/resolver"
	"go.mondoo.com/cnquery/motor/vault"
)

const (
	DiscoveryGroup   = "group"
	DiscoveryProject = "project"
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

	identifier, err := p.Identifier()
	if err != nil {
		return nil, err
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
			if name == "" {
				project, _ := p.Project()
				if project != nil {
					name = project.NameWithNamespace
				}
			}

			list = append(list, &asset.Asset{
				PlatformIds: []string{identifier},
				Name:        name,
				Platform:    pf,
				Connections: []*providers.Config{pCfg}, // pass-in the current config
				State:       asset.State_STATE_ONLINE,
			})
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
					clonedConfig.Options["group"] = grp.Name
					clonedConfig.Options["project"] = project.Name

					list = append(list, &asset.Asset{
						PlatformIds: []string{identifier},
						Name:        project.NameWithNamespace,
						Platform:    gitlab_provider.GitLabProjectPlatform,
						Connections: []*providers.Config{clonedConfig}, // pass-in the current config
						State:       asset.State_STATE_ONLINE,
					})
				}
			}
		}
	}
	return list, nil
}
