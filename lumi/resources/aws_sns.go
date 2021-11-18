package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
	aws_transport "go.mondoo.io/mondoo/motor/transports/aws"
)

func (s *lumiAwsSns) id() (string, error) {
	return "aws.sns", nil
}

func (s *lumiAwsSnsTopic) id() (string, error) {
	return s.Arn()
}

func (s *lumiAwsSnsSubscription) id() (string, error) {
	return s.Arn()
}

func (s *lumiAwsSns) GetTopics() ([]interface{}, error) {
	at, err := awstransport(s.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(s.getTopics(at), 5)
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
func (s *lumiAwsSns) getTopics(at *aws_transport.Transport) []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {

			svc := at.Sns(regionVal)
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
					lumiTopic, err := s.Runtime.CreateResource("aws.sns.topic",
						"arn", toString(topic.TopicArn),
						"region", regionVal,
					)
					if err != nil {
						return nil, err
					}
					res = append(res, lumiTopic)
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

func (s *lumiAwsSnsTopic) GetAttributes() (interface{}, error) {
	arn, err := s.Arn()
	if err != nil {
		return false, err
	}
	region, err := s.Region()
	if err != nil {
		return nil, err
	}
	at, err := awstransport(s.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	svc := at.Sns(region)
	ctx := context.Background()

	topicAttributes, err := svc.GetTopicAttributes(ctx, &sns.GetTopicAttributesInput{TopicArn: &arn})
	if err != nil {
		return nil, err
	}
	return jsonToDict(topicAttributes.Attributes)
}
