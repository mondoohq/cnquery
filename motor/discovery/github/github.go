package github

import (
	"context"
	"errors"

	"github.com/google/go-github/v45/github"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/credentials"
	"go.mondoo.io/mondoo/motor/providers"
	github_provider "go.mondoo.io/mondoo/motor/providers/github"
	"go.mondoo.io/mondoo/motor/providers/resolver"
)

const (
	DiscoveryAll        = "all"
	DiscoveryRepository = "repository"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "GitHub Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{DiscoveryAll, DiscoveryRepository}
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, pCfg *providers.Config, cfn credentials.CredentialFn, sfn credentials.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
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
	case "github-user":
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
	case "github-org":
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

		if pCfg.IncludesDiscoveryTarget(DiscoveryAll) || pCfg.IncludesDiscoveryTarget(DiscoveryRepository) {
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
					PlatformIds: []string{github_provider.NewGitubRepoIdentifier(owner, repoName)},
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
