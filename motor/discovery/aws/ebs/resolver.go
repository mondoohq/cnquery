package ebs

import (
	"context"

	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/aws"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/vault/credentials_resolver"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "AWS EC2 EBS Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{}
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, pCfg *providers.Config, credsResolver credentials_resolver.Resolver, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	pCfg.Backend = providers.ProviderType_AWS_EC2_EBS
	assetInfo := &asset.Asset{
		Name:        pCfg.Options["id"],
		Connections: []*providers.Config{pCfg},
		State:       asset.State_STATE_ONLINE,
		PlatformIds: []string{pCfg.PlatformId},
		Labels:      map[string]string{aws.EBSScanLabel: "true", aws.RegionLabel: pCfg.Options["region"], "mondoo.com/item-type": pCfg.Options["type"]},
	}
	// If there's a root-provided name, use that to overwrite
	if root.Name != "" {
		assetInfo.Name = root.Name
	}

	return []*asset.Asset{assetInfo}, nil
}
