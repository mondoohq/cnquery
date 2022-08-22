package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"go.mondoo.io/mondoo/resources/library/jobpool"
	aws_transport "go.mondoo.io/mondoo/motor/providers/aws"
	"go.mondoo.io/mondoo/resources/packs/core"
)

const (
	elbv1LbArnPattern = "arn:aws:elasticloadbalancing:%s:%s:loadbalancer/classic/%s"
)

func (e *mqlAwsElb) id() (string, error) {
	return "aws.elb", nil
}

func (e *mqlAwsElb) GetClassicLoadBalancers() ([]interface{}, error) {
	at, err := awstransport(e.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(e.getClassicLoadBalancers(at), 5)
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

func (e *mqlAwsElb) getClassicLoadBalancers(at *aws_transport.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	account, err := at.Account()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := at.Elb(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			var marker *string
			for {
				lbs, err := svc.DescribeLoadBalancers(ctx, &elasticloadbalancing.DescribeLoadBalancersInput{Marker: marker})
				if err != nil {
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
	at, err := awstransport(e.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(e.getLoadBalancers(at), 5)
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

func (e *mqlAwsElb) getLoadBalancers(at *aws_transport.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := at.Elbv2(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			var marker *string
			for {
				lbs, err := svc.DescribeLoadBalancers(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{Marker: marker})
				if err != nil {
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

func (e *mqlAwsElbLoadbalancer) GetListenerDescriptions() ([]interface{}, error) {
	at, err := awstransport(e.MotorRuntime.Motor.Provider)
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
	svc := at.Elbv2(region)
	ctx := context.Background()
	listeners, err := svc.DescribeListeners(ctx, &elasticloadbalancingv2.DescribeListenersInput{LoadBalancerArn: &arn})
	if err != nil {
		return nil, err
	}
	return core.JsonToDictSlice(listeners.Listeners)
}

func (e *mqlAwsElbLoadbalancer) GetAttributes() ([]interface{}, error) {
	at, err := awstransport(e.MotorRuntime.Motor.Provider)
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
	svc := at.Elbv2(region)
	ctx := context.Background()
	attributes, err := svc.DescribeLoadBalancerAttributes(ctx, &elasticloadbalancingv2.DescribeLoadBalancerAttributesInput{LoadBalancerArn: &arn})
	if err != nil {
		return nil, err
	}
	return core.JsonToDictSlice(attributes.Attributes)
}
