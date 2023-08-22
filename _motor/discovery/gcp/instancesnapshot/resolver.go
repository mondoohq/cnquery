// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package instancesnapshot

import (
	"context"
	"errors"

	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/gcpinstancesnapshot"
	"go.mondoo.com/cnquery/motor/vault"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "GCP Compute Instance Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{"auto"}
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, pCfg *providers.Config, credsResolver vault.Resolver, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	pCfg.Backend = providers.ProviderType_GCP_COMPUTE_INSTANCE_SNAPSHOT

	// determine the platform mrn
	sc, err := gcpinstancesnapshot.NewSnapshotCreator()
	if err != nil {
		return nil, err
	}

	target := gcpinstancesnapshot.ParseTarget(pCfg)
	var assetInfo *asset.Asset

	switch target.TargetType {
	case "instance":
		instanceInfo, err := sc.InstanceInfo(target.ProjectID, target.Zone, target.InstanceName)
		if err != nil {
			return nil, err
		}

		assetInfo = &asset.Asset{
			Name:        instanceInfo.InstanceName,
			Connections: []*providers.Config{pCfg},
			State:       asset.State_STATE_ONLINE,
			Labels:      map[string]string{},
			PlatformIds: []string{instanceInfo.PlatformMrn},
		}
	case "snapshot":
		snapshotInfo, err := sc.SnapshotInfo(target.ProjectID, target.SnapshotName)
		if err != nil {
			return nil, err
		}

		assetInfo = &asset.Asset{
			Name:        snapshotInfo.SnapshotName,
			Connections: []*providers.Config{pCfg},
			State:       asset.State_STATE_ONLINE,
			Labels:      map[string]string{},
			PlatformIds: []string{snapshotInfo.PlatformMrn},
		}
	default:
		return nil, errors.New("GCP compute discovery does not support asset type " + target.TargetType)
	}

	// If there's a root-provided name, use that to overwrite
	if root.Name != "" {
		assetInfo.Name = root.Name
	}

	return []*asset.Asset{assetInfo}, nil
}
