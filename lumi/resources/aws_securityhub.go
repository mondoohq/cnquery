package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/securityhub"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
)

func (s *lumiAwsSecurityhub) id() (string, error) {
	return "aws.securityhub", nil
}

func (s *lumiAwsSecurityhub) GetHubs() ([]interface{}, error) {
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(s.getHubs(), 5)
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

func (s *lumiAwsSecurityhub) getHubs() []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
	at, err := awstransport(s.Runtime.Motor.Transport)
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

			svc := at.Securityhub(regionVal)
			ctx := context.Background()
			res := []interface{}{}
			secHub, err := svc.DescribeHubRequest(&securityhub.DescribeHubInput{}).Send(ctx)
			isAwsErr, code := IsAwsCode(err)
			if err != nil {
				if isAwsErr && code == "InvalidAccessException" {
					return jobpool.JobResult(nil), nil
				}
				return nil, err
			}
			lumiHub, err := s.Runtime.CreateResource("aws.securityhub.hub",
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
