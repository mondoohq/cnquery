// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package ms365

import (
	"context"
	"errors"

	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/providers"
	microsoft "go.mondoo.com/cnquery/motor/providers/microsoft"
	"go.mondoo.com/cnquery/motor/providers/resolver"
	"go.mondoo.com/cnquery/motor/vault"
)

const DiscoveryTenant = "tenants"

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Microsoft 365 Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{common.DiscoveryAuto, common.DiscoveryAll, DiscoveryTenant}
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, cc *providers.Config, credsResolver vault.Resolver, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	// Note: we use the resolver instead of the direct ms365_provider.New to resolve credentials properly
	m, err := resolver.NewMotorConnection(ctx, cc, credsResolver)
	if err != nil {
		return nil, err
	}
	defer m.Close()
	provider, ok := m.Provider.(*microsoft.Provider)
	if !ok {
		return nil, errors.New("could not create ms 365 transport")
	}

	identifier, err := provider.Identifier()
	if err != nil {
		return nil, err
	}

	// try getting a token, this wil err out if the pem/pfx file are wrong or if the password
	// is wrong. allows for returning an err early
	_, err = provider.GetTokenCredential()
	if err != nil {
		return nil, err
	}
	// detect platform info for the asset
	pf, err := m.Platform()
	if err != nil {
		return nil, err
	}

	if cc.IncludesOneOfDiscoveryTarget(common.DiscoveryAuto, common.DiscoveryAll, DiscoveryTenant) {
		resolved = append(resolved, &asset.Asset{
			PlatformIds: []string{identifier},
			Name:        "Microsoft 365 tenant " + provider.TenantID(),
			Platform:    pf,
			Connections: []*providers.Config{cc}, // pass-in the current config
			Labels: map[string]string{
				"azure.com/tenant": provider.TenantID(),
			},
			State: asset.State_STATE_ONLINE,
		})
	}

	return resolved, nil
}
