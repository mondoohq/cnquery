package gitlab

import (
	"errors"

	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/credentials"
	"go.mondoo.io/mondoo/motor/providers"
	gitlab_transport "go.mondoo.io/mondoo/motor/providers/gitlab"
	"go.mondoo.io/mondoo/motor/providers/resolver"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Gitlab Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{}
}

func (r *Resolver) Resolve(root *asset.Asset, tc *providers.Config, cfn credentials.CredentialFn, sfn credentials.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	// establish connection to GitLab
	m, err := resolver.NewMotorConnection(tc, cfn)
	if err != nil {
		return nil, err
	}
	defer m.Close()

	trans, ok := m.Transport.(*gitlab_transport.Provider)
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

	return []*asset.Asset{{
		PlatformIds: []string{identifier},
		Name:        name,
		Platform:    pf,
		Connections: []*providers.Config{tc}, // pass-in the current config
		State:       asset.State_STATE_ONLINE,
	}}, nil
}
