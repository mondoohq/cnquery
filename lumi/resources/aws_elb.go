package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
	aws_transport "go.mondoo.io/mondoo/motor/providers/aws"
)

const (
	elbv1LbArnPattern = "arn:aws:elasticloadbalancing:%s:%s:loadbalancer/classic/%s"
)

func (e *lumiAwsElb) id() (string, error) {
	return "aws.elb", nil
}

func (e *lumiAwsElb) GetClassicLoadBalancers() ([]interface{}, error) {
	at, err := awstransport(e.MotorRuntime.Motor.Transport)
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

func (e *lumiAwsElb) getClassicLoadBalancers(at *aws_transport.Transport) []*jobpool.Job {
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
					jsonListeners, err := jsonToDictSlice(lb.ListenerDescriptions)
					if err != nil {
						return nil, err
					}
					lumiLb, err := e.MotorRuntime.CreateResource("aws.elb.loadbalancer",
						"arn", fmt.Sprintf(elbv1LbArnPattern, regionVal, account.ID, toString(lb.LoadBalancerName)),
						"listenerDescriptions", jsonListeners,
						"dnsName", toString(lb.DNSName),
						"name", toString(lb.LoadBalancerName),
						"scheme", toString(lb.Scheme),
					)
					if err != nil {
						return nil, err
					}
					res = append(res, lumiLb)
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

func (e *lumiAwsElbLoadbalancer) id() (string, error) {
	return e.Arn()
}

func (e *lumiAwsElb) GetLoadBalancers() ([]interface{}, error) {
	at, err := awstransport(e.MotorRuntime.Motor.Transport)
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

func (e *lumiAwsElb) getLoadBalancers(at *aws_transport.Transport) []*jobpool.Job {
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
					lumiLb, err := e.MotorRuntime.CreateResource("aws.elb.loadbalancer",
						"arn", toString(lb.LoadBalancerArn),
						"dnsName", toString(lb.DNSName),
						"name", toString(lb.LoadBalancerName),
						"scheme", string(lb.Scheme),
					)
					if err != nil {
						return nil, err
					}
					res = append(res, lumiLb)
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

func (e *lumiAwsElbLoadbalancer) GetListenerDescriptions() ([]interface{}, error) {
	at, err := awstransport(e.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	arn, err := e.Arn()
	if err != nil {
		return nil, err
	}
	region, err := getRegionFromArn(arn)
	if err != nil {
		return nil, err
	}
	svc := at.Elbv2(region)
	ctx := context.Background()
	listeners, err := svc.DescribeListeners(ctx, &elasticloadbalancingv2.DescribeListenersInput{LoadBalancerArn: &arn})
	if err != nil {
		return nil, err
	}
	return jsonToDictSlice(listeners.Listeners)
}

func (e *lumiAwsElbLoadbalancer) GetAttributes() ([]interface{}, error) {
	at, err := awstransport(e.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	arn, err := e.Arn()
	if err != nil {
		return nil, err
	}
	region, err := getRegionFromArn(arn)
	if err != nil {
		return nil, err
	}
	svc := at.Elbv2(region)
	ctx := context.Background()
	attributes, err := svc.DescribeLoadBalancerAttributes(ctx, &elasticloadbalancingv2.DescribeLoadBalancerAttributesInput{LoadBalancerArn: &arn})
	if err != nil {
		return nil, err
	}
	return jsonToDictSlice(attributes.Attributes)
}
