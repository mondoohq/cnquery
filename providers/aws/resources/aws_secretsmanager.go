// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	secretstypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"

	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAwsSecretsmanager) id() (string, error) {
	return "aws.secretsmanager", nil
}

func (a *mqlAwsSecretsmanager) secrets() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getSecrets(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}
	return res, nil
}

func (a *mqlAwsSecretsmanagerSecret) id() (string, error) {
	return a.Arn.Data, nil
}

func initAwsSecretsmanagerSecret(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch secretsmanager secret")
	}

	obj, err := CreateResource(runtime, ResourceAwsSecretsmanager, map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	sm := obj.(*mqlAwsSecretsmanager)

	rawResources := sm.GetSecrets()
	if rawResources.Error != nil {
		return nil, nil, rawResources.Error
	}

	arnVal := args["arn"].Value.(string)
	for _, rawResource := range rawResources.Data {
		secret := rawResource.(*mqlAwsSecretsmanagerSecret)
		if secret.Arn.Data == arnVal {
			return args, secret, nil
		}
	}
	return nil, nil, errors.New("secretsmanager secret does not exist")
}

func (a *mqlAwsSecretsmanagerSecret) kmsKey() (*mqlAwsKmsKey, error) {
	return a.KmsKey.Data, nil
}

func (a *mqlAwsSecretsmanager) getSecrets(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Secretsmanager(region)
			ctx := context.Background()

			res := []any{}

			params := &secretsmanager.ListSecretsInput{}
			paginator := secretsmanager.NewListSecretsPaginator(svc, params)
			for paginator.HasMorePages() {
				secrets, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, secret := range secrets.SecretList {
					args := map[string]*llx.RawData{
						"arn":              llx.StringDataPtr(secret.ARN),
						"createdAt":        llx.TimeDataPtr(secret.CreatedDate),
						"description":      llx.StringDataPtr(secret.Description),
						"lastAccessedDate": llx.TimeDataPtr(secret.LastAccessedDate),
						"lastChangedDate":  llx.TimeDataPtr(secret.LastChangedDate),
						"lastRotatedDate":  llx.TimeDataPtr(secret.LastRotatedDate),
						"name":             llx.StringDataPtr(secret.Name),
						"nextRotationDate": llx.TimeDataPtr(secret.NextRotationDate),
						"owningService":    llx.StringDataPtr(secret.OwningService),
						"primaryRegion":    llx.StringDataPtr(secret.PrimaryRegion),
						"rotationEnabled":  llx.BoolData(convert.ToValue(secret.RotationEnabled)),
						"tags":             llx.MapData(secretTagsToMap(secret.Tags), types.String),
					}

					// add kms key if there is one
					if secret.KmsKeyId != nil {
						mqlKeyResource, err := NewResource(a.MqlRuntime, ResourceAwsKmsKey,
							map[string]*llx.RawData{
								"arn": llx.StringDataPtr(secret.KmsKeyId),
							})
						if err != nil {
							args["kmsKey"] = &llx.RawData{Type: types.Resource(ResourceAwsKmsKey), Error: err}
						} else {
							mqlKey := mqlKeyResource.(*mqlAwsKmsKey)
							args["kmsKey"] = llx.ResourceData(mqlKey, mqlKey.MqlName())
						}
					} else {
						args["kmsKey"] = llx.NilData
					}

					// add rotation lambda if there is one
					if secret.RotationLambdaARN != nil {
						mqlLambdaResource, err := NewResource(a.MqlRuntime, ResourceAwsLambdaFunction,
							map[string]*llx.RawData{
								"arn": llx.StringDataPtr(secret.RotationLambdaARN),
							})
						if err != nil {
							args["rotationLambda"] = &llx.RawData{Type: types.Resource(ResourceAwsLambdaFunction), Error: err}
						} else {
							mqlLambda := mqlLambdaResource.(*mqlAwsLambdaFunction)
							args["rotationLambda"] = llx.ResourceData(mqlLambda, mqlLambda.MqlName())
						}
					} else {
						args["rotationLambda"] = llx.NilData
					}

					// add rotation rules if configured
					if secret.RotationRules != nil {
						var automaticallyAfterDays int64
						if secret.RotationRules.AutomaticallyAfterDays != nil {
							automaticallyAfterDays = *secret.RotationRules.AutomaticallyAfterDays
						}
						mqlRotationRules, err := CreateResource(a.MqlRuntime, ResourceAwsSecretsmanagerSecretRotationRules,
							map[string]*llx.RawData{
								"__id":                   llx.StringData(convert.ToValue(secret.ARN) + "/rotationRules"),
								"automaticallyAfterDays": llx.IntData(automaticallyAfterDays),
								"duration":               llx.StringDataPtr(secret.RotationRules.Duration),
								"scheduleExpression":     llx.StringDataPtr(secret.RotationRules.ScheduleExpression),
							})
						if err != nil {
							args["rotationRules"] = &llx.RawData{Type: types.Resource(ResourceAwsSecretsmanagerSecretRotationRules), Error: err}
						} else {
							mqlRules := mqlRotationRules.(*mqlAwsSecretsmanagerSecretRotationRules)
							args["rotationRules"] = llx.ResourceData(mqlRules, mqlRules.MqlName())
						}
					} else {
						args["rotationRules"] = llx.NilData
					}

					mqlSecret, err := CreateResource(a.MqlRuntime, ResourceAwsSecretsmanagerSecret, args)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlSecret)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func secretTagsToMap(tags []secretstypes.Tag) map[string]any {
	tagsMap := make(map[string]any)

	if len(tags) > 0 {
		for i := range tags {
			tag := tags[i]
			tagsMap[convert.ToValue(tag.Key)] = convert.ToValue(tag.Value)
		}
	}

	return tagsMap
}
