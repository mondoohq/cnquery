// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package vcd

import (
	"context"
	"errors"

	"go.mondoo.com/cnquery/v9/motor/asset"
	"go.mondoo.com/cnquery/v9/motor/discovery/common"
	"go.mondoo.com/cnquery/v9/motor/providers"
	"go.mondoo.com/cnquery/v9/motor/providers/resolver"
	vcd_provider "go.mondoo.com/cnquery/v9/motor/providers/vcd"
	"go.mondoo.com/cnquery/v9/motor/vault"
)

type Resolver struct{}

func (k *Resolver) Name() string {
	return "VMware Cloud Director Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{common.DiscoveryAuto, common.DiscoveryAll}
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, cc *providers.Config, credsResolver vault.Resolver, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	m, err := resolver.NewMotorConnection(ctx, cc, credsResolver)
	if err != nil {
		return nil, err
	}
	defer m.Close()

	provider, ok := m.Provider.(*vcd_provider.Provider)
	if !ok {
		return nil, errors.New("could not create VMware Cloud Director provider")
	}

	identifier, err := provider.Identifier()
	if err != nil {
		return nil, err
	}

	// detect platform info for the asset
	pf, err := m.Platform()
	if err != nil {
		return nil, err
	}

	if cc.IncludesOneOfDiscoveryTarget(common.DiscoveryAuto, common.DiscoveryAll) {
		resolved = append(resolved, &asset.Asset{
			PlatformIds: []string{identifier},
			Name:        "VMware Cloud Director",
			Platform:    pf,
			Connections: []*providers.Config{cc}, // pass-in the current config
			Labels:      map[string]string{},
		})
	}

	return resolved, nil
}
