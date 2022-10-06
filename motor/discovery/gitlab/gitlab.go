package gitlab

import (
	"context"
	"errors"

	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/providers"
	gitlab_transport "go.mondoo.com/cnquery/motor/providers/gitlab"
	"go.mondoo.com/cnquery/motor/providers/resolver"
)

const DiscoveryGroup = "group"

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Gitlab Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{common.DiscoveryAuto, DiscoveryGroup}
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, tc *providers.Config, cfn common.CredentialFn, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	// establish connection to GitLab
	m, err := resolver.NewMotorConnection(ctx, tc, cfn)
	if err != nil {
		return nil, err
	}
	defer m.Close()

	trans, ok := m.Provider.(*gitlab_transport.Provider)
	if !ok {
		return nil, errors.New("could not initialize gitlab transport")
	}

	identifier, err := trans.Identifier()
	if err != nil {
		return nil, err
	}

	pf, err := m.Platform()
	if err != nil {
		return nil, err
	}

	var assets []*asset.Asset
	if tc.IncludesDiscoveryTarget(common.DiscoveryAuto) || tc.IncludesDiscoveryTarget(DiscoveryGroup) {
		name := root.Name
		if name == "" {
			grp, err := trans.Group()
			if err != nil {
				return nil, err
			}
			if grp != nil {
				name = "GitLab Group " + grp.Name
			}
		}

		assets = append(assets, &asset.Asset{
			PlatformIds: []string{identifier},
			Name:        name,
			Platform:    pf,
			Connections: []*providers.Config{tc}, // pass-in the current config
			State:       asset.State_STATE_ONLINE,
		})
	}

	return assets, nil
}
