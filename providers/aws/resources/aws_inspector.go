// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/inspector2"
	"github.com/aws/aws-sdk-go-v2/service/inspector2/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v11/providers/aws/connection"
	llxtypes "go.mondoo.com/cnquery/v11/types"
)

func (a *mqlAwsInspector) id() (string, error) {
	return "aws.inspector", nil
}

func (a *mqlAwsInspectorCoverage) id() (string, error) {
	return a.AccountId.Data + "/" + a.ResourceId.Data, nil
}

func (a *mqlAwsInspector) coverages() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getCoverage(conn), 5)
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

func (a *mqlAwsInspector) getCoverage(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Inspector(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			nextToken := aws.String("no_token_to_start_with")
			params := &inspector2.ListCoverageInput{}
			for nextToken != nil {
				coverages, err := svc.ListCoverage(ctx, params)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, coverage := range coverages.CoveredResources {
					if coverage.AccountId == nil || coverage.ResourceId == nil {
						continue
					}
					mqlCoverage, err := CreateResource(a.MqlRuntime, "aws.inspector.coverage",
						map[string]*llx.RawData{
							"accountId":     llx.StringDataPtr(coverage.AccountId),
							"resourceId":    llx.StringDataPtr(coverage.ResourceId),
							"resourceType":  llx.StringData(string(coverage.ResourceType)),
							"lastScannedAt": llx.TimeDataPtr(coverage.LastScannedAt),
							"statusReason":  llx.StringData(string(coverage.ScanStatus.Reason)),
							"statusCode":    llx.StringData(string(coverage.ScanStatus.StatusCode)),
							"scanType":      llx.StringData(string(coverage.ScanType)),
							"region":        llx.StringData(regionVal),
						},
					)
					if err != nil {
						return nil, err
					}
					mqlCoverage.(*mqlAwsInspectorCoverage).cacheCoverage = &coverage
					res = append(res, mqlCoverage)
				}
				nextToken = coverages.NextToken
				if coverages.NextToken != nil {
					params.NextToken = nextToken
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsInspectorCoverageInternal struct {
	cacheCoverage *types.CoveredResource
}

type mqlAwsInspectorCoverageInstanceInternal struct {
	cacheAmiId string
}

func (a *mqlAwsInspectorCoverageInstance) id() (string, error) {
	strTags := ""
	for k, v := range a.Tags.Data {
		strTags = strTags + k + "/" + v.(string) + "/"
	}
	return a.Region.Data + "/" + strTags, nil
}

func (a *mqlAwsInspectorCoverage) ec2Instance() (*mqlAwsInspectorCoverageInstance, error) {
	if a.cacheCoverage != nil && a.cacheCoverage.ResourceMetadata != nil && a.cacheCoverage.ResourceMetadata.Ec2 != nil {
		conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
		args := map[string]*llx.RawData{
			"platform": llx.StringData(string(a.cacheCoverage.ResourceMetadata.Ec2.Platform)),
			"tags":     llx.MapData(mapConversion(a.cacheCoverage.ResourceMetadata.Ec2.Tags), llxtypes.String),
			"region":   llx.StringData(a.Region.Data),
		}
		image, err := NewResource(a.MqlRuntime, "aws.ec2.image", map[string]*llx.RawData{
			"arn": llx.StringData(fmt.Sprintf(imageArnPattern, a.Region.Data, conn.AccountId(), convert.ToString(a.cacheCoverage.ResourceMetadata.Ec2.AmiId))),
		})
		if err == nil {
			args["image"] = llx.ResourceData(image, "aws.ec2.image")
		}
		mqlEc2Instance, err := CreateResource(a.MqlRuntime, "aws.inspector.coverage.instance", args)
		if err == nil {
			mqlEc2Instance.(*mqlAwsInspectorCoverageInstance).cacheAmiId = *a.cacheCoverage.ResourceMetadata.Ec2.AmiId
			return mqlEc2Instance.(*mqlAwsInspectorCoverageInstance), err
		}
	}
	a.Ec2Instance.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

func mapConversion(m map[string]string) map[string]interface{} {
	newMap := make(map[string]interface{})
	for k, v := range m {
		newMap[k] = v
	}
	return newMap
}

func listMapConversion(m []string) map[string]interface{} {
	newMap := make(map[string]interface{})
	for _, k := range m {
		newMap[k] = ""
	}
	return newMap
}

func (a *mqlAwsInspectorCoverageImage) id() (string, error) {
	tagString := ""
	for k, v := range a.Tags.Data {
		tagString = tagString + k + "/" + v.(string) + "/"
	}
	return a.Region.Data + "/" + tagString, nil
}

func (a *mqlAwsInspectorCoverage) ecrImage() (*mqlAwsInspectorCoverageImage, error) {
	if a.cacheCoverage != nil && a.cacheCoverage.ResourceMetadata != nil && a.cacheCoverage.ResourceMetadata.EcrImage != nil {
		mqlEcr, err := CreateResource(a.MqlRuntime, "aws.inspector.coverage.image", map[string]*llx.RawData{
			"tags":          llx.MapData(listMapConversion(a.cacheCoverage.ResourceMetadata.EcrImage.Tags), llxtypes.String),
			"imagePulledAt": llx.TimeDataPtr(a.cacheCoverage.ResourceMetadata.EcrImage.ImagePulledAt),
			"region":        llx.StringData(a.Region.Data),
		})
		if err == nil {
			return mqlEcr.(*mqlAwsInspectorCoverageImage), err
		}
	}
	a.EcrImage.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

func (a *mqlAwsInspectorCoverageRepository) id() (string, error) {
	return a.Region.Data + "/" + a.Name.Data, nil
}

func (a *mqlAwsInspectorCoverage) ecrRepo() (*mqlAwsInspectorCoverageRepository, error) {
	if a.cacheCoverage != nil && a.cacheCoverage.ResourceMetadata != nil && a.cacheCoverage.ResourceMetadata.EcrRepository != nil {
		mqlEcr, err := CreateResource(a.MqlRuntime, "aws.inspector.coverage.repository", map[string]*llx.RawData{
			"name":          llx.StringDataPtr(a.cacheCoverage.ResourceMetadata.EcrRepository.Name),
			"scanFrequency": llx.StringData(string(a.cacheCoverage.ResourceMetadata.EcrRepository.ScanFrequency)),
			"region":        llx.StringData(a.Region.Data),
		})
		if err == nil {
			return mqlEcr.(*mqlAwsInspectorCoverageRepository), err
		}
	}
	a.EcrRepo.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

func (a *mqlAwsInspectorCoverage) lambda() (*mqlAwsLambdaFunction, error) {
	if a.cacheCoverage != nil && a.cacheCoverage.ResourceMetadata != nil && a.cacheCoverage.ResourceMetadata.LambdaFunction != nil {
		l, err := NewResource(a.MqlRuntime, "aws.lambda.function",
			map[string]*llx.RawData{
				"name":   llx.StringData(*a.cacheCoverage.ResourceMetadata.LambdaFunction.FunctionName),
				"region": llx.StringData(a.Region.Data),
			})
		if err == nil {
			return l.(*mqlAwsLambdaFunction), nil
		}
	}
	a.Lambda.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}
