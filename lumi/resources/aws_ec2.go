package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/library/jobpool"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
)

const (
	ec2InstanceArnPattern   = "arn:aws:ec2:%s:%s:instance/%s"
	securityGroupArnPattern = "arn:aws:ec2:%s:%s:security-group/%s"
)

func (e *lumiAwsEc2) id() (string, error) {
	return "aws.ec2", nil
}

func ec2TagsToMap(tags []ec2.Tag) map[string]interface{} {
	var tagsMap map[string]interface{}

	if len(tags) > 0 {
		tagsMap := map[string]interface{}{}
		for i := range tags {
			tag := tags[i]
			tagsMap[toString(tag.Key)] = toString(tag.Value)
		}
	}

	return tagsMap
}

func (s *lumiAwsEc2Securitygroup) GetIsAttachedToNetworkInterface() (bool, error) {
	sgId, err := s.Id()
	if err != nil {
		return false, errors.Wrap(err, "unable to parse instance id")
	}
	region, err := s.Region()
	if err != nil {
		return false, errors.Wrap(err, "unable to parse instance id")
	}
	at, err := awstransport(s.Runtime.Motor.Transport)
	if err != nil {
		return false, nil
	}
	svc := at.Ec2(region)
	ctx := context.Background()

	networkinterfaces, err := svc.DescribeNetworkInterfacesRequest(&ec2.DescribeNetworkInterfacesInput{Filters: []ec2.Filter{
		{Name: aws.String("group-id"), Values: []string{sgId}},
	}}).Send(ctx)
	if err != nil {
		return false, err
	}
	if len(networkinterfaces.NetworkInterfaces) > 0 {
		return true, nil
	}
	return false, nil
}

func (s *lumiAwsEc2) getSecurityGroups() []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
	at, err := awstransport(s.Runtime.Motor.Transport)
	if err != nil {
		return []*jobpool.Job{&jobpool.Job{Err: err}} // return the error
	}

	account, err := at.Account()
	if err != nil {
		return []*jobpool.Job{&jobpool.Job{Err: err}} // return the error
	}

	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{&jobpool.Job{Err: err}} // return the error
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := at.Ec2(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			nextToken := aws.String("no_token_to_start_with")
			params := &ec2.DescribeSecurityGroupsInput{}
			for nextToken != nil {
				securityGroups, err := svc.DescribeSecurityGroupsRequest(params).Send(ctx)
				if err != nil {
					return nil, err
				}
				nextToken = securityGroups.NextToken
				if securityGroups.NextToken != nil {
					params.NextToken = nextToken
				}

				for i := range securityGroups.SecurityGroups {
					group := securityGroups.SecurityGroups[i]

					lumiIpPermissions := []interface{}{}
					for p := range group.IpPermissions {
						permission := group.IpPermissions[p]

						ipRanges := []interface{}{}
						for r := range permission.IpRanges {
							iprange := permission.IpRanges[r]
							if iprange.CidrIp != nil {
								ipRanges = append(ipRanges, *iprange.CidrIp)
							}
						}

						ipv6Ranges := []interface{}{}
						for r := range permission.Ipv6Ranges {
							iprange := permission.Ipv6Ranges[r]
							if iprange.CidrIpv6 != nil {
								ipRanges = append(ipRanges, *iprange.CidrIpv6)
							}
						}

						lumiSecurityGroupIpPermission, err := s.Runtime.CreateResource("aws.ec2.securitygroup.ippermission",
							"id", toString(group.GroupId)+"-"+strconv.Itoa(p),
							"fromPort", toInt64(permission.FromPort),
							"toPort", toInt64(permission.ToPort),
							"ipProtocol", toString(permission.IpProtocol),
							"ipRanges", ipRanges,
							"ipv6Ranges", ipv6Ranges,
							// prefixListIds
							// userIdGroupPairs
						)
						if err != nil {
							return nil, err
						}

						lumiIpPermissions = append(lumiIpPermissions, lumiSecurityGroupIpPermission)
					}

					// NOTE: this will create the resource and determine the data in its init method
					lumiVpc, err := s.Runtime.CreateResource("aws.vpc",
						"arn", fmt.Sprintf(vpcArnPattern, regionVal, account.ID, toString(group.VpcId)),
					)
					if err != nil {
						return nil, err
					}
					lumiS3SecurityGroup, err := s.Runtime.CreateResource("aws.ec2.securitygroup",
						"arn", fmt.Sprintf(securityGroupArnPattern, regionVal, account.ID, toString(group.GroupId)),
						"id", toString(group.GroupId),
						"name", toString(group.GroupName),
						"description", toString(group.Description),
						"tag", ec2TagsToMap(group.Tags),
						"vpc", lumiVpc,
						"ipPermissions", lumiIpPermissions,
						"ipPermissionsEgress", []interface{}{},
						"region", regionVal,
					)
					if err != nil {
						return nil, err
					}
					res = append(res, lumiS3SecurityGroup)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (s *lumiAwsEc2) GetSecurityGroups() ([]interface{}, error) {
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(s.getSecurityGroups(), 5)
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

type ebsEncryption struct {
	region                 string `json:"region"`
	ebsEncryptionByDefault bool   `json:"ebsEncryptionByDefault"`
}

func (s *lumiAwsEc2) GetEbsEncryptionByDefault() (map[string]interface{}, error) {
	res := make(map[string]interface{})
	poolOfJobs := jobpool.CreatePool(s.getEbsEncryptionPerRegion(), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		jobResult := poolOfJobs.Jobs[i].Result.(ebsEncryption)
		res[jobResult.region] = jobResult.ebsEncryptionByDefault
	}
	return res, nil
}

func (s *lumiAwsEc2) getEbsEncryptionPerRegion() []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
	at, err := awstransport(s.Runtime.Motor.Transport)
	if err != nil {
		return []*jobpool.Job{&jobpool.Job{Err: err}} // return the error
	}
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{&jobpool.Job{Err: err}} // return the error
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := at.Ec2(regionVal)
			ctx := context.Background()

			ebsEncryptionRes, err := svc.GetEbsEncryptionByDefaultRequest(&ec2.GetEbsEncryptionByDefaultInput{}).Send(ctx)
			if err != nil {
				return nil, err
			}
			structVal := ebsEncryption{
				region:                 regionVal,
				ebsEncryptionByDefault: toBool(ebsEncryptionRes.EbsEncryptionByDefault),
			}
			return jobpool.JobResult(structVal), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (s *lumiAwsEc2) GetInstances() ([]interface{}, error) {
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(s.getInstances(), 5)
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

func (s *lumiAwsEc2) getImdsv2Instances(ctx context.Context, svc *ec2.Client, filterName string) ([]ec2.Reservation, error) {
	res := []ec2.Reservation{}
	nextToken := aws.String("no_token_to_start_with")
	params := &ec2.DescribeInstancesInput{
		Filters: []ec2.Filter{
			{Name: &filterName, Values: []string{"required"}},
		},
	}
	for nextToken != nil {
		instances, err := svc.DescribeInstancesRequest(params).Send(ctx)
		if err != nil {
			return nil, err
		}
		nextToken = instances.NextToken
		if instances.NextToken != nil {
			params.NextToken = nextToken
		}
		res = append(res, instances.Reservations...)
	}
	return res, nil
}

func (s *lumiAwsEc2) getImdsv1Instances(ctx context.Context, svc *ec2.Client, filterName string) ([]ec2.Reservation, error) {
	res := []ec2.Reservation{}
	nextToken := aws.String("no_token_to_start_with")
	params := &ec2.DescribeInstancesInput{
		Filters: []ec2.Filter{
			{Name: &filterName, Values: []string{"optional"}},
		},
	}
	for nextToken != nil {
		instances, err := svc.DescribeInstancesRequest(params).Send(ctx)
		if err != nil {
			return nil, err
		}
		nextToken = instances.NextToken
		if instances.NextToken != nil {
			params.NextToken = nextToken
		}
		res = append(res, instances.Reservations...)
	}
	return res, nil
}

func (s *lumiAwsEc2) getInstances() []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
	at, err := awstransport(s.Runtime.Motor.Transport)
	if err != nil {
		return []*jobpool.Job{&jobpool.Job{Err: err}} // return the error
	}
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{&jobpool.Job{Err: err}} // return the error
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := at.Ec2(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			// the value for http tokens is not available on api output i've been able to find, so here
			// we make two calls to get the instances, one with the imdsv1 filter and another with the imdsv2 filter
			filterName := "metadata-options.http-tokens"
			imdsv2Instances, err := s.getImdsv2Instances(ctx, svc, filterName)
			if err != nil {
				return nil, err
			}
			res, err = s.gatherInstanceInfo(imdsv2Instances, 2, regionVal)
			if err != nil {
				return nil, err
			}

			imdsv1Instances, err := s.getImdsv1Instances(ctx, svc, filterName)
			if err != nil {
				return nil, err
			}
			imdsv1Res, err := s.gatherInstanceInfo(imdsv1Instances, 1, regionVal)
			if err != nil {
				return nil, err
			}
			res = append(res, imdsv1Res...)

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (s *lumiAwsEc2) gatherInstanceInfo(instances []ec2.Reservation, imdsvVersion int, regionVal string) ([]interface{}, error) {
	at, err := awstransport(s.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	account, err := at.Account()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	httpTokens := "required"
	if imdsvVersion == 1 {
		httpTokens = "optional"
	}
	for _, reservation := range instances {
		for _, instance := range reservation.Instances {
			lumiDevices := []interface{}{}
			for i := range instance.BlockDeviceMappings {
				device := instance.BlockDeviceMappings[i]
				deviceStatus, err := device.Ebs.Status.MarshalValue()
				if err != nil {
					return nil, err
				}
				lumiInstanceDevice, err := s.Runtime.CreateResource("aws.ec2.instance.device",
					"deleteOnTermination", toBool(device.Ebs.DeleteOnTermination),
					"status", deviceStatus,
					"volumeId", toString(device.Ebs.VolumeId),
					"deviceName", toString(device.DeviceName),
				)
				if err != nil {
					return nil, err
				}
				lumiDevices = append(lumiDevices, lumiInstanceDevice)
			}
			stateName, err := instance.State.Name.MarshalValue()
			if err != nil {
				return nil, err
			}
			detailedMonitoring, err := instance.Monitoring.State.MarshalValue()
			if err != nil {
				return nil, err
			}
			sgs := []interface{}{}
			for i := range instance.SecurityGroups {
				// NOTE: this will create the resource and determine the data in its init method
				lumiSg, err := s.Runtime.CreateResource("aws.ec2.securitygroup",
					"arn", fmt.Sprintf(securityGroupArnPattern, regionVal, account.ID, toString(instance.SecurityGroups[i].GroupId)),
				)
				if err != nil {
					return nil, err
				}
				sgs = append(sgs, lumiSg)
			}

			stateReason, err := jsonToDict(instance.StateReason)
			if err != nil {
				return nil, err
			}

			args := []interface{}{
				"arn", fmt.Sprintf(ec2InstanceArnPattern, regionVal, account.ID, toString(instance.InstanceId)),
				"instanceId", toString(instance.InstanceId),
				"region", regionVal,
				"publicIp", toString(instance.PublicIpAddress),
				"detailedMonitoring", detailedMonitoring,
				"httpTokens", httpTokens,
				"state", stateName,
				"deviceMappings", lumiDevices,
				"securityGroups", sgs,
				"publicDnsName", toString(instance.PublicDnsName),
				"stateReason", stateReason,
				"stateTransitionReason", toString(instance.StateTransitionReason),
			}

			// add vpc if there is one
			if instance.VpcId != nil {
				// NOTE: this will create the resource and determine the data in its init method
				lumiVpcResource, err := s.Runtime.CreateResource("aws.vpc",
					"arn", fmt.Sprintf(vpcArnPattern, regionVal, account.ID, toString(instance.VpcId)),
				)
				if err != nil {
					return nil, err
				}
				lumiVpc := lumiVpcResource.(AwsVpc)
				args = append(args, "vpc", lumiVpc)
			}

			lumiEc2Instance, err := s.Runtime.CreateResource("aws.ec2.instance", args...)
			if err != nil {
				return nil, err
			}
			res = append(res, lumiEc2Instance)
		}
	}
	return res, nil
}

func (s *lumiAwsEc2Securitygroup) id() (string, error) {
	return s.Arn()
}

func (p *lumiAwsEc2Securitygroup) init(args *lumi.Args) (*lumi.Args, AwsEc2Securitygroup, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if (*args)["arn"] == nil && (*args)["id"] == nil {
		return nil, nil, errors.New("arn or id required to fetch aws security group")
	}

	// load all security groups
	obj, err := p.Runtime.CreateResource("aws.ec2")
	if err != nil {
		return nil, nil, err
	}
	awsEc2 := obj.(AwsEc2)

	rawResources, err := awsEc2.SecurityGroups()
	if err != nil {
		return nil, nil, err
	}

	var match func(secGroup AwsEc2Securitygroup) bool

	if (*args)["arn"] != nil {
		arnVal := (*args)["arn"].(string)
		match = func(secGroup AwsEc2Securitygroup) bool {
			lumiSecArn, err := secGroup.Arn()
			if err != nil {
				log.Error().Err(err).Msg("security group is not properly initialized")
				return false
			}
			return lumiSecArn == arnVal
		}
	}

	if (*args)["id"] != nil {
		idVal := (*args)["id"].(string)
		match = func(secGroup AwsEc2Securitygroup) bool {
			lumiSecId, err := secGroup.Id()
			if err != nil {
				log.Error().Err(err).Msg("security group is not properly initialized")
				return false
			}
			return lumiSecId == idVal
		}
	}

	for i := range rawResources {
		securityGroup := rawResources[i].(AwsEc2Securitygroup)
		if match(securityGroup) {
			return args, securityGroup, nil
		}
	}

	return nil, nil, errors.New("security group does not exist")
}

func (s *lumiAwsEc2SecuritygroupIppermission) id() (string, error) {
	return s.Id()
}

func (s *lumiAwsEc2InstanceDevice) id() (string, error) {
	return s.VolumeId()
}

func (s *lumiAwsEc2Instance) id() (string, error) {
	return s.Arn()
}

func (s *lumiAwsEc2Instance) GetVpc() (interface{}, error) {
	// this indicated that no vpc is attached since we set the value when we construct the resource
	// we return nil here to make it easier for users to compare:
	// aws.ec2.instances.where(state != "terminated") { vpc != null }
	return nil, nil
}

func (s *lumiAwsEc2Instance) GetSsm() (interface{}, error) {
	instanceId, err := s.InstanceId()
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse instance id")
	}
	region, err := s.Region()
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse instance region")
	}
	at, err := awstransport(s.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	svc := at.Ssm(region)
	ctx := context.Background()
	resourceTypeFilter := "ResourceType"
	instanceIdFilter := "InstanceIds"
	params := &ssm.DescribeInstanceInformationInput{
		Filters: []ssm.InstanceInformationStringFilter{
			{Key: &resourceTypeFilter, Values: []string{"ManagedInstance"}},
			{Key: &instanceIdFilter, Values: []string{instanceId}},
		},
	}
	ssmInstanceInfo, err := svc.DescribeInstanceInformationRequest(params).Send(ctx)
	if err != nil {
		return nil, err
	}
	res, err := jsonToDict(ssmInstanceInfo)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *lumiAwsEc2Instance) GetPatchState() (interface{}, error) {
	var res interface{}
	instanceId, err := s.InstanceId()
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse instance id")
	}
	region, err := s.Region()
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse instance region")
	}
	at, err := awstransport(s.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	svc := at.Ssm(region)
	ctx := context.Background()

	ssmPatchInfo, err := svc.DescribeInstancePatchStatesRequest(&ssm.DescribeInstancePatchStatesInput{InstanceIds: []string{instanceId}}).Send(ctx)
	if err != nil {
		return nil, err
	}
	if len(ssmPatchInfo.InstancePatchStates) > 0 {
		if instanceId == toString(ssmPatchInfo.InstancePatchStates[0].InstanceId) {
			res, err = jsonToDict(ssmPatchInfo.InstancePatchStates[0])
			if err != nil {
				return nil, err
			}
		}
	}
	return res, nil
}

func (s *lumiAwsEc2Instance) GetInstanceStatus() (interface{}, error) {
	var res interface{}
	instanceId, err := s.InstanceId()
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse instance id")
	}
	region, err := s.Region()
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse instance region")
	}
	at, err := awstransport(s.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	svc := at.Ec2(region)
	ctx := context.Background()

	instanceStatus, err := svc.DescribeInstanceStatusRequest(&ec2.DescribeInstanceStatusInput{
		InstanceIds:         []string{instanceId},
		IncludeAllInstances: aws.Bool(true),
	}).Send(ctx)
	if err != nil {
		return nil, err
	}

	if len(instanceStatus.InstanceStatuses) > 0 {
		if instanceId == toString(instanceStatus.InstanceStatuses[0].InstanceId) {
			res, err = jsonToDict(instanceStatus.InstanceStatuses[0])
			if err != nil {
				return nil, err
			}
		}
	}

	return res, nil
}
