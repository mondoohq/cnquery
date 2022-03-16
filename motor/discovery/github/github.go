package github

import (
	"errors"

	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/credentials"
	"go.mondoo.io/mondoo/motor/transports"
	github_transport "go.mondoo.io/mondoo/motor/transports/github"
	"go.mondoo.io/mondoo/motor/transports/resolver"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Github Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{}
}

func (r *Resolver) Resolve(tc *transports.TransportConfig, cfn credentials.CredentialFn, sfn credentials.QuerySecretFn, userIdDetectors ...transports.PlatformIdDetector) ([]*asset.Asset, error) {
	// establish connection to Github
	m, err := resolver.NewMotorConnection(tc, cfn)
	if err != nil {
		return nil, err
	}
	defer m.Close()

	trans, ok := m.Transport.(*github_transport.Transport)
	if !ok {
		return nil, errors.New("could not initialize github transport")
	}

	identifier, err := trans.Identifier()
	if err != nil {
		return nil, err
	}

	pf, err := m.Platform()
	if err != nil {
		return nil, err
	}

	name := ""
	org, err := trans.Organization()
	if err != nil {
		return nil, err
	}
	if org != nil && org.Name != nil {
		name = *org.Name
	}

	return []*asset.Asset{{
		PlatformIds: []string{identifier},
		Name:        "Github Organization " + name,
		Platform:    pf,
		Connections: []*transports.TransportConfig{tc}, // pass-in the current config
		State:       asset.State_STATE_ONLINE,
	}}, nil
}
