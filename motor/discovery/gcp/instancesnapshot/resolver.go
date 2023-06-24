package instancesnapshot

import (
	"context"

	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/providers"
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
	assetInfo := &asset.Asset{
		Name:        pCfg.Options["id"],
		Connections: []*providers.Config{pCfg},
		State:       asset.State_STATE_ONLINE,
		PlatformIds: []string{pCfg.PlatformId},
		Labels:      map[string]string{},
	}
	// If there's a root-provided name, use that to overwrite
	if root.Name != "" {
		assetInfo.Name = root.Name
	}

	return []*asset.Asset{assetInfo}, nil
}
