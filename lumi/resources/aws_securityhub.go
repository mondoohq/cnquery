package resources

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/securityhub"
	"github.com/aws/aws-sdk-go-v2/service/securityhub/types"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
	aws_transport "go.mondoo.io/mondoo/motor/providers/aws"
)

func (s *lumiAwsSecurityhub) id() (string, error) {
	return "aws.securityhub", nil
}

func (s *lumiAwsSecurityhub) GetHubs() ([]interface{}, error) {
	at, err := awstransport(s.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(s.getHubs(at), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]interface{})...)
		}
	}
	return res, nil
}

func (s *lumiAwsSecurityhub) getHubs(at *aws_transport.Transport) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := at.Securityhub(regionVal)
			ctx := context.Background()
			res := []interface{}{}
			secHub, err := svc.DescribeHub(ctx, &securityhub.DescribeHubInput{})
			if err != nil {
				var notFoundErr *types.InvalidAccessException
				if errors.As(err, &notFoundErr) {
					return nil, nil
				}
				return nil, err
			}
			lumiHub, err := s.MotorRuntime.CreateResource("aws.securityhub.hub",
				"arn", toString(secHub.HubArn),
				"subscribedAt", toString(secHub.SubscribedAt),
			)
			if err != nil {
				return nil, err
			}
			res = append(res, lumiHub)
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (s *lumiAwsSecurityhubHub) id() (string, error) {
	return s.Arn()
}
