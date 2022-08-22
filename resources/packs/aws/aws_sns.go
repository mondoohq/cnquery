package aws

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/smithy-go/transport/http"
	aws_provider "go.mondoo.io/mondoo/motor/providers/aws"
	"go.mondoo.io/mondoo/resources"
	"go.mondoo.io/mondoo/resources/library/jobpool"
	"go.mondoo.io/mondoo/resources/packs/core"
)

func (s *mqlAwsSns) id() (string, error) {
	return "aws.sns", nil
}

func (s *mqlAwsSnsTopic) id() (string, error) {
	return s.Arn()
}

func (s *mqlAwsSnsSubscription) id() (string, error) {
	return s.Arn()
}

func (s *mqlAwsSns) GetTopics() ([]interface{}, error) {
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(s.getTopics(provider), 5)
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

func (s *mqlAwsSnsTopic) init(args *resources.Args) (*resources.Args, AwsSnsTopic, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if (*args)["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch sns topic")
	}
	arnVal := (*args)["arn"].(string)
	arn, err := arn.Parse(arnVal)
	if err != nil {
		return nil, nil, nil
	}
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}
	svc := provider.Sns(arn.Region)
	ctx := context.Background()

	tags, err := getSNSTags(ctx, svc, &arnVal)
	if err != nil {
		return nil, nil, err
	}
	(*args)["arn"] = arnVal
	(*args)["region"] = arn.Region
	(*args)["tags"] = tags
	return args, nil, nil
}

func (s *mqlAwsSns) getTopics(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := provider.Sns(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			nextToken := aws.String("no_token_to_start_with")
			params := &sns.ListTopicsInput{}
			for nextToken != nil {
				topics, err := svc.ListTopics(ctx, params)
				if err != nil {
					return nil, err
				}
				for _, topic := range topics.Topics {
					tags, err := getSNSTags(ctx, svc, topic.TopicArn)
					if err != nil {
						return nil, err
					}
					mqlTopic, err := s.MotorRuntime.CreateResource("aws.sns.topic",
						"arn", core.ToString(topic.TopicArn),
						"region", regionVal,
						"tags", tags,
					)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlTopic)
				}
				nextToken = topics.NextToken
				if topics.NextToken != nil {
					params.NextToken = nextToken
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (s *mqlAwsSnsTopic) GetAttributes() (interface{}, error) {
	arn, err := s.Arn()
	if err != nil {
		return false, err
	}
	region, err := s.Region()
	if err != nil {
		return nil, err
	}
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.Sns(region)
	ctx := context.Background()

	topicAttributes, err := svc.GetTopicAttributes(ctx, &sns.GetTopicAttributesInput{TopicArn: &arn})
	if err != nil {
		return nil, err
	}
	return core.JsonToDict(topicAttributes.Attributes)
}

func getSNSTags(ctx context.Context, svc *sns.Client, arn *string) (map[string]interface{}, error) {
	resp, err := svc.ListTagsForResource(ctx, &sns.ListTagsForResourceInput{ResourceArn: arn})
	var respErr *http.ResponseError
	if err != nil {
		if errors.As(err, &respErr) {
			if respErr.HTTPStatusCode() == 404 {
				return nil, nil
			}
		}
		return nil, err
	}
	tags := make(map[string]interface{})
	for _, t := range resp.Tags {
		tags[*t.Key] = *t.Value
	}
	return tags, nil
}
