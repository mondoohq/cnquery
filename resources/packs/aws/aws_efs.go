package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/aws/aws-sdk-go-v2/service/efs/types"
	"github.com/aws/smithy-go/transport/http"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
	aws_transport "go.mondoo.io/mondoo/motor/providers/aws"
	"go.mondoo.io/mondoo/resources/packs/core"
)

func (e *lumiAwsEfs) id() (string, error) {
	return "aws.efs", nil
}

func (e *lumiAwsEfsFilesystem) id() (string, error) {
	return e.Arn()
}

func (e *lumiAwsEfs) GetFilesystems() ([]interface{}, error) {
	at, err := awstransport(e.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(e.getFilesystems(at), 5)
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

func (e *lumiAwsEfs) getFilesystems(at *aws_transport.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := at.Efs(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			var marker *string
			for {
				describeFileSystemsRes, err := svc.DescribeFileSystems(ctx, &efs.DescribeFileSystemsInput{Marker: marker})
				if err != nil {
					return nil, err
				}

				for i := range describeFileSystemsRes.FileSystems {
					fs := describeFileSystemsRes.FileSystems[i]
					args := []interface{}{
						"id", core.ToString(fs.FileSystemId),
						"arn", core.ToString(fs.FileSystemArn),
						"name", core.ToString(fs.Name),
						"encrypted", core.ToBool(fs.Encrypted),
						"region", regionVal,
						"tags", efsTagsToMap(fs.Tags),
					}
					// add kms key if there is one
					if fs.KmsKeyId != nil {
						lumiKeyResource, err := e.MotorRuntime.CreateResource("aws.kms.key",
							"arn", core.ToString(fs.KmsKeyId),
						)
						if err != nil {
							return nil, err
						}
						lumiKey := lumiKeyResource.(AwsKmsKey)
						args = append(args, "kmsKey", lumiKey)
					}
					lumiFilesystem, err := e.MotorRuntime.CreateResource("aws.efs.filesystem", args...)
					if err != nil {
						return nil, err
					}
					res = append(res, lumiFilesystem)
				}
				if describeFileSystemsRes.NextMarker == nil {
					break
				}
				marker = describeFileSystemsRes.NextMarker
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func efsTagsToMap(tags []types.Tag) map[string]interface{} {
	tagsMap := make(map[string]interface{})

	if len(tags) > 0 {
		for i := range tags {
			tag := tags[i]
			tagsMap[core.ToString(tag.Key)] = core.ToString(tag.Value)
		}
	}

	return tagsMap
}

func (e *lumiAwsEfsFilesystem) GetKmsKey() (interface{}, error) {
	// no key id on the log group object
	return nil, nil
}

func (e *lumiAwsEfsFilesystem) GetBackupPolicy() (interface{}, error) {
	id, err := e.Id()
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse id")
	}
	region, err := e.Region()
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse instance region")
	}
	at, err := awstransport(e.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	svc := at.Efs(region)
	ctx := context.Background()

	backupPolicy, err := svc.DescribeBackupPolicy(ctx, &efs.DescribeBackupPolicyInput{
		FileSystemId: &id,
	})
	var respErr *http.ResponseError
	if err != nil && errors.As(err, &respErr) {
		if respErr.HTTPStatusCode() == 404 {
			return nil, nil
		}
	} else if err != nil {
		return nil, err
	}
	res, err := core.JsonToDict(backupPolicy)
	if err != nil {
		return nil, err
	}
	return res, nil
}
