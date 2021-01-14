package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
)

func (e *lumiAwsEfs) id() (string, error) {
	return "aws.efs", nil
}

func (e *lumiAwsEfsFilesystem) id() (string, error) {
	return e.Arn()
}

func (e *lumiAwsEfs) GetFilesystems() ([]interface{}, error) {
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(e.getFilesystems(), 5)
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

func (e *lumiAwsEfs) getFilesystems() []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
	at, err := awstransport(e.Runtime.Motor.Transport)
	if err != nil {
		return []*jobpool.Job{&jobpool.Job{Err: err}} // return the error
	}
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{&jobpool.Job{Err: err}} // return the error
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
				describeFileSystemsRes, err := svc.DescribeFileSystemsRequest(&efs.DescribeFileSystemsInput{Marker: marker}).Send(ctx)
				if err != nil {
					return nil, err
				}

				for i := range describeFileSystemsRes.FileSystems {
					fs := describeFileSystemsRes.FileSystems[i]
					args := []interface{}{
						"id", toString(fs.FileSystemId),
						"arn", toString(fs.FileSystemArn),
						"name", toString(fs.Name),
						"encrypted", toBool(fs.Encrypted),
						"region", regionVal,
					}
					// add kms key if there is one
					if fs.KmsKeyId != nil {
						lumiKeyResource, err := e.Runtime.CreateResource("aws.kms.key",
							"arn", toString(fs.KmsKeyId),
						)
						if err != nil {
							return nil, err
						}
						lumiKey := lumiKeyResource.(AwsKmsKey)
						args = append(args, "kmsKey", lumiKey)
					}
					lumiFilesystem, err := e.Runtime.CreateResource("aws.efs.filesystem", args...)
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
	at, err := awstransport(e.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	svc := at.Efs(region)
	ctx := context.Background()

	backupPolicy, err := svc.DescribeBackupPolicyRequest(&efs.DescribeBackupPolicyInput{
		FileSystemId: &id,
	}).Send(ctx)
	if err != nil {
		return nil, err
	}
	res, err := jsonToDict(backupPolicy)
	if err != nil {
		return nil, err
	}
	return res, nil
}
