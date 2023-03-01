package aws

import (
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/rs/zerolog/log"

	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/motorid/awsec2"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/vault"
)

func NewSSMManagedInstancesDiscovery(m *MqlDiscovery, cfg *providers.Config, account string) (*SSMManagedInstances, error) {
	return &SSMManagedInstances{mqlDiscovery: m, providerConfig: cfg.Clone(), account: account}, nil
}

type SSMManagedInstances struct {
	FilterOptions  Ec2InstancesFilters
	profile        string
	mqlDiscovery   *MqlDiscovery
	providerConfig *providers.Config
	account        string
}

func (ssmi *SSMManagedInstances) Name() string {
	return "AWS SSM Discover"
}

func (ssmi *SSMManagedInstances) List() ([]*asset.Asset, error) {
	ssmInstanceAssets, err := ssmInstances(ssmi.mqlDiscovery, ssmi.account, ssmi.providerConfig, whereFilter(ssmi.FilterOptions))
	if err != nil {
		return nil, err
	}
	assetsWithConnecion := []*asset.Asset{}
	for i := range ssmInstanceAssets {
		assetsWithConnecion = append(assetsWithConnecion, ssmi.addConnectionInfoToSSMAsset(ssmInstanceAssets[i]))
	}
	return assetsWithConnecion, nil
}

func assetHasLabels(a *asset.Asset, labels map[string]string) bool {
	if len(labels) == 0 {
		return true
	}
	for k, v := range labels {
		if a.Labels[k] == v {
			return true
		}
	}
	return false
}

func mapSmmManagedPingStateCode(pingStatus string) asset.State {
	switch pingStatus {
	case string(types.PingStatusOnline):
		return asset.State_STATE_RUNNING
	case string(types.PingStatusConnectionLost):
		return asset.State_STATE_PENDING
	case string(types.PingStatusInactive):
		return asset.State_STATE_STOPPED
	default:
		return asset.State_STATE_UNKNOWN
	}
}

func (ssmi *SSMManagedInstances) addConnectionInfoToSSMAsset(instance *asset.Asset) *asset.Asset {
	asset := instance
	creds := []*vault.Credential{
		{
			User: getProbableUsernameFromSSMPlatformName(strings.ToLower(instance.Labels[PlatformLabel])),
		},
	}
	instanceId := instance.Labels[InstanceLabel]
	ipAddress := instance.Labels[common.IPLabel]
	region := instance.Labels[RegionLabel]
	ping := instance.Labels[SSMPingLabel]

	if strings.HasPrefix(instanceId, "i-") {
		creds[0].Type = vault.CredentialType_aws_ec2_ssm_session // this will only work for ec2 instances
	} else {
		log.Warn().Str("asset", asset.Name).Str("id", instanceId).Msg("cannot use ssm session credentials")
	}
	host := instanceId
	if ipAddress != "" {
		host = ipAddress
	}
	asset.PlatformIds = []string{awsec2.MondooInstanceID(ssmi.account, region, instanceId)}
	asset.Platform = &platform.Platform{
		Kind:    providers.Kind_KIND_VIRTUAL_MACHINE,
		Runtime: providers.RUNTIME_AWS_SSM_MANAGED,
	}
	asset.Connections = []*providers.Config{{
		Backend:     providers.ProviderType_SSH,
		Host:        host,
		Insecure:    true,
		Runtime:     providers.RUNTIME_AWS_EC2,
		Credentials: creds,
		Options: map[string]string{
			"region":   region,
			"profile":  ssmi.profile,
			"instance": instanceId,
		},
	}}
	asset.State = mapSmmManagedPingStateCode(ping)

	return asset
}

func getProbableUsernameFromSSMPlatformName(name string) string {
	if strings.HasPrefix(name, "centos") {
		return "centos"
	}
	if strings.HasPrefix(name, "ubuntu") {
		return "ubuntu"
	}
	return "ec2-user"
}
