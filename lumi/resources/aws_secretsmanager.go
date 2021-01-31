package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
)

func (e *lumiAwsSecretsmanager) id() (string, error) {
	return "aws.secretsmanager", nil
}

func (e *lumiAwsSecretsmanager) GetSecrets() ([]interface{}, error) {
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(e.getSecrets(), 5)
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

func (e *lumiAwsSecretsmanagerSecret) id() (string, error) {
	return e.Arn()
}

func (e *lumiAwsSecretsmanager) getSecrets() []*jobpool.Job {
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

			svc := at.Secretsmanager(regionVal)
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
					lumiSecret, err := e.Runtime.CreateResource("aws.secretsmanager.secret",
						"arn", toString(secret.ARN),
						"name", toString(secret.Name),
						"rotationEnabled", secret.RotationEnabled,
					)
					if err != nil {
						return nil, err
					}
					res = append(res, lumiSecret)
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
