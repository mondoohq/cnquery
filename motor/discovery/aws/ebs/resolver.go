package ebs

import (
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/aws"
	"go.mondoo.io/mondoo/motor/discovery/credentials"
	"go.mondoo.io/mondoo/motor/providers"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Aws Ec2 Ebs Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{}
}

func (r *Resolver) Resolve(root *asset.Asset, tc *providers.TransportConfig, cfn credentials.CredentialFn, sfn credentials.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	tc.Backend = providers.TransportBackend_CONNECTION_AWS_EC2_EBS
	assetInfo := &asset.Asset{
		Name:        tc.Options["id"],
		Connections: []*providers.TransportConfig{tc},
		State:       asset.State_STATE_ONLINE,
		PlatformIds: []string{tc.PlatformId},
		Labels:      map[string]string{aws.EBSScanLabel: "true", aws.RegionLabel: tc.Options["region"], "mondoo.com/item-type": tc.Options["type"]},
	}

	return []*asset.Asset{assetInfo}, nil
}
