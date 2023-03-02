package aws

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/motorid/awsec2"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/vault"
)

func NewEc2Discovery(m *MqlDiscovery, cfg *providers.Config, account string) (*Ec2Instances, error) {
	return &Ec2Instances{mqlDiscovery: m, providerConfig: cfg.Clone(), account: account}, nil
}

type Ec2Instances struct {
	Insecure       bool
	FilterOptions  Ec2InstancesFilters
	profile        string
	mqlDiscovery   *MqlDiscovery
	providerConfig *providers.Config
	account        string
	PassInLabels   map[string]string
}

type Ec2InstanceState int

const (
	Running Ec2InstanceState = iota
	All
)

type Ec2InstancesFilters struct {
	InstanceIds   []string
	Tags          map[string]string
	Regions       []string
	InstanceState Ec2InstanceState
}

func (ec2i *Ec2Instances) Name() string {
	return "AWS EC2 Discover"
}

func (ec2i *Ec2Instances) List() ([]*asset.Asset, error) {
	ec2InstanceAssets, err := ec2Instances(ec2i.mqlDiscovery, ec2i.account, ec2i.providerConfig, whereFilter(ec2i.FilterOptions))
	if err != nil {
		return nil, err
	}
	assetsWithConnecion := []*asset.Asset{}
	for i := range ec2InstanceAssets {
		assetsWithConnecion = append(assetsWithConnecion, ec2i.addConnectionInfoToEc2Asset(ec2InstanceAssets[i]))
	}
	return assetsWithConnecion, nil
}

func (ec2i *Ec2Instances) addConnectionInfoToEc2Asset(instance *asset.Asset) *asset.Asset {
	asset := instance
	instanceId := instance.Labels[InstanceLabel]
	ipAddress := instance.Labels[common.IPLabel]
	region := instance.Labels[RegionLabel]
	state := instance.Labels[StateLabel]
	imageName := instance.Labels[ImageNameLabel]

	asset.PlatformIds = []string{awsec2.MondooInstanceID(ec2i.account, region, instanceId)}
	asset.IdDetector = []string{providers.AWSEc2Detector.String()}
	asset.Platform = &platform.Platform{
		Kind:    providers.Kind_KIND_VIRTUAL_MACHINE,
		Runtime: providers.RUNTIME_AWS_EC2,
	}
	asset.State = mapEc2InstanceStateCode(state)
	// if there is a public ip, we assume ssh is an option
	if ipAddress != "" {
		asset.Connections = []*providers.Config{{
			Backend:  providers.ProviderType_SSH,
			Host:     ipAddress,
			Insecure: ec2i.Insecure,
			Runtime:  providers.RUNTIME_AWS_EC2,
			Credentials: []*vault.Credential{
				{
					Type: vault.CredentialType_aws_ec2_instance_connect,
					User: getProbableUsernameFromImageName(imageName),
				},
			},
			Options: map[string]string{
				"region":   region,
				"profile":  ec2i.profile,
				"instance": instanceId,
			},
		}}
	} else {
		log.Warn().Str("asset", asset.Name).Msg("no public ip address found")
	}
	if len(ec2i.PassInLabels) > 0 {
		for k, v := range ec2i.PassInLabels {
			asset.Labels[k] = v
		}
	}
	return asset
}

func whereFilter(filters Ec2InstancesFilters) string {
	where := []string{}
	if len(filters.Regions) > 0 {
		for i := range filters.Regions {
			r := filters.Regions[i]
			if i > 0 {
				where = append(where, " || ")
			}
			where = append(where, fmt.Sprintf(`region == "%s"`, r))
		}
	}
	if len(filters.InstanceIds) > 0 {
		if len(filters.Regions) > 0 {
			where = append(where, " && ")
		}
		for i := range filters.InstanceIds {
			r := filters.InstanceIds[i]
			if i > 0 {
				where = append(where, " || ")
			}
			where = append(where, fmt.Sprintf(`instanceId == "%s"`, r))
		}
	}
	if len(filters.Tags) > 0 {
		if len(filters.Regions) > 0 || len(filters.InstanceIds) > 0 {
			where = append(where, " && ")
		}
		count := 0
		for k, v := range filters.Tags {
			if count > 0 {
				where = append(where, " || ")
			}
			where = append(where, fmt.Sprintf(`tags["%s"] == "%s"`, k, v))
			count++
		}
	}
	return strings.Join(where, "")
}

const AWSNameLabel = "Name"

type awsec2id struct {
	Account  string
	Region   string
	Instance string
}

func ParseEc2PlatformID(uri string) *awsec2id {
	// aws://ec2/v1/accounts/{account}/regions/{region}/instances/{instanceid}
	awsec2 := regexp.MustCompile(`^\/\/platformid.api.mondoo.app\/runtime\/aws\/ec2\/v1\/accounts\/(.*)\/regions\/(.*)\/instances\/(.*)$`)
	m := awsec2.FindStringSubmatch(uri)
	if len(m) == 0 {
		return nil
	}

	return &awsec2id{
		Account:  m[1],
		Region:   m[2],
		Instance: m[3],
	}
}

func mapEc2InstanceStateCode(state string) asset.State {
	switch state {
	case string(types.InstanceStateNameRunning):
		return asset.State_STATE_RUNNING
	case string(types.InstanceStateNamePending):
		return asset.State_STATE_PENDING
	case string(types.InstanceStateNameShuttingDown): // 32 is shutting down, which is the step before terminated, assume terminated if we get shutting down
		return asset.State_STATE_TERMINATED
	case string(types.InstanceStateNameStopping):
		return asset.State_STATE_STOPPING
	case string(types.InstanceStateNameStopped):
		return asset.State_STATE_STOPPED
	case string(types.InstanceStateNameTerminated):
		return asset.State_STATE_TERMINATED
	default:
		log.Warn().Str("state", string(state)).Msg("unknown ec2 state")
		return asset.State_STATE_UNKNOWN
	}
}

func InstanceIsInRunningOrStoppedState(state *types.InstanceState) bool {
	// instance state 16 == running, 80 == stopped
	if state == nil {
		return false
	}
	return *state.Code == 16 || *state.Code == 80
}

func InstanceIsInRunningState(state *types.InstanceState) bool {
	// instance state 16 == running
	if state == nil {
		return false
	}
	return *state.Code == 16
}

func getProbableUsernameFromImageName(name string) string {
	if strings.Contains(name, "centos") {
		return "centos"
	}
	if strings.Contains(name, "ubuntu") {
		return "ubuntu"
	}
	return "ec2-user"
}
