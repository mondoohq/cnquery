// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	secretstypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v10/providers/aws/connection"

	"go.mondoo.com/cnquery/v10/types"
)

func (a *mqlAwsSecretsmanager) id() (string, error) {
	return "aws.secretsmanager", nil
}

func (a *mqlAwsSecretsmanager) secrets() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getSecrets(conn), 5)
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

func (a *mqlAwsSecretsmanagerSecret) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSecretsmanager) getSecrets(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Secretsmanager(regionVal)
			ctx := context.Background()

			res := []interface{}{}

			nextToken := aws.String("no_token_to_start_with")
			params := &secretsmanager.ListSecretsInput{}
			for nextToken != nil {
				secrets, err := svc.ListSecrets(ctx, params)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, secret := range secrets.SecretList {
					mqlSecret, err := CreateResource(a.MqlRuntime, "aws.secretsmanager.secret",
						map[string]*llx.RawData{
							"arn":              llx.StringDataPtr(secret.ARN),
							"createdAt":        llx.TimeDataPtr(secret.CreatedDate),
							"description":      llx.StringDataPtr(secret.Description),
							"lastChangedDate":  llx.TimeDataPtr(secret.LastChangedDate),
							"lastRotatedDate":  llx.TimeDataPtr(secret.LastRotatedDate),
							"name":             llx.StringDataPtr(secret.Name),
							"nextRotationDate": llx.TimeDataPtr(secret.NextRotationDate),
							"primaryRegion":    llx.StringDataPtr(secret.PrimaryRegion),
							"rotationEnabled":  llx.BoolData(convert.ToBool(secret.RotationEnabled)),
							"tags":             llx.MapData(secretTagsToMap(secret.Tags), types.String),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlSecret)
				}
				nextToken = secrets.NextToken
				if secrets.NextToken != nil {
					params.NextToken = nextToken
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func secretTagsToMap(tags []secretstypes.Tag) map[string]interface{} {
	tagsMap := make(map[string]interface{})

	if len(tags) > 0 {
		for i := range tags {
			tag := tags[i]
			tagsMap[convert.ToString(tag.Key)] = convert.ToString(tag.Value)
		}
	}

	return tagsMap
}
