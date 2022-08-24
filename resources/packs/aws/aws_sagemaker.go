package aws

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/aws/smithy-go/transport/http"
	aws_provider "go.mondoo.com/cnquery/motor/providers/aws"
	"go.mondoo.com/cnquery/resources/library/jobpool"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (s *mqlAwsSagemaker) id() (string, error) {
	return "aws.sagemaker", nil
}

func (s *mqlAwsSagemaker) GetEndpoints() ([]interface{}, error) {
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(s.getEndpoints(provider), 5)
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

func (s *mqlAwsSagemaker) getEndpoints(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := provider.Sagemaker(regionVal)
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
					mqlEndpoint, err := s.MotorRuntime.CreateResource("aws.sagemaker.endpoint",
						"arn", core.ToString(endpoint.EndpointArn),
						"name", core.ToString(endpoint.EndpointName),
						"region", regionVal,
						"tags", tags,
					)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlEndpoint)
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

func (s *mqlAwsSagemakerEndpoint) GetConfig() (map[string]interface{}, error) {
	name, err := s.Name()
	if err != nil {
		return nil, err
	}
	region, err := s.Region()
	if err != nil {
		return nil, err
	}
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	svc := provider.Sagemaker(region)
	ctx := context.Background()
	config, err := svc.DescribeEndpointConfig(ctx, &sagemaker.DescribeEndpointConfigInput{EndpointConfigName: &name})
	if err != nil {
		return nil, err
	}
	return core.JsonToDict(config)
}

func (s *mqlAwsSagemaker) GetNotebookInstances() ([]interface{}, error) {
	at, err := awsProvider(s.MotorRuntime.Motor.Provider)
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

func (s *mqlAwsSagemaker) getNotebookInstances(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := provider.Sagemaker(regionVal)
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
					mqlEndpoint, err := s.MotorRuntime.CreateResource("aws.sagemaker.notebookinstance",
						"arn", core.ToString(instance.NotebookInstanceArn),
						"name", core.ToString(instance.NotebookInstanceName),
						"region", regionVal,
						"tags", tags,
					)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlEndpoint)
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

func (s *mqlAwsSagemakerNotebookinstance) GetDetails() (interface{}, error) {
	name, err := s.Name()
	if err != nil {
		return nil, err
	}
	region, err := s.Region()
	if err != nil {
		return nil, err
	}
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	svc := provider.Sagemaker(region)
	ctx := context.Background()
	instanceDetails, err := svc.DescribeNotebookInstance(ctx, &sagemaker.DescribeNotebookInstanceInput{NotebookInstanceName: &name})
	if err != nil {
		return nil, err
	}
	mqlKeyResource, err := s.MotorRuntime.CreateResource("aws.kms.key",
		"arn", core.ToString(instanceDetails.KmsKeyId),
	)
	if err != nil {
		return nil, err
	}
	mqlInstanceDetails, err := s.MotorRuntime.CreateResource("aws.sagemaker.notebookinstance.details",
		"arn", core.ToString(instanceDetails.NotebookInstanceArn),
		"kmsKey", mqlKeyResource,
		"directInternetAccess", string(instanceDetails.DirectInternetAccess),
	)
	if err != nil {
		return nil, err
	}
	return mqlInstanceDetails, nil
}

func (s *mqlAwsSagemakerEndpoint) id() (string, error) {
	return s.Arn()
}

func (s *mqlAwsSagemakerNotebookinstance) id() (string, error) {
	return s.Arn()
}

func (s *mqlAwsSagemakerNotebookinstanceDetails) id() (string, error) {
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
