package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
	aws_transport "go.mondoo.io/mondoo/motor/transports/aws"
)

func (a *lumiAwsAutoscaling) id() (string, error) {
	return "aws.autoscaling", nil
}

func (a *lumiAwsAutoscalingGroup) id() (string, error) {
	return a.Arn()
}

func (a *lumiAwsAutoscaling) GetGroups() ([]interface{}, error) {
	at, err := awstransport(a.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getGroups(at), 5)
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

func (a *lumiAwsAutoscaling) getGroups(at *aws_transport.Transport) []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {

			svc := at.Autoscaling(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			nextToken := aws.String("no_token_to_start_with")
			params := &autoscaling.DescribeAutoScalingGroupsInput{}
			for nextToken != nil {
				groups, err := svc.DescribeAutoScalingGroups(ctx, params)
				if err != nil {
					return nil, err
				}
				for _, group := range groups.AutoScalingGroups {
					lbNames := []interface{}{}
					for _, name := range group.LoadBalancerNames {
						lbNames = append(lbNames, name)
					}
					lumiGroup, err := a.Runtime.CreateResource("aws.autoscaling.group",
						"arn", toString(group.AutoScalingGroupARN),
						"name", toString(group.AutoScalingGroupName),
						"loadBalancerNames", lbNames,
						"healthCheckType", toString(group.HealthCheckType),
						"tags", autoscalingTagsToMap(group.Tags),
					)
					if err != nil {
						return nil, err
					}
					res = append(res, lumiGroup)
				}
				nextToken = groups.NextToken
				if groups.NextToken != nil {
					params.NextToken = nextToken
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func autoscalingTagsToMap(tags []types.TagDescription) map[string]interface{} {
	tagsMap := make(map[string]interface{})

	if len(tags) > 0 {
		for i := range tags {
			tag := tags[i]
			tagsMap[toString(tag.Key)] = toString(tag.Value)
		}
	}

	return tagsMap
}
