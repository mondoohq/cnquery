// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/efs"
	efstypes "github.com/aws/aws-sdk-go-v2/service/efs/types"
	"github.com/aws/smithy-go/transport/http"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v10/providers/aws/connection"

	"go.mondoo.com/cnquery/v10/types"
)

func (a *mqlAwsEfsFilesystem) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsEfs) filesystems() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getFilesystems(conn), 5)
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

func (a *mqlAwsEfs) getFilesystems(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for i := range regions {
		regionVal := regions[i]
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("efs>getFilesystems>calling aws with region %s", regionVal)

			svc := conn.Efs(regionVal)
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
					args := map[string]*llx.RawData{
						"id":               llx.StringDataPtr(fs.FileSystemId),
						"arn":              llx.StringDataPtr(fs.FileSystemArn),
						"name":             llx.StringDataPtr(fs.Name),
						"encrypted":        llx.BoolData(convert.ToBool(fs.Encrypted)),
						"region":           llx.StringData(regionVal),
						"availabilityZone": llx.StringDataPtr(fs.AvailabilityZoneName),
						"createdAt":        llx.TimeDataPtr(fs.CreationTime),
						"tags":             llx.MapData(efsTagsToMap(fs.Tags), types.String),
					}
					// add kms key if there is one
					if fs.KmsKeyId != nil {
						mqlKeyResource, err := NewResource(a.MqlRuntime, "aws.kms.key", map[string]*llx.RawData{
							"arn": llx.StringDataPtr(fs.KmsKeyId),
						})
						if err != nil {
							log.Error().Err(err).Msg("cannot create kms key resource")
						} else {
							args["kmsKey"] = llx.ResourceData(mqlKeyResource, mqlKeyResource.MqlName())
						}
					}
					mqlFilesystem, err := CreateResource(a.MqlRuntime, "aws.efs.filesystem", args)
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

func (a *mqlAwsEfsFilesystem) kmsKey() (*mqlAwsKmsKey, error) {
	return a.GetKmsKey().Data, nil
}

func initAwsEfsFilesystem(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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
		return nil, nil, errors.New("arn required to fetch efs filesystem")
	}

	// load all efs filesystems
	obj, err := CreateResource(runtime, "aws.efs", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	efs := obj.(*mqlAwsEfs)

	rawResources := efs.GetFilesystems()
	if err != nil {
		return nil, nil, err
	}

	arnVal := args["arn"].Value.(string)
	for i := range rawResources.Data {
		fs := rawResources.Data[i].(*mqlAwsEfsFilesystem)
		if fs.Arn.Data == arnVal {
			return args, fs, nil
		}
	}
	return nil, nil, errors.New("rds db instance does not exist")
}

func (a *mqlAwsEfsFilesystem) backupPolicy() (interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	id := a.Id.Data
	region := a.Region.Data

	svc := conn.Efs(region)
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
	res, err := convert.JsonToDict(backupPolicy)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func efsTagsToMap(tags []efstypes.Tag) map[string]interface{} {
	tagsMap := make(map[string]interface{})

	if len(tags) > 0 {
		for i := range tags {
			tag := tags[i]
			tagsMap[convert.ToString(tag.Key)] = convert.ToString(tag.Value)
		}
	}

	return tagsMap
}
