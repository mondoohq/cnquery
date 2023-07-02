package aws

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/aws/smithy-go"
	"github.com/rs/zerolog/log"
	aws_provider "go.mondoo.com/cnquery/motor/providers/aws"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/library/jobpool"
	"go.mondoo.com/cnquery/resources/packs/core"
)

const (
	ec2InstanceArnPattern   = "arn:aws:ec2:%s:%s:instance/%s"
	securityGroupArnPattern = "arn:aws:ec2:%s:%s:security-group/%s"
	volumeArnPattern        = "arn:aws:ec2:%s:%s:volume/%s"
	snapshotArnPattern      = "arn:aws:ec2:%s:%s:snapshot/%s"
	internetGwArnPattern    = "arn:aws:ec2:%s:%s:gateway/%s"
	vpnConnArnPattern       = "arn:aws:ec2:%s:%s:vpn-connection/%s"
	networkAclArnPattern    = "arn:aws:ec2:%s:%s:network-acl/%s"
	imageArnPattern         = "arn:aws:ec2:%s:%s:image/%s"
	keypairArnPattern       = "arn:aws:ec2:%s:%s:keypair/%s"
)

func (e *mqlAwsEc2) id() (string, error) {
	return "aws.ec2", nil
}

func Ec2TagsToMap(tags []types.Tag) map[string]interface{} {
	tagsMap := make(map[string]interface{})

	if len(tags) > 0 {
		for i := range tags {
			tag := tags[i]
			tagsMap[core.ToString(tag.Key)] = core.ToString(tag.Value)
		}
	}

	return tagsMap
}

func (s *mqlAwsEc2Networkacl) id() (string, error) {
	return s.Arn()
}

func (s *mqlAwsEc2) GetNetworkAcls() ([]interface{}, error) {
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(s.getNetworkACLs(provider), 5)
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

func (s *mqlAwsEc2) getNetworkACLs(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)

	account, err := provider.Account()
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}

	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := provider.Ec2(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			nextToken := aws.String("no_token_to_start_with")
			params := &ec2.DescribeNetworkAclsInput{}
			for nextToken != nil {
				networkAcls, err := svc.DescribeNetworkAcls(ctx, params)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				nextToken = networkAcls.NextToken
				if networkAcls.NextToken != nil {
					params.NextToken = nextToken
				}

				for i := range networkAcls.NetworkAcls {
					acl := networkAcls.NetworkAcls[i]
					mqlNetworkAcl, err := s.MotorRuntime.CreateResource("aws.ec2.networkacl",
						"arn", fmt.Sprintf(networkAclArnPattern, regionVal, account.ID, core.ToString(acl.NetworkAclId)),
						"id", core.ToString(acl.NetworkAclId),
						"region", regionVal,
					)
					if err != nil {
						return nil, err
					}

					res = append(res, mqlNetworkAcl)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (s *mqlAwsEc2NetworkaclEntry) id() (string, error) {
	return s.Id()
}

func (s *mqlAwsEc2NetworkaclEntryPortrange) id() (string, error) {
	return s.Id()
}

func (s *mqlAwsEc2Networkacl) GetEntries() ([]interface{}, error) {
	id, err := s.Id()
	if err != nil {
		return nil, errors.Join(err, errors.New("unable to parse id"))
	}
	region, err := s.Region()
	if err != nil {
		return nil, errors.Join(err, errors.New("unable to region"))
	}
	at, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	svc := at.Ec2(region)
	ctx := context.Background()
	networkacls, err := svc.DescribeNetworkAcls(ctx, &ec2.DescribeNetworkAclsInput{NetworkAclIds: []string{id}})
	if err != nil {
		return nil, err
	}

	if len(networkacls.NetworkAcls) == 0 {
		return nil, errors.New("aws network acl not found")
	}

	res := []interface{}{}
	for _, entry := range networkacls.NetworkAcls[0].Entries {
		args := []interface{}{
			"egress", entry.Egress,
			"ruleAction", string(entry.RuleAction),
			"id", id + "-" + strconv.Itoa(core.ToIntFrom32(entry.RuleNumber)),
		}
		if entry.PortRange != nil {
			mqlPortEntry, err := s.MotorRuntime.CreateResource("aws.ec2.networkacl.entry.portrange",
				"from", entry.PortRange.From,
				"to", entry.PortRange.To,
				"id", id+"-"+strconv.Itoa(core.ToIntFrom32(entry.RuleNumber))+"-"+strconv.Itoa(core.ToIntFrom32(entry.PortRange.From)),
			)
			if err != nil {
				return nil, err
			}
			args = append(args, mqlPortEntry)
		}

		mqlAclEntry, err := s.MotorRuntime.CreateResource("aws.ec2.networkacl.entry", args...)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAclEntry)
	}

	return res, nil
}

func (s *mqlAwsEc2NetworkaclEntry) GetPortRange() (interface{}, error) {
	return nil, nil
}

func (s *mqlAwsEc2Securitygroup) GetIsAttachedToNetworkInterface() (bool, error) {
	sgId, err := s.Id()
	if err != nil {
		return false, errors.Join(err, errors.New("unable to parse instance id"))
	}
	region, err := s.Region()
	if err != nil {
		return false, errors.Join(err, errors.New("unable to parse instance id"))
	}
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return false, nil
	}
	svc := provider.Ec2(region)
	ctx := context.Background()

	networkinterfaces, err := svc.DescribeNetworkInterfaces(ctx, &ec2.DescribeNetworkInterfacesInput{Filters: []types.Filter{
		{Name: aws.String("group-id"), Values: []string{sgId}},
	}})
	if err != nil {
		return false, err
	}
	if len(networkinterfaces.NetworkInterfaces) > 0 {
		return true, nil
	}
	return false, nil
}

func (s *mqlAwsEc2) getSecurityGroups(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)

	account, err := provider.Account()
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}

	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := provider.Ec2(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			nextToken := aws.String("no_token_to_start_with")
			params := &ec2.DescribeSecurityGroupsInput{}
			for nextToken != nil {
				securityGroups, err := svc.DescribeSecurityGroups(ctx, params)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				nextToken = securityGroups.NextToken
				if securityGroups.NextToken != nil {
					params.NextToken = nextToken
				}

				for i := range securityGroups.SecurityGroups {
					group := securityGroups.SecurityGroups[i]

					mqlIpPermissions := []interface{}{}
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
						mqlSecurityGroupIpPermission, err := s.MotorRuntime.CreateResource("aws.ec2.securitygroup.ippermission",
							"id", core.ToString(group.GroupId)+"-"+strconv.Itoa(p),
							"fromPort", core.ToInt64From32(permission.FromPort),
							"toPort", core.ToInt64From32(permission.ToPort),
							"ipProtocol", core.ToString(permission.IpProtocol),
							"ipRanges", ipRanges,
							"ipv6Ranges", ipv6Ranges,
							// prefixListIds
							// userIdGroupPairs
						)
						if err != nil {
							return nil, err
						}

						mqlIpPermissions = append(mqlIpPermissions, mqlSecurityGroupIpPermission)
					}

					// NOTE: this will create the resource and determine the data in its init method
					mqlVpc, err := s.MotorRuntime.CreateResource("aws.vpc",
						"arn", fmt.Sprintf(vpcArnPattern, regionVal, account.ID, core.ToString(group.VpcId)),
					)
					if err != nil {
						return nil, err
					}
					mqlS3SecurityGroup, err := s.MotorRuntime.CreateResource("aws.ec2.securitygroup",
						"arn", fmt.Sprintf(securityGroupArnPattern, regionVal, account.ID, core.ToString(group.GroupId)),
						"id", core.ToString(group.GroupId),
						"name", core.ToString(group.GroupName),
						"description", core.ToString(group.Description),
						"tags", Ec2TagsToMap(group.Tags),
						"vpc", mqlVpc,
						"ipPermissions", mqlIpPermissions,
						"ipPermissionsEgress", []interface{}{},
						"region", regionVal,
					)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlS3SecurityGroup)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (s *mqlAwsEc2) GetKeypairs() ([]interface{}, error) {
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(s.getKeypairs(provider), 5)
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

func (s *mqlAwsEc2Keypair) id() (string, error) {
	return s.Arn()
}

func (s *mqlAwsEc2) getKeypairs(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	account, err := provider.Account()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := provider.Ec2(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			params := &ec2.DescribeKeyPairsInput{}
			keyPairs, err := svc.DescribeKeyPairs(ctx, params)
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
					return res, nil
				}
				return nil, err
			}

			for i := range keyPairs.KeyPairs {
				kp := keyPairs.KeyPairs[i]
				mqlKeypair, err := s.MotorRuntime.CreateResource("aws.ec2.keypair",
					"arn", fmt.Sprintf(keypairArnPattern, account.ID, regionVal, core.ToString(kp.KeyPairId)),
					"fingerprint", core.ToString(kp.KeyFingerprint),
					"name", core.ToString(kp.KeyName),
					"type", string(kp.KeyType),
					"tags", Ec2TagsToMap(kp.Tags),
					"region", regionVal,
				)
				if err != nil {
					return nil, err
				}
				res = append(res, mqlKeypair)
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (i *mqlAwsEc2Keypair) init(args *resources.Args) (*resources.Args, AwsEc2Keypair, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if (*args)["name"] == nil {
		return nil, nil, errors.New("name required to fetch aws ec2 keypair")
	}
	n := (*args)["name"].(string)
	if n == "" {
		return nil, nil, errors.New("ec2 keypair name cannot be empty")
	}
	if (*args)["region"] == nil {
		return nil, nil, errors.New("region required to fetch aws ec2 keypair")
	}
	r := (*args)["region"].(string)

	provider, err := awsProvider(i.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}
	account, err := provider.Account()
	if err != nil {
		return nil, nil, err
	}
	svc := provider.Ec2(r)
	ctx := context.Background()
	kps, err := svc.DescribeKeyPairs(ctx, &ec2.DescribeKeyPairsInput{KeyNames: []string{n}})
	if err != nil {
		// it is quite common for instances to get created with a keypair and then that keypair be deleted
		var ae smithy.APIError
		if errors.As(err, &ae) {
			if ae.ErrorCode() == "InvalidKeyPair.NotFound" {
				log.Warn().Msgf("key %s does not exist in %s region", n, r)
				return args, nil, nil
			}
		}
		log.Error().Err(err).Msg("cannot fetch keypair")
		return nil, nil, err
	}

	if len(kps.KeyPairs) > 0 {
		kp := kps.KeyPairs[0]
		(*args)["fingerprint"] = core.ToString(kp.KeyFingerprint)
		(*args)["name"] = core.ToString(kp.KeyName)
		(*args)["type"] = string(kp.KeyType)
		(*args)["tags"] = Ec2TagsToMap(kp.Tags)
		(*args)["region"] = r
		(*args)["arn"] = fmt.Sprintf(keypairArnPattern, account.ID, r, core.ToString(kp.KeyPairId))
		return args, nil, nil
	}

	(*args)["fingerprint"] = ""
	(*args)["name"] = n
	(*args)["type"] = ""
	(*args)["tags"] = ""
	(*args)["region"] = r
	(*args)["arn"] = ""
	return args, nil, nil
}

func (s *mqlAwsEc2) GetSecurityGroups() ([]interface{}, error) {
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(s.getSecurityGroups(provider), 5)
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
	region                 string
	ebsEncryptionByDefault bool
}

func (s *mqlAwsEc2) GetEbsEncryptionByDefault() (map[string]interface{}, error) {
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := make(map[string]interface{})
	poolOfJobs := jobpool.CreatePool(s.getEbsEncryptionPerRegion(provider), 5)
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

func (s *mqlAwsEc2) getEbsEncryptionPerRegion(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)

	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := provider.Ec2(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			ebsEncryptionRes, err := svc.GetEbsEncryptionByDefault(ctx, &ec2.GetEbsEncryptionByDefaultInput{})
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
					return res, nil
				}
				return nil, err
			}
			structVal := ebsEncryption{
				region:                 regionVal,
				ebsEncryptionByDefault: core.ToBool(ebsEncryptionRes.EbsEncryptionByDefault),
			}
			return jobpool.JobResult(structVal), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (s *mqlAwsEc2) GetInstances() ([]interface{}, error) {
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(s.getInstances(provider), 5)
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

func (s *mqlAwsEc2) getImdsv2Instances(ctx context.Context, svc *ec2.Client, filterName string) ([]types.Reservation, error) {
	res := []types.Reservation{}
	nextToken := aws.String("no_token_to_start_with")
	params := &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{Name: &filterName, Values: []string{"required"}},
		},
	}
	for nextToken != nil {
		instances, err := svc.DescribeInstances(ctx, params)
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

func (s *mqlAwsEc2) getImdsv1Instances(ctx context.Context, svc *ec2.Client, filterName string) ([]types.Reservation, error) {
	res := []types.Reservation{}
	nextToken := aws.String("no_token_to_start_with")
	params := &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{Name: &filterName, Values: []string{"optional"}},
		},
	}
	for nextToken != nil {
		instances, err := svc.DescribeInstances(ctx, params)
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

func (s *mqlAwsEc2) getInstances(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := provider.Ec2(regionVal)
			ctx := context.Background()
			var res []interface{}

			// the value for http tokens is not available on api output i've been able to find, so here
			// we make two calls to get the instances, one with the imdsv1 filter and another with the imdsv2 filter
			filterName := "metadata-options.http-tokens"
			imdsv2Instances, err := s.getImdsv2Instances(ctx, svc, filterName)
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
					return res, nil
				}
				return nil, err
			}
			res, err = s.gatherInstanceInfo(imdsv2Instances, 2, regionVal)
			if err != nil {
				return nil, err
			}

			imdsv1Instances, err := s.getImdsv1Instances(ctx, svc, filterName)
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
					return res, nil
				}
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

func (s *mqlAwsEc2) gatherInstanceInfo(instances []types.Reservation, imdsvVersion int, regionVal string) ([]interface{}, error) {
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	account, err := provider.Account()
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
			mqlDevices := []interface{}{}
			for i := range instance.BlockDeviceMappings {
				device := instance.BlockDeviceMappings[i]

				mqlInstanceDevice, err := s.MotorRuntime.CreateResource("aws.ec2.instance.device",
					"deleteOnTermination", core.ToBool(device.Ebs.DeleteOnTermination),
					"status", string(device.Ebs.Status),
					"volumeId", core.ToString(device.Ebs.VolumeId),
					"deviceName", core.ToString(device.DeviceName),
				)
				if err != nil {
					return nil, err
				}
				mqlDevices = append(mqlDevices, mqlInstanceDevice)
			}
			sgs := []interface{}{}
			for i := range instance.SecurityGroups {
				// NOTE: this will create the resource and determine the data in its init method
				mqlSg, err := s.MotorRuntime.CreateResource("aws.ec2.securitygroup",
					"arn", fmt.Sprintf(securityGroupArnPattern, regionVal, account.ID, core.ToString(instance.SecurityGroups[i].GroupId)),
				)
				if err != nil {
					return nil, err
				}
				sgs = append(sgs, mqlSg)
			}

			stateReason, err := core.JsonToDict(instance.StateReason)
			if err != nil {
				return nil, err
			}

			mqlImage, err := s.MotorRuntime.CreateResource("aws.ec2.image",
				"arn", fmt.Sprintf(imageArnPattern, regionVal, account.ID, core.ToString(instance.ImageId)),
			)
			if err != nil {
				return nil, err
			}

			args := []interface{}{
				"platformDetails", core.ToString(instance.PlatformDetails),
				"arn", fmt.Sprintf(ec2InstanceArnPattern, regionVal, account.ID, core.ToString(instance.InstanceId)),
				"instanceId", core.ToString(instance.InstanceId),
				"region", regionVal,
				"publicIp", core.ToString(instance.PublicIpAddress),
				"detailedMonitoring", string(instance.Monitoring.State),
				"httpTokens", httpTokens,
				"state", string(instance.State.Name),
				"deviceMappings", mqlDevices,
				"securityGroups", sgs,
				"publicDnsName", core.ToString(instance.PublicDnsName),
				"stateReason", stateReason,
				"stateTransitionReason", core.ToString(instance.StateTransitionReason),
				"ebsOptimized", core.ToBool(instance.EbsOptimized),
				"instanceType", string(instance.InstanceType),
				"tags", Ec2TagsToMap(instance.Tags),
				"image", mqlImage,
				"launchTime", instance.LaunchTime,
				"privateIp", core.ToString(instance.PrivateIpAddress),
				"privateDnsName", core.ToString(instance.PrivateDnsName),
			}

			// add vpc if there is one
			if instance.VpcId != nil {
				// NOTE: this will create the resource and determine the data in its init method
				mqlVpcResource, err := s.MotorRuntime.CreateResource("aws.vpc",
					"arn", fmt.Sprintf(vpcArnPattern, regionVal, account.ID, core.ToString(instance.VpcId)),
				)
				if err != nil {
					return nil, err
				}
				mqlVpc := mqlVpcResource.(AwsVpc)
				args = append(args, "vpc", mqlVpc)
			}

			// only add a keypair if the ec2 instance has one attached
			if instance.KeyName != nil {
				mqlKeyPair, err := s.MotorRuntime.CreateResource("aws.ec2.keypair",
					"region", regionVal,
					"name", core.ToString(instance.KeyName),
				)
				if err == nil {
					mqlKp := mqlKeyPair.(AwsEc2Keypair)
					args = append(args, "keypair", mqlKp)
				}
			}

			mqlEc2Instance, err := s.MotorRuntime.CreateResource("aws.ec2.instance", args...)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlEc2Instance)
		}
	}
	return res, nil
}

func (i *mqlAwsEc2Image) id() (string, error) {
	return i.Arn()
}

func (i *mqlAwsEc2Image) init(args *resources.Args) (*resources.Args, AwsEc2Image, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if (*args)["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch aws ec2 image")
	}

	arnVal := (*args)["arn"].(string)
	arn, err := arn.Parse(arnVal)
	if err != nil {
		return nil, nil, nil
	}
	resource := strings.Split(arn.Resource, "/")
	provider, err := awsProvider(i.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}
	svc := provider.Ec2(arn.Region)
	ctx := context.Background()
	images, err := svc.DescribeImages(ctx, &ec2.DescribeImagesInput{ImageIds: []string{resource[1]}})
	if err != nil {
		return nil, nil, err
	}

	if len(images.Images) > 0 {
		image := images.Images[0]
		(*args)["arn"] = arnVal
		(*args)["id"] = resource[1]
		(*args)["name"] = core.ToString(image.Name)
		(*args)["architecture"] = string(image.Architecture)
		(*args)["ownerId"] = core.ToString(image.OwnerId)
		(*args)["ownerAlias"] = core.ToString(image.ImageOwnerAlias)
		return args, nil, nil
	}

	(*args)["arn"] = arnVal
	(*args)["id"] = resource[1]
	(*args)["name"] = ""
	(*args)["architecture"] = ""
	(*args)["ownerId"] = ""
	(*args)["ownerAlias"] = ""
	return args, nil, nil
}

func (s *mqlAwsEc2Securitygroup) id() (string, error) {
	return s.Arn()
}

func (p *mqlAwsEc2Securitygroup) init(args *resources.Args) (*resources.Args, AwsEc2Securitygroup, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if len(*args) == 0 {
		if ids := getAssetIdentifier(p.MqlResource().MotorRuntime); ids != nil {
			(*args)["name"] = ids.name
			(*args)["arn"] = ids.arn
		}
	}

	if (*args)["arn"] == nil && (*args)["id"] == nil {
		return nil, nil, errors.New("arn or id required to fetch aws security group")
	}

	// load all security groups
	obj, err := p.MotorRuntime.CreateResource("aws.ec2")
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
			mqlSecArn, err := secGroup.Arn()
			if err != nil {
				log.Error().Err(err).Msg("security group is not properly initialized")
				return false
			}
			return mqlSecArn == arnVal
		}
	}

	if (*args)["id"] != nil {
		idVal := (*args)["id"].(string)
		match = func(secGroup AwsEc2Securitygroup) bool {
			mqlSecId, err := secGroup.Id()
			if err != nil {
				log.Error().Err(err).Msg("security group is not properly initialized")
				return false
			}
			return mqlSecId == idVal
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

func (s *mqlAwsEc2SecuritygroupIppermission) id() (string, error) {
	return s.Id()
}

func (s *mqlAwsEc2InstanceDevice) id() (string, error) {
	return s.VolumeId()
}

func (s *mqlAwsEc2Instance) id() (string, error) {
	return s.Arn()
}

func (s *mqlAwsEc2Instance) GetVpc() (interface{}, error) {
	// this indicated that no vpc is attached since we set the value when we construct the resource
	// we return nil here to make it easier for users to compare:
	// aws.ec2.instances.where(state != "terminated") { vpc != null }
	return nil, nil
}

func (s *mqlAwsEc2Instance) GetKeypair() (interface{}, error) {
	// this indicated that no keypair is assigned to the ec2instance since we set the value when we construct the resource
	// we return nil here to make it easier for users to compare, e.g.:
	// aws.ec2.instances.where(keypair != null) { instanceId }
	return nil, nil
}

func (s *mqlAwsEc2Instance) GetSsm() (interface{}, error) {
	instanceId, err := s.InstanceId()
	if err != nil {
		return nil, errors.Join(err, errors.New("unable to parse instance id"))
	}
	region, err := s.Region()
	if err != nil {
		return nil, errors.Join(err, errors.New("unable to parse instance region"))
	}
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	svc := provider.Ssm(region)
	ctx := context.Background()
	instanceIdFilter := "InstanceIds"
	params := &ssm.DescribeInstanceInformationInput{
		Filters: []ssmtypes.InstanceInformationStringFilter{
			{Key: &instanceIdFilter, Values: []string{instanceId}},
		},
	}
	ssmInstanceInfo, err := svc.DescribeInstanceInformation(ctx, params)
	if err != nil {
		return nil, err
	}
	res, err := core.JsonToDict(ssmInstanceInfo)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *mqlAwsEc2Instance) GetPatchState() (interface{}, error) {
	var res interface{}
	instanceId, err := s.InstanceId()
	if err != nil {
		return nil, errors.Join(err, errors.New("unable to parse instance id"))
	}
	region, err := s.Region()
	if err != nil {
		return nil, errors.Join(err, errors.New("unable to parse instance region"))
	}
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	svc := provider.Ssm(region)
	ctx := context.Background()

	ssmPatchInfo, err := svc.DescribeInstancePatchStates(ctx, &ssm.DescribeInstancePatchStatesInput{InstanceIds: []string{instanceId}})
	if err != nil {
		return nil, err
	}
	if len(ssmPatchInfo.InstancePatchStates) > 0 {
		if instanceId == core.ToString(ssmPatchInfo.InstancePatchStates[0].InstanceId) {
			res, err = core.JsonToDict(ssmPatchInfo.InstancePatchStates[0])
			if err != nil {
				return nil, err
			}
		}
	}
	return res, nil
}

func (s *mqlAwsEc2Instance) GetInstanceStatus() (interface{}, error) {
	var res interface{}
	instanceId, err := s.InstanceId()
	if err != nil {
		return nil, errors.Join(err, errors.New("unable to parse instance id"))
	}
	region, err := s.Region()
	if err != nil {
		return nil, errors.Join(err, errors.New("unable to parse instance region"))
	}
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.Ec2(region)
	ctx := context.Background()

	instanceStatus, err := svc.DescribeInstanceStatus(ctx, &ec2.DescribeInstanceStatusInput{
		InstanceIds:         []string{instanceId},
		IncludeAllInstances: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	if len(instanceStatus.InstanceStatuses) > 0 {
		if instanceId == core.ToString(instanceStatus.InstanceStatuses[0].InstanceId) {
			res, err = core.JsonToDict(instanceStatus.InstanceStatuses[0])
			if err != nil {
				return nil, err
			}
		}
	}

	return res, nil
}

func (s *mqlAwsEc2) GetVolumes() ([]interface{}, error) {
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(s.getVolumes(provider), 5)
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

func (s *mqlAwsEc2) getVolumes(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)

	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	account, err := provider.Account()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := provider.Ec2(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			nextToken := aws.String("no_token_to_start_with")
			params := &ec2.DescribeVolumesInput{}
			for nextToken != nil {
				volumes, err := svc.DescribeVolumes(ctx, params)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, vol := range volumes.Volumes {
					jsonAttachments, err := core.JsonToDictSlice(vol.Attachments)
					if err != nil {
						return nil, err
					}
					mqlVol, err := s.MotorRuntime.CreateResource("aws.ec2.volume",
						"arn", fmt.Sprintf(volumeArnPattern, region, account.ID, core.ToString(vol.VolumeId)),
						"id", core.ToString(vol.VolumeId),
						"attachments", jsonAttachments,
						"encrypted", core.ToBool(vol.Encrypted),
						"state", string(vol.State),
						"tags", Ec2TagsToMap(vol.Tags),
						"availabilityZone", core.ToString(vol.AvailabilityZone),
						"volumeType", string(vol.VolumeType),
						"createTime", vol.CreateTime,
						"region", regionVal,
					)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlVol)
				}
				nextToken = volumes.NextToken
				if volumes.NextToken != nil {
					params.NextToken = nextToken
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (p *mqlAwsEc2Volume) init(args *resources.Args) (*resources.Args, AwsEc2Volume, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if len(*args) == 0 {
		if ids := getAssetIdentifier(p.MqlResource().MotorRuntime); ids != nil {
			(*args)["id"] = ids.name
			(*args)["arn"] = ids.arn
		}
	}

	if (*args)["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch aws volume")
	}

	// load all security groups
	obj, err := p.MotorRuntime.CreateResource("aws.ec2")
	if err != nil {
		return nil, nil, err
	}
	awsEc2 := obj.(AwsEc2)

	rawResources, err := awsEc2.Volumes()
	if err != nil {
		return nil, nil, err
	}

	var match func(secGroup AwsEc2Volume) bool

	if (*args)["arn"] != nil {
		arnVal := (*args)["arn"].(string)
		match = func(vol AwsEc2Volume) bool {
			mqlVolArn, err := vol.Arn()
			if err != nil {
				log.Error().Err(err).Msg("volume is not properly initialized")
				return false
			}
			return mqlVolArn == arnVal
		}
	}

	for i := range rawResources {
		volume := rawResources[i].(AwsEc2Volume)
		if match(volume) {
			return args, volume, nil
		}
	}

	return nil, nil, errors.New("volume does not exist")
}

func (d *mqlAwsEc2Instance) init(args *resources.Args) (*resources.Args, AwsEc2Instance, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if len(*args) == 0 {
		if ids := getAssetIdentifier(d.MqlResource().MotorRuntime); ids != nil {
			(*args)["arn"] = ids.arn
		}
	}

	if (*args)["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch ec2 instance")
	}

	obj, err := d.MotorRuntime.CreateResource("aws.ec2")
	if err != nil {
		return nil, nil, err
	}
	ec2 := obj.(AwsEc2)

	rawResources, err := ec2.Instances()
	if err != nil {
		return nil, nil, err
	}

	arnVal := (*args)["arn"].(string)
	for i := range rawResources {
		instance := rawResources[i].(AwsEc2Instance)
		mqlInstArn, err := instance.Arn()
		if err != nil {
			return nil, nil, errors.New("ec2 instance does not exist")
		}
		if mqlInstArn == arnVal {
			return args, instance, nil
		}
	}
	return nil, nil, errors.New("ec2 instance does not exist")
}

func (p *mqlAwsEc2Snapshot) init(args *resources.Args) (*resources.Args, AwsEc2Snapshot, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if len(*args) == 0 {
		if ids := getAssetIdentifier(p.MqlResource().MotorRuntime); ids != nil {
			(*args)["arn"] = ids.arn
		}
	}

	if (*args)["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch aws snapshot")
	}

	// load all security groups
	obj, err := p.MotorRuntime.CreateResource("aws.ec2")
	if err != nil {
		return nil, nil, err
	}
	awsEc2 := obj.(AwsEc2)

	rawResources, err := awsEc2.Snapshots()
	if err != nil {
		return nil, nil, err
	}

	var match func(snapshot AwsEc2Snapshot) bool

	if (*args)["arn"] != nil {
		arnVal := (*args)["arn"].(string)
		match = func(snapshot AwsEc2Snapshot) bool {
			mqlSnapArn, err := snapshot.Arn()
			if err != nil {
				log.Error().Err(err).Msg("snapshot is not properly initialized")
				return false
			}
			return mqlSnapArn == arnVal
		}
	}

	if (*args)["id"] != nil {
		idVal := (*args)["id"].(string)
		match = func(snap AwsEc2Snapshot) bool {
			mqlSnapId, err := snap.Id()
			if err != nil {
				log.Error().Err(err).Msg("snapshot is not properly initialized")
				return false
			}
			return mqlSnapId == idVal
		}
	}

	for i := range rawResources {
		snapshot := rawResources[i].(AwsEc2Snapshot)
		if match(snapshot) {
			return args, snapshot, nil
		}
	}

	return nil, nil, errors.New("snapshot does not exist")
}

func (s *mqlAwsEc2Volume) id() (string, error) {
	return s.Arn()
}

func (s *mqlAwsEc2Snapshot) id() (string, error) {
	return s.Arn()
}

func (s *mqlAwsEc2) GetVpnConnections() ([]interface{}, error) {
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(s.getVpnConnections(provider), 5)
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

func (s *mqlAwsEc2) getVpnConnections(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	account, err := provider.Account()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := provider.Ec2(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			vpnConnections, err := svc.DescribeVpnConnections(ctx, &ec2.DescribeVpnConnectionsInput{})
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
					return res, nil
				}
				return nil, err
			}
			for _, vpnConn := range vpnConnections.VpnConnections {
				mqlVgwT := []interface{}{}
				for _, vgwT := range vpnConn.VgwTelemetry {
					mqlVgwTelemetry, err := s.MotorRuntime.CreateResource("aws.ec2.vgwtelemetry",
						"outsideIpAddress", core.ToString(vgwT.OutsideIpAddress),
						"status", string(vgwT.Status),
						"statusMessage", core.ToString(vgwT.StatusMessage),
					)
					if err != nil {
						return nil, err
					}
					mqlVgwT = append(mqlVgwT, mqlVgwTelemetry)
				}
				mqlVpnConn, err := s.MotorRuntime.CreateResource("aws.ec2.vpnconnection",
					"arn", fmt.Sprintf(vpnConnArnPattern, regionVal, account.ID, core.ToString(vpnConn.VpnConnectionId)),
					"vgwTelemetry", mqlVgwT,
				)
				if err != nil {
					return nil, err
				}
				res = append(res, mqlVpnConn)
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (s *mqlAwsEc2) GetSnapshots() ([]interface{}, error) {
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(s.getSnapshots(provider), 5)
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

func (s *mqlAwsEc2) getSnapshots(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	account, err := provider.Account()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := provider.Ec2(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			nextToken := aws.String("no_token_to_start_with")
			params := &ec2.DescribeSnapshotsInput{Filters: []types.Filter{{Name: aws.String("owner-id"), Values: []string{account.ID}}}}
			for nextToken != nil {
				snapshots, err := svc.DescribeSnapshots(ctx, params)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, snapshot := range snapshots.Snapshots {
					mqlSnap, err := s.MotorRuntime.CreateResource("aws.ec2.snapshot",
						"arn", fmt.Sprintf(snapshotArnPattern, regionVal, account.ID, core.ToString(snapshot.SnapshotId)),
						"id", core.ToString(snapshot.SnapshotId),
						"region", regionVal,
						"volumeId", core.ToString(snapshot.VolumeId),
						"startTime", snapshot.StartTime,
						"tags", Ec2TagsToMap(snapshot.Tags),
						"state", string(snapshot.State),
					)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlSnap)
				}
				nextToken = snapshots.NextToken
				if snapshots.NextToken != nil {
					params.NextToken = nextToken
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (s *mqlAwsEc2Snapshot) GetCreateVolumePermission() ([]interface{}, error) {
	id, err := s.Id()
	if err != nil {
		return nil, errors.Join(err, errors.New("unable to parse instance id"))
	}
	region, err := s.Region()
	if err != nil {
		return nil, errors.Join(err, errors.New("unable to parse instance region"))
	}
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.Ec2(region)
	ctx := context.Background()

	attribute, err := svc.DescribeSnapshotAttribute(ctx, &ec2.DescribeSnapshotAttributeInput{SnapshotId: &id, Attribute: types.SnapshotAttributeNameCreateVolumePermission})
	if err != nil {
		return nil, err
	}

	return core.JsonToDictSlice(attribute.CreateVolumePermissions)
}

func (s *mqlAwsEc2) GetInternetGateways() ([]interface{}, error) {
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(s.getInternetGateways(provider), 5)
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

func (s *mqlAwsEc2) getInternetGateways(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := provider.Ec2(regionVal)
			ctx := context.Background()
			params := &ec2.DescribeInternetGatewaysInput{}
			res := []interface{}{}
			nextToken := aws.String("no_token_to_start_with")
			for nextToken != nil {
				internetGws, err := svc.DescribeInternetGateways(ctx, params)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, gateway := range internetGws.InternetGateways {
					jsonAttachments, err := core.JsonToDictSlice(gateway.Attachments)
					if err != nil {
						return nil, err
					}
					mqlInternetGw, err := s.MotorRuntime.CreateResource("aws.ec2.internetgateway",
						"arn", fmt.Sprintf(internetGwArnPattern, regionVal, core.ToString(gateway.OwnerId), core.ToString(gateway.InternetGatewayId)),
						"id", core.ToString(gateway.InternetGatewayId),
						"attachments", jsonAttachments,
					)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlInternetGw)
				}

				nextToken = internetGws.NextToken
				if internetGws.NextToken != nil {
					params.NextToken = nextToken
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (s *mqlAwsEc2Internetgateway) id() (string, error) {
	return s.Arn()
}

func (s *mqlAwsEc2Vpnconnection) id() (string, error) {
	return s.Arn()
}

func (s *mqlAwsEc2Vgwtelemetry) id() (string, error) {
	return s.OutsideIpAddress()
}
