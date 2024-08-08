// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/aws/smithy-go"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v11/providers/aws/connection"
	"go.mondoo.com/cnquery/v11/types"
)

func (e *mqlAwsEc2) id() (string, error) {
	return "aws.ec2", nil
}

func Ec2TagsToMap(tags []ec2types.Tag) map[string]interface{} {
	tagsMap := make(map[string]interface{})

	if len(tags) > 0 {
		for i := range tags {
			tag := tags[i]
			tagsMap[convert.ToString(tag.Key)] = convert.ToString(tag.Value)
		}
	}

	return tagsMap
}

func initAwsEc2Eip(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if args["publicIp"] == nil {
		return nil, nil, errors.New("publicIp required to fetch aws ec2 eip")
	}
	p := args["publicIp"].Value.(string)

	if args["region"] == nil {
		return nil, nil, errors.New("region required to fetch aws ec2 eip")
	}
	r := args["region"].Value.(string)

	conn := runtime.Connection.(*connection.AwsConnection)
	svc := conn.Ec2(r)
	ctx := context.Background()
	address, err := svc.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{Filters: []ec2types.Filter{{Name: aws.String("public-ip"), Values: []string{p}}}})
	if err != nil {
		return nil, nil, err
	}

	if len(address.Addresses) > 0 {
		add := address.Addresses[0]
		attached := add.AllocationId != nil
		args["publicIp"] = llx.StringDataPtr(add.PublicIp)
		args["attached"] = llx.BoolData(attached) // this is false if allocationId is null and true otherwise
		args["networkInterfaceId"] = llx.StringDataPtr(add.NetworkInterfaceId)
		args["networkInterfaceOwnerId"] = llx.StringDataPtr(add.NetworkInterfaceOwnerId)
		args["privateIpAddress"] = llx.StringDataPtr(add.PrivateIpAddress)
		args["publicIpv4Pool"] = llx.StringDataPtr(add.PublicIpv4Pool)
		args["tags"] = llx.MapData(Ec2TagsToMap(add.Tags), types.String)
		args["region"] = llx.StringData(r)
		return args, nil, nil
	}
	return args, nil, nil
}

func (a *mqlAwsEc2Eip) id() (string, error) {
	return a.NetworkInterfaceId.Data, nil
}

type mqlAwsEc2EipInternal struct {
	eipCache ec2types.Address
}

func (a *mqlAwsEc2Eip) instance() (*mqlAwsEc2Instance, error) {
	regionVal := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	if a.eipCache.InstanceId != nil {
		instanceId := a.eipCache.InstanceId
		mqlEc2Instance, err := NewResource(a.MqlRuntime, "aws.ec2.instance",
			map[string]*llx.RawData{
				"arn": llx.StringData(fmt.Sprintf(ec2InstanceArnPattern, regionVal, conn.AccountId(), convert.ToString(instanceId))),
			})
		if err != nil {
			return nil, err
		}
		return mqlEc2Instance.(*mqlAwsEc2Instance), err
	}
	a.Instance.State = plugin.StateIsNull | plugin.StateIsSet
	return nil, nil
}

func (a *mqlAwsEc2) eips() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getEIPs(conn), 5)
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

func (a *mqlAwsEc2) getEIPs(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)

	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("ec2>getEIPs>calling aws with region %s", regionVal)

			svc := conn.Ec2(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			params := &ec2.DescribeAddressesInput{} // no pagination
			addresses, err := svc.DescribeAddresses(ctx, params)
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
					return res, nil
				}
				return nil, err
			}

			for i := range addresses.Addresses {
				add := addresses.Addresses[i]
				attached := add.AllocationId != nil

				args := map[string]*llx.RawData{
					"publicIp":                llx.StringDataPtr(add.PublicIp),
					"attached":                llx.BoolData(attached), // this is false if allocationId is null and true otherwise
					"networkInterfaceId":      llx.StringDataPtr(add.NetworkInterfaceId),
					"networkInterfaceOwnerId": llx.StringDataPtr(add.NetworkInterfaceOwnerId),
					"privateIpAddress":        llx.StringDataPtr(add.PrivateIpAddress),
					"publicIpv4Pool":          llx.StringDataPtr(add.PublicIpv4Pool),
					"tags":                    llx.MapData(Ec2TagsToMap(add.Tags), types.String),
					"region":                  llx.StringData(regionVal),
				}
				mqlAddress, err := CreateResource(a.MqlRuntime, "aws.ec2.eip", args)
				if err != nil {
					return nil, err
				}
				mqlAddress.(*mqlAwsEc2Eip).eipCache = add

				res = append(res, mqlAddress)

			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsEc2Networkacl) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsEc2) networkAcls() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getNetworkACLs(conn), 5)
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

func (a *mqlAwsEc2) getNetworkACLs(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)

	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("ec2>getNetworkACLs>calling aws with region %s", regionVal)

			svc := conn.Ec2(regionVal)
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
					assoc := []interface{}{}
					for i := range acl.Associations {
						association := acl.Associations[i]
						mqlNetworkAclAssoc, err := CreateResource(a.MqlRuntime, "aws.ec2.networkacl.association",
							map[string]*llx.RawData{
								"associationId": llx.StringDataPtr(association.NetworkAclAssociationId),
								"networkAclId":  llx.StringDataPtr(association.NetworkAclId),
								"subnetId":      llx.StringDataPtr(association.SubnetId),
							})
						if err == nil {
							assoc = append(assoc, mqlNetworkAclAssoc)
						}
					}
					mqlNetworkAcl, err := CreateResource(a.MqlRuntime, "aws.ec2.networkacl",
						map[string]*llx.RawData{
							"arn":          llx.StringData(fmt.Sprintf(networkAclArnPattern, regionVal, conn.AccountId(), convert.ToString(acl.NetworkAclId))),
							"id":           llx.StringDataPtr(acl.NetworkAclId),
							"region":       llx.StringData(regionVal),
							"isDefault":    llx.BoolDataPtr(acl.IsDefault),
							"tags":         llx.MapData(Ec2TagsToMap(acl.Tags), types.String),
							"associations": llx.ArrayData(assoc, "aws.ec2.networkacl.association"),
						})
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

func (a *mqlAwsEc2NetworkaclEntry) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAwsEc2NetworkaclEntryPortrange) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAwsEc2Networkacl) entries() ([]interface{}, error) {
	id := a.Id.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Ec2(region)
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
		egress := convert.ToBool(entry.Egress)
		entryId := id + "-" + strconv.Itoa(convert.ToIntFrom32(entry.RuleNumber))
		if egress {
			entryId += "-egress"
		} else {
			entryId += "-ingress"
		}
		args := map[string]*llx.RawData{
			"egress":        llx.BoolData(egress),
			"ruleAction":    llx.StringData(string(entry.RuleAction)),
			"ruleNumber":    llx.IntDataDefault(entry.RuleNumber, 0),
			"cidrBlock":     llx.StringDataPtr(entry.CidrBlock),
			"ipv6CidrBlock": llx.StringDataPtr(entry.Ipv6CidrBlock),
			"id":            llx.StringData(entryId),
		}
		if entry.PortRange != nil {
			mqlPortRange, err := CreateResource(a.MqlRuntime, "aws.ec2.networkacl.entry.portrange",
				map[string]*llx.RawData{
					"from": llx.IntDataDefault(entry.PortRange.From, -1),
					"to":   llx.IntDataDefault(entry.PortRange.To, -1),
					"id":   llx.StringData(entryId + "-" + strconv.Itoa(convert.ToIntFrom32(entry.PortRange.From))),
				})
			if err != nil {
				return nil, err
			}
			args["portRange"] = llx.ResourceData(mqlPortRange, mqlPortRange.MqlName())
		} else {
			args["portRange"] = llx.NilData
		}

		mqlAclEntry, err := CreateResource(a.MqlRuntime, "aws.ec2.networkacl.entry", args)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAclEntry)
	}

	return res, nil
}

func (a *mqlAwsEc2NetworkaclEntry) portRange() (*mqlAwsEc2NetworkaclEntryPortrange, error) {
	return a.PortRange.Data, nil
}

func (a *mqlAwsEc2Securitygroup) isAttachedToNetworkInterface() (bool, error) {
	sgId := a.Id.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Ec2(region)
	ctx := context.Background()

	networkinterfaces, err := svc.DescribeNetworkInterfaces(ctx, &ec2.DescribeNetworkInterfacesInput{Filters: []ec2types.Filter{
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

type mqlAwsEc2SecuritygroupInternal struct {
	cacheIpPerms       []ec2types.IpPermission
	cacheIpPermsEgress []ec2types.IpPermission
	groupId            string
	region             string
	cacheVpc           *string
}

func (a *mqlAwsEc2) getSecurityGroups(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)

	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("ec2>getSecurityGroups>calling aws with region %s", regionVal)

			svc := conn.Ec2(regionVal)
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

					args := map[string]*llx.RawData{
						"arn":         llx.StringData(fmt.Sprintf(securityGroupArnPattern, regionVal, conn.AccountId(), convert.ToString(group.GroupId))),
						"id":          llx.StringDataPtr(group.GroupId),
						"name":        llx.StringDataPtr(group.GroupName),
						"description": llx.StringDataPtr(group.Description),
						"tags":        llx.MapData(Ec2TagsToMap(group.Tags), types.String),
						"region":      llx.StringData(regionVal),
					}

					mqlEc2SecurityGroup, err := CreateResource(a.MqlRuntime, "aws.ec2.securitygroup", args)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlEc2SecurityGroup)
					mqlEc2SecurityGroup.(*mqlAwsEc2Securitygroup).cacheIpPerms = group.IpPermissions
					mqlEc2SecurityGroup.(*mqlAwsEc2Securitygroup).cacheIpPermsEgress = group.IpPermissionsEgress
					mqlEc2SecurityGroup.(*mqlAwsEc2Securitygroup).groupId = *group.GroupId
					mqlEc2SecurityGroup.(*mqlAwsEc2Securitygroup).region = region
					mqlEc2SecurityGroup.(*mqlAwsEc2Securitygroup).cacheVpc = group.VpcId
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsEc2Securitygroup) vpc() (*mqlAwsVpc, error) {
	if a.cacheVpc != nil {
		conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
		mqlVpc, err := NewResource(a.MqlRuntime, "aws.vpc",
			map[string]*llx.RawData{
				"arn": llx.StringData(fmt.Sprintf(vpcArnPattern, a.region, conn.AccountId(), convert.ToString(a.cacheVpc))),
			})
		if err != nil {
			return nil, err
		}
		return mqlVpc.(*mqlAwsVpc), nil
	}
	a.Vpc.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

func (a *mqlAwsEc2Securitygroup) ipPermissions() ([]interface{}, error) {
	mqlIpPermissions := []interface{}{}
	for p := range a.cacheIpPerms {
		permission := a.cacheIpPerms[p]

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
		prefixListIds, err := convert.JsonToDictSlice(permission.PrefixListIds)
		if err != nil {
			return nil, err
		}
		userIdGroupPairs, err := convert.JsonToDictSlice(permission.UserIdGroupPairs)
		if err != nil {
			return nil, err
		}
		mqlSecurityGroupIpPermission, err := CreateResource(a.MqlRuntime, "aws.ec2.securitygroup.ippermission",
			map[string]*llx.RawData{
				"id":               llx.StringData(a.groupId + "-" + strconv.Itoa(p)),
				"fromPort":         llx.IntDataDefault(permission.FromPort, -1),
				"toPort":           llx.IntDataDefault(permission.ToPort, -1),
				"ipProtocol":       llx.StringDataPtr(permission.IpProtocol),
				"ipRanges":         llx.ArrayData(ipRanges, types.Any),
				"ipv6Ranges":       llx.ArrayData(ipv6Ranges, types.Any),
				"prefixListIds":    llx.ArrayData(prefixListIds, types.Any),
				"userIdGroupPairs": llx.ArrayData(userIdGroupPairs, types.Any),
			})
		if err != nil {
			return nil, err
		}

		mqlIpPermissions = append(mqlIpPermissions, mqlSecurityGroupIpPermission)
	}
	return mqlIpPermissions, nil
}

func (a *mqlAwsEc2Securitygroup) ipPermissionsEgress() ([]interface{}, error) {
	mqlIpPermissionsEgress := []interface{}{}
	for p := range a.cacheIpPermsEgress {
		permission := a.cacheIpPermsEgress[p]

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
		prefixListIds, err := convert.JsonToDictSlice(permission.PrefixListIds)
		if err != nil {
			return nil, err
		}
		userIdGroupPairs, err := convert.JsonToDictSlice(permission.UserIdGroupPairs)
		if err != nil {
			return nil, err
		}
		mqlSecurityGroupIpPermission, err := CreateResource(a.MqlRuntime, "aws.ec2.securitygroup.ippermission",
			map[string]*llx.RawData{
				"id":               llx.StringData(a.groupId + "-" + strconv.Itoa(p) + "-egress"),
				"fromPort":         llx.IntDataDefault(permission.FromPort, -1),
				"toPort":           llx.IntDataDefault(permission.ToPort, -1),
				"ipProtocol":       llx.StringDataPtr(permission.IpProtocol),
				"ipRanges":         llx.ArrayData(ipRanges, types.Any),
				"ipv6Ranges":       llx.ArrayData(ipv6Ranges, types.Any),
				"prefixListIds":    llx.ArrayData(prefixListIds, types.Any),
				"userIdGroupPairs": llx.ArrayData(userIdGroupPairs, types.Any),
			})
		if err != nil {
			return nil, err
		}

		mqlIpPermissionsEgress = append(mqlIpPermissionsEgress, mqlSecurityGroupIpPermission)
	}
	return mqlIpPermissionsEgress, nil
}

func (a *mqlAwsEc2) keypairs() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getKeypairs(conn), 5)
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

func (a *mqlAwsEc2Keypair) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsEc2) getKeypairs(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)

	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("ec2>getKeypairs>calling aws with region %s", regionVal)

			svc := conn.Ec2(regionVal)
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
				mqlKeypair, err := CreateResource(a.MqlRuntime, "aws.ec2.keypair",
					map[string]*llx.RawData{
						"arn":         llx.StringData(fmt.Sprintf(keypairArnPattern, conn.AccountId(), regionVal, convert.ToString(kp.KeyPairId))),
						"fingerprint": llx.StringDataPtr(kp.KeyFingerprint),
						"name":        llx.StringDataPtr(kp.KeyName),
						"type":        llx.StringData(string(kp.KeyType)),
						"tags":        llx.MapData(Ec2TagsToMap(kp.Tags), types.String),
						"region":      llx.StringData(regionVal),
						"createdAt":   llx.TimeDataPtr(kp.CreateTime),
					})
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

func initAwsEc2Keypair(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if args["name"] == nil {
		return nil, nil, errors.New("name required to fetch aws ec2 keypair")
	}
	n := args["name"].Value.(string)
	if n == "" {
		return nil, nil, errors.New("ec2 keypair name cannot be empty")
	}
	if args["region"] == nil {
		return nil, nil, errors.New("region required to fetch aws ec2 keypair")
	}
	r := args["region"].Value.(string)

	conn := runtime.Connection.(*connection.AwsConnection)
	svc := conn.Ec2(r)
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
		args["fingerprint"] = llx.StringData(convert.ToString(kp.KeyFingerprint))
		args["name"] = llx.StringData(convert.ToString(kp.KeyName))
		args["type"] = llx.StringData(string(kp.KeyType))
		args["tags"] = llx.MapData(Ec2TagsToMap(kp.Tags), types.String)
		args["region"] = llx.StringData(r)
		args["arn"] = llx.StringData(fmt.Sprintf(keypairArnPattern, conn.AccountId(), r, convert.ToString(kp.KeyPairId)))
		args["createdAt"] = llx.TimeDataPtr(kp.CreateTime)

		return args, nil, nil
	}
	return args, nil, nil
}

func (a *mqlAwsEc2) securityGroups() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getSecurityGroups(conn), 5)
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

func (a *mqlAwsEc2) ebsEncryptionByDefault() (map[string]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := make(map[string]interface{})
	poolOfJobs := jobpool.CreatePool(a.getEbsEncryptionPerRegion(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			jobResult := poolOfJobs.Jobs[i].Result.(ebsEncryption)
			res[jobResult.region] = jobResult.ebsEncryptionByDefault
		}
	}
	return res, nil
}

func (a *mqlAwsEc2) getEbsEncryptionPerRegion(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)

	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("ec2>getEbsEncryptionPerRegion>calling aws with region %s", regionVal)

			svc := conn.Ec2(regionVal)
			ctx := context.Background()

			ebsEncryptionRes, err := svc.GetEbsEncryptionByDefault(ctx, &ec2.GetEbsEncryptionByDefaultInput{})
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
					return nil, nil
				}
				return nil, err
			}
			structVal := ebsEncryption{
				region:                 regionVal,
				ebsEncryptionByDefault: convert.ToBool(ebsEncryptionRes.EbsEncryptionByDefault),
			}
			return jobpool.JobResult(structVal), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsEc2) instances() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getInstances(conn), 5)
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

func (a *mqlAwsEc2) getImdsv2Instances(ctx context.Context, svc *ec2.Client, filterName string) ([]ec2types.Reservation, error) {
	res := []ec2types.Reservation{}
	nextToken := aws.String("no_token_to_start_with")
	params := &ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{
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

func (a *mqlAwsEc2) getImdsv1Instances(ctx context.Context, svc *ec2.Client, filterName string) ([]ec2types.Reservation, error) {
	res := []ec2types.Reservation{}
	nextToken := aws.String("no_token_to_start_with")
	params := &ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{
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

func (a *mqlAwsEc2) getInstances(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	if len(conn.Filters.Ec2DiscoveryFilters.Regions) > 0 {
		regions = conn.Filters.Ec2DiscoveryFilters.Regions
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("ec2>getInstances>calling aws with region %s", regionVal)

			svc := conn.Ec2(regionVal)
			ctx := context.Background()
			var res []interface{}

			// the value for http tokens is not available on api output i've been able to find, so here
			// we make two calls to get the instances, one with the imdsv1 filter and another with the imdsv2 filter
			filterName := "metadata-options.http-tokens"
			imdsv2Instances, err := a.getImdsv2Instances(ctx, svc, filterName)
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
					return res, nil
				}
				return nil, err
			}
			res, err = a.gatherInstanceInfo(imdsv2Instances, 2, regionVal)
			if err != nil {
				return nil, err
			}

			imdsv1Instances, err := a.getImdsv1Instances(ctx, svc, filterName)
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
					return res, nil
				}
				return nil, err
			}
			imdsv1Res, err := a.gatherInstanceInfo(imdsv1Instances, 1, regionVal)
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

func (a *mqlAwsEc2) gatherInstanceInfo(instances []ec2types.Reservation, imdsvVersion int, regionVal string) ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
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

				mqlInstanceDevice, err := CreateResource(a.MqlRuntime, "aws.ec2.instance.device",
					map[string]*llx.RawData{
						"deleteOnTermination": llx.BoolData(convert.ToBool(device.Ebs.DeleteOnTermination)),
						"status":              llx.StringData(string(device.Ebs.Status)),
						"volumeId":            llx.StringData(convert.ToString(device.Ebs.VolumeId)),
						"deviceName":          llx.StringData(convert.ToString(device.DeviceName)),
					})
				if err != nil {
					return nil, err
				}
				mqlDevices = append(mqlDevices, mqlInstanceDevice)
			}

			stateReason, err := convert.JsonToDict(instance.StateReason)
			if err != nil {
				return nil, err
			}
			var stateTransitionTime time.Time
			reg := regexp.MustCompile(`.*\((\d{4}-\d{2}-\d{2}\s\d{2}:\d{2}:\d{2}) GMT\)`)
			timeString := reg.FindStringSubmatch(convert.ToString(instance.StateTransitionReason))
			if len(timeString) == 2 {
				stateTransitionTime, err = time.Parse(time.DateTime, timeString[1])
				if err != nil {
					log.Error().Err(err).Msg("cannot parse state transition time for ec2 instance")
					stateTransitionTime = llx.NeverPastTime
				}
			}
			args := map[string]*llx.RawData{
				"architecture":          llx.StringData(string(instance.Architecture)),
				"arn":                   llx.StringData(fmt.Sprintf(ec2InstanceArnPattern, regionVal, conn.AccountId(), convert.ToString(instance.InstanceId))),
				"detailedMonitoring":    llx.StringData(string(instance.Monitoring.State)),
				"deviceMappings":        llx.ArrayData(mqlDevices, types.Resource("aws.ec2.instance.device")),
				"ebsOptimized":          llx.BoolDataPtr(instance.EbsOptimized),
				"enaSupported":          llx.BoolDataPtr(instance.EnaSupport),
				"httpEndpoint":          llx.StringData(string(instance.MetadataOptions.HttpEndpoint)),
				"httpTokens":            llx.StringData(httpTokens),
				"hypervisor":            llx.StringData(string(instance.Hypervisor)),
				"instanceId":            llx.StringDataPtr(instance.InstanceId),
				"instanceLifecycle":     llx.StringData(string(instance.InstanceLifecycle)),
				"instanceType":          llx.StringData(string(instance.InstanceType)),
				"launchTime":            llx.TimeDataPtr(instance.LaunchTime),
				"platformDetails":       llx.StringDataPtr(instance.PlatformDetails),
				"privateDnsName":        llx.StringDataPtr(instance.PrivateDnsName),
				"privateIp":             llx.StringDataPtr(instance.PrivateIpAddress),
				"publicDnsName":         llx.StringDataPtr(instance.PublicDnsName),
				"publicIp":              llx.StringDataPtr(instance.PublicIpAddress),
				"region":                llx.StringData(regionVal),
				"rootDeviceName":        llx.StringDataPtr(instance.RootDeviceName),
				"rootDeviceType":        llx.StringData(string(instance.RootDeviceType)),
				"state":                 llx.StringData(string(instance.State.Name)),
				"stateReason":           llx.MapData(stateReason, types.Any),
				"stateTransitionReason": llx.StringDataPtr(instance.StateTransitionReason),
				"stateTransitionTime":   llx.TimeData(stateTransitionTime),
				"tags":                  llx.MapData(Ec2TagsToMap(instance.Tags), types.String),
				"tpmSupport":            llx.StringDataPtr(instance.TpmSupport),
			}

			// add vpc if there is one
			if instance.VpcId != nil {
				arn := fmt.Sprintf(vpcArnPattern, regionVal, conn.AccountId(), convert.ToString(instance.VpcId))
				args["vpcArn"] = llx.StringData(arn)
			} else {
				args["vpcArn"] = llx.NilData
			}

			mqlEc2Instance, err := CreateResource(a.MqlRuntime, "aws.ec2.instance", args)
			if err != nil {
				return nil, err
			}
			mqlEc2Instance.(*mqlAwsEc2Instance).instanceCache = instance
			res = append(res, mqlEc2Instance)
		}
	}
	return res, nil
}

type mqlAwsEc2InstanceInternal struct {
	instanceCache ec2types.Instance
}

func (i *mqlAwsEc2Instance) networkInterfaces() ([]interface{}, error) {
	conn := i.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Ec2(i.Region.Data)
	ctx := context.Background()
	nextToken := aws.String("no_token_to_start_with")
	params := &ec2.DescribeNetworkInterfacesInput{Filters: []ec2types.Filter{{Name: aws.String("attachment.instance-id"), Values: []string{i.InstanceId.Data}}}}
	res := []interface{}{}
	for nextToken != nil {
		nis, err := svc.DescribeNetworkInterfaces(ctx, params)
		if err != nil {
			return nil, err
		}
		for ni := range nis.NetworkInterfaces {
			n := nis.NetworkInterfaces[ni]
			args := map[string]*llx.RawData{
				"availabilityZone": llx.StringDataPtr(n.AvailabilityZone),
				"description":      llx.StringDataPtr(n.Description),
				"id":               llx.StringDataPtr(n.NetworkInterfaceId),
				"ipv6Native":       llx.BoolDataPtr(n.Ipv6Native),
				"macAddress":       llx.StringDataPtr(n.MacAddress),
				"privateDnsName":   llx.StringDataPtr(n.PrivateDnsName),
				"privateIpAddress": llx.StringDataPtr(n.PrivateIpAddress),
				"requesterManaged": llx.BoolDataPtr(n.RequesterManaged),
				"sourceDestCheck":  llx.BoolDataPtr(n.SourceDestCheck),
				"status":           llx.StringData(string(n.Status)),
				"tags":             llx.MapData(Ec2TagsToMap(n.TagSet), types.String),
			}
			mqlNetworkInterface, err := CreateResource(i.MqlRuntime, "aws.ec2.networkinterface", args)
			if err != nil {
				return nil, err
			}
			mqlNetworkInterface.(*mqlAwsEc2Networkinterface).networkInterfaceCache = n
			mqlNetworkInterface.(*mqlAwsEc2Networkinterface).region = i.Region.Data
			res = append(res, mqlNetworkInterface)
		}
		nextToken = nis.NextToken
		if nis.NextToken != nil {
			params.NextToken = nextToken
		}
	}
	return res, nil
}

type mqlAwsEc2NetworkinterfaceInternal struct {
	networkInterfaceCache ec2types.NetworkInterface
	region                string
}

func (i *mqlAwsEc2Networkinterface) securityGroups() ([]interface{}, error) {
	if i.networkInterfaceCache.Groups != nil {
		sgs := []interface{}{}
		conn := i.MqlRuntime.Connection.(*connection.AwsConnection)

		for nig := range i.networkInterfaceCache.Groups {
			g := *i.networkInterfaceCache.Groups[nig].GroupId

			mqlSg, err := NewResource(i.MqlRuntime, "aws.ec2.securitygroup",
				map[string]*llx.RawData{"arn": llx.StringData(fmt.Sprintf(securityGroupArnPattern, i.region, conn.AccountId(), g))})
			if err != nil {
				return nil, err
			}
			sgs = append(sgs, mqlSg)
		}
		return sgs, nil
	}
	i.SecurityGroups.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

func (i *mqlAwsEc2Networkinterface) subnet() (*mqlAwsVpcSubnet, error) {
	subn := i.networkInterfaceCache.SubnetId
	conn := i.MqlRuntime.Connection.(*connection.AwsConnection)
	if subn != nil {

		arn := fmt.Sprintf(subnetArnPattern, i.region, conn.AccountId(), *subn)
		res, err := NewResource(i.MqlRuntime, "aws.vpc.subnet", map[string]*llx.RawData{"arn": llx.StringData(arn)})
		if err != nil {
			return nil, err
		}
		return res.(*mqlAwsVpcSubnet), nil
	}
	i.Subnet.State = plugin.StateIsNull | plugin.StateIsSet
	return nil, nil
}

func (i *mqlAwsEc2Networkinterface) vpc() (*mqlAwsVpc, error) {
	vpcId := i.networkInterfaceCache.VpcId
	if vpcId != nil {
		conn := i.MqlRuntime.Connection.(*connection.AwsConnection)
		vpcArn := fmt.Sprintf(vpcArnPattern, i.region, conn.AccountId(), convert.ToString(vpcId))
		res, err := NewResource(i.MqlRuntime, "aws.vpc", map[string]*llx.RawData{"arn": llx.StringData(vpcArn)})
		if err != nil {
			return nil, err
		}
		return res.(*mqlAwsVpc), nil
	}
	i.Vpc.State = plugin.StateIsNull | plugin.StateIsSet
	return nil, nil
}

func (i *mqlAwsEc2Instance) securityGroups() ([]interface{}, error) {
	if i.instanceCache.SecurityGroups != nil {
		sgs := []interface{}{}
		conn := i.MqlRuntime.Connection.(*connection.AwsConnection)

		for j := range i.instanceCache.SecurityGroups {
			mqlSg, err := NewResource(i.MqlRuntime, "aws.ec2.securitygroup",
				map[string]*llx.RawData{"arn": llx.StringData(fmt.Sprintf(securityGroupArnPattern, i.Region.Data, conn.AccountId(), convert.ToString(i.instanceCache.SecurityGroups[j].GroupId)))})
			if err != nil {
				return nil, err
			}
			sgs = append(sgs, mqlSg)
		}
		return sgs, nil
	}
	i.SecurityGroups.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

func (i *mqlAwsEc2Instance) image() (*mqlAwsEc2Image, error) {
	if i.instanceCache.ImageId != nil {
		conn := i.MqlRuntime.Connection.(*connection.AwsConnection)

		mqlImage, err := NewResource(i.MqlRuntime, "aws.ec2.image",
			map[string]*llx.RawData{"arn": llx.StringData(fmt.Sprintf(imageArnPattern, i.Region.Data, conn.AccountId(), convert.ToString(i.instanceCache.ImageId)))})
		if err == nil {
			return mqlImage.(*mqlAwsEc2Image), nil
		}
	}
	i.Image.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

func (i *mqlAwsEc2Instance) keypair() (*mqlAwsEc2Keypair, error) {
	if i.instanceCache.KeyName != nil {
		mqlKeyPair, err := NewResource(i.MqlRuntime, "aws.ec2.keypair",
			map[string]*llx.RawData{
				"region": llx.StringData(i.Region.Data),
				"name":   llx.StringDataPtr(i.instanceCache.KeyName),
			})
		if err == nil {
			return mqlKeyPair.(*mqlAwsEc2Keypair), nil
		}
	}
	i.Keypair.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

func (i *mqlAwsEc2Image) id() (string, error) {
	return i.Arn.Data, nil
}

func initAwsEc2Image(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch aws ec2 image")
	}

	arnVal := args["arn"].Value.(string)
	arn, err := arn.Parse(arnVal)
	if err != nil {
		return nil, nil, err
	}
	resource := strings.Split(arn.Resource, "/")
	conn := runtime.Connection.(*connection.AwsConnection)
	svc := conn.Ec2(arn.Region)
	ctx := context.Background()
	images, err := svc.DescribeImages(ctx, &ec2.DescribeImagesInput{ImageIds: []string{resource[1]}})
	if err != nil {
		args["arn"] = llx.StringData(arnVal)
		args["id"] = llx.StringData(resource[1])
		args["name"] = llx.StringData("not found")
		args["architecture"] = llx.NilData
		args["ownerId"] = llx.NilData
		args["ownerAlias"] = llx.NilData
		args["createdAt"] = llx.NilData
		args["deprecatedAt"] = llx.NilData
		return args, nil, nil
	}

	if len(images.Images) > 0 {
		image := images.Images[0]
		args["arn"] = llx.StringData(arnVal)
		args["id"] = llx.StringData(resource[1])
		args["name"] = llx.StringData(convert.ToString(image.Name))
		args["architecture"] = llx.StringData(string(image.Architecture))
		args["ownerId"] = llx.StringData(convert.ToString(image.OwnerId))
		args["ownerAlias"] = llx.StringData(convert.ToString(image.ImageOwnerAlias))
		if image.CreationDate == nil {
			args["createdAt"] = llx.NilData
		} else {
			createdAt, err := time.Parse(time.RFC3339, *image.CreationDate)
			if err != nil {
				return nil, nil, err
			}
			args["createdAt"] = llx.TimeData(createdAt)
		}
		if image.DeprecationTime == nil {
			args["deprecatedAt"] = llx.NilData
		} else {
			deprecateTime, err := time.Parse(time.RFC3339, *image.DeprecationTime)
			if err != nil {
				return nil, nil, err
			}
			args["deprecatedAt"] = llx.TimeData(deprecateTime)
		}
		return args, nil, nil
	}

	return nil, nil, errors.New("image not found")
}

func (a *mqlAwsEc2Securitygroup) id() (string, error) {
	return a.Arn.Data, nil
}

func initAwsEc2Securitygroup(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	if args["arn"] == nil && args["id"] == nil {
		return nil, nil, errors.New("arn or id required to fetch aws security group")
	}

	// load all security groups
	obj, err := CreateResource(runtime, "aws.ec2", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	awsEc2 := obj.(*mqlAwsEc2)

	rawResources := awsEc2.GetSecurityGroups()
	if rawResources.Error != nil {
		return nil, nil, err
	}

	var match func(secGroup *mqlAwsEc2Securitygroup) bool

	if args["arn"] != nil {
		arnVal := args["arn"].Value.(string)
		match = func(secGroup *mqlAwsEc2Securitygroup) bool {
			return secGroup.Arn.Data == arnVal
		}
	}

	if args["id"] != nil {
		idVal := args["id"].Value.(string)
		match = func(secGroup *mqlAwsEc2Securitygroup) bool {
			return secGroup.Id.Data == idVal
		}
	}

	for i := range rawResources.Data {
		securityGroup := rawResources.Data[i].(*mqlAwsEc2Securitygroup)
		if match(securityGroup) {
			return args, securityGroup, nil
		}
	}

	return nil, nil, errors.New("security group does not exist")
}

func (a *mqlAwsEc2SecuritygroupIppermission) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAwsEc2InstanceDevice) id() (string, error) {
	return a.VolumeId.Data, nil
}

func (a *mqlAwsEc2Instance) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsEc2Instance) vpc() (*mqlAwsVpc, error) {
	vpcArn := a.VpcArn
	if vpcArn.State == plugin.StateIsNull {
		return nil, errors.New("ec2 instance has no vpc associated with it")
	} else if vpcArn.Error != nil {
		return nil, vpcArn.Error
	} else {
		res, err := NewResource(a.MqlRuntime, "aws.vpc", map[string]*llx.RawData{"arn": llx.StringData(vpcArn.Data)})
		if err != nil {
			return nil, err
		}
		return res.(*mqlAwsVpc), nil
	}
}

func (a *mqlAwsEc2Instance) ssm() (interface{}, error) {
	instanceId := a.InstanceId.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Ssm(region)
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
	res, err := convert.JsonToDict(ssmInstanceInfo)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (a *mqlAwsEc2Instance) patchState() (interface{}, error) {
	var res interface{}
	instanceId := a.InstanceId.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Ssm(region)
	ctx := context.Background()

	ssmPatchInfo, err := svc.DescribeInstancePatchStates(ctx, &ssm.DescribeInstancePatchStatesInput{InstanceIds: []string{instanceId}})
	if err != nil {
		return nil, err
	}
	if len(ssmPatchInfo.InstancePatchStates) > 0 {
		if instanceId == convert.ToString(ssmPatchInfo.InstancePatchStates[0].InstanceId) {
			res, err = convert.JsonToDict(ssmPatchInfo.InstancePatchStates[0])
			if err != nil {
				return nil, err
			}
		}
	}
	return res, nil
}

func (a *mqlAwsEc2Instance) instanceStatus() (interface{}, error) {
	var res interface{}
	instanceId := a.InstanceId.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Ec2(region)
	ctx := context.Background()

	instanceStatus, err := svc.DescribeInstanceStatus(ctx, &ec2.DescribeInstanceStatusInput{
		InstanceIds:         []string{instanceId},
		IncludeAllInstances: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	if len(instanceStatus.InstanceStatuses) > 0 {
		if instanceId == convert.ToString(instanceStatus.InstanceStatuses[0].InstanceId) {
			res, err = convert.JsonToDict(instanceStatus.InstanceStatuses[0])
			if err != nil {
				return nil, err
			}
		}
	}

	return res, nil
}

func (a *mqlAwsEc2) volumes() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getVolumes(conn), 5)
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

func (a *mqlAwsEc2) getVolumes(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)

	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Ec2(regionVal)
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
					jsonAttachments, err := convert.JsonToDictSlice(vol.Attachments)
					if err != nil {
						return nil, err
					}
					mqlVol, err := CreateResource(a.MqlRuntime, "aws.ec2.volume",
						map[string]*llx.RawData{
							"arn":                llx.StringData(fmt.Sprintf(volumeArnPattern, region, conn.AccountId(), convert.ToString(vol.VolumeId))),
							"attachments":        llx.ArrayData(jsonAttachments, types.Any),
							"availabilityZone":   llx.StringDataPtr(vol.AvailabilityZone),
							"createTime":         llx.TimeDataPtr(vol.CreateTime),
							"encrypted":          llx.BoolDataPtr(vol.Encrypted),
							"id":                 llx.StringDataPtr(vol.VolumeId),
							"iops":               llx.IntDataDefault(vol.Iops, 0),
							"multiAttachEnabled": llx.BoolDataPtr(vol.MultiAttachEnabled),
							"region":             llx.StringData(regionVal),
							"size":               llx.IntDataDefault(vol.Size, 0),
							"state":              llx.StringData(string(vol.State)),
							"tags":               llx.MapData(Ec2TagsToMap(vol.Tags), types.String),
							"throughput":         llx.IntDataDefault(vol.Throughput, 0),
							"volumeType":         llx.StringData(string(vol.VolumeType)),
						})
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

func initAwsEc2Volume(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["id"] = llx.StringData(ids.name)
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch aws volume")
	}

	// load all security groups
	obj, err := CreateResource(runtime, "aws.ec2", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	awsEc2 := obj.(*mqlAwsEc2)

	rawResources := awsEc2.GetVolumes()
	if rawResources.Error != nil {
		return nil, nil, err
	}

	var match func(secGroup *mqlAwsEc2Volume) bool

	if args["arn"] != nil {
		arnVal := args["arn"].Value.(string)
		match = func(vol *mqlAwsEc2Volume) bool {
			return vol.Arn.Data == arnVal
		}
	}

	for i := range rawResources.Data {
		volume := rawResources.Data[i].(*mqlAwsEc2Volume)
		if match(volume) {
			return args, volume, nil
		}
	}

	return nil, nil, errors.New("volume does not exist")
}

func initAwsEc2Instance(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	log.Debug().Msg("init an ec2 instance")
	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch ec2 instance")
	}

	obj, err := CreateResource(runtime, "aws.ec2", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	ec2 := obj.(*mqlAwsEc2)

	rawResources := ec2.GetInstances()
	if rawResources.Error != nil {
		return nil, nil, err
	}

	arnVal := args["arn"].Value.(string)
	for i := range rawResources.Data {
		instance := rawResources.Data[i].(*mqlAwsEc2Instance)
		if instance.Arn.Data == arnVal {
			return args, instance, nil
		}
	}
	return nil, nil, errors.New("ec2 instance does not exist")
}

func initAwsEc2Snapshot(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch aws snapshot")
	}

	// load all security groups
	obj, err := CreateResource(runtime, "aws.ec2", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	awsEc2 := obj.(*mqlAwsEc2)

	rawResources := awsEc2.GetSnapshots()
	if rawResources.Error != nil {
		return nil, nil, err
	}
	var match func(snapshot *mqlAwsEc2Snapshot) bool

	if args["arn"] != nil {
		arnVal := args["arn"].Value.(string)
		match = func(snapshot *mqlAwsEc2Snapshot) bool {
			return snapshot.Arn.Data == arnVal
		}
	}

	if args["id"] != nil {
		idVal := args["id"].Value.(string)
		match = func(snap *mqlAwsEc2Snapshot) bool {
			return snap.Id.Data == idVal
		}
	}

	for i := range rawResources.Data {
		snapshot := rawResources.Data[i].(*mqlAwsEc2Snapshot)
		if match(snapshot) {
			return args, snapshot, nil
		}
	}

	return nil, nil, errors.New("snapshot does not exist")
}

func (a *mqlAwsEc2Volume) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsEc2Snapshot) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsEc2) vpnConnections() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getVpnConnections(conn), 5)
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

func (a *mqlAwsEc2) getVpnConnections(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Ec2(regionVal)
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
					mqlVgwTelemetry, err := CreateResource(a.MqlRuntime, "aws.ec2.vgwtelemetry",
						map[string]*llx.RawData{
							"outsideIpAddress": llx.StringData(convert.ToString(vgwT.OutsideIpAddress)),
							"status":           llx.StringData(string(vgwT.Status)),
							"statusMessage":    llx.StringData(convert.ToString(vgwT.StatusMessage)),
						})
					if err != nil {
						return nil, err
					}
					mqlVgwT = append(mqlVgwT, mqlVgwTelemetry)
				}
				mqlVpnConn, err := CreateResource(a.MqlRuntime, "aws.ec2.vpnconnection",
					map[string]*llx.RawData{
						"arn":          llx.StringData(fmt.Sprintf(vpnConnArnPattern, regionVal, conn.AccountId(), convert.ToString(vpnConn.VpnConnectionId))),
						"vgwTelemetry": llx.ArrayData(mqlVgwT, types.Resource("aws.ec2.vgwtelemetry")),
					})
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

func (a *mqlAwsEc2) snapshots() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getSnapshots(conn), 5)
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

func (a *mqlAwsEc2) getSnapshots(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Ec2(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			nextToken := aws.String("no_token_to_start_with")
			params := &ec2.DescribeSnapshotsInput{Filters: []ec2types.Filter{{Name: aws.String("owner-id"), Values: []string{conn.AccountId()}}}}
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
					mqlSnap, err := CreateResource(a.MqlRuntime, "aws.ec2.snapshot",
						map[string]*llx.RawData{
							"arn":         llx.StringData(fmt.Sprintf(snapshotArnPattern, regionVal, conn.AccountId(), convert.ToString(snapshot.SnapshotId))),
							"description": llx.StringDataPtr(snapshot.Description),
							"encrypted":   llx.BoolDataPtr(snapshot.Encrypted),
							"id":          llx.StringDataPtr(snapshot.SnapshotId),
							"region":      llx.StringData(regionVal),
							"startTime":   llx.TimeDataPtr(snapshot.StartTime),
							"state":       llx.StringData(string(snapshot.State)),
							"tags":        llx.MapData(Ec2TagsToMap(snapshot.Tags), types.String),
							"volumeId":    llx.StringDataPtr(snapshot.VolumeId),
							"volumeSize":  llx.IntDataDefault(snapshot.VolumeSize, 0),
						})
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

func (a *mqlAwsEc2Snapshot) createVolumePermission() ([]interface{}, error) {
	id := a.Id.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Ec2(region)
	ctx := context.Background()

	attribute, err := svc.DescribeSnapshotAttribute(ctx, &ec2.DescribeSnapshotAttributeInput{SnapshotId: &id, Attribute: ec2types.SnapshotAttributeNameCreateVolumePermission})
	if err != nil {
		return nil, err
	}

	return convert.JsonToDictSlice(attribute.CreateVolumePermissions)
}

func (a *mqlAwsEc2) internetGateways() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getInternetGateways(conn), 5)
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

func (a *mqlAwsEc2) getInternetGateways(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Ec2(regionVal)
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
					jsonAttachments, err := convert.JsonToDictSlice(gateway.Attachments)
					if err != nil {
						return nil, err
					}
					mqlInternetGw, err := CreateResource(a.MqlRuntime, "aws.ec2.internetgateway",
						map[string]*llx.RawData{
							"arn":         llx.StringData(fmt.Sprintf(internetGwArnPattern, regionVal, convert.ToString(gateway.OwnerId), convert.ToString(gateway.InternetGatewayId))),
							"id":          llx.StringData(convert.ToString(gateway.InternetGatewayId)),
							"attachments": llx.ArrayData(jsonAttachments, types.Any),
						})
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

func (a *mqlAwsEc2Internetgateway) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsEc2Vpnconnection) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsEc2Vgwtelemetry) id() (string, error) {
	return a.OutsideIpAddress.Data, nil
}
