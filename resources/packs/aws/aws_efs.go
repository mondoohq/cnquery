package aws

import (
	"context"

	"errors"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/aws/aws-sdk-go-v2/service/efs/types"
	"github.com/aws/smithy-go/transport/http"
	"github.com/rs/zerolog/log"
	aws_provider "go.mondoo.com/cnquery/motor/providers/aws"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/library/jobpool"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (e *mqlAwsEfs) id() (string, error) {
	return "aws.efs", nil
}

func (e *mqlAwsEfsFilesystem) id() (string, error) {
	return e.Arn()
}

func (e *mqlAwsEfs) GetFilesystems() ([]interface{}, error) {
	provider, err := awsProvider(e.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(e.getFilesystems(provider), 5)
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

func (e *mqlAwsEfs) getFilesystems(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := provider.Efs(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			var marker *string
			for {
				describeFileSystemsRes, err := svc.DescribeFileSystems(ctx, &efs.DescribeFileSystemsInput{Marker: marker})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
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
						mqlKeyResource, err := e.MotorRuntime.CreateResource("aws.kms.key",
							"arn", core.ToString(fs.KmsKeyId),
						)
						if err != nil {
							return nil, err
						}
						mqlKey := mqlKeyResource.(AwsKmsKey)
						args = append(args, "kmsKey", mqlKey)
					}
					mqlFilesystem, err := e.MotorRuntime.CreateResource("aws.efs.filesystem", args...)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlFilesystem)
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

func (d *mqlAwsEfsFilesystem) init(args *resources.Args) (*resources.Args, AwsEfsFilesystem, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if len(*args) == 0 {
		if ids := getAssetIdentifier(d.MqlResource().MotorRuntime); ids != nil {
			(*args)["name"] = ids.name
			(*args)["arn"] = ids.arn
		}
	}

	if (*args)["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch efs filesystem")
	}

	obj, err := d.MotorRuntime.CreateResource("aws.efs")
	if err != nil {
		return nil, nil, err
	}
	efs := obj.(AwsEfs)

	rawResources, err := efs.Filesystems()
	if err != nil {
		return nil, nil, err
	}

	arnVal := (*args)["arn"].(string)
	for i := range rawResources {
		fs := rawResources[i].(AwsEfsFilesystem)
		mqlFsArn, err := fs.Arn()
		if err != nil {
			return nil, nil, errors.New("efs filesystem does not exist")
		}
		if mqlFsArn == arnVal {
			return args, fs, nil
		}
	}
	return nil, nil, errors.New("efs filesystem does not exist")
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

func (e *mqlAwsEfsFilesystem) GetKmsKey() (interface{}, error) {
	// no key id on the log group object
	return nil, nil
}

func (e *mqlAwsEfsFilesystem) GetBackupPolicy() (interface{}, error) {
	id, err := e.Id()
	if err != nil {
		return nil, errors.Join(err, errors.New("unable to parse id"))
	}
	region, err := e.Region()
	if err != nil {
		return nil, errors.Join(err, errors.New("unable to parse instance region"))
	}
	provider, err := awsProvider(e.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	svc := provider.Efs(region)
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
