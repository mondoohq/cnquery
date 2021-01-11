package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
)

func (s *lumiAwsSagemaker) id() (string, error) {
	return "aws.sagemaker", nil
}

func (s *lumiAwsSagemaker) GetEndpoints() ([]interface{}, error) {
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(s.getEndpoints(), 5)
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
func (s *lumiAwsSagemaker) getEndpoints() []*jobpool.Job {
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
			svc := at.Sagemaker(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			nextToken := aws.String("no_token_to_start_with")
			params := &sagemaker.ListEndpointsInput{}
			for nextToken != nil {
				endpoints, err := svc.ListEndpointsRequest(params).Send(ctx)
				if err != nil {
					return nil, err
				}

				for _, endpoint := range endpoints.Endpoints {
					lumiEndpoint, err := s.Runtime.CreateResource("aws.sagemaker.endpoint",
						"arn", toString(endpoint.EndpointArn),
						"name", toString(endpoint.EndpointName),
						"region", regionVal,
					)
					if err != nil {
						return nil, err
					}
					res = append(res, lumiEndpoint)
				}
				nextToken = endpoints.NextToken
				if endpoints.NextToken != nil {
					params.NextToken = nextToken
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (s *lumiAwsSagemakerEndpoint) GetConfig() (map[string]interface{}, error) {
	name, err := s.Name()
	if err != nil {
		return nil, err
	}
	region, err := s.Region()
	if err != nil {
		return nil, err
	}
	at, err := awstransport(s.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	svc := at.Sagemaker(region)
	ctx := context.Background()
	config, err := svc.DescribeEndpointConfigRequest(&sagemaker.DescribeEndpointConfigInput{EndpointConfigName: &name}).Send(ctx)
	if err != nil {
		return nil, err
	}
	return jsonToDict(config)
}

func (s *lumiAwsSagemaker) GetNotebookInstances() ([]interface{}, error) {
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(s.getNotebookInstances(), 5)
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

func (s *lumiAwsSagemaker) getNotebookInstances() []*jobpool.Job {
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
			svc := at.Sagemaker(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			nextToken := aws.String("no_token_to_start_with")
			params := &sagemaker.ListNotebookInstancesInput{}
			for nextToken != nil {
				notebookInstances, err := svc.ListNotebookInstancesRequest(&sagemaker.ListNotebookInstancesInput{}).Send(ctx)
				if err != nil {
					return nil, err
				}
				for _, instance := range notebookInstances.NotebookInstances {
					lumiEndpoint, err := s.Runtime.CreateResource("aws.sagemaker.notebookinstance",
						"arn", toString(instance.NotebookInstanceArn),
						"name", toString(instance.NotebookInstanceName),
						"region", regionVal,
					)
					if err != nil {
						return nil, err
					}
					res = append(res, lumiEndpoint)
				}
				nextToken = notebookInstances.NextToken
				if notebookInstances.NextToken != nil {
					params.NextToken = nextToken
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (s *lumiAwsSagemakerNotebookinstance) GetKmsKeyId() (string, error) {
	name, err := s.Name()
	if err != nil {
		return "", err
	}
	region, err := s.Region()
	if err != nil {
		return "", err
	}
	at, err := awstransport(s.Runtime.Motor.Transport)
	if err != nil {
		return "", err
	}
	svc := at.Sagemaker(region)
	ctx := context.Background()
	instanceDetails, err := svc.DescribeNotebookInstanceRequest(&sagemaker.DescribeNotebookInstanceInput{NotebookInstanceName: &name}).Send(ctx)
	if err != nil {
		return "", err
	}
	return toString(instanceDetails.KmsKeyId), nil
}

func (s *lumiAwsSagemakerEndpoint) id() (string, error) {
	return s.Arn()
}

func (s *lumiAwsSagemakerNotebookinstance) id() (string, error) {
	return s.Arn()
}
