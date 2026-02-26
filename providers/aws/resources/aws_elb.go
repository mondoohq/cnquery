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
	elbv1types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbtypes "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAwsElb) id() (string, error) {
	return ResourceAwsElb, nil
}

func (a *mqlAwsElb) classicLoadBalancers() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getClassicLoadBalancers(conn), 5)
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

func (a *mqlAwsElb) getClassicLoadBalancers(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Elb(region)
			ctx := context.Background()
			res := []any{}

			params := &elasticloadbalancing.DescribeLoadBalancersInput{}
			paginator := elasticloadbalancing.NewDescribeLoadBalancersPaginator(svc, params)
			for paginator.HasMorePages() {
				lbs, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, lb := range lbs.LoadBalancerDescriptions {
					jsonListeners, err := convert.JsonToDictSlice(lb.ListenerDescriptions)
					if err != nil {
						return nil, err
					}
					lbName := convert.ToValue(lb.LoadBalancerName)

					availabilityZones := []any{}
					for _, zone := range lb.AvailabilityZones {
						availabilityZones = append(availabilityZones, zone)
					}

					sgs := []any{}
					for _, sg := range lb.SecurityGroups {
						mqlSg, err := NewResource(a.MqlRuntime, ResourceAwsEc2Securitygroup,
							map[string]*llx.RawData{
								"arn": llx.StringData(fmt.Sprintf(securityGroupArnPattern, region, conn.AccountId(), sg)),
							})
						if err != nil {
							// When tag filters are active, the security group may not be in the
							// filtered list. Log and continue rather than failing the entire LB.
							log.Debug().Str("sg", sg).Err(err).Msg("could not resolve security group for classic ELB")
							continue
						}
						sgs = append(sgs, mqlSg)
					}

					args := map[string]*llx.RawData{
						"arn":                  llx.StringData(fmt.Sprintf(elbv1LbArnPattern, region, conn.AccountId(), lbName)),
						"availabilityZones":    llx.ArrayData(availabilityZones, types.String),
						"createdTime":          llx.TimeDataPtr(lb.CreatedTime),
						"createdAt":            llx.TimeDataPtr(lb.CreatedTime),
						"dnsName":              llx.StringDataPtr(lb.DNSName),
						"elbType":              llx.StringData("classic"),
						"hostedZoneId":         llx.StringDataPtr(lb.CanonicalHostedZoneNameID),
						"listenerDescriptions": llx.AnyData(jsonListeners),
						"name":                 llx.StringDataPtr(lb.LoadBalancerName),
						"region":               llx.StringData(region),
						"scheme":               llx.StringDataPtr(lb.Scheme),
						"securityGroups":       llx.ArrayData(sgs, types.Resource(ResourceAwsEc2Securitygroup)),
						"vpcId":                llx.StringDataPtr(lb.VPCId),
						"vpc":                  llx.NilData,
					}

					if lb.VPCId != nil {
						mqlVpc, err := NewResource(a.MqlRuntime, ResourceAwsVpc,
							map[string]*llx.RawData{
								"arn": llx.StringData(fmt.Sprintf(vpcArnPattern, region, conn.AccountId(), convert.ToValue(lb.VPCId))),
							})
						if err != nil {
							// When tag filters are active, the VPC may not be in the filtered list.
							log.Debug().Str("vpcId", convert.ToValue(lb.VPCId)).Err(err).Msg("could not resolve VPC for classic ELB")
						} else {
							args["vpc"] = llx.ResourceData(mqlVpc, mqlVpc.MqlName())
						}
					}

					mqlLb, err := CreateResource(a.MqlRuntime, ResourceAwsElbLoadbalancer, args)
					if err != nil {
						return nil, err
					}
					// keeps the tags lazy unless the filters need to be evaluated
					if conn.Filters.General.HasTags() {
						tags, err := mqlLb.(*mqlAwsElbLoadbalancer).tags()
						if err != nil {
							return nil, err
						}
						if conn.Filters.General.IsFilteredOutByTags(mapStringInterfaceToStringString(tags)) {
							continue
						}
					}

					res = append(res, mqlLb)
				}
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

func (a *mqlAwsElb) loadBalancers() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getLoadBalancers(conn), 5)
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

func (a *mqlAwsElb) getLoadBalancers(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Elbv2(region)
			ctx := context.Background()
			res := []any{}

			params := &elasticloadbalancingv2.DescribeLoadBalancersInput{}
			paginator := elasticloadbalancingv2.NewDescribeLoadBalancersPaginator(svc, params)
			for paginator.HasMorePages() {
				lbs, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, lb := range lbs.LoadBalancers {
					availabilityZones := []any{}
					for _, zone := range lb.AvailabilityZones {
						availabilityZones = append(availabilityZones, convert.ToValue(zone.ZoneName))
					}

					sgs := []any{}
					for _, sg := range lb.SecurityGroups {
						mqlSg, err := NewResource(a.MqlRuntime, ResourceAwsEc2Securitygroup,
							map[string]*llx.RawData{
								"arn": llx.StringData(fmt.Sprintf(securityGroupArnPattern, region, conn.AccountId(), sg)),
							})
						if err != nil {
							// When tag filters are active, the security group may not be in the
							// filtered list. Log and continue rather than failing the entire LB.
							log.Debug().Str("sg", sg).Err(err).Msg("could not resolve security group for ELB")
							continue
						}
						sgs = append(sgs, mqlSg)
					}

					args := map[string]*llx.RawData{
						"arn":               llx.StringDataPtr(lb.LoadBalancerArn),
						"availabilityZones": llx.ArrayData(availabilityZones, types.String),
						"createdTime":       llx.TimeDataPtr(lb.CreatedTime),
						"createdAt":         llx.TimeDataPtr(lb.CreatedTime),
						"dnsName":           llx.StringDataPtr(lb.DNSName),
						"hostedZoneId":      llx.StringDataPtr(lb.CanonicalHostedZoneId),
						"name":              llx.StringDataPtr(lb.LoadBalancerName),
						"scheme":            llx.StringData(string(lb.Scheme)),
						"securityGroups":    llx.ArrayData(sgs, types.Resource(ResourceAwsEc2Securitygroup)),
						"vpcId":             llx.StringDataPtr(lb.VpcId),
						"elbType":           llx.StringData(string(lb.Type)),
						"ipAddressType":     llx.StringData(string(lb.IpAddressType)),
						"region":            llx.StringData(region),
						"vpc":               llx.NilData, // set vpc to nil as default, if vpc is not set
					}

					if lb.VpcId != nil {
						mqlVpc, err := NewResource(a.MqlRuntime, ResourceAwsVpc,
							map[string]*llx.RawData{
								"arn": llx.StringData(fmt.Sprintf(vpcArnPattern, region, conn.AccountId(), convert.ToValue(lb.VpcId))),
							})
						if err != nil {
							// When tag filters are active, the VPC may not be in the filtered list.
							log.Debug().Str("vpcId", convert.ToValue(lb.VpcId)).Err(err).Msg("could not resolve VPC for ELB")
						} else {
							// update the vpc setting
							args["vpc"] = llx.ResourceData(mqlVpc, mqlVpc.MqlName())
						}
					}

					mqlLb, err := CreateResource(a.MqlRuntime, ResourceAwsElbLoadbalancer, args)
					if err != nil {
						return nil, err
					}
					// keeps the tags lazy unless the filters need to be evaluated
					if conn.Filters.General.HasTags() {
						tags, err := mqlLb.(*mqlAwsElbLoadbalancer).tags()
						if err != nil {
							return nil, err
						}
						if conn.Filters.General.IsFilteredOutByTags(mapStringInterfaceToStringString(tags)) {
							continue
						}
					}

					res = append(res, mqlLb)
				}
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

	arnVal := args["arn"].Value.(string)

	// Quick check: if the ARN doesn't belong to elasticloadbalancing, this asset
	// is not an ELB. This happens when the query runs against non-ELB discovered
	// assets (e.g., DynamoDB tables, IAM users, S3 buckets).
	if arnVal == "" || !strings.Contains(arnVal, ":elasticloadbalancing:") {
		return nil, nil, errors.New("elb load balancer does not exist")
	}

	obj, err := CreateResource(runtime, ResourceAwsElb, map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	elb := obj.(*mqlAwsElb)

	rawResources := elb.GetLoadBalancers()
	if rawResources.Error != nil {
		return nil, nil, rawResources.Error
	}
	for _, rawResource := range rawResources.Data {
		lb := rawResource.(*mqlAwsElbLoadbalancer)
		if lb.Arn.Data == arnVal {
			return args, lb, nil
		}
	}

	classicResources := elb.GetClassicLoadBalancers()
	if classicResources.Error != nil {
		return nil, nil, classicResources.Error
	}
	for _, rawResource := range classicResources.Data {
		lb := rawResource.(*mqlAwsElbLoadbalancer)
		if lb.Arn.Data == arnVal {
			return args, lb, nil
		}
	}

	return nil, nil, errors.New("elb load balancer does not exist")
}

func (a *mqlAwsElbLoadbalancer) listenerDescriptions() ([]any, error) {
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

func (a *mqlAwsElbLoadbalancer) attributes() ([]any, error) {
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
		return []any{j}, nil
	}
	svc := conn.Elbv2(region)
	attributes, err := svc.DescribeLoadBalancerAttributes(ctx, &elasticloadbalancingv2.DescribeLoadBalancerAttributesInput{LoadBalancerArn: &arn})
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(attributes.Attributes)
}

func (a *mqlAwsElbListener) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsElbLoadbalancer) listeners() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	arnVal := a.Arn.Data

	if isV1LoadBalancerArn(arnVal) {
		return []any{}, nil
	}

	region, err := GetRegionFromArn(arnVal)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	svc := conn.Elbv2(region)

	res := []any{}
	params := &elasticloadbalancingv2.DescribeListenersInput{LoadBalancerArn: &arnVal}
	paginator := elasticloadbalancingv2.NewDescribeListenersPaginator(svc, params)
	for paginator.HasMorePages() {
		listenersResp, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, l := range listenersResp.Listeners {
			defaultActions, err := convert.JsonToDictSlice(l.DefaultActions)
			if err != nil {
				return nil, err
			}
			certificates, err := convert.JsonToDictSlice(l.Certificates)
			if err != nil {
				return nil, err
			}

			args := map[string]*llx.RawData{
				"__id":            llx.StringDataPtr(l.ListenerArn),
				"arn":             llx.StringDataPtr(l.ListenerArn),
				"loadBalancerArn": llx.StringDataPtr(l.LoadBalancerArn),
				"port":            llx.IntDataPtr(l.Port),
				"protocol":        llx.StringData(string(l.Protocol)),
				"sslPolicy":       llx.StringDataPtr(l.SslPolicy),
				"defaultActions":  llx.ArrayData(defaultActions, types.Dict),
				"certificates":    llx.ArrayData(certificates, types.Dict),
				"alpnPolicy":      llx.ArrayData(llx.TArr2Raw(l.AlpnPolicy), types.String),
			}

			mqlListener, err := CreateResource(a.MqlRuntime, "aws.elb.listener", args)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlListener)
		}
	}
	return res, nil
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

func elbv2TagsToMap(tags []elbtypes.Tag) map[string]any {
	tagsMap := make(map[string]any)
	for _, tag := range tags {
		tagsMap[convert.ToValue(tag.Key)] = convert.ToValue(tag.Value)
	}
	return tagsMap
}

func elbv1TagsToMap(tags []elbv1types.Tag) map[string]any {
	tagsMap := make(map[string]any)
	for _, tag := range tags {
		tagsMap[convert.ToValue(tag.Key)] = convert.ToValue(tag.Value)
	}
	return tagsMap
}

func (a *mqlAwsElbLoadbalancer) tags() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	lbArn := a.Arn.Data
	name := a.Name.Data

	region, err := GetRegionFromArn(lbArn)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()

	if isV1LoadBalancerArn(lbArn) {
		svc := conn.Elb(region)
		resp, err := svc.DescribeTags(ctx, &elasticloadbalancing.DescribeTagsInput{LoadBalancerNames: []string{name}})
		if err != nil {
			return nil, err
		}
		for _, desc := range resp.TagDescriptions {
			if convert.ToValue(desc.LoadBalancerName) == name {
				return elbv1TagsToMap(desc.Tags), nil
			}
		}
		return map[string]any{}, nil
	}

	svc := conn.Elbv2(region)
	resp, err := svc.DescribeTags(ctx, &elasticloadbalancingv2.DescribeTagsInput{ResourceArns: []string{lbArn}})
	if err != nil {
		return nil, err
	}
	for _, desc := range resp.TagDescriptions {
		if convert.ToValue(desc.ResourceArn) == lbArn {
			return elbv2TagsToMap(desc.Tags), nil
		}
	}
	return map[string]any{}, nil
}

func (a *mqlAwsElbTargetgroup) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsElbLoadbalancer) targetGroups() ([]any, error) {
	// Classic load balancers don't have target groups
	if isV1LoadBalancerArn(a.Arn.Data) {
		return []any{}, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	regionVal := a.Region.Data
	svc := conn.Elbv2(regionVal)
	ctx := context.Background()
	res := []any{}

	params := &elasticloadbalancingv2.DescribeTargetGroupsInput{LoadBalancerArn: aws.String(a.Arn.Data)}
	paginator := elasticloadbalancingv2.NewDescribeTargetGroupsPaginator(svc, params)
	for paginator.HasMorePages() {
		tgs, err := paginator.NextPage(ctx)
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
				"healthyThresholdCount":      llx.IntDataPtr(tg.HealthyThresholdCount),
			}

			mqlLb, err := CreateResource(a.MqlRuntime, ResourceAwsElbTargetgroup, args)
			if err != nil {
				return nil, err
			}
			mqlLb.(*mqlAwsElbTargetgroup).targetGroup = tg
			mqlLb.(*mqlAwsElbTargetgroup).region = regionVal
			res = append(res, mqlLb)
		}
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
	mqlVpc, err := NewResource(a.MqlRuntime, ResourceAwsVpc,
		map[string]*llx.RawData{
			"arn": llx.StringData(fmt.Sprintf(vpcArnPattern, a.region, conn.AccountId(), *a.targetGroup.VpcId)),
		})
	if err != nil {
		return nil, err
	}
	return mqlVpc.(*mqlAwsVpc), nil
}

func (a *mqlAwsElbTargetgroup) attributes() (*mqlAwsElbTargetgroupAttributes, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	tgArn := a.Arn.Data

	region := a.region
	svc := conn.Elbv2(region)
	ctx := context.Background()

	resp, err := svc.DescribeTargetGroupAttributes(ctx, &elasticloadbalancingv2.DescribeTargetGroupAttributesInput{
		TargetGroupArn: &tgArn,
	})
	if err != nil {
		return nil, err
	}

	// Build a lookup map from the key/value attributes
	attrMap := make(map[string]string)
	for _, attr := range resp.Attributes {
		if attr.Key != nil && attr.Value != nil {
			attrMap[*attr.Key] = *attr.Value
		}
	}

	args := map[string]*llx.RawData{
		"__id":                                         llx.StringData(tgArn + "/attributes"),
		"targetGroupArn":                               llx.StringData(tgArn),
		"deregistrationDelayTimeoutSeconds":            llx.IntData(attrMapInt(attrMap, "deregistration_delay.timeout_seconds")),
		"stickinessEnabled":                            llx.BoolData(attrMapBool(attrMap, "stickiness.enabled")),
		"stickinessType":                               llx.StringData(attrMap["stickiness.type"]),
		"loadBalancingAlgorithmType":                   llx.StringData(attrMap["load_balancing.algorithm.type"]),
		"loadBalancingAlgorithmAnomalyMitigation":      llx.StringData(attrMap["load_balancing.algorithm.anomaly_mitigation"]),
		"slowStartDurationSeconds":                     llx.IntData(attrMapInt(attrMap, "slow_start.duration_seconds")),
		"crossZoneEnabled":                             llx.StringData(attrMap["load_balancing.cross_zone.enabled"]),
		"proxyProtocolV2Enabled":                       llx.BoolData(attrMapBool(attrMap, "proxy_protocol_v2.enabled")),
		"preserveClientIpEnabled":                      llx.BoolData(attrMapBool(attrMap, "preserve_client_ip.enabled")),
		"connectionTerminationOnDeregistrationEnabled": llx.BoolData(attrMapBool(attrMap, "deregistration_delay.connection_termination.enabled")),
		"connectionTerminationOnUnhealthyEnabled":      llx.BoolData(attrMapBool(attrMap, "target_health_state.unhealthy.connection_termination.enabled")),
		"lambdaMultiValueHeadersEnabled":               llx.BoolData(attrMapBool(attrMap, "lambda.multi_value_headers.enabled")),
		"targetFailoverOnDeregistration":               llx.StringData(attrMap["target_failover.on_deregistration"]),
		"targetFailoverOnUnhealthy":                    llx.StringData(attrMap["target_failover.on_unhealthy"]),
	}

	res, err := CreateResource(a.MqlRuntime, ResourceAwsElbTargetgroupAttributes, args)
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsElbTargetgroupAttributes), nil
}

func (a *mqlAwsElbTargetgroupAttributes) id() (string, error) {
	return a.TargetGroupArn.Data + "/attributes", nil
}

// attrMapBool parses a string value from the attribute map as a boolean.
func attrMapBool(m map[string]string, key string) bool {
	return m[key] == "true"
}

// attrMapInt parses a string value from the attribute map as an integer.
// Returns 0 if the key is missing or the value is not a valid integer.
func attrMapInt(m map[string]string, key string) int64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	var i int64
	if _, err := fmt.Sscanf(v, "%d", &i); err != nil {
		return 0
	}
	return i
}

func (a *mqlAwsElbLoadbalancerAttribute) id() (string, error) {
	return a.LoadBalancerArn.Data + "/attributes", nil
}

func (a *mqlAwsElbLoadbalancer) attribute() (*mqlAwsElbLoadbalancerAttribute, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	arnVal := a.Arn.Data

	if isV1LoadBalancerArn(arnVal) {
		a.Attribute.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	region, err := GetRegionFromArn(arnVal)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	svc := conn.Elbv2(region)

	resp, err := svc.DescribeLoadBalancerAttributes(ctx, &elasticloadbalancingv2.DescribeLoadBalancerAttributesInput{
		LoadBalancerArn: &arnVal,
	})
	if err != nil {
		return nil, err
	}

	attrMap := make(map[string]string)
	for _, attr := range resp.Attributes {
		if attr.Key != nil && attr.Value != nil {
			attrMap[*attr.Key] = *attr.Value
		}
	}

	args := map[string]*llx.RawData{
		"__id":                      llx.StringData(arnVal + "/attributes"),
		"loadBalancerArn":           llx.StringData(arnVal),
		"deletionProtectionEnabled": llx.BoolData(attrMapBool(attrMap, "deletion_protection.enabled")),
		"crossZoneEnabled":          llx.StringData(attrMap["load_balancing.cross_zone.enabled"]),
		"accessLogsEnabled":         llx.BoolData(attrMapBool(attrMap, "access_logs.s3.enabled")),
		"accessLogsBucket":          llx.StringData(attrMap["access_logs.s3.bucket"]),
		"accessLogsPrefix":          llx.StringData(attrMap["access_logs.s3.prefix"]),
		"idleTimeoutSeconds":        llx.IntData(attrMapInt(attrMap, "idle_timeout.timeout_seconds")),
		"desyncMitigationMode":      llx.StringData(attrMap["routing.http.desync_mitigation_mode"]),
		"dropInvalidHeaderFields":   llx.BoolData(attrMapBool(attrMap, "routing.http.drop_invalid_header_fields.enabled")),
		"preserveHostHeader":        llx.BoolData(attrMapBool(attrMap, "routing.http.preserve_host_header.enabled")),
		"http2Enabled":              llx.BoolData(attrMapBool(attrMap, "routing.http2.enabled")),
		"wafFailOpenEnabled":        llx.BoolData(attrMapBool(attrMap, "waf.fail_open.enabled")),
		"zonalShiftEnabled":         llx.BoolData(attrMapBool(attrMap, "zonal_shift.config.enabled")),
		"connectionLogsEnabled":     llx.BoolData(attrMapBool(attrMap, "connection_logs.s3.enabled")),
		"connectionLogsBucket":      llx.StringData(attrMap["connection_logs.s3.bucket"]),
	}

	resource, err := CreateResource(a.MqlRuntime, "aws.elb.loadbalancer.attribute", args)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAwsElbLoadbalancerAttribute), nil
}

func (a *mqlAwsElbLoadbalancer) instances() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	arnVal := a.Arn.Data

	if !isV1LoadBalancerArn(arnVal) {
		// ALB/NLB/GLB don't have direct instances; use target groups instead
		return []any{}, nil
	}

	region, err := GetRegionFromArn(arnVal)
	if err != nil {
		return nil, err
	}

	name := a.Name.Data
	svc := conn.Elb(region)
	ctx := context.Background()

	resp, err := svc.DescribeLoadBalancers(ctx, &elasticloadbalancing.DescribeLoadBalancersInput{
		LoadBalancerNames: []string{name},
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	if len(resp.LoadBalancerDescriptions) == 0 {
		return res, nil
	}

	for _, inst := range resp.LoadBalancerDescriptions[0].Instances {
		if inst.InstanceId == nil {
			continue
		}
		mqlInst, err := NewResource(a.MqlRuntime, ResourceAwsEc2Instance,
			map[string]*llx.RawData{
				"arn": llx.StringData(fmt.Sprintf(ec2InstanceArnPattern, region, conn.AccountId(), *inst.InstanceId)),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInst)
	}
	return res, nil
}

func (a *mqlAwsElbTargetgroup) ec2Targets() ([]any, error) {
	// TODO
	return nil, nil
}

func (a *mqlAwsElbTargetgroup) lambdaTargets() ([]any, error) {
	// TODO
	return nil, nil
}
