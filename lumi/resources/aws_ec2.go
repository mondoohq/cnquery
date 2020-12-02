package resources

import (
	"context"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
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
	at, err := awstransport(s.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	svc := at.Ec2()
	ctx := context.Background()

	// todo: add pagination
	vpcs, err := svc.DescribeVpcsRequest(&ec2.DescribeVpcsInput{}).Send(ctx)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
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
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiVpc)
	}
	return res, nil
}

func (s *lumiAwsEc2) GetSecurityGroups() ([]interface{}, error) {
	at, err := awstransport(s.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	svc := at.Ec2()
	ctx := context.Background()

	// TODO: iterate over each region? pagination is needed.
	securityGroups, err := svc.DescribeSecurityGroupsRequest(&ec2.DescribeSecurityGroupsInput{}).Send(ctx)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
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
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiS3SecurityGroup)
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
