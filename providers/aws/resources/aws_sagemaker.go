// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/aws/smithy-go/transport/http"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v11/providers/aws/connection"

	"go.mondoo.com/cnquery/v11/types"
)

func (a *mqlAwsSagemaker) id() (string, error) {
	return "aws.sagemaker", nil
}

func (a *mqlAwsSagemaker) endpoints() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getEndpoints(conn), 5)
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

func (a *mqlAwsSagemaker) getEndpoints(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Sagemaker(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			nextToken := aws.String("no_token_to_start_with")
			params := &sagemaker.ListEndpointsInput{}
			for nextToken != nil {
				endpoints, err := svc.ListEndpoints(ctx, params)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, endpoint := range endpoints.Endpoints {
					tags, err := getSagemakerTags(ctx, svc, endpoint.EndpointArn)
					if err != nil {
						return nil, err
					}
					mqlEndpoint, err := CreateResource(a.MqlRuntime, "aws.sagemaker.endpoint",
						map[string]*llx.RawData{
							"arn":    llx.StringDataPtr(endpoint.EndpointArn),
							"name":   llx.StringDataPtr(endpoint.EndpointName),
							"region": llx.StringData(regionVal),
							"tags":   llx.MapData(tags, types.String),
						})
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

func (a *mqlAwsSagemakerEndpoint) config() (map[string]interface{}, error) {
	name := a.Name.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Sagemaker(region)
	ctx := context.Background()
	config, err := svc.DescribeEndpointConfig(ctx, &sagemaker.DescribeEndpointConfigInput{EndpointConfigName: &name})
	if err != nil {
		return nil, err
	}
	return convert.JsonToDict(config)
}

func (a *mqlAwsSagemaker) notebookInstances() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getNotebookInstances(conn), 5)
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

func (a *mqlAwsSagemaker) getNotebookInstances(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Sagemaker(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			nextToken := aws.String("no_token_to_start_with")
			params := &sagemaker.ListNotebookInstancesInput{}
			for nextToken != nil {
				notebookInstances, err := svc.ListNotebookInstances(ctx, &sagemaker.ListNotebookInstancesInput{})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, instance := range notebookInstances.NotebookInstances {
					tags, err := getSagemakerTags(ctx, svc, instance.NotebookInstanceArn)
					if err != nil {
						return nil, err
					}
					mqlEndpoint, err := CreateResource(a.MqlRuntime, "aws.sagemaker.notebookinstance",
						map[string]*llx.RawData{
							"arn":    llx.StringData(convert.ToValue(instance.NotebookInstanceArn)),
							"name":   llx.StringData(convert.ToValue(instance.NotebookInstanceName)),
							"region": llx.StringData(regionVal),
							"tags":   llx.MapData(tags, types.String),
						})
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

func initAwsSagemakerNotebookinstance(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch sagemaker notebookinstance")
	}

	obj, err := CreateResource(runtime, "aws.sagemaker", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	sm := obj.(*mqlAwsSagemaker)

	rawResources := sm.GetNotebookInstances()
	if rawResources.Error != nil {
		return nil, nil, rawResources.Error
	}

	arnVal := args["arn"].Value.(string)
	for i := range rawResources.Data {
		ni := rawResources.Data[i].(*mqlAwsSagemakerNotebookinstance)
		if ni.Arn.Data == arnVal {
			return args, ni, nil
		}
	}
	return nil, nil, errors.New("sagemaker notebookinstance does not exist")
}

func (a *mqlAwsSagemakerNotebookinstance) details() (*mqlAwsSagemakerNotebookinstancedetails, error) {
	name := a.Name.Data
	region := a.Region.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sagemaker(region)
	ctx := context.Background()
	instanceDetails, err := svc.DescribeNotebookInstance(ctx, &sagemaker.DescribeNotebookInstanceInput{NotebookInstanceName: &name})
	if err != nil {
		return nil, err
	}
	args := map[string]*llx.RawData{
		"arn":                  llx.StringDataPtr(instanceDetails.NotebookInstanceArn),
		"directInternetAccess": llx.StringData(string(instanceDetails.DirectInternetAccess)),
	}

	mqlInstanceDetails, err := CreateResource(a.MqlRuntime, "aws.sagemaker.notebookinstancedetails", args)
	if err != nil {
		return nil, err
	}
	mqlInstanceDetails.(*mqlAwsSagemakerNotebookinstancedetails).cacheKmsKey = instanceDetails.KmsKeyId
	return mqlInstanceDetails.(*mqlAwsSagemakerNotebookinstancedetails), nil
}

type mqlAwsSagemakerNotebookinstancedetailsInternal struct {
	cacheKmsKey *string
}

func (a *mqlAwsSagemakerNotebookinstancedetails) kmsKey() (*mqlAwsKmsKey, error) {
	if a.cacheKmsKey != nil && *a.cacheKmsKey != "" {
		mqlKeyResource, err := NewResource(a.MqlRuntime, "aws.kms.key",
			map[string]*llx.RawData{"arn": llx.StringData(convert.ToValue(a.cacheKmsKey))},
		)
		if err != nil {
			return nil, err
		}
		return mqlKeyResource.(*mqlAwsKmsKey), nil
	}
	a.KmsKey.State = plugin.StateIsNull | plugin.StateIsSet
	return nil, nil
}

func (a *mqlAwsSagemakerEndpoint) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerNotebookinstance) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerNotebookinstancedetails) id() (string, error) {
	return a.Arn.Data, nil
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
		if t.Key != nil && t.Value != nil {
			tags[*t.Key] = *t.Value
		}
	}
	return tags, nil
}
