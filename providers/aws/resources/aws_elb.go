// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbtypes "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v11/providers/aws/connection"
	"go.mondoo.com/cnquery/v11/types"
)

func (a *mqlAwsElb) id() (string, error) {
	return "aws.elb", nil
}

func (a *mqlAwsElb) classicLoadBalancers() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getClassicLoadBalancers(conn), 5)
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

func (a *mqlAwsElb) getClassicLoadBalancers(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Elb(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			var marker *string
			for {
				lbs, err := svc.DescribeLoadBalancers(ctx, &elasticloadbalancing.DescribeLoadBalancersInput{Marker: marker})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, lb := range lbs.LoadBalancerDescriptions {
					jsonListeners, err := convert.JsonToDictSlice(lb.ListenerDescriptions)
					if err != nil {
						return nil, err
					}
					mqlLb, err := CreateResource(a.MqlRuntime, "aws.elb.loadbalancer",
						map[string]*llx.RawData{
							"arn":                  llx.StringData(fmt.Sprintf(elbv1LbArnPattern, regionVal, conn.AccountId(), convert.ToString(lb.LoadBalancerName))),
							"createdTime":          llx.TimeDataPtr(lb.CreatedTime),
							"dnsName":              llx.StringDataPtr(lb.DNSName),
							"elbType":              llx.StringData("classic"),
							"listenerDescriptions": llx.AnyData(jsonListeners),
							"name":                 llx.StringDataPtr(lb.LoadBalancerName),
							"region":               llx.StringData(regionVal),
							"scheme":               llx.StringDataPtr(lb.Scheme),
							"vpcId":                llx.StringDataPtr(lb.VPCId),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlLb)
				}
				if lbs.NextMarker == nil {
					break
				}
				marker = lbs.NextMarker
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsElbLoadbalancer) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsElb) loadBalancers() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getLoadBalancers(conn), 5)
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

func (a *mqlAwsElb) getLoadBalancers(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Elbv2(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			var marker *string
			for {
				lbs, err := svc.DescribeLoadBalancers(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{Marker: marker})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, lb := range lbs.LoadBalancers {
					availabilityZones := []interface{}{}
					for _, zone := range lb.AvailabilityZones {
						availabilityZones = append(availabilityZones, convert.ToString(zone.ZoneName))
					}

					sgs := []interface{}{}
					for i := range lb.SecurityGroups {
						sg := lb.SecurityGroups[i]
						mqlSg, err := NewResource(a.MqlRuntime, "aws.ec2.securitygroup",
							map[string]*llx.RawData{
								"arn": llx.StringData(fmt.Sprintf(securityGroupArnPattern, regionVal, conn.AccountId(), sg)),
							})
						if err != nil {
							return nil, err
						}
						sgs = append(sgs, mqlSg)
					}

					args := map[string]*llx.RawData{
						"arn":               llx.StringDataPtr(lb.LoadBalancerArn),
						"availabilityZones": llx.ArrayData(availabilityZones, types.String),
						"createdTime":       llx.TimeDataPtr(lb.CreatedTime),
						"dnsName":           llx.StringDataPtr(lb.DNSName),
						"hostedZoneId":      llx.StringDataPtr(lb.CanonicalHostedZoneId),
						"name":              llx.StringDataPtr(lb.LoadBalancerName),
						"scheme":            llx.StringData(string(lb.Scheme)),
						"securityGroups":    llx.ArrayData(sgs, types.Resource("aws.ec2.securitygroup")),
						"vpcId":             llx.StringDataPtr(lb.VpcId),
						"elbType":           llx.StringData(string(lb.Type)),
						"region":            llx.StringData(regionVal),
						"vpc":               llx.NilData, // set vpc to nil as default, if vpc is not set
					}

					if lb.VpcId != nil {
						mqlVpc, err := NewResource(a.MqlRuntime, "aws.vpc",
							map[string]*llx.RawData{
								"arn": llx.StringData(fmt.Sprintf(vpcArnPattern, regionVal, conn.AccountId(), convert.ToString(lb.VpcId))),
							})
						if err != nil {
							return nil, err
						}
						// update the vpc setting
						args["vpc"] = llx.ResourceData(mqlVpc, mqlVpc.MqlName())
					}

					mqlLb, err := CreateResource(a.MqlRuntime, "aws.elb.loadbalancer", args)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlLb)
				}
				if lbs.NextMarker == nil {
					break
				}
				marker = lbs.NextMarker
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func initAwsElbLoadbalancer(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch elb loadbalancer")
	}

	obj, err := CreateResource(runtime, "aws.elb", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	elb := obj.(*mqlAwsElb)

	rawResources := elb.GetLoadBalancers()
	if rawResources.Error != nil {
		return nil, nil, rawResources.Error
	}

	arnVal := args["arn"].Value.(string)
	for i := range rawResources.Data {
		lb := rawResources.Data[i].(*mqlAwsElbLoadbalancer)
		if lb.Arn.Data == arnVal {
			return args, lb, nil
		}
	}
	return nil, nil, errors.New("elb load balancer does not exist")
}

func (a *mqlAwsElbLoadbalancer) listenerDescriptions() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	arn := a.Arn.Data

	region, err := GetRegionFromArn(arn)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()

	if isV1LoadBalancerArn(arn) {
		return a.ListenerDescriptions.Data, nil
	}
	svc := conn.Elbv2(region)
	listeners, err := svc.DescribeListeners(ctx, &elasticloadbalancingv2.DescribeListenersInput{LoadBalancerArn: &arn})
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(listeners.Listeners)
}

func (a *mqlAwsElbLoadbalancer) attributes() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	arn := a.Arn.Data
	name := a.Name.Data

	region, err := GetRegionFromArn(arn)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()

	if isV1LoadBalancerArn(arn) {
		svc := conn.Elb(region)
		attributes, err := svc.DescribeLoadBalancerAttributes(ctx, &elasticloadbalancing.DescribeLoadBalancerAttributesInput{LoadBalancerName: &name})
		if err != nil {
			return nil, err
		}
		j, err := convert.JsonToDict(attributes.LoadBalancerAttributes)
		if err != nil {
			return nil, err
		}
		return []interface{}{j}, nil
	}
	svc := conn.Elbv2(region)
	attributes, err := svc.DescribeLoadBalancerAttributes(ctx, &elasticloadbalancingv2.DescribeLoadBalancerAttributesInput{LoadBalancerArn: &arn})
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(attributes.Attributes)
}

func isV1LoadBalancerArn(a string) bool {
	arnVal, err := arn.Parse(a)
	if err != nil {
		return false
	}
	if strings.Contains(arnVal.Resource, "classic") {
		return true
	}
	return false
}

func (a *mqlAwsElbTargetgroup) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsElbLoadbalancer) targetGroups() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	regionVal := a.Region.Data
	svc := conn.Elbv2(regionVal)
	ctx := context.Background()
	res := []interface{}{}

	var marker *string
	for {
		tgs, err := svc.DescribeTargetGroups(ctx, &elasticloadbalancingv2.DescribeTargetGroupsInput{LoadBalancerArn: aws.String(a.Arn.Data), Marker: marker})
		if err != nil {
			if Is400AccessDeniedError(err) {
				log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
				return res, nil
			}
			return nil, err
		}
		for _, tg := range tgs.TargetGroups {
			args := map[string]*llx.RawData{
				"arn":                        llx.StringDataPtr(tg.TargetGroupArn),
				"name":                       llx.StringDataPtr(tg.TargetGroupName),
				"port":                       llx.IntDataPtr(tg.Port),
				"protocol":                   llx.StringData(string(tg.Protocol)),
				"protocolVersion":            llx.StringDataPtr(tg.ProtocolVersion),
				"ipAddressType":              llx.StringData(string(tg.IpAddressType)),
				"healthCheckEnabled":         llx.BoolDataPtr(tg.HealthCheckEnabled),
				"healthCheckIntervalSeconds": llx.IntDataPtr(tg.HealthCheckIntervalSeconds),
				"healthCheckPath":            llx.StringDataPtr(tg.HealthCheckPath),
				"healthCheckPort":            llx.StringDataPtr(tg.HealthCheckPort),
				"healthCheckProtocol":        llx.StringData(string(tg.HealthCheckProtocol)),
				"healthCheckTimeoutSeconds":  llx.IntDataPtr(tg.HealthCheckTimeoutSeconds),
				"targetType":                 llx.StringData(string(tg.TargetType)),
				"unhealthyThresholdCount":    llx.IntDataPtr(tg.UnhealthyThresholdCount),
			}

			mqlLb, err := CreateResource(a.MqlRuntime, "aws.elb.targetgroup", args)
			if err != nil {
				return nil, err
			}
			mqlLb.(*mqlAwsElbTargetgroup).targetGroup = tg
			mqlLb.(*mqlAwsElbTargetgroup).region = regionVal
			res = append(res, mqlLb)
		}
		if tgs.NextMarker == nil {
			break
		}
		marker = tgs.NextMarker
	}
	return res, nil
}

type mqlAwsElbTargetgroupInternal struct {
	targetGroup elbtypes.TargetGroup
	region      string
}

func (a *mqlAwsElbTargetgroup) vpc() (*mqlAwsVpc, error) {
	if a.targetGroup.VpcId == nil {
		a.Vpc.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	mqlVpc, err := NewResource(a.MqlRuntime, "aws.vpc",
		map[string]*llx.RawData{
			"arn": llx.StringData(fmt.Sprintf(vpcArnPattern, a.region, conn.AccountId(), *a.targetGroup.VpcId)),
		})
	if err != nil {
		return nil, err
	}
	return mqlVpc.(*mqlAwsVpc), nil
}

func (a *mqlAwsElbTargetgroup) ec2Targets() ([]interface{}, error) {
	// TODO
	return nil, nil
}

func (a *mqlAwsElbTargetgroup) lambdaTargets() ([]interface{}, error) {
	// TODO
	return nil, nil
}
