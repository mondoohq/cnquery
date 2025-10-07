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

	region := ""
	if arn.IsARN(a) {
		if val, err := arn.Parse(a); err == nil {
			region = val.Region
		}
	}
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

	region := ""
	clusterName := ""
	if arn.IsARN(a) {
		if val, err := arn.Parse(a); err == nil {
			region = val.Region
			if res := strings.Split(val.Resource, "/"); len(res) == 3 {
				clusterName = res[1]
			}
		}
	}
	svc := conn.Ecs(region)
	ctx := context.Background()
	params := &ecs.DescribeTasksInput{Tasks: []string{a}, Cluster: &clusterName, Include: []ecstypes.TaskField{ecstypes.TaskFieldTags}}
	params.Cluster = &clusterName
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
