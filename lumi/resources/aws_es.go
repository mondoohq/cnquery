package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/elasticsearchservice"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
)

func (e *lumiAwsEs) id() (string, error) {
	return "aws.es", nil
}

func (e *lumiAwsEs) GetDomains() ([]interface{}, error) {
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(e.getDomains(), 5)
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

func (e *lumiAwsEs) getDomains() []*jobpool.Job {
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
			svc := at.Es(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			domains, err := svc.ListDomainNamesRequest(&elasticsearchservice.ListDomainNamesInput{}).Send(ctx)
			if err != nil {
				return nil, err
			}

			for _, domain := range domains.DomainNames {
				domainDetails, err := svc.DescribeElasticsearchDomainRequest(&elasticsearchservice.DescribeElasticsearchDomainInput{DomainName: domain.DomainName}).Send(ctx)
				if err != nil {
					return nil, err
				}
				lumiDomain, err := e.Runtime.CreateResource("aws.es.domain",
					"arn", toString(domainDetails.DomainStatus.ARN),
					"encryptionAtRestEnabled", toBool(domainDetails.DomainStatus.EncryptionAtRestOptions.Enabled),
					"nodeToNodeEncryptionEnabled", toBool(domainDetails.DomainStatus.NodeToNodeEncryptionOptions.Enabled),
					"endpoint", toString(domainDetails.DomainStatus.Endpoint),
					"name", toString(domainDetails.DomainStatus.DomainName),
				)
				if err != nil {
					return nil, err
				}
				res = append(res, lumiDomain)
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (e *lumiAwsEsDomain) id() (string, error) {
	return e.Arn()
}
