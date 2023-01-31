package aws

import (
	"context"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ecsservice "github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go/aws/arn"

	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/motorid/awsec2"
	"go.mondoo.com/cnquery/motor/motorid/containerid"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
	aws_provider "go.mondoo.com/cnquery/motor/providers/aws"
	"go.mondoo.com/cnquery/resources/library/jobpool"
	"go.mondoo.com/cnquery/stringx"
)

func NewECSContainersDiscovery(cfg aws.Config) (*ECSContainers, error) {
	return &ECSContainers{config: cfg}, nil
}

type ECSContainers struct {
	config aws.Config
}

func (ecs *ECSContainers) Name() string {
	return "AWS ECS Discover"
}

func (ecs *ECSContainers) List() ([]*asset.Asset, error) {
	identityResp, err := aws_provider.CheckIam(ecs.config)
	if err != nil {
		return nil, err
	}

	account := *identityResp.Account

	instances := []*asset.Asset{}
	poolOfJobs := jobpool.CreatePool(ecs.getECSContainers(account), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		instances = append(instances, poolOfJobs.Jobs[i].Result.([]*asset.Asset)...)
	}

	return instances, nil
}

func (ecs *ECSContainers) getRegions() ([]string, error) {
	regions := []string{}

	ec2svc := ec2.NewFromConfig(ecs.config)
	ctx := context.Background()

	res, err := ec2svc.DescribeRegions(ctx, &ec2.DescribeRegionsInput{})
	if err != nil {
		return regions, err
	}
	for _, region := range res.Regions {
		regions = append(regions, *region.RegionName)
	}
	return regions, nil
}

func (ecs *ECSContainers) getECSContainers(account string) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	var err error

	regions, err := ecs.getRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	log.Debug().Msgf("regions being called for ecs container instances list are: %v", regions)
	for ri := range regions {
		region := regions[ri]
		f := func() (jobpool.JobResult, error) {
			// get client for region
			clonedConfig := ecs.config.Copy()
			clonedConfig.Region = region
			containerInstancesAssets := []*asset.Asset{}
			containersAssets := []*asset.Asset{}
			svc := ecsservice.NewFromConfig(clonedConfig)
			ctx := context.Background()

			resp, err := svc.ListClusters(ctx, &ecsservice.ListClustersInput{})
			if err != nil {
				return nil, errors.Wrap(err, "could not gather ecs cluster information")
			}
			for _, cluster := range resp.ClusterArns {
				clusterName := ""
				clusterDescription, err := svc.DescribeClusters(ctx, &ecsservice.DescribeClustersInput{Clusters: []string{cluster}})
				if err != nil {
					log.Error().Err(err).Msg("could not gather ecs cluster information")
					continue
				}
				if len(clusterDescription.Clusters) == 1 {
					clusterName = *clusterDescription.Clusters[0].ClusterName
				} else {
					log.Warn().Msg("found more than one ecs cluster when filtering by arn")
					continue
				}
				containerInstances, err := svc.ListContainerInstances(ctx, &ecsservice.ListContainerInstancesInput{Cluster: &cluster, Status: types.ContainerInstanceStatusActive})
				if err != nil {
					log.Error().Err(err).Msg("cannot list container instances")
				} else if len(containerInstances.ContainerInstanceArns) > 0 {
					containerInstancesDetail, err := svc.DescribeContainerInstances(ctx, &ecsservice.DescribeContainerInstancesInput{Cluster: &cluster, ContainerInstances: containerInstances.ContainerInstanceArns})
					if err == nil {
						for _, ci := range containerInstancesDetail.ContainerInstances {
							// container instance assets
							if containerInstanceAsset := ecsContainerInstanceToAsset(account, region, ci, clonedConfig); containerInstanceAsset != nil {
								containerInstancesAssets = append(containerInstancesAssets, containerInstanceAsset)
							}
						}
					} else {
						log.Error().Err(err).Msg("could not gather ecs container instances")
					}
				}
				tasks, err := svc.ListTasks(ctx, &ecsservice.ListTasksInput{Cluster: &cluster})
				if err != nil {
					return nil, errors.Wrap(err, "could not gather ecs tasks for cluster")
				}
				if len(tasks.TaskArns) > 0 {
					taskdescriptions, err := svc.DescribeTasks(ctx, &ecsservice.DescribeTasksInput{Tasks: tasks.TaskArns, Cluster: &cluster})
					if err != nil {
						log.Error().Err(err).Msg("could not describe ecs tasks for cluster")
						continue
					}
					for _, task := range taskdescriptions.Tasks {
						for _, c := range task.Containers {
							// container assets
							if containerAsset := ecsContainerToAsset(ctx, account, region, c, clonedConfig, task, clusterName); containerAsset != nil {
								containersAssets = append(containersAssets, containerAsset)
							}
						}
					}
				}
			}

			log.Debug().Str("account", account).Str("region", clonedConfig.Region).Int("ecs containers count", len(containersAssets)).Msg("found ecs containers")
			log.Debug().Str("account", account).Str("region", clonedConfig.Region).Int("ecs container instances count", len(containerInstancesAssets)).Msg("found ecs container instances")
			res := containersAssets
			res = append(res, containerInstancesAssets...)
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
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

func ecsContainerToAsset(ctx context.Context, account string, region string, c ecstypes.Container, clonedConfig aws.Config, task ecstypes.Task, clusterName string) *asset.Asset {
	if c.RuntimeId == nil { // container is not running, do not include it for now
		return nil
	}

	asset := &asset.Asset{
		Name:        *c.Name,
		PlatformIds: []string{containerid.MondooContainerID(*c.RuntimeId)},
		Platform: &platform.Platform{
			Kind:    providers.Kind_KIND_CONTAINER,
			Runtime: providers.RUNTIME_AWS_ECS,
			Family:  []string{*task.PlatformFamily},
			Version: *task.PlatformVersion,
		},
		Connections: []*providers.Config{},
		Labels:      make(map[string]string),
		State:       mapContainerState(*c.LastStatus),
	}
	containerAttachmentIds := []string{}
	for _, ca := range c.NetworkInterfaces {
		containerAttachmentIds = append(containerAttachmentIds, *ca.AttachmentId)
	}
	taskId := ""
	if arn.IsARN(*c.TaskArn) {
		if parsed, err := arn.Parse(*c.TaskArn); err == nil {
			if taskIds := strings.Split(parsed.Resource, "/"); len(taskIds) > 1 {
				taskId = taskIds[len(taskIds)-1]
			}
		}
	}
	var publicIp string
	for _, a := range task.Attachments {
		if stringx.Contains(containerAttachmentIds, *a.Id) {
			for _, a := range task.Attachments {
				for _, detail := range a.Details {
					if *detail.Name == "networkInterfaceId" {
						publicIp = getPublicIpForContainer(ctx, clonedConfig, *detail.Value)
					}

					// add connections here
					asset.Connections = append(asset.Connections, &providers.Config{
						Backend: providers.ProviderType_SSH, // looking into ecs-exec for this, if we leave this out the scan assumes its local
						Host:    publicIp,
						Options: map[string]string{
							"region":      region,
							ContainerName: *c.Name,
							TaskId:        taskId,
						},
					})
				}
			}
		}
	}

	asset.Labels[common.IPLabel] = publicIp

	if publicIp != "" {
		asset.Name = *c.Name + "-" + publicIp
	}
	if c.Image != nil {
		asset.Labels[ImageLabel] = *c.Image
	}
	for j := range task.Tags {
		tag := task.Tags[j]
		if tag.Key != nil {
			key := ImportedFromAWSTagKeyPrefix + *tag.Key
			value := ""
			if tag.Value != nil {
				value = *tag.Value
			}
			asset.Labels[key] = value
		}
	}
	asset.Labels[ClusterNameLabel] = clusterName
	asset.Labels[TaskDefinitionArnLabel] = *task.TaskDefinitionArn
	return asset
}

const (
	ContainerName          = "container_name"
	ClusterNameLabel       = "cluster_name"
	TaskDefinitionArnLabel = "task_definition_arn"
	TaskId                 = "task_id"
	ImageLabel             = "image"
)

func getPublicIpForContainer(ctx context.Context, cfg aws.Config, nii string) string {
	svc := ec2.NewFromConfig(cfg)
	ni, err := svc.DescribeNetworkInterfaces(ctx, &ec2.DescribeNetworkInterfacesInput{NetworkInterfaceIds: []string{nii}})
	if err == nil {
		if len(ni.NetworkInterfaces) == 1 {
			if ni.NetworkInterfaces[0].Association != nil {
				return *ni.NetworkInterfaces[0].Association.PublicIp
			}
		}
	}
	return ""
}

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

func ecsContainerInstanceToAsset(account string, region string, ci ecstypes.ContainerInstance, clonedConfig aws.Config) *asset.Asset {
	// Ec2InstanceId: The ID of the container instance. For Amazon EC2 instances, this value is the
	// Amazon EC2 instance ID. For external instances, this value is the Amazon Web
	// Services Systems Manager managed instance ID.
	if strings.HasPrefix(*ci.Ec2InstanceId, "i-") {
		// use ec2 discovery if it's an ec2 instance
		ec2i, err := NewEc2Discovery(clonedConfig)
		if err != nil {
			return nil
		}
		log.Debug().Msg("using ec2 discovery for container instance")
		ec2i.FilterOptions = Ec2InstancesFilters{InstanceIds: []string{*ci.Ec2InstanceId}, Regions: []string{region}}
		assets, err := ec2i.List()
		if err == nil {
			if len(assets) == 1 {
				return assets[0]
			}
		}
	}
	if strings.HasPrefix(*ci.Ec2InstanceId, "mi-") {
		// use ec2 discovery if it's an ec2 instance
		ec2i, err := NewSSMManagedInstancesDiscovery(clonedConfig)
		if err != nil {
			return nil
		}
		log.Debug().Msg("using ssm discovery for container instance")
		ec2i.FilterOptions = Ec2InstancesFilters{InstanceIds: []string{*ci.Ec2InstanceId}, Regions: []string{region}}
		assets, err := ec2i.List()
		if err == nil {
			if len(assets) == 1 {
				return assets[0]
			}
		}
	}
	asset := &asset.Asset{
		PlatformIds: []string{awsec2.MondooInstanceID(account, region, *ci.Ec2InstanceId)},
		Name:        *ci.Ec2InstanceId,
		Platform: &platform.Platform{
			Runtime: providers.RUNTIME_AWS_ECS,
		},
		State:       mapContainerInstanceState(ci.Status),
		Connections: []*providers.Config{},
		Labels:      make(map[string]string),
	}

	for j := range ci.Tags {
		tag := ci.Tags[j]
		if tag.Key != nil {
			key := ImportedFromAWSTagKeyPrefix + *tag.Key
			value := ""
			if tag.Value != nil {
				value = *tag.Value
			}
			asset.Labels[key] = value
		}
	}

	// add AWS metadata labels
	if label, ok := asset.Labels[ImportedFromAWSTagKeyPrefix+AWSNameLabel]; ok {
		asset.Name = label
	}
	return asset
}
