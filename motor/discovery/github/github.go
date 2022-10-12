package github

import (
	"context"
	"errors"

	"github.com/google/go-github/v47/github"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/providers"
	github_provider "go.mondoo.com/cnquery/motor/providers/github"
	"go.mondoo.com/cnquery/motor/providers/resolver"
)

const (
	DiscoveryRepository   = "repository"
	DiscoveryUser         = "user"
	DiscoveryOrganization = "organization"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "GitHub Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{common.DiscoveryAuto, common.DiscoveryAll, DiscoveryRepository, DiscoveryUser}
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, pCfg *providers.Config, cfn common.CredentialFn, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	// establish connection to GitHub
	m, err := resolver.NewMotorConnection(ctx, pCfg, cfn)
	if err != nil {
		return nil, err
	}
	defer m.Close()

	p, ok := m.Provider.(*github_provider.Provider)
	if !ok {
		return nil, errors.New("could not initialize github transport")
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
	case "github-repo":
		if pCfg.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryRepository) {
			name := defaultName
			if name == "" {
				repo, _ := p.Repository()
				if repo != nil && repo.GetOwner() != nil {
					name = repo.GetOwner().GetLogin() + "/" + repo.GetName()
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
	case "github-user":
		if pCfg.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryUser) {
			name := defaultName
			if name == "" {
				user, _ := p.User()
				if user != nil {
					name = user.GetName()
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
	case "github-org":
		if pCfg.IncludesDiscoveryTarget(common.DiscoveryAll) ||
			pCfg.IncludesDiscoveryTarget(common.DiscoveryAuto) ||
			pCfg.IncludesDiscoveryTarget(DiscoveryOrganization) {
			name := defaultName
			if name == "" {
				org, _ := p.Organization()
				if org != nil {
					name = org.GetName()
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

		if pCfg.IncludesDiscoveryTarget(common.DiscoveryAll) || pCfg.IncludesDiscoveryTarget(DiscoveryRepository) {
			org, err := p.Organization()
			if err != nil {
				return nil, err
			}

			repos, _, err := p.Client().Repositories.List(context.Background(), org.GetLogin(), &github.RepositoryListOptions{})
			if err != nil {
				return nil, err
			}

			for _, repo := range repos {
				clonedConfig := pCfg.Clone()
				if clonedConfig.Options == nil {
					clonedConfig.Options = map[string]string{}
				}

				owner := repo.GetOwner().GetLogin()
				repoName := repo.GetName()
				clonedConfig.Options["owner"] = owner
				clonedConfig.Options["repository"] = repoName
				delete(clonedConfig.Options, "organization")
				delete(clonedConfig.Options, "user")

				list = append(list, &asset.Asset{
					PlatformIds: []string{github_provider.NewGitHubRepoIdentifier(owner, repoName)},
					Name:        owner + "/" + repoName,
					Platform:    github_provider.GithubRepoPlatform,
					Connections: []*providers.Config{clonedConfig}, // pass-in the current config
					State:       asset.State_STATE_ONLINE,
				})
			}
		}
	}

	return list, nil
}
