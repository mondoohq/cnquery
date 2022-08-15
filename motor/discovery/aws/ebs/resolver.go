package ebs

import (
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/aws"
	"go.mondoo.io/mondoo/motor/discovery/credentials"
	"go.mondoo.io/mondoo/motor/providers"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "AWS EC2 EBS Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{}
}

func (r *Resolver) Resolve(root *asset.Asset, pCfg *providers.Config, cfn credentials.CredentialFn, sfn credentials.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
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
