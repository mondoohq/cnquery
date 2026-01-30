// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecsservice "github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v12/providers/aws/connection"
	"go.mondoo.com/cnquery/v12/types"
	"go.mondoo.com/cnquery/v12/utils/stringx"
)

func (a *mqlAwsEcs) id() (string, error) {
	return "aws.ecs", nil
}

func (a *mqlAwsEcs) containers() ([]any, error) {
	obj, err := CreateResource(a.MqlRuntime, "aws.ecs", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	ecs := obj.(*mqlAwsEcs)

	clusters, err := ecs.clusters()
	if err != nil {
		return nil, err
	}
	containers := []any{}

	for i := range clusters {
		tasks, err := clusters[i].(*mqlAwsEcsCluster).tasks()
		if err != nil {
			return nil, err
		}
		for i := range tasks {
			t := tasks[i].(*mqlAwsEcsTask)
			c := t.GetContainers()
			if c.Error != nil {
				return nil, c.Error
			}
			containers = append(containers, c.Data...)
		}
	}
	return containers, nil
}

func (a *mqlAwsEcs) containerInstances() ([]any, error) {
	obj, err := CreateResource(a.MqlRuntime, "aws.ecs", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	ecs := obj.(*mqlAwsEcs)

	clusters, err := ecs.clusters()
	if err != nil {
		return nil, err
	}
	containerInstances := []any{}

	for i := range clusters {
		c := clusters[i].(*mqlAwsEcsCluster)
		ci := c.GetContainerInstances()
		if ci.Error != nil {
			return nil, ci.Error
		}
		containerInstances = append(containerInstances, ci.Data...)

	}
	return containerInstances, nil
}

func (a *mqlAwsEcsInstance) ec2Instance() (*mqlAwsEc2Instance, error) {
	return a.Ec2Instance.Data, nil
}

func (a *mqlAwsEcs) clusters() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getECSClusters(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsEcs) getECSClusters(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	var err error
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	log.Debug().Msgf("regions being called for ecs clusters list are: %v", regions)
	for ri := range regions {
		region := regions[ri]
		f := func() (jobpool.JobResult, error) {
			svc := conn.Ecs(region)
			ctx := context.Background()
			res := []any{}

			params := &ecsservice.ListClustersInput{}
			paginator := ecsservice.NewListClustersPaginator(svc, params)
			for paginator.HasMorePages() {
				resp, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, errors.Wrap(err, "could not gather ecs cluster information")
				}
				for _, clusterArn := range resp.ClusterArns {
					mqlCluster, err := NewResource(a.MqlRuntime, "aws.ecs.cluster",
						map[string]*llx.RawData{
							"arn": llx.StringData(clusterArn),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlCluster)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func initAwsEcsCluster(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch ecs cluster")
	}
	a := args["arn"].Value.(string)
	conn := runtime.Connection.(*connection.AwsConnection)

	// Validate and parse ARN if provided
	parsedARN, err := validateAndParseARN(a, "ecs")
	if err != nil {
		return nil, nil, err
	}

	region := parsedARN.Region

	svc := conn.Ecs(region)
	ctx := context.Background()
	clusterDetails, err := svc.DescribeClusters(ctx, &ecs.DescribeClustersInput{Clusters: []string{a}, Include: []ecstypes.ClusterField{ecstypes.ClusterFieldTags}})
	if err != nil {
		return nil, nil, err
	}
	if len(clusterDetails.Clusters) != 1 {
		return nil, nil, errors.Newf("only expected one cluster, got %d", len(clusterDetails.Clusters))
	}
	c := clusterDetails.Clusters[0]
	configuration, err := convert.JsonToDict(c.Configuration)
	if err != nil {
		return nil, nil, err
	}
	args["activeServicesCount"] = llx.IntData(int64(c.ActiveServicesCount))
	args["configuration"] = llx.MapData(configuration, types.String)
	args["name"] = llx.StringDataPtr(c.ClusterName)
	args["pendingTasksCount"] = llx.IntData(int64(c.PendingTasksCount))
	args["region"] = llx.StringData(region)
	args["registeredContainerInstancesCount"] = llx.IntData(int64(c.RegisteredContainerInstancesCount))
	args["runningTasksCount"] = llx.IntData(int64(c.RunningTasksCount))
	args["status"] = llx.StringDataPtr(c.Status)
	args["tags"] = llx.MapData(ecsTagsToMap(c.Tags), types.String)
	return args, nil, nil
}

func (a *mqlAwsEcsCluster) containerInstances() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	clustera := a.Arn.Data
	region := ""
	if arn.IsARN(clustera) {
		if val, err := arn.Parse(clustera); err == nil {
			region = val.Region
		}
	}
	svc := conn.Ecs(region)
	ctx := context.Background()
	res := []any{}

	params := &ecsservice.ListContainerInstancesInput{Cluster: &clustera}
	containerInstances, err := svc.ListContainerInstances(ctx, params)
	if err != nil {
		log.Error().Err(err).Msg("cannot list container instances") // no fail
	} else if len(containerInstances.ContainerInstanceArns) > 0 {
		containerInstancesDetail, err := svc.DescribeContainerInstances(ctx, &ecsservice.DescribeContainerInstancesInput{Cluster: &clustera, ContainerInstances: containerInstances.ContainerInstanceArns})
		if err == nil {
			for _, ci := range containerInstancesDetail.ContainerInstances {
				// container instance assets
				args := map[string]*llx.RawData{
					"arn":              llx.StringData(convert.ToValue(ci.ContainerInstanceArn)),
					"agentConnected":   llx.BoolData(ci.AgentConnected),
					"id":               llx.StringData(convert.ToValue(ci.Ec2InstanceId)),
					"capacityProvider": llx.StringData(convert.ToValue(ci.CapacityProviderName)),
					"region":           llx.StringData(region),
				}
				if strings.HasPrefix(convert.ToValue(ci.Ec2InstanceId), "i-") {
					mqlInstanceResource, err := CreateResource(a.MqlRuntime, "aws.ec2.instance",
						map[string]*llx.RawData{
							"arn": llx.StringData(fmt.Sprintf(ec2InstanceArnPattern, region, conn.AccountId(), convert.ToValue(ci.Ec2InstanceId))),
						})
					if err == nil && mqlInstanceResource != nil {
						mqlInstance := mqlInstanceResource.(*mqlAwsEc2Instance)
						args["ec2Instance"] = llx.ResourceData(mqlInstance, mqlInstance.MqlName())
					}
				} else {
					args["ec2Instance"] = llx.NilData
				}

				mqlEcsInstance, err := CreateResource(a.MqlRuntime, "aws.ecs.instance", args)
				if err != nil {
					return nil, err
				}
				res = append(res, mqlEcsInstance)
			}
		} else {
			log.Error().Err(err).Msg("could not gather ecs container instances")
		}
	}
	return res, nil
}

func (s *mqlAwsEcsInstance) id() (string, error) {
	return s.Arn.Data, nil
}

func (s *mqlAwsEcsCluster) id() (string, error) {
	return s.Arn.Data, nil
}

func (a *mqlAwsEcsCluster) tasks() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	clustera := a.Arn.Data
	name := a.Arn.Data

	region := ""
	if arn.IsARN(clustera) {
		if val, err := arn.Parse(clustera); err == nil {
			region = val.Region
		}
	}
	svc := conn.Ecs(region)
	ctx := context.Background()
	res := []any{}

	params := &ecsservice.ListTasksInput{Cluster: &clustera}
	paginator := ecsservice.NewListTasksPaginator(svc, params)
	for paginator.HasMorePages() {
		resp, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not gather ecs tasks information")
		}
		for _, taskArn := range resp.TaskArns {
			mqlTask, err := NewResource(a.MqlRuntime, "aws.ecs.task",
				map[string]*llx.RawData{
					"arn":         llx.StringData(taskArn),
					"clusterName": llx.StringData(name),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlTask)
		}
	}
	return res, nil
}

func (s *mqlAwsEcsTask) id() (string, error) {
	return s.Arn.Data, nil
}

func initAwsEcsTask(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch ecs task")
	}
	a := args["arn"].Value.(string)
	conn := runtime.Connection.(*connection.AwsConnection)

	parsedARN, err := validateAndParseARN(a, "ecs")
	if err != nil {
		return nil, nil, err
	}

	region := parsedARN.Region
	clusterName := ""
	if res := strings.Split(parsedARN.Resource, "/"); len(res) == 3 {
		clusterName = res[1]
	}

	svc := conn.Ecs(region)
	ctx := context.Background()
	params := &ecs.DescribeTasksInput{Tasks: []string{a}, Cluster: &clusterName, Include: []ecstypes.TaskField{ecstypes.TaskFieldTags}}
	taskDetails, err := svc.DescribeTasks(ctx, params)
	if err != nil {
		return nil, nil, err
	}
	if len(taskDetails.Tasks) != 1 {
		return nil, nil, errors.Newf("only expected one task, got %d", len(taskDetails.Tasks))
	}

	t := taskDetails.Tasks[0]
	args["clusterName"] = llx.StringData(clusterName)
	args["connectivity"] = llx.StringData(string(t.Connectivity))
	args["lastStatus"] = llx.StringData(convert.ToValue(t.LastStatus))
	args["platformFamily"] = llx.StringData(convert.ToValue(t.PlatformFamily))
	args["platformVersion"] = llx.StringData(convert.ToValue(t.PlatformVersion))
	args["tags"] = llx.MapData(ecsTagsToMap(t.Tags), types.String)
	res, err := CreateResource(runtime, "aws.ecs.task", args)
	if err != nil {
		return args, nil, err
	}
	res.(*mqlAwsEcsTask).cacheContainers = t.Containers
	res.(*mqlAwsEcsTask).region = region
	res.(*mqlAwsEcsTask).attachments = t.Attachments
	res.(*mqlAwsEcsTask).clusterName = clusterName
	res.(*mqlAwsEcsTask).taskDefArn = t.TaskDefinitionArn

	return args, res, nil
}

type mqlAwsEcsTaskInternal struct {
	cacheContainers []ecstypes.Container
	region          string
	attachments     []ecstypes.Attachment
	clusterName     string
	taskDefArn      *string
}

func (t *mqlAwsEcsTask) containers() ([]any, error) {
	conn := t.MqlRuntime.Connection.(*connection.AwsConnection)
	ctx := context.Background()

	svc := conn.Ecs(t.region)
	definition, err := svc.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{TaskDefinition: t.taskDefArn})
	if err != nil {
		return nil, err
	}
	containerLogDriverMap := make(map[string]string)
	containerCommandMap := make(map[string][]string)

	for i := range definition.TaskDefinition.ContainerDefinitions {
		cd := definition.TaskDefinition.ContainerDefinitions[i]
		if cd.Name != nil {
			containerCommandMap[*cd.Name] = cd.Command
			if cd.LogConfiguration != nil {
				containerLogDriverMap[*cd.Name] = string(cd.LogConfiguration.LogDriver)
			} else {
				containerLogDriverMap[*cd.Name] = "none"
			}
		}
	}

	containers := []any{}
	for _, c := range t.cacheContainers {
		containerLogDriverMap := make(map[string]string)
		containerCommandMap := make(map[string]string)
		cmds := []any{}
		for i := range containerCommandMap[convert.ToValue(c.Name)] {
			cmds = append(cmds, containerCommandMap[convert.ToValue(c.Name)][i])
		}
		publicIp := getContainerIP(ctx, conn, t.attachments, c, t.region)
		name := convert.ToValue(c.Name)
		if publicIp != "" {
			name = name + "-" + publicIp
		}

		if !conn.Filters.Ecs.MatchesOnlyRunningContainers(convert.ToValue(c.LastStatus)) {
			log.Debug().Str("container", name).Str("state", convert.ToValue(c.LastStatus)).Msg("skipping ecs container due to not being in a running state")
			continue
		}

		mqlContainer, err := CreateResource(t.MqlRuntime, "aws.ecs.container",
			map[string]*llx.RawData{
				"arn":               llx.StringDataPtr(c.ContainerArn),
				"clusterName":       llx.StringData(t.clusterName),
				"command":           llx.ArrayData(cmds, types.Any),
				"containerName":     llx.StringDataPtr(c.Name),
				"cpuUnits":          llx.StringDataPtr(c.Cpu),
				"image":             llx.StringData(convert.ToValue(c.Image)),
				"logDriver":         llx.StringData(containerLogDriverMap[convert.ToValue(c.Name)]),
				"name":              llx.StringData(name),
				"platformFamily":    llx.StringData(t.PlatformFamily.Data),
				"platformVersion":   llx.StringData(t.PlatformVersion.Data),
				"publicIp":          llx.StringData(publicIp),
				"region":            llx.StringData(t.region),
				"runtimeId":         llx.StringDataPtr(c.RuntimeId),
				"status":            llx.StringDataPtr(c.LastStatus),
				"taskArn":           llx.StringData(t.Arn.Data),
				"taskDefinitionArn": llx.StringData(t.Arn.Data),
				"memorySoftLimit":   llx.StringDataPtr(c.MemoryReservation),
				"memoryHardLimit":   llx.StringDataPtr(c.Memory),
				"reason":            llx.StringDataPtr(c.Reason),
			})
		if err != nil {
			return nil, err
		}
		containers = append(containers, mqlContainer)
	}
	return containers, nil
}

func getContainerIP(ctx context.Context, conn *connection.AwsConnection, attachments []ecstypes.Attachment, c ecstypes.Container, region string) string {
	containerAttachmentIds := []string{}
	for _, ca := range c.NetworkInterfaces {
		containerAttachmentIds = append(containerAttachmentIds, *ca.AttachmentId)
	}
	var publicIp string
	for _, a := range attachments {
		if stringx.Contains(containerAttachmentIds, *a.Id) {
			for _, detail := range a.Details {
				if *detail.Name == "networkInterfaceId" {
					publicIp = getPublicIpForContainer(ctx, conn, *detail.Value, region)
				}
			}
		}
	}
	return publicIp
}

func getPublicIpForContainer(ctx context.Context, conn *connection.AwsConnection, nii string, region string) string {
	svc := conn.Ec2(region)
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

func (s *mqlAwsEcsContainer) id() (string, error) {
	return s.Arn.Data, nil
}

func ecsTagsToMap(tags []ecstypes.Tag) map[string]any {
	res := map[string]any{}
	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			res[convert.ToValue(tag.Key)] = convert.ToValue(tag.Value)
		}
	}
	return res
}

// validateAndParseARN validates that the given string is a valid ECS ARN
// and returns the parsed ARN structure. Returns an error if the ARN is malformed
// or does not belong to the expectedService.
func validateAndParseARN(arnStr, expectedService string) (*arn.ARN, error) {
	if !strings.HasPrefix(arnStr, "arn:") {
		return nil, errors.Newf("invalid ARN format: %s", arnStr)
	}

	parsedArn, err := arn.Parse(arnStr)
	if err != nil {
		return nil, err
	}

	if parsedArn.Service != expectedService {
		return nil, errors.Newf("invalid ARN (service is %s, expected %s): %s",
			parsedArn.Service, expectedService, arnStr)
	}

	return &parsedArn, nil
}

func (a *mqlAwsEcs) taskDefinitions() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getECSTaskDefinitions(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsEcs) getECSTaskDefinitions(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	var err error
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	log.Debug().Msgf("regions being called for ecs task definitions list are: %v", regions)
	for ri := range regions {
		region := regions[ri]
		f := func() (jobpool.JobResult, error) {
			svc := conn.Ecs(region)
			ctx := context.Background()
			res := []any{}

			params := &ecsservice.ListTaskDefinitionsInput{}
			paginator := ecsservice.NewListTaskDefinitionsPaginator(svc, params)
			for paginator.HasMorePages() {
				resp, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, errors.Wrap(err, "could not gather ecs task definition information")
				}
				for _, taskDefArn := range resp.TaskDefinitionArns {
					// Describe each task definition to get full details
					describeResp, err := svc.DescribeTaskDefinition(ctx, &ecsservice.DescribeTaskDefinitionInput{
						TaskDefinition: &taskDefArn,
					})
					if err != nil {
						if Is400AccessDeniedError(err) {
							log.Warn().Str("region", region).Str("taskDef", taskDefArn).Msg("error accessing task definition")
							continue
						}
						return nil, errors.Wrapf(err, "could not describe task definition %s", taskDefArn)
					}

					if describeResp.TaskDefinition == nil {
						continue
					}

					td := describeResp.TaskDefinition

					// Fetch tags using ListTagsForResource API
					tags := make(map[string]any)
					if td.TaskDefinitionArn != nil {
						tagsResp, err := svc.ListTagsForResource(ctx, &ecsservice.ListTagsForResourceInput{
							ResourceArn: td.TaskDefinitionArn,
						})
						if err != nil {
							if Is400AccessDeniedError(err) {
								log.Warn().Str("region", region).Str("taskDef", *td.TaskDefinitionArn).Msg("access denied when fetching tags for task definition")
							} else {
								log.Warn().Err(err).Str("taskDef", *td.TaskDefinitionArn).Msg("could not fetch tags for task definition")
							}
						} else if tagsResp != nil && tagsResp.Tags != nil {
							tags = ecsTagsToMap(tagsResp.Tags)
						}
					}

					mqlTaskDef, err := a.createTaskDefinitionResource(region, td, tags)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlTaskDef)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsEcs) createTaskDefinitionResource(region string, td *ecstypes.TaskDefinition, tags map[string]any) (any, error) {
	// Extract basic fields
	arn := ""
	if td.TaskDefinitionArn != nil {
		arn = *td.TaskDefinitionArn
	}
	family := ""
	if td.Family != nil {
		family = *td.Family
	}
	revision := int64(td.Revision)
	status := string(td.Status)
	networkMode := ""
	if td.NetworkMode != "" {
		networkMode = string(td.NetworkMode)
	}
	pidMode := ""
	if td.PidMode != "" {
		pidMode = string(td.PidMode)
	}
	ipcMode := ""
	if td.IpcMode != "" {
		ipcMode = string(td.IpcMode)
	}
	taskRoleArn := ""
	if td.TaskRoleArn != nil {
		taskRoleArn = *td.TaskRoleArn
	}
	executionRoleArn := ""
	if td.ExecutionRoleArn != nil {
		executionRoleArn = *td.ExecutionRoleArn
	}

	// Create container definitions
	containerDefs := []any{}
	for i := range td.ContainerDefinitions {
		mqlContainerDef, err := a.createContainerDefinitionResource(arn, &td.ContainerDefinitions[i])
		if err != nil {
			return nil, err
		}
		containerDefs = append(containerDefs, mqlContainerDef)
	}

	// Create volumes
	volumes := []any{}
	for i := range td.Volumes {
		mqlVolume, err := a.createVolumeResource(&td.Volumes[i])
		if err != nil {
			return nil, err
		}
		volumes = append(volumes, mqlVolume)
	}

	// Create ephemeral storage
	var ephemeralStorage any
	if td.EphemeralStorage != nil {
		mqlEphemeralStorage, err := a.createEphemeralStorageResource(td.EphemeralStorage)
		if err != nil {
			return nil, err
		}
		ephemeralStorage = mqlEphemeralStorage
	} else {
		// Create empty ephemeral storage resource
		mqlEphemeralStorage, err := CreateResource(a.MqlRuntime, "aws.ecs.taskDefinition.ephemeralStorage",
			map[string]*llx.RawData{
				"__id":      llx.StringData(arn + "/ephemeralStorage"),
				"sizeInGiB": llx.IntData(0),
			})
		if err != nil {
			return nil, err
		}
		ephemeralStorage = mqlEphemeralStorage
	}

	// Tags are passed as parameter (fetched via ListTagsForResource)
	// Type assert ephemeralStorage to Resource
	ephemeralStorageResource, ok := ephemeralStorage.(plugin.Resource)
	if !ok {
		return nil, errors.New("failed to convert ephemeralStorage to Resource")
	}

	return CreateResource(a.MqlRuntime, "aws.ecs.taskDefinition",
		map[string]*llx.RawData{
			"__id":                 llx.StringData(arn),
			"arn":                  llx.StringData(arn),
			"family":               llx.StringData(family),
			"revision":             llx.IntData(revision),
			"status":               llx.StringData(status),
			"networkMode":          llx.StringData(networkMode),
			"pidMode":              llx.StringData(pidMode),
			"ipcMode":              llx.StringData(ipcMode),
			"taskRoleArn":          llx.StringData(taskRoleArn),
			"executionRoleArn":     llx.StringData(executionRoleArn),
			"containerDefinitions": llx.ArrayData(containerDefs, types.Resource("aws.ecs.taskDefinition.containerDefinition")),
			"volumes":              llx.ArrayData(volumes, types.Resource("aws.ecs.taskDefinition.volume")),
			"ephemeralStorage":     llx.ResourceData(ephemeralStorageResource, "aws.ecs.taskDefinition.ephemeralStorage"),
			"tags":                 llx.MapData(tags, types.String),
			"region":               llx.StringData(region),
		})
}

func (a *mqlAwsEcs) createContainerDefinitionResource(taskDefArn string, cd *ecstypes.ContainerDefinition) (any, error) {
	name := ""
	if cd.Name != nil {
		name = *cd.Name
	}
	image := ""
	if cd.Image != nil {
		image = *cd.Image
	}
	privileged := false
	if cd.Privileged != nil {
		privileged = *cd.Privileged
	}
	readonlyRootFilesystem := false
	if cd.ReadonlyRootFilesystem != nil {
		readonlyRootFilesystem = *cd.ReadonlyRootFilesystem
	}
	user := ""
	if cd.User != nil {
		user = *cd.User
	}
	memory := int64(0)
	if cd.Memory != nil {
		memory = int64(*cd.Memory)
	}
	cpu := int64(cd.Cpu)

	// Create environment variables
	envVars := []any{}
	if cd.Environment != nil {
		for _, env := range cd.Environment {
			envName := ""
			envValue := ""
			if env.Name != nil {
				envName = *env.Name
			}
			if env.Value != nil {
				envValue = *env.Value
			}
			mqlEnv, err := CreateResource(a.MqlRuntime, ResourceAwsEcsTaskDefinitionContainerDefinitionEnvironmentVariable,
				map[string]*llx.RawData{
					"__id":  llx.StringData(taskDefArn + "/container/" + name + "/env/" + envName),
					"name":  llx.StringData(envName),
					"value": llx.StringData(envValue),
				})
			if err != nil {
				return nil, err
			}
			envVars = append(envVars, mqlEnv)
		}
	}

	// Create secrets
	secrets := []any{}
	if cd.Secrets != nil {
		for _, secret := range cd.Secrets {
			secretName := ""
			valueFrom := ""
			if secret.Name != nil {
				secretName = *secret.Name
			}
			if secret.ValueFrom != nil {
				valueFrom = *secret.ValueFrom
			}
			mqlSecret, err := CreateResource(a.MqlRuntime, ResourceAwsEcsTaskDefinitionContainerDefinitionSecret,
				map[string]*llx.RawData{
					"__id":      llx.StringData(taskDefArn + "/container/" + name + "/secret/" + secretName),
					"name":      llx.StringData(secretName),
					"valueFrom": llx.StringData(valueFrom),
				})
			if err != nil {
				return nil, err
			}
			secrets = append(secrets, mqlSecret)
		}
	}

	// Create log configuration
	var logConfig any
	if cd.LogConfiguration != nil {
		logDriver := string(cd.LogConfiguration.LogDriver)
		options := make(map[string]any)
		if cd.LogConfiguration.Options != nil {
			for k, v := range cd.LogConfiguration.Options {
				options[k] = v
			}
		}
		mqlLogConfig, err := CreateResource(a.MqlRuntime, ResourceAwsEcsTaskDefinitionContainerDefinitionLogConfiguration,
			map[string]*llx.RawData{
				"__id":      llx.StringData(taskDefArn + "/container/" + name + "/logConfiguration"),
				"logDriver": llx.StringData(logDriver),
				"options":   llx.MapData(options, types.String),
			})
		if err != nil {
			return nil, err
		}
		logConfig = mqlLogConfig
	} else {
		// Create empty log configuration
		mqlLogConfig, err := CreateResource(a.MqlRuntime, ResourceAwsEcsTaskDefinitionContainerDefinitionLogConfiguration,
			map[string]*llx.RawData{
				"__id":      llx.StringData(taskDefArn + "/container/" + name + "/logConfiguration"),
				"logDriver": llx.StringData(""),
				"options":   llx.MapData(map[string]any{}, types.String),
			})
		if err != nil {
			return nil, err
		}
		logConfig = mqlLogConfig
	}

	// Create port mappings
	portMappings := []any{}
	if cd.PortMappings != nil {
		for _, pm := range cd.PortMappings {
			containerPort := int64(0)
			if pm.ContainerPort != nil {
				containerPort = int64(*pm.ContainerPort)
			}
			hostPort := int64(0)
			if pm.HostPort != nil {
				hostPort = int64(*pm.HostPort)
			}
			protocol := string(pm.Protocol)
			mqlPortMapping, err := CreateResource(a.MqlRuntime, ResourceAwsEcsTaskDefinitionContainerDefinitionPortMapping,
				map[string]*llx.RawData{
					"__id":          llx.StringData(fmt.Sprintf("%s/container/%s/port/%d", taskDefArn, name, containerPort)),
					"containerPort": llx.IntData(containerPort),
					"hostPort":      llx.IntData(hostPort),
					"protocol":      llx.StringData(protocol),
				})
			if err != nil {
				return nil, err
			}
			portMappings = append(portMappings, mqlPortMapping)
		}
	}

	// Type assert logConfig to Resource
	logConfigResource, ok := logConfig.(plugin.Resource)
	if !ok {
		return nil, errors.New("failed to convert logConfig to Resource")
	}

	return CreateResource(a.MqlRuntime, ResourceAwsEcsTaskDefinitionContainerDefinition,
		map[string]*llx.RawData{
			"__id":                   llx.StringData(taskDefArn + "/container/" + name),
			"name":                   llx.StringData(name),
			"image":                  llx.StringData(image),
			"privileged":             llx.BoolData(privileged),
			"readonlyRootFilesystem": llx.BoolData(readonlyRootFilesystem),
			"user":                   llx.StringData(user),
			"environment":            llx.ArrayData(envVars, types.Resource("aws.ecs.taskDefinition.containerDefinition.environmentVariable")),
			"secrets":                llx.ArrayData(secrets, types.Resource("aws.ecs.taskDefinition.containerDefinition.secret")),
			"logConfiguration":       llx.ResourceData(logConfigResource, "aws.ecs.taskDefinition.containerDefinition.logConfiguration"),
			"memory":                 llx.IntData(memory),
			"cpu":                    llx.IntData(cpu),
			"portMappings":           llx.ArrayData(portMappings, types.Resource("aws.ecs.taskDefinition.containerDefinition.portMapping")),
		})
}

func (a *mqlAwsEcs) createVolumeResource(vol *ecstypes.Volume) (any, error) {
	volName := ""
	if vol.Name != nil {
		volName = *vol.Name
	}

	// Create EFS volume configuration
	var efsVolConfig any
	if vol.EfsVolumeConfiguration != nil {
		efsConfig := vol.EfsVolumeConfiguration
		fileSystemId := ""
		if efsConfig.FileSystemId != nil {
			fileSystemId = *efsConfig.FileSystemId
		}
		rootDirectory := ""
		if efsConfig.RootDirectory != nil {
			rootDirectory = *efsConfig.RootDirectory
		}
		transitEncryption := string(efsConfig.TransitEncryption)
		transitEncryptionPort := int64(0)
		if efsConfig.TransitEncryptionPort != nil {
			transitEncryptionPort = int64(*efsConfig.TransitEncryptionPort)
		}

		// Create authorization config
		var authConfig any
		if efsConfig.AuthorizationConfig != nil {
			accessPointId := ""
			if efsConfig.AuthorizationConfig.AccessPointId != nil {
				accessPointId = *efsConfig.AuthorizationConfig.AccessPointId
			}
			iam := string(efsConfig.AuthorizationConfig.Iam)
			mqlAuthConfig, err := CreateResource(a.MqlRuntime, "aws.ecs.taskDefinition.volume.efsVolumeConfiguration.authorizationConfig",
				map[string]*llx.RawData{
					"__id":          llx.StringData(volName + "/efs/auth"),
					"accessPointId": llx.StringData(accessPointId),
					"iam":           llx.StringData(iam),
				})
			if err != nil {
				return nil, err
			}
			authConfig = mqlAuthConfig
		} else {
			// Create empty authorization config
			mqlAuthConfig, err := CreateResource(a.MqlRuntime, "aws.ecs.taskDefinition.volume.efsVolumeConfiguration.authorizationConfig",
				map[string]*llx.RawData{
					"__id":          llx.StringData(volName + "/efs/auth"),
					"accessPointId": llx.StringData(""),
					"iam":           llx.StringData(""),
				})
			if err != nil {
				return nil, err
			}
			authConfig = mqlAuthConfig
		}

		// Type assert authConfig to Resource
		authConfigResource, ok := authConfig.(plugin.Resource)
		if !ok {
			return nil, errors.New("failed to convert authConfig to Resource")
		}
		mqlEfsConfig, err := CreateResource(a.MqlRuntime, "aws.ecs.taskDefinition.volume.efsVolumeConfiguration",
			map[string]*llx.RawData{
				"__id":                  llx.StringData(volName + "/efs"),
				"fileSystemId":          llx.StringData(fileSystemId),
				"rootDirectory":         llx.StringData(rootDirectory),
				"transitEncryption":     llx.StringData(transitEncryption),
				"transitEncryptionPort": llx.IntData(transitEncryptionPort),
				"authorizationConfig":   llx.ResourceData(authConfigResource, "aws.ecs.taskDefinition.volume.efsVolumeConfiguration.authorizationConfig"),
			})
		if err != nil {
			return nil, err
		}
		efsVolConfig = mqlEfsConfig
	} else {
		// Create empty EFS config
		// Create empty authorization config for empty EFS config
		emptyAuthConfig, err := CreateResource(a.MqlRuntime, "aws.ecs.taskDefinition.volume.efsVolumeConfiguration.authorizationConfig",
			map[string]*llx.RawData{
				"__id":          llx.StringData(volName + "/efs/auth"),
				"accessPointId": llx.StringData(""),
				"iam":           llx.StringData(""),
			})
		if err != nil {
			return nil, err
		}
		mqlEfsConfig, err := CreateResource(a.MqlRuntime, "aws.ecs.taskDefinition.volume.efsVolumeConfiguration",
			map[string]*llx.RawData{
				"__id":                  llx.StringData(volName + "/efs"),
				"fileSystemId":          llx.StringData(""),
				"rootDirectory":         llx.StringData(""),
				"transitEncryption":     llx.StringData(""),
				"transitEncryptionPort": llx.IntData(0),
				"authorizationConfig":   llx.ResourceData(emptyAuthConfig, "aws.ecs.taskDefinition.volume.efsVolumeConfiguration.authorizationConfig"),
			})
		if err != nil {
			return nil, err
		}
		efsVolConfig = mqlEfsConfig
	}

	// Create host volume configuration
	var hostConfig any
	if vol.Host != nil {
		sourcePath := ""
		if vol.Host.SourcePath != nil {
			sourcePath = *vol.Host.SourcePath
		}
		mqlHost, err := CreateResource(a.MqlRuntime, "aws.ecs.taskDefinition.volume.host",
			map[string]*llx.RawData{
				"__id":       llx.StringData(volName + "/host"),
				"sourcePath": llx.StringData(sourcePath),
			})
		if err != nil {
			return nil, err
		}
		hostConfig = mqlHost
	} else {
		// Create empty host config
		mqlHost, err := CreateResource(a.MqlRuntime, "aws.ecs.taskDefinition.volume.host",
			map[string]*llx.RawData{
				"__id":       llx.StringData(volName + "/host"),
				"sourcePath": llx.StringData(""),
			})
		if err != nil {
			return nil, err
		}
		hostConfig = mqlHost
	}

	// Create docker volume configuration
	var dockerConfig any
	if vol.DockerVolumeConfiguration != nil {
		dockerVolConfig := vol.DockerVolumeConfiguration
		scope := string(dockerVolConfig.Scope)
		autoprovision := false
		if dockerVolConfig.Autoprovision != nil {
			autoprovision = *dockerVolConfig.Autoprovision
		}
		driver := ""
		if dockerVolConfig.Driver != nil {
			driver = *dockerVolConfig.Driver
		}
		driverOpts := make(map[string]any)
		if dockerVolConfig.DriverOpts != nil {
			for k, v := range dockerVolConfig.DriverOpts {
				driverOpts[k] = v
			}
		}
		labels := make(map[string]any)
		if dockerVolConfig.Labels != nil {
			for k, v := range dockerVolConfig.Labels {
				labels[k] = v
			}
		}
		mqlDocker, err := CreateResource(a.MqlRuntime, "aws.ecs.taskDefinition.volume.dockerVolumeConfiguration",
			map[string]*llx.RawData{
				"__id":          llx.StringData(volName + "/docker"),
				"scope":         llx.StringData(scope),
				"autoprovision": llx.BoolData(autoprovision),
				"driver":        llx.StringData(driver),
				"driverOpts":    llx.MapData(driverOpts, types.String),
				"labels":        llx.MapData(labels, types.String),
			})
		if err != nil {
			return nil, err
		}
		dockerConfig = mqlDocker
	} else {
		// Create empty docker config
		mqlDocker, err := CreateResource(a.MqlRuntime, "aws.ecs.taskDefinition.volume.dockerVolumeConfiguration",
			map[string]*llx.RawData{
				"__id":          llx.StringData(volName + "/docker"),
				"scope":         llx.StringData(""),
				"autoprovision": llx.BoolData(false),
				"driver":        llx.StringData(""),
				"driverOpts":    llx.MapData(map[string]any{}, types.String),
				"labels":        llx.MapData(map[string]any{}, types.String),
			})
		if err != nil {
			return nil, err
		}
		dockerConfig = mqlDocker
	}

	// Type assert volume configs to Resource
	efsVolConfigResource, ok := efsVolConfig.(plugin.Resource)
	if !ok {
		return nil, errors.New("failed to convert efsVolConfig to Resource")
	}
	hostConfigResource, ok := hostConfig.(plugin.Resource)
	if !ok {
		return nil, errors.New("failed to convert hostConfig to Resource")
	}
	dockerConfigResource, ok := dockerConfig.(plugin.Resource)
	if !ok {
		return nil, errors.New("failed to convert dockerConfig to Resource")
	}

	return CreateResource(a.MqlRuntime, "aws.ecs.taskDefinition.volume",
		map[string]*llx.RawData{
			"__id":                      llx.StringData(volName),
			"name":                      llx.StringData(volName),
			"efsVolumeConfiguration":    llx.ResourceData(efsVolConfigResource, "aws.ecs.taskDefinition.volume.efsVolumeConfiguration"),
			"host":                      llx.ResourceData(hostConfigResource, "aws.ecs.taskDefinition.volume.host"),
			"dockerVolumeConfiguration": llx.ResourceData(dockerConfigResource, "aws.ecs.taskDefinition.volume.dockerVolumeConfiguration"),
		})
}

func (a *mqlAwsEcs) createEphemeralStorageResource(es *ecstypes.EphemeralStorage) (any, error) {
	sizeInGiB := int64(es.SizeInGiB)

	return CreateResource(a.MqlRuntime, "aws.ecs.taskDefinition.ephemeralStorage",
		map[string]*llx.RawData{
			"__id":      llx.StringData("ephemeralStorage"),
			"sizeInGiB": llx.IntData(sizeInGiB),
		})
}

// Getter methods for task definition resources
func (a *mqlAwsEcsTaskDefinition) containerDefinitions() ([]any, error) {
	if !a.ContainerDefinitions.IsSet() {
		return nil, nil
	}
	if a.ContainerDefinitions.Error != nil {
		return nil, a.ContainerDefinitions.Error
	}
	return a.ContainerDefinitions.Data, nil
}

func (a *mqlAwsEcsTaskDefinition) volumes() ([]any, error) {
	if !a.Volumes.IsSet() {
		return nil, nil
	}
	if a.Volumes.Error != nil {
		return nil, a.Volumes.Error
	}
	return a.Volumes.Data, nil
}

func (a *mqlAwsEcsTaskDefinition) ephemeralStorage() (*mqlAwsEcsTaskDefinitionEphemeralStorage, error) {
	if !a.EphemeralStorage.IsSet() {
		return nil, nil
	}
	if a.EphemeralStorage.Error != nil {
		return nil, a.EphemeralStorage.Error
	}
	return a.EphemeralStorage.Data, nil
}

// id() methods for task definition resources
func (a *mqlAwsEcsTaskDefinition) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsEcsTaskDefinitionContainerDefinition) environment() ([]any, error) {
	if !a.Environment.IsSet() {
		return nil, nil
	}
	if a.Environment.Error != nil {
		return nil, a.Environment.Error
	}
	return a.Environment.Data, nil
}

func (a *mqlAwsEcsTaskDefinitionContainerDefinition) secrets() ([]any, error) {
	if !a.Secrets.IsSet() {
		return nil, nil
	}
	if a.Secrets.Error != nil {
		return nil, a.Secrets.Error
	}
	return a.Secrets.Data, nil
}

func (a *mqlAwsEcsTaskDefinitionContainerDefinition) logConfiguration() (*mqlAwsEcsTaskDefinitionContainerDefinitionLogConfiguration, error) {
	if !a.LogConfiguration.IsSet() {
		return nil, nil
	}
	if a.LogConfiguration.Error != nil {
		return nil, a.LogConfiguration.Error
	}
	return a.LogConfiguration.Data, nil
}

func (a *mqlAwsEcsTaskDefinitionContainerDefinition) portMappings() ([]any, error) {
	if !a.PortMappings.IsSet() {
		return nil, nil
	}
	if a.PortMappings.Error != nil {
		return nil, a.PortMappings.Error
	}
	return a.PortMappings.Data, nil
}

func (a *mqlAwsEcsTaskDefinitionContainerDefinition) id() (string, error) {
	return a.Name.Data, nil
}

func (a *mqlAwsEcsTaskDefinitionVolume) efsVolumeConfiguration() (*mqlAwsEcsTaskDefinitionVolumeEfsVolumeConfiguration, error) {
	if !a.EfsVolumeConfiguration.IsSet() {
		return nil, nil
	}
	if a.EfsVolumeConfiguration.Error != nil {
		return nil, a.EfsVolumeConfiguration.Error
	}
	return a.EfsVolumeConfiguration.Data, nil
}

func (a *mqlAwsEcsTaskDefinitionVolume) host() (*mqlAwsEcsTaskDefinitionVolumeHost, error) {
	if !a.Host.IsSet() {
		return nil, nil
	}
	if a.Host.Error != nil {
		return nil, a.Host.Error
	}
	return a.Host.Data, nil
}

func (a *mqlAwsEcsTaskDefinitionVolume) dockerVolumeConfiguration() (*mqlAwsEcsTaskDefinitionVolumeDockerVolumeConfiguration, error) {
	if !a.DockerVolumeConfiguration.IsSet() {
		return nil, nil
	}
	if a.DockerVolumeConfiguration.Error != nil {
		return nil, a.DockerVolumeConfiguration.Error
	}
	return a.DockerVolumeConfiguration.Data, nil
}

func (a *mqlAwsEcsTaskDefinitionVolume) id() (string, error) {
	return a.Name.Data, nil
}

func (a *mqlAwsEcsTaskDefinitionEphemeralStorage) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAwsEcsTaskDefinitionContainerDefinitionEnvironmentVariable) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAwsEcsTaskDefinitionContainerDefinitionSecret) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAwsEcsTaskDefinitionContainerDefinitionLogConfiguration) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAwsEcsTaskDefinitionContainerDefinitionPortMapping) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAwsEcsTaskDefinitionVolumeEfsVolumeConfiguration) authorizationConfig() (*mqlAwsEcsTaskDefinitionVolumeEfsVolumeConfigurationAuthorizationConfig, error) {
	if !a.AuthorizationConfig.IsSet() {
		return nil, nil
	}
	if a.AuthorizationConfig.Error != nil {
		return nil, a.AuthorizationConfig.Error
	}
	return a.AuthorizationConfig.Data, nil
}

func (a *mqlAwsEcsTaskDefinitionVolumeEfsVolumeConfiguration) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAwsEcsTaskDefinitionVolumeEfsVolumeConfigurationAuthorizationConfig) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAwsEcsTaskDefinitionVolumeHost) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAwsEcsTaskDefinitionVolumeDockerVolumeConfiguration) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAwsEcsCluster) services() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	clusterArn := a.Arn.Data
	region := ""
	if arn.IsARN(clusterArn) {
		if val, err := arn.Parse(clusterArn); err == nil {
			region = val.Region
		}
	}
	svc := conn.Ecs(region)
	ctx := context.Background()
	res := []any{}

	// List services in this cluster
	serviceParams := &ecsservice.ListServicesInput{
		Cluster: &clusterArn,
	}
	servicePaginator := ecsservice.NewListServicesPaginator(svc, serviceParams)
	serviceArns := []string{}
	for servicePaginator.HasMorePages() {
		serviceResp, err := servicePaginator.NextPage(ctx)
		if err != nil {
			if Is400AccessDeniedError(err) {
				log.Warn().Str("region", region).Str("cluster", clusterArn).Msg("error accessing cluster for services")
				return res, nil
			}
			return nil, errors.Wrap(err, "could not gather ecs services information")
		}
		serviceArns = append(serviceArns, serviceResp.ServiceArns...)
	}

	// Describe services in batches (AWS allows up to 10 services per DescribeServices call)
	batchSize := 10
	for i := 0; i < len(serviceArns); i += batchSize {
		end := i + batchSize
		if end > len(serviceArns) {
			end = len(serviceArns)
		}
		batch := serviceArns[i:end]

		describeParams := &ecsservice.DescribeServicesInput{
			Cluster:  &clusterArn,
			Services: batch,
			Include:  []ecstypes.ServiceField{ecstypes.ServiceFieldTags},
		}
		describeResp, err := svc.DescribeServices(ctx, describeParams)
		if err != nil {
			if Is400AccessDeniedError(err) {
				log.Warn().Str("region", region).Str("cluster", clusterArn).Msg("error describing services")
				continue
			}
			return nil, errors.Wrap(err, "could not describe ecs services")
		}

		for _, service := range describeResp.Services {
			mqlService, err := NewResource(a.MqlRuntime, ResourceAwsEcsService,
				map[string]*llx.RawData{
					"arn": llx.StringData(convert.ToValue(service.ServiceArn)),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlService)
		}
	}

	return res, nil
}

func (s *mqlAwsEcsService) id() (string, error) {
	return s.Arn.Data, nil
}

func (s *mqlAwsEcsService) deploymentConfiguration() (*mqlAwsEcsServiceDeploymentConfiguration, error) {
	if !s.DeploymentConfiguration.IsSet() {
		return nil, errors.New("deploymentConfiguration not initialized")
	}
	if s.DeploymentConfiguration.Error != nil {
		return nil, s.DeploymentConfiguration.Error
	}
	return s.DeploymentConfiguration.Data, nil
}

func (s *mqlAwsEcsService) networkConfiguration() (*mqlAwsEcsServiceNetworkConfiguration, error) {
	if !s.NetworkConfiguration.IsSet() {
		return nil, errors.New("networkConfiguration not initialized")
	}
	if s.NetworkConfiguration.Error != nil {
		return nil, s.NetworkConfiguration.Error
	}
	return s.NetworkConfiguration.Data, nil
}

func (d *mqlAwsEcsServiceDeploymentConfiguration) deploymentCircuitBreaker() (*mqlAwsEcsServiceDeploymentConfigurationDeploymentCircuitBreaker, error) {
	if !d.DeploymentCircuitBreaker.IsSet() {
		return nil, errors.New("deploymentCircuitBreaker not initialized")
	}
	if d.DeploymentCircuitBreaker.Error != nil {
		return nil, d.DeploymentCircuitBreaker.Error
	}
	return d.DeploymentCircuitBreaker.Data, nil
}

func (n *mqlAwsEcsServiceNetworkConfiguration) awsVpcConfiguration() (*mqlAwsEcsServiceNetworkConfigurationAwsVpcConfiguration, error) {
	if !n.AwsVpcConfiguration.IsSet() {
		return nil, errors.New("awsVpcConfiguration not initialized")
	}
	if n.AwsVpcConfiguration.Error != nil {
		return nil, n.AwsVpcConfiguration.Error
	}
	return n.AwsVpcConfiguration.Data, nil
}

func initAwsEcsService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch ecs service")
	}
	a := args["arn"].Value.(string)
	conn := runtime.Connection.(*connection.AwsConnection)

	// Validate and parse ARN if provided
	parsedARN, err := validateAndParseARN(a, "ecs")
	if err != nil {
		return nil, nil, err
	}

	region := parsedARN.Region
	clusterName := ""
	serviceName := ""
	if res := strings.Split(parsedARN.Resource, "/"); len(res) >= 2 {
		clusterName = res[1]
		if len(res) >= 3 {
			serviceName = res[2]
		}
	}

	// Extract service name from ARN

	clusterArn := fmt.Sprintf("arn:aws:ecs:%s:%s:cluster/%s", region, parsedARN.AccountID, clusterName)

	svc := conn.Ecs(region)
	ctx := context.Background()

	serviceDetails, err := svc.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  &clusterArn,
		Services: []string{serviceName},
		Include:  []ecstypes.ServiceField{ecstypes.ServiceFieldTags},
	})
	if err != nil {
		return nil, nil, err
	}
	if len(serviceDetails.Services) != 1 {
		return nil, nil, errors.Newf("only expected one service, got %d", len(serviceDetails.Services))
	}

	s := serviceDetails.Services[0]

	// Create deployment configuration resource
	var deploymentConfigResource any
	if s.DeploymentConfiguration != nil {
		var err error
		deploymentConfigResource, err = createDeploymentConfigurationResource(runtime, s.DeploymentConfiguration, a)
		if err != nil {
			return nil, nil, err
		}
	}

	// Create network configuration resource
	var networkConfigResource any
	if s.NetworkConfiguration != nil {
		var err error
		networkConfigResource, err = createNetworkConfigurationResource(runtime, s.NetworkConfiguration, a)
		if err != nil {
			return nil, nil, err
		}
	}

	// Extract launch type
	launchType := ""
	if s.LaunchType != "" {
		launchType = string(s.LaunchType)
	}

	// Extract task definition ARN
	taskDefinition := ""
	if s.TaskDefinition != nil {
		taskDefinition = *s.TaskDefinition
	}

	// Extract createdBy
	createdBy := ""
	if s.CreatedBy != nil {
		createdBy = *s.CreatedBy
	}

	args["name"] = llx.StringDataPtr(s.ServiceName)
	args["clusterArn"] = llx.StringDataPtr(s.ClusterArn)
	args["status"] = llx.StringDataPtr(s.Status)
	args["desiredCount"] = llx.IntData(int64(s.DesiredCount))
	args["runningCount"] = llx.IntData(int64(s.RunningCount))
	args["taskDefinition"] = llx.StringData(taskDefinition)
	args["launchType"] = llx.StringData(launchType)
	// Always set deploymentConfiguration - AWS services should always have this, but handle nil case
	if deploymentConfigResource != nil {
		args["deploymentConfiguration"] = llx.ResourceData(deploymentConfigResource.(plugin.Resource), ResourceAwsEcsServiceDeploymentConfiguration)
	} else {
		// AWS should always return deploymentConfiguration, but if nil, set to nil explicitly
		args["deploymentConfiguration"] = llx.NilData
	}
	// Always set networkConfiguration - AWS services should always have this, but handle nil case
	if networkConfigResource != nil {
		args["networkConfiguration"] = llx.ResourceData(networkConfigResource.(plugin.Resource), ResourceAwsEcsServiceNetworkConfiguration)
	} else {
		// AWS should always return networkConfiguration, but if nil, set to nil explicitly
		args["networkConfiguration"] = llx.NilData
	}
	args["tags"] = llx.MapData(ecsTagsToMap(s.Tags), types.String)
	args["createdAt"] = llx.TimeDataPtr(s.CreatedAt)
	args["createdBy"] = llx.StringData(createdBy)

	return args, nil, nil
}

func createDeploymentConfigurationResource(runtime *plugin.Runtime, dc *ecstypes.DeploymentConfiguration, serviceArn string) (any, error) {
	// Create deployment circuit breaker resource
	var circuitBreakerResource any
	if dc.DeploymentCircuitBreaker != nil {
		cb, err := CreateResource(runtime, ResourceAwsEcsServiceDeploymentConfigurationDeploymentCircuitBreaker,
			map[string]*llx.RawData{
				"__id":     llx.StringData(serviceArn + "/deploymentCircuitBreaker"),
				"enable":   llx.BoolData(dc.DeploymentCircuitBreaker.Enable),
				"rollback": llx.BoolData(dc.DeploymentCircuitBreaker.Rollback),
			})
		if err != nil {
			return nil, err
		}
		circuitBreakerResource = cb
	}

	// Convert optional fields to dicts
	var alarmsDict map[string]any
	if dc.Alarms != nil {
		var err error
		alarmsDict, err = convert.JsonToDict(dc.Alarms)
		if err != nil {
			return nil, err
		}
	}

	var canaryConfigDict map[string]any
	if dc.CanaryConfiguration != nil {
		var err error
		canaryConfigDict, err = convert.JsonToDict(dc.CanaryConfiguration)
		if err != nil {
			return nil, err
		}
	}

	var lifecycleHooksDict map[string]any
	if dc.LifecycleHooks != nil {
		var err error
		lifecycleHooksDict, err = convert.JsonToDict(dc.LifecycleHooks)
		if err != nil {
			return nil, err
		}
	}

	var linearConfigDict map[string]any
	if dc.LinearConfiguration != nil {
		var err error
		linearConfigDict, err = convert.JsonToDict(dc.LinearConfiguration)
		if err != nil {
			return nil, err
		}
	}

	args := map[string]*llx.RawData{
		"__id":                  llx.StringData(serviceArn + "/deploymentConfiguration"),
		"maximumPercent":        llx.IntDataPtr(dc.MaximumPercent),
		"minimumHealthyPercent": llx.IntDataPtr(dc.MinimumHealthyPercent),
		"bakeTimeInMinutes":     llx.IntDataPtr(dc.BakeTimeInMinutes),
		"strategy":              llx.StringData(string(dc.Strategy)),
	}
	// Always set deploymentCircuitBreaker, even if nil
	if circuitBreakerResource != nil {
		args["deploymentCircuitBreaker"] = llx.ResourceData(circuitBreakerResource.(plugin.Resource), ResourceAwsEcsServiceDeploymentConfigurationDeploymentCircuitBreaker)
	} else {
		args["deploymentCircuitBreaker"] = llx.NilData
	}
	if alarmsDict != nil {
		args["alarms"] = llx.MapData(alarmsDict, types.String)
	}
	if canaryConfigDict != nil {
		args["canaryConfiguration"] = llx.MapData(canaryConfigDict, types.String)
	}
	if lifecycleHooksDict != nil {
		args["lifecycleHooks"] = llx.MapData(lifecycleHooksDict, types.String)
	}
	if linearConfigDict != nil {
		args["linearConfiguration"] = llx.MapData(linearConfigDict, types.String)
	}

	return CreateResource(runtime, ResourceAwsEcsServiceDeploymentConfiguration, args)
}

func createNetworkConfigurationResource(runtime *plugin.Runtime, nc *ecstypes.NetworkConfiguration, serviceArn string) (any, error) {
	// Create awsvpc configuration resource
	var awsvpcResource any
	if nc.AwsvpcConfiguration != nil {
		awsvpc := nc.AwsvpcConfiguration
		subnets := []any{}
		for _, subnet := range awsvpc.Subnets {
			subnets = append(subnets, subnet)
		}
		securityGroups := []any{}
		for _, sg := range awsvpc.SecurityGroups {
			securityGroups = append(securityGroups, sg)
		}
		awsvpcRes, err := CreateResource(runtime, ResourceAwsEcsServiceNetworkConfigurationAwsVpcConfiguration,
			map[string]*llx.RawData{
				"__id":           llx.StringData(serviceArn + "/networkConfiguration/awsVpc"),
				"subnets":        llx.ArrayData(subnets, types.String),
				"securityGroups": llx.ArrayData(securityGroups, types.String),
				"assignPublicIp": llx.StringData(string(awsvpc.AssignPublicIp)),
			})
		if err != nil {
			return nil, err
		}
		awsvpcResource = awsvpcRes
	}

	args := map[string]*llx.RawData{
		"__id": llx.StringData(serviceArn + "/networkConfiguration"),
	}
	// Always set awsVpcConfiguration, even if nil
	if awsvpcResource != nil {
		args["awsVpcConfiguration"] = llx.ResourceData(awsvpcResource.(plugin.Resource), ResourceAwsEcsServiceNetworkConfigurationAwsVpcConfiguration)
	} else {
		args["awsVpcConfiguration"] = llx.NilData
	}

	return CreateResource(runtime, ResourceAwsEcsServiceNetworkConfiguration, args)
}
