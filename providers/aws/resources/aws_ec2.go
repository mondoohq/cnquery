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
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v12/providers/aws/connection"
	"go.mondoo.com/cnquery/v12/types"
)

func (e *mqlAwsEc2) id() (string, error) {
	return ResourceAwsEc2, nil
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
		args["tags"] = llx.MapData(toInterfaceMap(ec2TagsToMap(add.Tags)), types.String)
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
		mqlEc2Instance, err := NewResource(a.MqlRuntime, ResourceAwsEc2Instance,
			map[string]*llx.RawData{
				"arn": llx.StringData(fmt.Sprintf(ec2InstanceArnPattern, regionVal, conn.AccountId(), convert.ToValue(instanceId))),
			})
		if err != nil {
			return nil, err
		}
		return mqlEc2Instance.(*mqlAwsEc2Instance), err
	}
	a.Instance.State = plugin.StateIsNull | plugin.StateIsSet
	return nil, nil
}

func (a *mqlAwsEc2) eips() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getEIPs(conn), 5)
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

func (a *mqlAwsEc2) getEIPs(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)

	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("ec2>getEIPs>calling aws with region %s", region)

			svc := conn.Ec2(region)
			ctx := context.Background()
			res := []any{}

			params := &ec2.DescribeAddressesInput{
				Filters: conn.Filters.General.ToServerSideEc2Filters(),
			} // no pagination
			addresses, err := svc.DescribeAddresses(ctx, params)
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", region).Msg("error accessing region for AWS API")
					return res, nil
				}
				return nil, err
			}

			for _, add := range addresses.Addresses {
				if conn.Filters.General.MatchesExcludeTags(ec2TagsToMap(add.Tags)) {
					log.Debug().Interface("acl", add.AllocationId).Msg("excluding elastic ip address due to filters")
					continue
				}

				attached := add.AllocationId != nil
				args := map[string]*llx.RawData{
					"publicIp":                llx.StringDataPtr(add.PublicIp),
					"attached":                llx.BoolData(attached), // this is false if allocationId is null and true otherwise
					"networkInterfaceId":      llx.StringDataPtr(add.NetworkInterfaceId),
					"networkInterfaceOwnerId": llx.StringDataPtr(add.NetworkInterfaceOwnerId),
					"privateIpAddress":        llx.StringDataPtr(add.PrivateIpAddress),
					"publicIpv4Pool":          llx.StringDataPtr(add.PublicIpv4Pool),
					"tags":                    llx.MapData(toInterfaceMap(ec2TagsToMap(add.Tags)), types.String),
					"region":                  llx.StringData(region),
				}
				mqlAddress, err := CreateResource(a.MqlRuntime, ResourceAwsEc2Eip, args)
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

func (a *mqlAwsEc2) networkAcls() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getNetworkACLs(conn), 5)
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

func (a *mqlAwsEc2) getNetworkACLs(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)

	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("ec2>getNetworkACLs>calling aws with region %s", region)

			svc := conn.Ec2(region)
			ctx := context.Background()
			res := []any{}

			params := &ec2.DescribeNetworkAclsInput{
				Filters: conn.Filters.General.ToServerSideEc2Filters(),
			}
			paginator := ec2.NewDescribeNetworkAclsPaginator(svc, params)
			for paginator.HasMorePages() {
				networkAcls, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, acl := range networkAcls.NetworkAcls {
					if conn.Filters.General.MatchesExcludeTags(ec2TagsToMap(acl.Tags)) {
						log.Debug().Interface("acl", acl.NetworkAclId).Msg("excluding network acl due to filters")
						continue
					}
					assoc := []any{}
					for _, association := range acl.Associations {
						mqlNetworkAclAssoc, err := CreateResource(a.MqlRuntime, ResourceAwsEc2NetworkaclAssociation,
							map[string]*llx.RawData{
								"associationId": llx.StringDataPtr(association.NetworkAclAssociationId),
								"networkAclId":  llx.StringDataPtr(association.NetworkAclId),
								"subnetId":      llx.StringDataPtr(association.SubnetId),
							})
						if err == nil {
							assoc = append(assoc, mqlNetworkAclAssoc)
						}
					}

					mqlNetworkAcl, err := CreateResource(a.MqlRuntime, ResourceAwsEc2Networkacl,
						map[string]*llx.RawData{
							"arn":          llx.StringData(fmt.Sprintf(networkAclArnPattern, region, conn.AccountId(), convert.ToValue(acl.NetworkAclId))),
							"id":           llx.StringDataPtr(acl.NetworkAclId),
							"region":       llx.StringData(region),
							"isDefault":    llx.BoolDataPtr(acl.IsDefault),
							"tags":         llx.MapData(toInterfaceMap(ec2TagsToMap(acl.Tags)), types.String),
							"associations": llx.ArrayData(assoc, types.Type(ResourceAwsEc2NetworkaclAssociation)),
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

func (a *mqlAwsEc2Networkacl) entries() ([]any, error) {
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

	res := []any{}
	for _, entry := range networkacls.NetworkAcls[0].Entries {
		egress := convert.ToValue(entry.Egress)
		entryId := fmt.Sprintf("%s-%d", id, convert.ToValue(entry.RuleNumber))
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
			mqlPortRange, err := CreateResource(a.MqlRuntime, ResourceAwsEc2NetworkaclEntryPortrange,
				map[string]*llx.RawData{
					"from": llx.IntDataDefault(entry.PortRange.From, -1),
					"to":   llx.IntDataDefault(entry.PortRange.To, -1),
					"id":   llx.StringData(fmt.Sprintf("%s-%d", entryId, convert.ToValue(entry.PortRange.From))),
				})
			if err != nil {
				return nil, err
			}
			args["portRange"] = llx.ResourceData(mqlPortRange, mqlPortRange.MqlName())
		} else {
			args["portRange"] = llx.NilData
		}

		mqlAclEntry, err := CreateResource(a.MqlRuntime, ResourceAwsEc2NetworkaclEntry, args)
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
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("ec2>getSecurityGroups>calling aws with region %s", region)

			svc := conn.Ec2(region)
			ctx := context.Background()
			res := []any{}

			params := &ec2.DescribeSecurityGroupsInput{
				Filters: conn.Filters.General.ToServerSideEc2Filters(),
			}
			paginator := ec2.NewDescribeSecurityGroupsPaginator(svc, params)
			for paginator.HasMorePages() {
				securityGroups, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, group := range securityGroups.SecurityGroups {
					if conn.Filters.General.MatchesExcludeTags(ec2TagsToMap(group.Tags)) {
						log.Debug().Interface("securitygroup", group.GroupId).Msg("excluding security group due to filters")
						continue
					}

					args := map[string]*llx.RawData{
						"arn":         llx.StringData(fmt.Sprintf(securityGroupArnPattern, region, conn.AccountId(), convert.ToValue(group.GroupId))),
						"id":          llx.StringDataPtr(group.GroupId),
						"name":        llx.StringDataPtr(group.GroupName),
						"description": llx.StringDataPtr(group.Description),
						"tags":        llx.MapData(toInterfaceMap(ec2TagsToMap(group.Tags)), types.String),
						"region":      llx.StringData(region),
					}

					mqlEc2SecurityGroup, err := CreateResource(a.MqlRuntime, ResourceAwsEc2Securitygroup, args)
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
		mqlVpc, err := NewResource(a.MqlRuntime, ResourceAwsVpc,
			map[string]*llx.RawData{
				"arn": llx.StringData(fmt.Sprintf(vpcArnPattern, a.region, conn.AccountId(), convert.ToValue(a.cacheVpc))),
			})
		if err != nil {
			return nil, err
		}
		return mqlVpc.(*mqlAwsVpc), nil
	}
	a.Vpc.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

func (a *mqlAwsEc2Securitygroup) ipPermissions() ([]any, error) {
	mqlIpPermissions := []any{}
	for p, permission := range a.cacheIpPerms {
		ipRanges := []any{}
		for r := range permission.IpRanges {
			iprange := permission.IpRanges[r]
			if iprange.CidrIp != nil {
				ipRanges = append(ipRanges, *iprange.CidrIp)
			}
		}

		ipv6Ranges := []any{}
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
		mqlSecurityGroupIpPermission, err := CreateResource(a.MqlRuntime, ResourceAwsEc2SecuritygroupIppermission,
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

func (a *mqlAwsEc2Securitygroup) ipPermissionsEgress() ([]any, error) {
	mqlIpPermissionsEgress := []any{}
	for p := range a.cacheIpPermsEgress {
		permission := a.cacheIpPermsEgress[p]

		ipRanges := []any{}
		for r := range permission.IpRanges {
			iprange := permission.IpRanges[r]
			if iprange.CidrIp != nil {
				ipRanges = append(ipRanges, *iprange.CidrIp)
			}
		}

		ipv6Ranges := []any{}
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
		mqlSecurityGroupIpPermission, err := CreateResource(a.MqlRuntime, ResourceAwsEc2SecuritygroupIppermission,
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

func (a *mqlAwsEc2) keypairs() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getKeypairs(conn), 5)
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
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("ec2>getKeypairs>calling aws with region %s", region)

			svc := conn.Ec2(region)
			ctx := context.Background()
			res := []any{}

			params := &ec2.DescribeKeyPairsInput{
				Filters: conn.Filters.General.ToServerSideEc2Filters(),
			}
			keyPairs, err := svc.DescribeKeyPairs(ctx, params)
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", region).Msg("error accessing region for AWS API")
					return res, nil
				}
				return nil, err
			}

			for _, kp := range keyPairs.KeyPairs {
				if conn.Filters.General.MatchesExcludeTags(ec2TagsToMap(kp.Tags)) {
					log.Debug().Interface("keypair", kp.KeyPairId).Msg("excluding keypair due to filters")
					continue
				}
				mqlKeypair, err := CreateResource(a.MqlRuntime, ResourceAwsEc2Keypair,
					map[string]*llx.RawData{
						"arn":         llx.StringData(fmt.Sprintf(keypairArnPattern, conn.AccountId(), region, convert.ToValue(kp.KeyPairId))),
						"fingerprint": llx.StringDataPtr(kp.KeyFingerprint),
						"name":        llx.StringDataPtr(kp.KeyName),
						"type":        llx.StringData(string(kp.KeyType)),
						"tags":        llx.MapData(toInterfaceMap(ec2TagsToMap(kp.Tags)), types.String),
						"region":      llx.StringData(region),
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
		args["fingerprint"] = llx.StringData(convert.ToValue(kp.KeyFingerprint))
		args["name"] = llx.StringData(convert.ToValue(kp.KeyName))
		args["type"] = llx.StringData(string(kp.KeyType))
		args["tags"] = llx.MapData(toInterfaceMap(ec2TagsToMap(kp.Tags)), types.String)
		args["region"] = llx.StringData(r)
		args["arn"] = llx.StringData(fmt.Sprintf(keypairArnPattern, conn.AccountId(), r, convert.ToValue(kp.KeyPairId)))
		args["createdAt"] = llx.TimeDataPtr(kp.CreateTime)

		return args, nil, nil
	}
	return args, nil, nil
}

func (a *mqlAwsEc2) images() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	var res []any
	poolOfJobs := jobpool.CreatePool(a.getImagesJob(conn), 5)

	poolOfJobs.Run()
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}

	for _, job := range poolOfJobs.Jobs {
		res = append(res, job.Result.([]any)...)
	}

	return res, nil
}

// createBlockDeviceMappings converts the AWS BlockDeviceMapping slice to MQL resources
func createBlockDeviceMappings(runtime *plugin.Runtime, imageArn string, mappings []ec2types.BlockDeviceMapping) ([]any, error) {
	result := make([]any, 0, len(mappings))
	for _, mapping := range mappings {
		deviceName := convert.ToValue(mapping.DeviceName)
		mappingID := fmt.Sprintf("%s/device/%s", imageArn, deviceName)

		args := map[string]*llx.RawData{
			"__id":        llx.StringData(mappingID),
			"deviceName":  llx.StringDataPtr(mapping.DeviceName),
			"virtualName": llx.StringDataPtr(mapping.VirtualName),
			"noDevice":    llx.BoolData(mapping.NoDevice != nil && *mapping.NoDevice != ""),
		}

		// Create an EBS block device resource if present
		if mapping.Ebs != nil {
			ebsID := fmt.Sprintf("%s/ebs", mappingID)
			mqlEbs, err := CreateResource(runtime, ResourceAwsEc2ImageEbsBlockDevice,
				map[string]*llx.RawData{
					"__id":                llx.StringData(ebsID),
					"encrypted":           llx.BoolDataPtr(mapping.Ebs.Encrypted),
					"snapshotId":          llx.StringDataPtr(mapping.Ebs.SnapshotId),
					"volumeSize":          llx.IntDataDefault(mapping.Ebs.VolumeSize, 0),
					"volumeType":          llx.StringData(string(mapping.Ebs.VolumeType)),
					"kmsKeyId":            llx.StringDataPtr(mapping.Ebs.KmsKeyId),
					"iops":                llx.IntDataDefault(mapping.Ebs.Iops, 0),
					"throughput":          llx.IntDataDefault(mapping.Ebs.Throughput, 0),
					"deleteOnTermination": llx.BoolDataPtr(mapping.Ebs.DeleteOnTermination),
				})
			if err != nil {
				return nil, err
			}
			args["ebs"] = llx.ResourceData(mqlEbs, mqlEbs.MqlName())
		} else {
			args["ebs"] = llx.NilData
		}

		mqlMapping, err := CreateResource(runtime, ResourceAwsEc2ImageBlockDeviceMapping, args)
		if err != nil {
			return nil, err
		}
		result = append(result, mqlMapping)
	}
	return result, nil
}

func (a *mqlAwsEc2) getImagesJob(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Str("region", region).Msgf("ec2>getImagesJob>calling aws with region")

			svc := conn.Ec2(region)
			ctx := context.Background()
			var res []any

			// Only fetch images owned by this account
			params := &ec2.DescribeImagesInput{
				Owners: []string{"self"},
			}
			paginator := ec2.NewDescribeImagesPaginator(svc, params)
			for paginator.HasMorePages() {
				images, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, image := range images.Images {
					if conn.Filters.General.MatchesExcludeTags(ec2TagsToMap(image.Tags)) {
						log.Debug().Interface("image", image.ImageId).Msg("excluding image due to filters")
						continue
					}

					// Create block device mapping MQL resources
					imageArn := fmt.Sprintf(imageArnPattern, region, conn.AccountId(), convert.ToValue(image.ImageId))
					blockDeviceMappings, err := createBlockDeviceMappings(a.MqlRuntime, imageArn, image.BlockDeviceMappings)
					if err != nil {
						return nil, err
					}

					// Parse creation date
					var createdAt *time.Time
					if image.CreationDate != nil {
						t, err := time.Parse(time.RFC3339, *image.CreationDate)
						if err != nil {
							log.Warn().Str("imageId", convert.ToValue(image.ImageId)).Err(err).
								Str("bad_value", *image.CreationDate).Msg("failed to parse image CreationDate")
						} else {
							createdAt = &t
						}
					}
					// Parse deprecation date
					var deprecatedAt *time.Time
					if image.DeprecationTime != nil {
						t, err := time.Parse(time.RFC3339, *image.DeprecationTime)
						if err != nil {
							log.Warn().Str("imageId", convert.ToValue(image.ImageId)).Err(err).
								Str("bad_value", *image.DeprecationTime).Msg("failed to parse image DeprecationTime")
						} else {
							deprecatedAt = &t
						}
					}
					mqlImage, err := CreateResource(a.MqlRuntime, ResourceAwsEc2Image,
						map[string]*llx.RawData{
							"arn":                 llx.StringData(imageArn),
							"id":                  llx.StringDataPtr(image.ImageId),
							"name":                llx.StringDataPtr(image.Name),
							"architecture":        llx.StringData(string(image.Architecture)),
							"ownerId":             llx.StringDataPtr(image.OwnerId),
							"ownerAlias":          llx.StringDataPtr(image.ImageOwnerAlias),
							"createdAt":           llx.TimeDataPtr(createdAt),
							"deprecatedAt":        llx.TimeDataPtr(deprecatedAt),
							"enaSupport":          llx.BoolDataPtr(image.EnaSupport),
							"tpmSupport":          llx.StringData(string(image.TpmSupport)),
							"state":               llx.StringData(string(image.State)),
							"public":              llx.BoolDataPtr(image.Public),
							"rootDeviceType":      llx.StringData(string(image.RootDeviceType)),
							"virtualizationType":  llx.StringData(string(image.VirtualizationType)),
							"blockDeviceMappings": llx.ArrayData(blockDeviceMappings, types.Resource(ResourceAwsEc2ImageBlockDeviceMapping)),
							"tags":                llx.MapData(toInterfaceMap(ec2TagsToMap(image.Tags)), types.String),
							"region":              llx.StringData(region),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlImage)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsEc2) securityGroups() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getSecurityGroups(conn), 5)
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

type ebsEncryption struct {
	region                 string
	ebsEncryptionByDefault bool
}

func (a *mqlAwsEc2) ebsEncryptionByDefault() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := make(map[string]any)
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
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("ec2>getEbsEncryptionPerRegion>calling aws with region %s", region)

			svc := conn.Ec2(region)
			ctx := context.Background()

			ebsEncryptionRes, err := svc.GetEbsEncryptionByDefault(ctx, &ec2.GetEbsEncryptionByDefaultInput{})
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", region).Msg("error accessing region for AWS API")
					return nil, nil
				}
				return nil, err
			}
			structVal := ebsEncryption{
				region:                 region,
				ebsEncryptionByDefault: convert.ToValue(ebsEncryptionRes.EbsEncryptionByDefault),
			}
			return jobpool.JobResult(structVal), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsEc2) instances() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getInstances(conn), 5)
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

func (a *mqlAwsEc2) getEc2Instances(ctx context.Context, svc *ec2.Client, filters connection.DiscoveryFilters) ([]ec2types.Instance, error) {
	res := []ec2types.Instance{}
	paginator := ec2.NewDescribeInstancesPaginator(svc, &ec2.DescribeInstancesInput{
		Filters:     filters.General.ToServerSideEc2Filters(),
		InstanceIds: filters.Ec2.InstanceIds,
	})
	for paginator.HasMorePages() {
		reservations, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, reservation := range reservations.Reservations {
			for _, instance := range reservation.Instances {
				if shouldExcludeInstance(instance, filters) {
					log.Debug().Interface("instance", instance.InstanceId).Msg("excluding ec2 instance due to filters")
					continue
				}
				res = append(res, instance)
			}
		}
	}
	return res, nil
}

func (a *mqlAwsEc2) getInstances(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("ec2>getInstances>calling aws with region %s", region)

			svc := conn.Ec2(region)
			ctx := context.Background()
			var res []any

			instances, err := a.getEc2Instances(ctx, svc, conn.Filters)
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", region).Msg("error accessing region for AWS API")
					return res, nil
				}
				// AWS returns an error response when trying to find an instance with a specific identifier if it cannot find it in some region.
				// we do not propagate this error upward because an instance can be found in one region and return an error for all others which
				// would be the expected behavior.
				if Is400InstanceNotFoundError(err) {
					log.Debug().Str("region", region).Msg("could not find instance in region")
					return res, nil
				}
				return nil, err
			}
			res, err = a.gatherInstanceInfo(instances, region)
			if err != nil {
				return nil, err
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsEc2) gatherInstanceInfo(instances []ec2types.Instance, regionVal string) ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	for _, instance := range instances {
		mqlDevices := []any{}
		for _, device := range instance.BlockDeviceMappings {
			mqlInstanceDevice, err := CreateResource(a.MqlRuntime, ResourceAwsEc2InstanceDevice,
				map[string]*llx.RawData{
					"deleteOnTermination": llx.BoolData(convert.ToValue(device.Ebs.DeleteOnTermination)),
					"status":              llx.StringData(string(device.Ebs.Status)),
					"volumeId":            llx.StringData(convert.ToValue(device.Ebs.VolumeId)),
					"deviceName":          llx.StringData(convert.ToValue(device.DeviceName)),
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
		timeString := reg.FindStringSubmatch(convert.ToValue(instance.StateTransitionReason))
		if len(timeString) == 2 {
			stateTransitionTime, err = time.Parse(time.DateTime, timeString[1])
			if err != nil {
				log.Error().Err(err).Msg("cannot parse state transition time for ec2 instance")
				stateTransitionTime = llx.NeverPastTime
			}
		}
		args := map[string]*llx.RawData{
			"architecture":       llx.StringData(string(instance.Architecture)),
			"arn":                llx.StringData(fmt.Sprintf(ec2InstanceArnPattern, regionVal, conn.AccountId(), convert.ToValue(instance.InstanceId))),
			"detailedMonitoring": llx.StringData(string(instance.Monitoring.State)),
			"deviceMappings":     llx.ArrayData(mqlDevices, types.Type(ResourceAwsEc2InstanceDevice)),
			"ebsOptimized":       llx.BoolDataPtr(instance.EbsOptimized),
			"enaSupported":       llx.BoolDataPtr(instance.EnaSupport),
			"hypervisor":         llx.StringData(string(instance.Hypervisor)),
			"instanceId":         llx.StringDataPtr(instance.InstanceId),
			"instanceLifecycle":  llx.StringData(string(instance.InstanceLifecycle)),
			"instanceType":       llx.StringData(string(instance.InstanceType)),
			"launchTime":         llx.TimeDataPtr(instance.LaunchTime),
			"platformDetails":    llx.StringDataPtr(instance.PlatformDetails),
			"privateDnsName":     llx.StringDataPtr(instance.PrivateDnsName),
			"privateIp":          llx.StringDataPtr(instance.PrivateIpAddress),
			"publicDnsName":      llx.StringDataPtr(instance.PublicDnsName),
			"publicIp":           llx.StringDataPtr(instance.PublicIpAddress),
			"region":             llx.StringData(regionVal),
			"rootDeviceName":     llx.StringDataPtr(instance.RootDeviceName),
			"rootDeviceType":     llx.StringData(string(instance.RootDeviceType)),
			"state":              llx.StringData(string(instance.State.Name)),
			"stateReason":        llx.MapData(stateReason, types.Any),
			// "iamInstanceProfile":    llx.MapData(iamInstanceProfile, types.Any),
			"stateTransitionReason": llx.StringDataPtr(instance.StateTransitionReason),
			"stateTransitionTime":   llx.TimeData(stateTransitionTime),
			"tags":                  llx.MapData(toInterfaceMap(ec2TagsToMap(instance.Tags)), types.String),
			"tpmSupport":            llx.StringDataPtr(instance.TpmSupport),
		}

		if instance.MetadataOptions != nil {
			args["httpEndpoint"] = llx.StringData(string(instance.MetadataOptions.HttpEndpoint))
			args["httpTokens"] = llx.StringData(string(instance.MetadataOptions.HttpTokens))
		} else {
			args["httpEndpoint"] = llx.NilData
			args["httpTokens"] = llx.NilData
		}
		// add vpc if there is one
		if instance.VpcId != nil {
			arn := fmt.Sprintf(vpcArnPattern, regionVal, conn.AccountId(), convert.ToValue(instance.VpcId))
			args["vpcArn"] = llx.StringData(arn)
		} else {
			args["vpcArn"] = llx.NilData
		}

		mqlEc2Instance, err := CreateResource(a.MqlRuntime, ResourceAwsEc2Instance, args)
		if err != nil {
			return nil, err
		}
		mqlEc2Instance.(*mqlAwsEc2Instance).instanceCache = instance
		res = append(res, mqlEc2Instance)
	}
	return res, nil
}

type mqlAwsEc2InstanceInternal struct {
	instanceCache ec2types.Instance
}

func (i *mqlAwsEc2Instance) networkInterfaces() ([]any, error) {
	conn := i.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Ec2(i.Region.Data)
	ctx := context.Background()
	filters := conn.Filters.General.ToServerSideEc2Filters()
	filters = append(filters, ec2types.Filter{Name: aws.String("attachment.instance-id"), Values: []string{i.InstanceId.Data}})
	params := &ec2.DescribeNetworkInterfacesInput{Filters: filters}
	res := []any{}
	paginator := ec2.NewDescribeNetworkInterfacesPaginator(svc, params)
	for paginator.HasMorePages() {
		nis, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, networkingInterface := range nis.NetworkInterfaces {
			if conn.Filters.General.MatchesExcludeTags(ec2TagsToMap(networkingInterface.TagSet)) {
				log.Debug().Interface("networkInterface", networkingInterface.NetworkInterfaceId).Msg("excluding network interface due to filters")
				continue
			}
			args := map[string]*llx.RawData{
				"availabilityZone": llx.StringDataPtr(networkingInterface.AvailabilityZone),
				"description":      llx.StringDataPtr(networkingInterface.Description),
				"id":               llx.StringDataPtr(networkingInterface.NetworkInterfaceId),
				"ipv6Native":       llx.BoolDataPtr(networkingInterface.Ipv6Native),
				"macAddress":       llx.StringDataPtr(networkingInterface.MacAddress),
				"privateDnsName":   llx.StringDataPtr(networkingInterface.PrivateDnsName),
				"privateIpAddress": llx.StringDataPtr(networkingInterface.PrivateIpAddress),
				"requesterManaged": llx.BoolDataPtr(networkingInterface.RequesterManaged),
				"sourceDestCheck":  llx.BoolDataPtr(networkingInterface.SourceDestCheck),
				"status":           llx.StringData(string(networkingInterface.Status)),
				"tags":             llx.MapData(toInterfaceMap(ec2TagsToMap(networkingInterface.TagSet)), types.String),
			}
			mqlNetworkInterface, err := CreateResource(i.MqlRuntime, ResourceAwsEc2Networkinterface, args)
			if err != nil {
				return nil, err
			}
			mqlNetworkInterface.(*mqlAwsEc2Networkinterface).networkInterfaceCache = networkingInterface
			mqlNetworkInterface.(*mqlAwsEc2Networkinterface).region = i.Region.Data
			res = append(res, mqlNetworkInterface)
		}
	}
	return res, nil
}

type mqlAwsEc2NetworkinterfaceInternal struct {
	networkInterfaceCache ec2types.NetworkInterface
	region                string
}

func (i *mqlAwsEc2Networkinterface) securityGroups() ([]any, error) {
	if i.networkInterfaceCache.Groups != nil {
		sgs := []any{}
		conn := i.MqlRuntime.Connection.(*connection.AwsConnection)

		for _, group := range i.networkInterfaceCache.Groups {
			mqlSg, err := NewResource(i.MqlRuntime, ResourceAwsEc2Securitygroup,
				map[string]*llx.RawData{"arn": llx.StringData(fmt.Sprintf(securityGroupArnPattern, i.region, conn.AccountId(), *group.GroupId))})
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
		res, err := NewResource(i.MqlRuntime, ResourceAwsVpcSubnet, map[string]*llx.RawData{"arn": llx.StringData(arn)})
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
		vpcArn := fmt.Sprintf(vpcArnPattern, i.region, conn.AccountId(), convert.ToValue(vpcId))
		res, err := NewResource(i.MqlRuntime, ResourceAwsVpc, map[string]*llx.RawData{"arn": llx.StringData(vpcArn)})
		if err != nil {
			return nil, err
		}
		return res.(*mqlAwsVpc), nil
	}
	i.Vpc.State = plugin.StateIsNull | plugin.StateIsSet
	return nil, nil
}

func (i *mqlAwsEc2Instance) securityGroups() ([]any, error) {
	if i.instanceCache.SecurityGroups != nil {
		sgs := []any{}
		conn := i.MqlRuntime.Connection.(*connection.AwsConnection)

		for _, sg := range i.instanceCache.SecurityGroups {
			mqlSg, err := NewResource(i.MqlRuntime, ResourceAwsEc2Securitygroup,
				map[string]*llx.RawData{"arn": llx.StringData(fmt.Sprintf(securityGroupArnPattern, i.Region.Data, conn.AccountId(), convert.ToValue(sg.GroupId)))})
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

		mqlImage, err := NewResource(i.MqlRuntime, ResourceAwsEc2Image,
			map[string]*llx.RawData{"arn": llx.StringData(fmt.Sprintf(imageArnPattern, i.Region.Data, conn.AccountId(), convert.ToValue(i.instanceCache.ImageId)))})
		if err == nil {
			return mqlImage.(*mqlAwsEc2Image), nil
		}
	}
	i.Image.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

func (i *mqlAwsEc2Instance) keypair() (*mqlAwsEc2Keypair, error) {
	if i.instanceCache.KeyName != nil {
		mqlKeyPair, err := NewResource(i.MqlRuntime, ResourceAwsEc2Keypair,
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

func (i *mqlAwsEc2Image) launchPermissions() ([]interface{}, error) {
	imageId := i.Id.Data
	region := i.Region.Data
	conn := i.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Ec2(region)
	ctx := context.Background()

	result, err := svc.DescribeImageAttribute(ctx, &ec2.DescribeImageAttributeInput{
		ImageId:   aws.String(imageId),
		Attribute: ec2types.ImageAttributeNameLaunchPermission,
	})
	if err != nil {
		return nil, err
	}

	imageArn := i.Arn.Data
	permissions := make([]interface{}, 0, len(result.LaunchPermissions))
	for _, perm := range result.LaunchPermissions {
		// Build unique ID based on which field is set
		var permId string
		switch {
		case perm.UserId != nil:
			permId = fmt.Sprintf("%s/user/%s", imageArn, *perm.UserId)
		case perm.Group != "":
			permId = fmt.Sprintf("%s/group/%s", imageArn, string(perm.Group))
		case perm.OrganizationArn != nil:
			permId = fmt.Sprintf("%s/org/%s", imageArn, *perm.OrganizationArn)
		case perm.OrganizationalUnitArn != nil:
			permId = fmt.Sprintf("%s/ou/%s", imageArn, *perm.OrganizationalUnitArn)
		default:
			permId = fmt.Sprintf("%s/unknown", imageArn)
		}

		mqlPermission, err := CreateResource(i.MqlRuntime, ResourceAwsEc2ImageLaunchPermission,
			map[string]*llx.RawData{
				"__id":                  llx.StringData(permId),
				"userId":                llx.StringDataPtr(perm.UserId),
				"group":                 llx.StringData(string(perm.Group)),
				"organizationArn":       llx.StringDataPtr(perm.OrganizationArn),
				"organizationalUnitArn": llx.StringDataPtr(perm.OrganizationalUnitArn),
			})
		if err != nil {
			return nil, err
		}
		permissions = append(permissions, mqlPermission)
	}

	return permissions, nil
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
		args["tpmSupport"] = llx.NilData
		args["enaSupport"] = llx.NilData
		args["state"] = llx.NilData
		args["public"] = llx.NilData
		args["rootDeviceType"] = llx.NilData
		args["virtualizationType"] = llx.NilData
		args["blockDeviceMappings"] = llx.NilData
		args["tags"] = llx.NilData
		args["region"] = llx.StringData(arn.Region)
		return args, nil, nil
	}

	if len(images.Images) > 0 {
		image := images.Images[0]

		// Create block device mapping MQL resources
		blockDeviceMappings, err := createBlockDeviceMappings(runtime, arnVal, image.BlockDeviceMappings)
		if err != nil {
			return nil, nil, err
		}

		args["arn"] = llx.StringData(arnVal)
		args["id"] = llx.StringData(resource[1])
		args["name"] = llx.StringDataPtr(image.Name)
		args["architecture"] = llx.StringData(string(image.Architecture))
		args["ownerId"] = llx.StringDataPtr(image.OwnerId)
		args["ownerAlias"] = llx.StringDataPtr(image.ImageOwnerAlias)
		args["enaSupport"] = llx.BoolDataPtr(image.EnaSupport)
		args["tpmSupport"] = llx.StringData(string(image.TpmSupport))
		args["state"] = llx.StringData(string(image.State))
		args["public"] = llx.BoolDataPtr(image.Public)
		args["rootDeviceType"] = llx.StringData(string(image.RootDeviceType))
		args["virtualizationType"] = llx.StringData(string(image.VirtualizationType))
		args["blockDeviceMappings"] = llx.ArrayData(blockDeviceMappings, types.Resource(ResourceAwsEc2ImageBlockDeviceMapping))
		args["tags"] = llx.MapData(toInterfaceMap(ec2TagsToMap(image.Tags)), types.String)
		args["region"] = llx.StringData(arn.Region)
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
	obj, err := CreateResource(runtime, ResourceAwsEc2, map[string]*llx.RawData{})
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

	for _, rawResource := range rawResources.Data {
		securityGroup := rawResource.(*mqlAwsEc2Securitygroup)
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

func (a *mqlAwsEc2Instance) ssm() (any, error) {
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

func (a *mqlAwsEc2Instance) patchState() (any, error) {
	var res any
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
		if instanceId == convert.ToValue(ssmPatchInfo.InstancePatchStates[0].InstanceId) {
			res, err = convert.JsonToDict(ssmPatchInfo.InstancePatchStates[0])
			if err != nil {
				return nil, err
			}
		}
	}
	return res, nil
}

func (a *mqlAwsEc2Instance) instanceStatus() (any, error) {
	var res any
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
		if instanceId == convert.ToValue(instanceStatus.InstanceStatuses[0].InstanceId) {
			res, err = convert.JsonToDict(instanceStatus.InstanceStatuses[0])
			if err != nil {
				return nil, err
			}
		}
	}

	return res, nil
}

func (a *mqlAwsEc2Instance) disableApiTermination() (bool, error) {
	instanceId := a.InstanceId.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Ec2(region)
	ctx := context.Background()

	result, err := svc.DescribeInstanceAttribute(ctx, &ec2.DescribeInstanceAttributeInput{
		InstanceId: aws.String(instanceId),
		Attribute:  ec2types.InstanceAttributeNameDisableApiTermination,
	})
	if err != nil {
		return false, err
	}

	if result.DisableApiTermination != nil && result.DisableApiTermination.Value != nil {
		return *result.DisableApiTermination.Value, nil
	}

	return false, nil
}

// # go.mondoo.com/cnquery/v12/providers/aws/resources
// resources/aws.lr.go:15420:12: c.iamRole undefined (type *mqlAwsIamInstanceProfile has no field or method iamRole, but does have field IamRole)
// make[1]: *** [providers/build/aws] Error 1

// x failed to build provider error="exit status 2" provider=aws

func (a *mqlAwsEc2Instance) iamInstanceProfile() (*mqlAwsIamInstanceProfile, error) {
	if a.instanceCache.IamInstanceProfile == nil {
		a.IamInstanceProfile.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	arn := a.instanceCache.IamInstanceProfile.Arn

	res, err := NewResource(a.MqlRuntime, ResourceAwsIamInstanceProfile, map[string]*llx.RawData{
		"arn": llx.StringDataPtr(arn),
	})
	if err != nil {
		return nil, err
	}

	return res.(*mqlAwsIamInstanceProfile), nil
}

func (a *mqlAwsEc2) volumes() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getVolumes(conn), 5)
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

func (a *mqlAwsEc2) getVolumes(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)

	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Ec2(region)
			ctx := context.Background()
			res := []any{}

			params := &ec2.DescribeVolumesInput{
				Filters: conn.Filters.General.ToServerSideEc2Filters(),
			}
			paginator := ec2.NewDescribeVolumesPaginator(svc, params)
			for paginator.HasMorePages() {
				volumes, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, vol := range volumes.Volumes {
					if conn.Filters.General.MatchesExcludeTags(ec2TagsToMap(vol.Tags)) {
						log.Debug().Interface("volume", vol.VolumeId).Msg("excluding volume due to filters")
						continue
					}
					jsonAttachments, err := convert.JsonToDictSlice(vol.Attachments)
					if err != nil {
						return nil, err
					}
					mqlVol, err := CreateResource(a.MqlRuntime, ResourceAwsEc2Volume,
						map[string]*llx.RawData{
							"arn":                llx.StringData(fmt.Sprintf(volumeArnPattern, region, conn.AccountId(), convert.ToValue(vol.VolumeId))),
							"attachments":        llx.ArrayData(jsonAttachments, types.Any),
							"availabilityZone":   llx.StringDataPtr(vol.AvailabilityZone),
							"createTime":         llx.TimeDataPtr(vol.CreateTime),
							"encrypted":          llx.BoolDataPtr(vol.Encrypted),
							"id":                 llx.StringDataPtr(vol.VolumeId),
							"iops":               llx.IntDataDefault(vol.Iops, 0),
							"multiAttachEnabled": llx.BoolDataPtr(vol.MultiAttachEnabled),
							"region":             llx.StringData(region),
							"size":               llx.IntDataDefault(vol.Size, 0),
							"state":              llx.StringData(string(vol.State)),
							"tags":               llx.MapData(toInterfaceMap(ec2TagsToMap(vol.Tags)), types.String),
							"throughput":         llx.IntDataDefault(vol.Throughput, 0),
							"volumeType":         llx.StringData(string(vol.VolumeType)),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlVol)
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
	obj, err := CreateResource(runtime, ResourceAwsEc2, map[string]*llx.RawData{})
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

	for _, rawResource := range rawResources.Data {
		volume := rawResource.(*mqlAwsEc2Volume)
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

	obj, err := CreateResource(runtime, ResourceAwsEc2, map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	ec2 := obj.(*mqlAwsEc2)

	rawResources := ec2.GetInstances()
	if rawResources.Error != nil {
		return nil, nil, err
	}

	arnVal := args["arn"].Value.(string)
	for _, rawResource := range rawResources.Data {
		instance := rawResource.(*mqlAwsEc2Instance)
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
	obj, err := CreateResource(runtime, ResourceAwsEc2, map[string]*llx.RawData{})
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

	for _, rawResource := range rawResources.Data {
		snapshot := rawResource.(*mqlAwsEc2Snapshot)
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

func (a *mqlAwsEc2) vpnConnections() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getVpnConnections(conn), 5)
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

func (a *mqlAwsEc2) getVpnConnections(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Ec2(region)
			ctx := context.Background()
			res := []any{}

			vpnConnections, err := svc.DescribeVpnConnections(ctx, &ec2.DescribeVpnConnectionsInput{
				Filters: conn.Filters.General.ToServerSideEc2Filters(),
			})
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", region).Msg("error accessing region for AWS API")
					return res, nil
				}
				return nil, err
			}
			for _, vpnConn := range vpnConnections.VpnConnections {
				if conn.Filters.General.MatchesExcludeTags(ec2TagsToMap(vpnConn.Tags)) {
					log.Debug().Interface("vpnConnection", vpnConn.VpnConnectionId).Msg("excluding vpn connection due to filters")
					continue
				}
				mqlVgwT := []any{}
				for _, vgwT := range vpnConn.VgwTelemetry {
					mqlVgwTelemetry, err := CreateResource(a.MqlRuntime, ResourceAwsEc2Vgwtelemetry,
						map[string]*llx.RawData{
							"outsideIpAddress": llx.StringData(convert.ToValue(vgwT.OutsideIpAddress)),
							"status":           llx.StringData(string(vgwT.Status)),
							"statusMessage":    llx.StringData(convert.ToValue(vgwT.StatusMessage)),
						})
					if err != nil {
						return nil, err
					}
					mqlVgwT = append(mqlVgwT, mqlVgwTelemetry)
				}
				mqlVpnConn, err := CreateResource(a.MqlRuntime, ResourceAwsEc2Vpnconnection,
					map[string]*llx.RawData{
						"arn":          llx.StringData(fmt.Sprintf(vpnConnArnPattern, region, conn.AccountId(), convert.ToValue(vpnConn.VpnConnectionId))),
						"vgwTelemetry": llx.ArrayData(mqlVgwT, types.Resource(ResourceAwsEc2Vgwtelemetry)),
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

func (a *mqlAwsEc2) snapshots() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getSnapshots(conn), 5)
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

func (a *mqlAwsEc2) getSnapshots(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Ec2(region)
			ctx := context.Background()
			res := []any{}

			filters := conn.Filters.General.ToServerSideEc2Filters()
			filters = append(filters, ec2types.Filter{Name: aws.String("owner-id"), Values: []string{conn.AccountId()}})
			params := &ec2.DescribeSnapshotsInput{
				Filters: filters,
			}
			paginator := ec2.NewDescribeSnapshotsPaginator(svc, params)
			for paginator.HasMorePages() {
				snapshots, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, snapshot := range snapshots.Snapshots {
					if conn.Filters.General.MatchesExcludeTags(ec2TagsToMap(snapshot.Tags)) {
						log.Debug().Interface("snapshot", snapshot.SnapshotId).Msg("excluding snapshot due to filters")
						continue
					}
					mqlSnap, err := CreateResource(a.MqlRuntime, ResourceAwsEc2Snapshot,
						map[string]*llx.RawData{
							"arn":            llx.StringData(fmt.Sprintf(snapshotArnPattern, region, conn.AccountId(), convert.ToValue(snapshot.SnapshotId))),
							"completionTime": llx.TimeDataPtr(snapshot.CompletionTime),
							"description":    llx.StringDataPtr(snapshot.Description),
							"encrypted":      llx.BoolDataPtr(snapshot.Encrypted),
							"id":             llx.StringDataPtr(snapshot.SnapshotId),
							"region":         llx.StringData(region),
							"startTime":      llx.TimeDataPtr(snapshot.StartTime),
							"state":          llx.StringData(string(snapshot.State)),
							"storageTier":    llx.StringData(string(snapshot.StorageTier)),
							"tags":           llx.MapData(toInterfaceMap(ec2TagsToMap(snapshot.Tags)), types.String),
							"volumeId":       llx.StringDataPtr(snapshot.VolumeId),
							"volumeSize":     llx.IntDataDefault(snapshot.VolumeSize, 0),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlSnap)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsEc2Snapshot) createVolumePermission() ([]any, error) {
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

func (a *mqlAwsEc2) internetGateways() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getInternetGateways(conn), 5)
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

func (a *mqlAwsEc2) getInternetGateways(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Ec2(region)
			ctx := context.Background()
			params := &ec2.DescribeInternetGatewaysInput{
				Filters: conn.Filters.General.ToServerSideEc2Filters(),
			}
			res := []any{}
			paginator := ec2.NewDescribeInternetGatewaysPaginator(svc, params)
			for paginator.HasMorePages() {
				internetGws, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, gateway := range internetGws.InternetGateways {
					if conn.Filters.General.MatchesExcludeTags(ec2TagsToMap(gateway.Tags)) {
						log.Debug().Interface("igw", gateway.InternetGatewayId).Msg("excluding internet gateway due to filters")
						continue
					}
					jsonAttachments, err := convert.JsonToDictSlice(gateway.Attachments)
					if err != nil {
						return nil, err
					}
					mqlInternetGw, err := CreateResource(a.MqlRuntime, ResourceAwsEc2Internetgateway,
						map[string]*llx.RawData{
							"arn":         llx.StringData(fmt.Sprintf(internetGwArnPattern, region, convert.ToValue(gateway.OwnerId), convert.ToValue(gateway.InternetGatewayId))),
							"id":          llx.StringData(convert.ToValue(gateway.InternetGatewayId)),
							"attachments": llx.ArrayData(jsonAttachments, types.Any),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlInternetGw)
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

// true if the instance should be excluded from results. filtering for excluded regions should happen before we retrieve the EC2 instance.
func shouldExcludeInstance(instance ec2types.Instance, filters connection.DiscoveryFilters) bool {
	hasExcludedId := filters.Ec2.MatchesExcludeInstanceIds(instance.InstanceId)
	hasExcludedTag := filters.General.MatchesExcludeTags(ec2TagsToMap(instance.Tags))
	return hasExcludedId || hasExcludedTag
}

// tags in AWS are guaranteed to have a unique key, so we can convert the slice to a map for easier processing
func ec2TagsToMap(tags []ec2types.Tag) map[string]string {
	result := make(map[string]string)
	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			result[*tag.Key] = *tag.Value
		}
	}
	return result
}
