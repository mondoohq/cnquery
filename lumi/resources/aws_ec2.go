package resources

import (
	"context"
	"strconv"

	"go.mondoo.io/mondoo/lumi/library/jobpool"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/rs/zerolog/log"
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

func (s *lumiAwsEc2) GetVpcs() ([]interface{}, error) {
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(s.getVpcs(), 5)
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

func (s *lumiAwsEc2) getVpcs() []*jobpool.Job {
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

			nextToken := aws.String("no_token_to_start_with")
			params := &ec2.DescribeVpcsInput{}
			for nextToken != nil {
				vpcs, err := svc.DescribeVpcsRequest(params).Send(ctx)
				if err != nil {
					return nil, err
				}
				nextToken = vpcs.NextToken
				if vpcs.NextToken != nil {
					params.NextToken = nextToken
				}

				for i := range vpcs.Vpcs {
					v := vpcs.Vpcs[i]
					stringState, err := ec2.VpcState.MarshalValue(v.State)
					if err != nil {
						return nil, err
					}
					lumiVpc, err := s.Runtime.CreateResource("aws.ec2.vpc",
						"id", toString(v.VpcId),
						"state", stringState,
						"isDefault", toBool(v.IsDefault),
						"region", regionVal,
					)
					if err != nil {
						return nil, err
					}
					res = append(res, lumiVpc)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (s *lumiAwsEc2) getSecurityGroups() []*jobpool.Job {
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

					lumiS3SecurityGroup, err := s.Runtime.CreateResource("aws.ec2.securitygroup",
						"id", toString(group.GroupId),
						"name", toString(group.GroupName),
						"description", toString(group.Description),
						"tag", ec2TagsToMap(group.Tags),
						// TODO: reference to vpc
						"vpcid", toString(group.VpcId),
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

func (s *lumiAwsEc2Securitygroup) id() (string, error) {
	return s.Id()
}

func (s *lumiAwsEc2SecuritygroupIppermission) id() (string, error) {
	return s.Id()
}

func (s *lumiAwsEc2Vpc) id() (string, error) {
	return s.Id()
}
