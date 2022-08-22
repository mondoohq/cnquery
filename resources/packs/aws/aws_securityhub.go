package aws

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/securityhub"
	"github.com/aws/aws-sdk-go-v2/service/securityhub/types"
	"go.mondoo.io/mondoo/resources/library/jobpool"
	aws_transport "go.mondoo.io/mondoo/motor/providers/aws"
	"go.mondoo.io/mondoo/resources/packs/core"
)

func (s *mqlAwsSecurityhub) id() (string, error) {
	return "aws.securityhub", nil
}

func (s *mqlAwsSecurityhub) GetHubs() ([]interface{}, error) {
	at, err := awstransport(s.MotorRuntime.Motor.Provider)
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

func (s *mqlAwsSecurityhub) getHubs(at *aws_transport.Provider) []*jobpool.Job {
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
			mqlHub, err := s.MotorRuntime.CreateResource("aws.securityhub.hub",
				"arn", core.ToString(secHub.HubArn),
				"subscribedAt", core.ToString(secHub.SubscribedAt),
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlHub)
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (s *mqlAwsSecurityhubHub) id() (string, error) {
	return s.Arn()
}
