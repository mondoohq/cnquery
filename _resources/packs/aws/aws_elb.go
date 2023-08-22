// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package aws

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/rs/zerolog/log"
	aws_provider "go.mondoo.com/cnquery/motor/providers/aws"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/library/jobpool"
	"go.mondoo.com/cnquery/resources/packs/core"
)

const (
	elbv1LbArnPattern = "arn:aws:elasticloadbalancing:%s:%s:loadbalancer/classic/%s"
)

func (e *mqlAwsElb) id() (string, error) {
	return "aws.elb", nil
}

func (e *mqlAwsElb) GetClassicLoadBalancers() ([]interface{}, error) {
	provider, err := awsProvider(e.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(e.getClassicLoadBalancers(provider), 5)
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

func (e *mqlAwsElb) getClassicLoadBalancers(provider *aws_provider.Provider) []*jobpool.Job {
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
			svc := provider.Elb(regionVal)
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
					jsonListeners, err := core.JsonToDictSlice(lb.ListenerDescriptions)
					if err != nil {
						return nil, err
					}
					mqlLb, err := e.MotorRuntime.CreateResource("aws.elb.loadbalancer",
						"arn", fmt.Sprintf(elbv1LbArnPattern, regionVal, account.ID, core.ToString(lb.LoadBalancerName)),
						"listenerDescriptions", jsonListeners,
						"dnsName", core.ToString(lb.DNSName),
						"name", core.ToString(lb.LoadBalancerName),
						"scheme", core.ToString(lb.Scheme),
					)
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

func (e *mqlAwsElbLoadbalancer) id() (string, error) {
	return e.Arn()
}

func (e *mqlAwsElb) GetLoadBalancers() ([]interface{}, error) {
	provider, err := awsProvider(e.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(e.getLoadBalancers(provider), 5)
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

func (e *mqlAwsElb) getLoadBalancers(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := provider.Elbv2(regionVal)
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
					mqlLb, err := e.MotorRuntime.CreateResource("aws.elb.loadbalancer",
						"arn", core.ToString(lb.LoadBalancerArn),
						"dnsName", core.ToString(lb.DNSName),
						"name", core.ToString(lb.LoadBalancerName),
						"scheme", string(lb.Scheme),
					)
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

func (d *mqlAwsElbLoadbalancer) init(args *resources.Args) (*resources.Args, AwsElbLoadbalancer, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if len(*args) == 0 {
		if ids := getAssetIdentifier(d.MqlResource().MotorRuntime); ids != nil {
			(*args)["name"] = ids.name
			(*args)["arn"] = ids.arn
		}
	}

	if (*args)["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch elb loadbalancer")
	}

	obj, err := d.MotorRuntime.CreateResource("aws.elb")
	if err != nil {
		return nil, nil, err
	}
	elb := obj.(AwsElb)

	rawResources, err := elb.LoadBalancers()
	if err != nil {
		return nil, nil, err
	}

	arnVal := (*args)["arn"].(string)
	for i := range rawResources {
		lb := rawResources[i].(AwsElbLoadbalancer)
		mqlLbArn, err := lb.Arn()
		if err != nil {
			return nil, nil, errors.New("elb loadbalancer does not exist")
		}
		if mqlLbArn == arnVal {
			return args, lb, nil
		}
	}
	return nil, nil, errors.New("elb load balancer does not exist")
}

func (e *mqlAwsElbLoadbalancer) GetListenerDescriptions() ([]interface{}, error) {
	provider, err := awsProvider(e.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	arn, err := e.Arn()
	if err != nil {
		return nil, err
	}
	region, err := GetRegionFromArn(arn)
	if err != nil {
		return nil, err
	}
	svc := provider.Elbv2(region)
	ctx := context.Background()
	listeners, err := svc.DescribeListeners(ctx, &elasticloadbalancingv2.DescribeListenersInput{LoadBalancerArn: &arn})
	if err != nil {
		return nil, err
	}
	return core.JsonToDictSlice(listeners.Listeners)
}

func (e *mqlAwsElbLoadbalancer) GetAttributes() ([]interface{}, error) {
	provider, err := awsProvider(e.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	arn, err := e.Arn()
	if err != nil {
		return nil, err
	}
	region, err := GetRegionFromArn(arn)
	if err != nil {
		return nil, err
	}
	svc := provider.Elbv2(region)
	ctx := context.Background()
	attributes, err := svc.DescribeLoadBalancerAttributes(ctx, &elasticloadbalancingv2.DescribeLoadBalancerAttributesInput{LoadBalancerArn: &arn})
	if err != nil {
		return nil, err
	}
	return core.JsonToDictSlice(attributes.Attributes)
}
