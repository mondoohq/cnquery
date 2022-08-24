package ebs

import (
	"context"

	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/aws"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/providers"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "AWS EC2 EBS Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{}
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, pCfg *providers.Config, cfn common.CredentialFn, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	pCfg.Backend = providers.ProviderType_AWS_EC2_EBS
	assetInfo := &asset.Asset{
		Name:        pCfg.Options["id"],
		Connections: []*providers.Config{pCfg},
		State:       asset.State_STATE_ONLINE,
		PlatformIds: []string{pCfg.PlatformId},
		Labels:      map[string]string{aws.EBSScanLabel: "true", aws.RegionLabel: pCfg.Options["region"], "mondoo.com/item-type": pCfg.Options["type"]},
	}

	return []*asset.Asset{assetInfo}, nil
}
