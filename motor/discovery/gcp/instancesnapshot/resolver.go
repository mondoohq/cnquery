package instancesnapshot

import (
	"context"

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
	instanceInfo, err := sc.InstanceInfo(target.ProjectID, target.Zone, target.InstanceName)
	if err != nil {
		return nil, err
	}

	assetInfo := &asset.Asset{
		Name:        instanceInfo.InstanceName,
		Connections: []*providers.Config{pCfg},
		State:       asset.State_STATE_ONLINE,
		Labels:      map[string]string{},
		PlatformIds: []string{instanceInfo.PlatformMrn},
	}
	// If there's a root-provided name, use that to overwrite
	if root.Name != "" {
		assetInfo.Name = root.Name
	}

	return []*asset.Asset{assetInfo}, nil
}
