package resources

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/aws/smithy-go/transport/http"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
	aws_transport "go.mondoo.io/mondoo/motor/providers/aws"
)

func (s *lumiAwsSagemaker) id() (string, error) {
	return "aws.sagemaker", nil
}

func (s *lumiAwsSagemaker) GetEndpoints() ([]interface{}, error) {
	at, err := awstransport(s.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(s.getEndpoints(at), 5)
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

func (s *lumiAwsSagemaker) getEndpoints(at *aws_transport.Transport) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
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
				endpoints, err := svc.ListEndpoints(ctx, params)
				if err != nil {
					return nil, err
				}

				for _, endpoint := range endpoints.Endpoints {
					tags, err := getSagemakerTags(ctx, svc, endpoint.EndpointArn)
					if err != nil {
						return nil, err
					}
					lumiEndpoint, err := s.MotorRuntime.CreateResource("aws.sagemaker.endpoint",
						"arn", toString(endpoint.EndpointArn),
						"name", toString(endpoint.EndpointName),
						"region", regionVal,
						"tags", tags,
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
	at, err := awstransport(s.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	svc := at.Sagemaker(region)
	ctx := context.Background()
	config, err := svc.DescribeEndpointConfig(ctx, &sagemaker.DescribeEndpointConfigInput{EndpointConfigName: &name})
	if err != nil {
		return nil, err
	}
	return jsonToDict(config)
}

func (s *lumiAwsSagemaker) GetNotebookInstances() ([]interface{}, error) {
	at, err := awstransport(s.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(s.getNotebookInstances(at), 5)
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

func (s *lumiAwsSagemaker) getNotebookInstances(at *aws_transport.Transport) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	at, err := awstransport(s.MotorRuntime.Motor.Transport)
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
				notebookInstances, err := svc.ListNotebookInstances(ctx, &sagemaker.ListNotebookInstancesInput{})
				if err != nil {
					return nil, err
				}
				for _, instance := range notebookInstances.NotebookInstances {
					tags, err := getSagemakerTags(ctx, svc, instance.NotebookInstanceArn)
					if err != nil {
						return nil, err
					}
					lumiEndpoint, err := s.MotorRuntime.CreateResource("aws.sagemaker.notebookinstance",
						"arn", toString(instance.NotebookInstanceArn),
						"name", toString(instance.NotebookInstanceName),
						"region", regionVal,
						"tags", tags,
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

func (s *lumiAwsSagemakerNotebookinstance) GetDetails() (interface{}, error) {
	name, err := s.Name()
	if err != nil {
		return nil, err
	}
	region, err := s.Region()
	if err != nil {
		return nil, err
	}
	at, err := awstransport(s.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	svc := at.Sagemaker(region)
	ctx := context.Background()
	instanceDetails, err := svc.DescribeNotebookInstance(ctx, &sagemaker.DescribeNotebookInstanceInput{NotebookInstanceName: &name})
	if err != nil {
		return nil, err
	}
	lumiKeyResource, err := s.MotorRuntime.CreateResource("aws.kms.key",
		"arn", toString(instanceDetails.KmsKeyId),
	)
	if err != nil {
		return nil, err
	}
	lumiInstanceDetails, err := s.MotorRuntime.CreateResource("aws.sagemaker.notebookinstance.details",
		"arn", toString(instanceDetails.NotebookInstanceArn),
		"kmsKey", lumiKeyResource,
		"directInternetAccess", string(instanceDetails.DirectInternetAccess),
	)
	if err != nil {
		return nil, err
	}
	return lumiInstanceDetails, nil
}

func (s *lumiAwsSagemakerEndpoint) id() (string, error) {
	return s.Arn()
}

func (s *lumiAwsSagemakerNotebookinstance) id() (string, error) {
	return s.Arn()
}

func (s *lumiAwsSagemakerNotebookinstanceDetails) id() (string, error) {
	return s.Arn()
}

func getSagemakerTags(ctx context.Context, svc *sagemaker.Client, arn *string) (map[string]interface{}, error) {
	resp, err := svc.ListTags(ctx, &sagemaker.ListTagsInput{ResourceArn: arn})
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
