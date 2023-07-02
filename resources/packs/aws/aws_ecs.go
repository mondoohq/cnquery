package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"

	"errors"
	ecsservice "github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/rs/zerolog/log"
	aws_provider "go.mondoo.com/cnquery/motor/providers/aws"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/library/jobpool"
	"go.mondoo.com/cnquery/resources/packs/core"
	"go.mondoo.com/cnquery/stringx"
)

func (e *mqlAwsEcs) id() (string, error) {
	return "aws.ecs", nil
}

func (e *mqlAwsEcs) GetContainers() ([]interface{}, error) {
	obj, err := e.MotorRuntime.CreateResource("aws.ecs")
	if err != nil {
		return nil, err
	}
	ecs := obj.(AwsEcs)

	clusters, err := ecs.Clusters()
	if err != nil {
		return nil, err
	}
	containers := []interface{}{}

	for i := range clusters {
		tasks, err := clusters[i].(AwsEcsCluster).Tasks()
		if err != nil {
			return nil, err
		}
		for i := range tasks {
			c, err := tasks[i].(AwsEcsTask).Containers()
			if err != nil {
				return nil, err
			}
			containers = append(containers, c...)
		}
	}
	return containers, nil
}

func (e *mqlAwsEcs) GetContainerInstances() ([]interface{}, error) {
	obj, err := e.MotorRuntime.CreateResource("aws.ecs")
	if err != nil {
		return nil, err
	}
	ecs := obj.(AwsEcs)

	clusters, err := ecs.Clusters()
	if err != nil {
		return nil, err
	}
	containerInstances := []interface{}{}

	for i := range clusters {
		ci, err := clusters[i].(AwsEcsCluster).ContainerInstances()
		if err != nil {
			return nil, err
		}
		containerInstances = append(containerInstances, ci...)

	}
	return containerInstances, nil
}

func (e *mqlAwsEcsInstance) GetEc2Instance() ([]interface{}, error) {
	return nil, nil
}

func (ecs *mqlAwsEcs) GetClusters() ([]interface{}, error) {
	provider, err := awsProvider(ecs.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(ecs.getECSClusters(provider), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]interface{})...)
	}

	return res, nil
}

func (ecs *mqlAwsEcs) getECSClusters(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	var err error
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	log.Debug().Msgf("regions being called for ecs clusters list are: %v", regions)
	for ri := range regions {
		region := regions[ri]
		f := func() (jobpool.JobResult, error) {
			svc := provider.Ecs(region)
			ctx := context.Background()
			res := []interface{}{}

			nextToken := aws.String("no_token_to_start_with")
			params := &ecsservice.ListClustersInput{}
			for nextToken != nil {
				resp, err := svc.ListClusters(ctx, params)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, errors.Join(err, errors.New("could not gather ecs cluster information"))
				}
				nextToken = resp.NextToken
				if resp.NextToken != nil {
					params.NextToken = nextToken
				}
				for _, cluster := range resp.ClusterArns {
					mqlCluster, err := ecs.MotorRuntime.CreateResource("aws.ecs.cluster",
						"arn", cluster,
					)
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

func (e *mqlAwsEcsCluster) init(args *resources.Args) (*resources.Args, AwsEcsCluster, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if len(*args) == 0 {
		if ids := getAssetIdentifier(e.MqlResource().MotorRuntime); ids != nil {
			(*args)["arn"] = ids.arn
		}
	}

	if (*args)["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch ecs cluster")
	}
	a := (*args)["arn"].(string)
	provider, err := awsProvider(e.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}
	region := ""
	if arn.IsARN(a) {
		if val, err := arn.Parse(a); err == nil {
			region = val.Region
		}
	}
	svc := provider.Ecs(region)
	ctx := context.Background()
	clusterDetails, err := svc.DescribeClusters(ctx, &ecs.DescribeClustersInput{Clusters: []string{a}})
	if err != nil {
		return nil, nil, err
	}
	if len(clusterDetails.Clusters) != 1 {
		return nil, nil, errors.New(fmt.Sprintf("only expected one cluster, got %d", len(clusterDetails.Clusters)))
	}
	c := clusterDetails.Clusters[0]
	configuration, err := core.JsonToDict(c.Configuration)

	(*args)["name"] = core.ToString(c.ClusterName)
	(*args)["tags"] = ecsTags(c.Tags)
	(*args)["runningTasksCount"] = int64(c.RunningTasksCount)
	(*args)["pendingTasksCount"] = int64(c.PendingTasksCount)
	(*args)["registeredContainerInstancesCount"] = int64(c.RegisteredContainerInstancesCount)
	(*args)["configuration"] = configuration
	(*args)["status"] = core.ToString(c.Status)
	return args, nil, nil
}

func ecsTags(t []ecstypes.Tag) map[string]interface{} {
	res := map[string]interface{}{}
	for i := range t {
		tag := t[i]
		if tag.Key != nil && tag.Value != nil {
			res[*tag.Key] = *tag.Value
		}
	}
	return res
}

func (ecs *mqlAwsEcsCluster) GetContainerInstances() ([]interface{}, error) {
	provider, err := awsProvider(ecs.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	account, err := provider.Account()
	if err != nil {
		return nil, err
	}
	clustera, err := ecs.Arn()
	if err != nil {
		return nil, err
	}
	region := ""
	if arn.IsARN(clustera) {
		if val, err := arn.Parse(clustera); err == nil {
			region = val.Region
		}
	}
	svc := provider.Ecs(region)
	ctx := context.Background()
	res := []interface{}{}

	params := &ecsservice.ListContainerInstancesInput{Cluster: &clustera}
	containerInstances, err := svc.ListContainerInstances(ctx, params)
	if err != nil {
		log.Error().Err(err).Msg("cannot list container instances") // no fail
	} else if len(containerInstances.ContainerInstanceArns) > 0 {
		containerInstancesDetail, err := svc.DescribeContainerInstances(ctx, &ecsservice.DescribeContainerInstancesInput{Cluster: &clustera, ContainerInstances: containerInstances.ContainerInstanceArns})
		if err == nil {
			for _, ci := range containerInstancesDetail.ContainerInstances {
				// container instance assets
				args := []interface{}{
					"arn", core.ToString(ci.ContainerInstanceArn),
					"agentConnected", ci.AgentConnected,
					"id", core.ToString(ci.Ec2InstanceId),
					"capacityProvider", core.ToString(ci.CapacityProviderName),
					"region", region,
				}
				if strings.HasPrefix(core.ToString(ci.Ec2InstanceId), "i-") {
					mqlInstanceResource, err := ecs.MotorRuntime.CreateResource("aws.ec2.instance",
						"arn", fmt.Sprintf(ec2InstanceArnPattern, region, account, core.ToString(ci.Ec2InstanceId)),
					)
					if err == nil && mqlInstanceResource != nil {
						mqlInstance := mqlInstanceResource.(AwsEc2Instance)
						args = append(args, "ec2Instance", mqlInstance)
					}
				}

				mqlEcsInstance, err := ecs.MotorRuntime.CreateResource("aws.ecs.instance", args...)
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
	return s.Arn()
}

func (s *mqlAwsEcsCluster) id() (string, error) {
	return s.Arn()
}

func (ecs *mqlAwsEcsCluster) GetTasks() ([]interface{}, error) {
	provider, err := awsProvider(ecs.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	clustera, err := ecs.Arn()
	if err != nil {
		return nil, err
	}
	name, err := ecs.Name()
	if err != nil {
		return nil, err
	}
	region := ""
	if arn.IsARN(clustera) {
		if val, err := arn.Parse(clustera); err == nil {
			region = val.Region
		}
	}
	svc := provider.Ecs(region)
	ctx := context.Background()
	res := []interface{}{}

	nextToken := aws.String("no_token_to_start_with")
	params := &ecsservice.ListTasksInput{Cluster: &clustera}
	for nextToken != nil {
		resp, err := svc.ListTasks(ctx, params)
		if err != nil {
			return nil, errors.Join(err, errors.New("could not gather ecs tasks information"))
		}
		nextToken = resp.NextToken
		if resp.NextToken != nil {
			params.NextToken = nextToken
		}
		for _, task := range resp.TaskArns {
			mqlTask, err := ecs.MotorRuntime.CreateResource("aws.ecs.task",
				"arn", task,
				"clusterName", name,
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlTask)
		}
	}
	return res, nil
}

func (s *mqlAwsEcsTask) id() (string, error) {
	return s.Arn()
}

func (e *mqlAwsEcsTask) init(args *resources.Args) (*resources.Args, AwsEcsTask, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if len(*args) == 0 {
		if ids := getAssetIdentifier(e.MqlResource().MotorRuntime); ids != nil {
			(*args)["arn"] = ids.arn
		}
	}

	if (*args)["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch ecs task")
	}
	a := (*args)["arn"].(string)

	provider, err := awsProvider(e.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}
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
	svc := provider.Ecs(region)
	ctx := context.Background()
	params := &ecs.DescribeTasksInput{Tasks: []string{a}}
	params.Cluster = &clusterName
	taskDetails, err := svc.DescribeTasks(ctx, params)
	if err != nil {
		return nil, nil, err
	}
	if len(taskDetails.Tasks) != 1 {
		return nil, nil, errors.New(fmt.Sprintf("only expected one task, got %d", len(taskDetails.Tasks)))
	}
	t := taskDetails.Tasks[0]
	taskDefinitionArn := t.TaskDefinitionArn
	definition, err := svc.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{TaskDefinition: taskDefinitionArn})
	if err != nil {
		return nil, nil, err
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

	(*args)["connectivity"] = string(t.Connectivity)
	(*args)["lastStatus"] = core.ToString(t.LastStatus)
	(*args)["platformFamily"] = core.ToString(t.PlatformFamily)
	(*args)["platformVersion"] = core.ToString(t.PlatformVersion)
	(*args)["tags"] = ecsTags(t.Tags)

	containers := []interface{}{}
	pf, _ := e.PlatformFamily()
	pv, _ := e.PlatformVersion()

	for _, c := range t.Containers {
		cmds := []interface{}{}
		for i := range containerCommandMap[core.ToString(c.Name)] {
			cmds = append(cmds, containerCommandMap[core.ToString(c.Name)][i])
		}
		publicIp := getContainerIP(ctx, provider, t.Attachments, c, region)
		name := core.ToString(c.Name)
		if publicIp != "" {
			name = name + "-" + publicIp
		}

		mqlContainer, err := e.MotorRuntime.CreateResource("aws.ecs.container",
			"name", name,
			"platformFamily", pf,
			"platformVersion", pv,
			"status", core.ToString(c.LastStatus),
			"publicIp", publicIp,
			"arn", core.ToString(c.ContainerArn),
			"logDriver", containerLogDriverMap[core.ToString(c.Name)],
			"image", core.ToString(c.Image),
			"clusterName", clusterName,
			"taskDefinitionArn", core.ToString(taskDefinitionArn),
			"region", region,
			"command", cmds,
			"taskArn", core.ToString(t.TaskArn),
			"runtimeId", core.ToString(c.RuntimeId),
			"containerName", core.ToString(c.Name),
		)
		if err != nil {
			return args, nil, err
		}
		containers = append(containers, mqlContainer)
	}
	(*args)["containers"] = containers

	return args, nil, nil
}

func (e *mqlAwsEcsContainer) init(args *resources.Args) (*resources.Args, AwsEcsContainer, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}
	if len(*args) == 0 {
		if ids := getAssetIdentifier(e.MqlResource().MotorRuntime); ids != nil {
			(*args)["arn"] = ids.arn
		}
	}
	obj, err := e.MotorRuntime.CreateResource("aws.ecs")
	if err != nil {
		return nil, nil, err
	}
	ecs := obj.(AwsEcs)

	rawResources, err := ecs.Containers()
	if err != nil {
		return nil, nil, err
	}

	arnVal := (*args)["arn"].(string)
	for i := range rawResources {
		container := rawResources[i].(AwsEcsContainer)
		mqlCArn, err := container.Arn()
		if err != nil {
			return nil, nil, errors.New("ecs container does not exist")
		}
		if mqlCArn == arnVal {
			return args, container, nil
		}
	}

	return nil, nil, errors.New("container does not exist")
}

func getContainerIP(ctx context.Context, provider *aws_provider.Provider, attachments []ecstypes.Attachment, c ecstypes.Container, region string) string {
	containerAttachmentIds := []string{}
	for _, ca := range c.NetworkInterfaces {
		containerAttachmentIds = append(containerAttachmentIds, *ca.AttachmentId)
	}
	var publicIp string
	for _, a := range attachments {
		if stringx.Contains(containerAttachmentIds, *a.Id) {
			for _, detail := range a.Details {
				if *detail.Name == "networkInterfaceId" {
					publicIp = getPublicIpForContainer(ctx, provider, *detail.Value, region)
				}
			}
		}
	}
	return publicIp
}

func getPublicIpForContainer(ctx context.Context, provider *aws_provider.Provider, nii string, region string) string {
	svc := provider.Ec2(region)
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
	return s.Arn()
}
