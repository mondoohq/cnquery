package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
)

const (
	elbv1LbArnPattern = "arn:aws:elasticloadbalancing:%s:%s:loadbalancer/classic/%s"
)

func (e *lumiAwsElb) id() (string, error) {
	return "aws.elb", nil
}

func (e *lumiAwsElb) GetClassicLoadBalancers() ([]interface{}, error) {
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(e.getClassicLoadBalancers(), 5)
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

func (e *lumiAwsElb) getClassicLoadBalancers() []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
	at, err := awstransport(e.Runtime.Motor.Transport)
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
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
				lbs, err := svc.DescribeLoadBalancersRequest(&elasticloadbalancing.DescribeLoadBalancersInput{Marker: marker}).Send(ctx)
				if err != nil {
					return nil, err
				}
				for _, lb := range lbs.LoadBalancerDescriptions {
					jsonListeners, err := jsonToDictSlice(lb.ListenerDescriptions)
					if err != nil {
						return nil, err
					}
					lumiLb, err := e.Runtime.CreateResource("aws.elb.loadbalancer",
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
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(e.getLoadBalancers(), 5)
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

func (e *lumiAwsElb) getLoadBalancers() []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
	at, err := awstransport(e.Runtime.Motor.Transport)
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
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
				lbs, err := svc.DescribeLoadBalancersRequest(&elasticloadbalancingv2.DescribeLoadBalancersInput{Marker: marker}).Send(ctx)
				if err != nil {
					return nil, err
				}
				for _, lb := range lbs.LoadBalancers {
					listeners, err := svc.DescribeListenersRequest(&elasticloadbalancingv2.DescribeListenersInput{LoadBalancerArn: lb.LoadBalancerArn}).Send(ctx)
					jsonListeners, err := jsonToDictSlice(listeners.Listeners)
					if err != nil {
						return nil, err
					}
					stringScheme, err := lb.Scheme.MarshalValue()
					if err != nil {
						return nil, err
					}
					lumiLb, err := e.Runtime.CreateResource("aws.elb.loadbalancer",
						"arn", toString(lb.LoadBalancerArn),
						"listenerDescriptions", jsonListeners,
						"dnsName", toString(lb.DNSName),
						"name", toString(lb.LoadBalancerName),
						"scheme", stringScheme,
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
