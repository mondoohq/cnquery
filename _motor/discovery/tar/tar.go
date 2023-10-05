// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package tar

import (
	"context"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v9/motor/asset"
	"go.mondoo.com/cnquery/v9/motor/discovery/common"
	"go.mondoo.com/cnquery/v9/motor/motorid"
	"go.mondoo.com/cnquery/v9/motor/providers"
	"go.mondoo.com/cnquery/v9/motor/providers/resolver"
	"go.mondoo.com/cnquery/v9/motor/vault"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Tar Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{common.DiscoveryAuto, common.DiscoveryAll}
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, tc *providers.Config, credsResolver vault.Resolver, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	assetObj := &asset.Asset{
		Name:        root.Name,
		Connections: []*providers.Config{tc},
		State:       asset.State_STATE_ONLINE,
	}

	m, err := resolver.NewMotorConnection(ctx, tc, credsResolver)
	if err != nil {
		return nil, err
	}
	defer m.Close()

	// determine platform information
	p, err := m.Platform()
	if err == nil {
		assetObj.Platform = p
	}

	fingerprint, err := motorid.IdentifyPlatform(m.Provider, p, userIdDetectors)
	if err != nil {
		return nil, err
	}

	assetObj.PlatformIds = fingerprint.PlatformIDs
	if assetObj.Name != "" {
		assetObj.Name = fingerprint.Name
	}

	log.Debug().Strs("identifier", assetObj.PlatformIds).Msg("motor connection")

	// use hostname as name if asset name was not explicitly provided
	if assetObj.Name == "" && tc.Options["path"] != "" {
		assetObj.Name = tc.Options["path"]
	}

	return []*asset.Asset{assetObj}, nil
}
