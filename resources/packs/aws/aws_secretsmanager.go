package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	aws_provider "go.mondoo.io/mondoo/motor/providers/aws"
	"go.mondoo.io/mondoo/resources/library/jobpool"
	"go.mondoo.io/mondoo/resources/packs/core"
)

func (e *mqlAwsSecretsmanager) id() (string, error) {
	return "aws.secretsmanager", nil
}

func (e *mqlAwsSecretsmanager) GetSecrets() ([]interface{}, error) {
	provider, err := awsProvider(e.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(e.getSecrets(provider), 5)
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

func (e *mqlAwsSecretsmanagerSecret) id() (string, error) {
	return e.Arn()
}

func (e *mqlAwsSecretsmanager) getSecrets(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := provider.Secretsmanager(regionVal)
			ctx := context.Background()

			res := []interface{}{}

			nextToken := aws.String("no_token_to_start_with")
			params := &secretsmanager.ListSecretsInput{}
			for nextToken != nil {
				secrets, err := svc.ListSecrets(ctx, params)
				if err != nil {
					return nil, err
				}
				for _, secret := range secrets.SecretList {
					mqlSecret, err := e.MotorRuntime.CreateResource("aws.secretsmanager.secret",
						"arn", core.ToString(secret.ARN),
						"name", core.ToString(secret.Name),
						"rotationEnabled", secret.RotationEnabled,
						"tags", secretTagsToMap(secret.Tags),
					)
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

func secretTagsToMap(tags []types.Tag) map[string]interface{} {
	tagsMap := make(map[string]interface{})

	if len(tags) > 0 {
		for i := range tags {
			tag := tags[i]
			tagsMap[core.ToString(tag.Key)] = core.ToString(tag.Value)
		}
	}

	return tagsMap
}
