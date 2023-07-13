package aws

import (
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/aws/aws-sdk-go-v2/aws/arn"

	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	awsecsid "go.mondoo.com/cnquery/motor/motorid/awsecs"
	"go.mondoo.com/cnquery/motor/motorid/containerid"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
)

func NewECSContainersDiscovery(m *MqlDiscovery, cfg *providers.Config, account string) (*ECSContainers, error) {
	return &ECSContainers{mqlDiscovery: m, providerConfig: cfg.Clone(), account: account}, nil
}

type ECSContainers struct {
	profile        string
	mqlDiscovery   *MqlDiscovery
	providerConfig *providers.Config
	account        string
	PassInLabels   map[string]string
}

func (ecs *ECSContainers) Name() string {
	return "AWS ECS Discover"
}

func (ecs *ECSContainers) List() ([]*asset.Asset, error) {
	ecsContainers, err := ecsContainers(ecs.mqlDiscovery, ecs.account, ecs.providerConfig)
	if err != nil {
		return nil, err
	}
	assetsWithConnection := []*asset.Asset{}
	for i := range ecsContainers {
		if a := ecs.addConnectionInfoToECSContainerAsset(ecsContainers[i]); a != nil {
			assetsWithConnection = append(assetsWithConnection, a)
		}
	}
	ecsInstances, err := ecsContainerInstances(ecs.mqlDiscovery, ecs.account, ecs.providerConfig)
	if err != nil {
		return nil, err
	}

	for i := range ecsInstances {
		if a := ecs.addConnectionInfoToECSContainerInstanceAsset(ecsInstances[i]); a != nil {
			assetsWithConnection = append(assetsWithConnection, a)
		}
	}

	return assetsWithConnection, nil
}

func mapContainerInstanceState(status *string) asset.State {
	if status == nil {
		return asset.State_STATE_UNKNOWN
	}
	switch *status {
	case "REGISTERING":
		return asset.State_STATE_PENDING
	case "REGISTRATION_FAILED":
		return asset.State_STATE_ERROR
	case "ACTIVE":
		return asset.State_STATE_ONLINE
	case "INACTIVE":
		return asset.State_STATE_OFFLINE
	case "DEREGISTERING":
		return asset.State_STATE_STOPPING
	case "DRAINING":
		return asset.State_STATE_STOPPING
	default:
		return asset.State_STATE_UNKNOWN
	}
}

func (ecs *ECSContainers) addConnectionInfoToECSContainerInstanceAsset(asset *asset.Asset) *asset.Asset {
	if asset == nil {
		return nil
	}
	if strings.HasPrefix(asset.Id, "i-") {
		ec2i, err := NewEc2Discovery(ecs.mqlDiscovery, ecs.providerConfig, ecs.account)
		if err == nil {
			return ec2i.addConnectionInfoToEc2Asset(asset)
		}
	}
	asset.Connections = []*providers.Config{{
		Backend: providers.ProviderType_SSH, // fallback to ssh
		Options: map[string]string{
			"region": asset.Labels[RegionLabel],
		},
	}}
	if len(ecs.PassInLabels) > 0 {
		for k, v := range ecs.PassInLabels {
			asset.Labels[k] = v
		}
	}
	return asset
}

func (ecs *ECSContainers) addConnectionInfoToECSContainerAsset(asset *asset.Asset) *asset.Asset {
	runtimeId := asset.Labels[RuntimeIdLabel]
	if runtimeId == "" {
		return nil
	}
	state := asset.Labels[StateLabel]
	containerArn := asset.Labels[ArnLabel]
	taskArn := asset.Labels[TaskDefinitionArnLabel]
	publicIp := asset.Labels[common.IPLabel]
	region := asset.Labels[RegionLabel]

	asset.PlatformIds = []string{containerid.MondooContainerID(runtimeId), awsecsid.MondooECSContainerID(containerArn)}
	asset.Platform = &platform.Platform{
		Kind:    providers.Kind_KIND_CONTAINER,
		Runtime: providers.RUNTIME_AWS_ECS,
	}
	asset.State = mapContainerState(state)
	taskId := ""
	if arn.IsARN(taskArn) {
		if parsed, err := arn.Parse(taskArn); err == nil {
			if taskIds := strings.Split(parsed.Resource, "/"); len(taskIds) > 1 {
				taskId = taskIds[len(taskIds)-1]
			}
		}
	}

	if publicIp != "" {
		asset.Connections = []*providers.Config{{
			Backend: providers.ProviderType_SSH, // looking into ecs-exec for this, if we leave this out the scan assumes its local
			Host:    publicIp,
			Options: map[string]string{
				"region":      region,
				ContainerName: asset.Labels[ContainerName],
				TaskId:        taskId,
			},
		}}
	} else {
		log.Warn().Str("asset", asset.Name).Msg("no public ip address found")
	}

	if len(ecs.PassInLabels) > 0 {
		for k, v := range ecs.PassInLabels {
			asset.Labels[k] = v
		}
	}

	return asset
}

const (
	DigestLabel            = "digest"
	RepoUrlLabel           = "repo-url"
	RuntimeIdLabel         = "runtime-id"
	ContainerName          = "container_name"
	ClusterNameLabel       = "cluster_name"
	TaskDefinitionArnLabel = "task_definition_arn"
	TaskId                 = "task_id"
	ImageLabel             = "image"
	AgentConnectedLabel    = "agent-connected"
	CapacityProviderLabel  = "capacity-provider"
)

func mapContainerState(state string) asset.State {
	switch strings.ToLower(state) {
	case "running":
		return asset.State_STATE_RUNNING
	case "created":
		return asset.State_STATE_PENDING
	case "paused":
		return asset.State_STATE_STOPPED
	case "exited":
		return asset.State_STATE_TERMINATED
	case "restarting":
		return asset.State_STATE_PENDING
	case "dead":
		return asset.State_STATE_ERROR
	default:
		log.Warn().Str("state", state).Msg("unknown container state")
		return asset.State_STATE_UNKNOWN
	}
}
