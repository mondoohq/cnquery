// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/elasticsearchservice"
	"github.com/aws/smithy-go/transport/http"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v11/providers/aws/connection"
	"go.mondoo.com/cnquery/v11/types"
)

func (a *mqlAwsEs) id() (string, error) {
	return "aws.es", nil
}

func (a *mqlAwsEs) domains() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getDomains(conn), 5)
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

func (a *mqlAwsEs) getDomains(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Es(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			domains, err := svc.ListDomainNames(ctx, &elasticsearchservice.ListDomainNamesInput{})
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
					return res, nil
				}
				return nil, err
			}

			for _, domain := range domains.DomainNames {
				// note: the api returns name and region here, so we just use that.
				// the arn is not returned until we get to the describe call
				mqlDomain, err := NewResource(a.MqlRuntime, "aws.es.domain",
					map[string]*llx.RawData{
						"name":   llx.StringDataPtr(domain.DomainName),
						"region": llx.StringData(regionVal),
					})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlDomain)
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func initAwsEsDomain(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if args["name"] == nil || args["region"] == nil {
		return nil, nil, errors.New("name and region required to fetch es domain")
	}

	name := args["name"].Value.(string)
	region := args["region"].Value.(string)

	conn := runtime.Connection.(*connection.AwsConnection)
	svc := conn.Es(region)
	ctx := context.Background()
	domainDetails, err := svc.DescribeElasticsearchDomain(ctx, &elasticsearchservice.DescribeElasticsearchDomainInput{DomainName: &name})
	if err != nil {
		return nil, nil, err
	}
	tags, err := getESTags(ctx, svc, domainDetails.DomainStatus.ARN)
	if err != nil {
		return nil, nil, err
	}
	args["encryptionAtRestEnabled"] = llx.BoolData(convert.ToValue(domainDetails.DomainStatus.EncryptionAtRestOptions.Enabled))
	args["nodeToNodeEncryptionEnabled"] = llx.BoolData(convert.ToValue(domainDetails.DomainStatus.NodeToNodeEncryptionOptions.Enabled))
	args["endpoint"] = llx.StringDataPtr(domainDetails.DomainStatus.Endpoint)
	args["arn"] = llx.StringDataPtr(domainDetails.DomainStatus.ARN)
	args["elasticsearchVersion"] = llx.StringDataPtr(domainDetails.DomainStatus.ElasticsearchVersion)
	args["domainId"] = llx.StringDataPtr(domainDetails.DomainStatus.DomainId)
	args["domainName"] = llx.StringDataPtr(domainDetails.DomainStatus.DomainName)
	args["tags"] = llx.MapData(tags, types.String)
	return args, nil, nil
}

func (a *mqlAwsEsDomain) id() (string, error) {
	return a.Arn.Data, nil
}

func getESTags(ctx context.Context, svc *elasticsearchservice.Client, arn *string) (map[string]interface{}, error) {
	resp, err := svc.ListTags(ctx, &elasticsearchservice.ListTagsInput{ARN: arn})
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
	for _, t := range resp.TagList {
		if t.Key != nil && t.Value != nil {
			tags[*t.Key] = *t.Value
		}
	}
	return tags, nil
}
