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
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v12/providers/aws/connection"

	"go.mondoo.com/cnquery/v12/types"
)

func (a *mqlAwsEfsFilesystem) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsEfs) filesystems() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getFilesystems(conn), 5)
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

func (a *mqlAwsEfs) getFilesystems(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("efs>getFilesystems>calling aws with region %s", region)

			svc := conn.Efs(region)
			ctx := context.Background()
			res := []any{}

			params := &efs.DescribeFileSystemsInput{}
			paginator := efs.NewDescribeFileSystemsPaginator(svc, params)
			for paginator.HasMorePages() {
				describeFileSystemsRes, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, fs := range describeFileSystemsRes.FileSystems {
					args := map[string]*llx.RawData{
						"id":               llx.StringDataPtr(fs.FileSystemId),
						"arn":              llx.StringDataPtr(fs.FileSystemArn),
						"name":             llx.StringDataPtr(fs.Name),
						"encrypted":        llx.BoolData(convert.ToValue(fs.Encrypted)),
						"region":           llx.StringData(region),
						"availabilityZone": llx.StringDataPtr(fs.AvailabilityZoneName),
						"createdAt":        llx.TimeDataPtr(fs.CreationTime),
						"tags":             llx.MapData(efsTagsToMap(fs.Tags), types.String),
					}
					mqlFilesystem, err := CreateResource(a.MqlRuntime, "aws.efs.filesystem", args)
					if err != nil {
						return nil, err
					}
					mqlFilesystem.(*mqlAwsEfsFilesystem).cacheKmsKeyID = fs.KmsKeyId

					res = append(res, mqlFilesystem)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsEfsFilesystemInternal struct {
	cacheKmsKeyID *string
}

func (a *mqlAwsEfsFilesystem) kmsKey() (*mqlAwsKmsKey, error) {
	// add kms key if there is one
	if a.cacheKmsKeyID != nil {
		mqlKeyResource, err := NewResource(a.MqlRuntime, "aws.kms.key", map[string]*llx.RawData{
			"arn": llx.StringDataPtr(a.cacheKmsKeyID),
		})
		return mqlKeyResource.(*mqlAwsKmsKey), err
	}
	a.KmsKey.State = plugin.StateIsSet | plugin.StateIsNull

	return nil, nil
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

	arnVal := args["arn"].Value.(string)
	for _, rawResource := range rawResources.Data {
		fs := rawResource.(*mqlAwsEfsFilesystem)
		if fs.Arn.Data == arnVal {
			return args, fs, nil
		}
	}
	return nil, nil, errors.New("efs filesystem does not exist")
}

func (a *mqlAwsEfsFilesystem) backupPolicy() (any, error) {
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

func efsTagsToMap(tags []efstypes.Tag) map[string]any {
	tagsMap := make(map[string]any)

	if len(tags) > 0 {
		for i := range tags {
			tag := tags[i]
			tagsMap[convert.ToValue(tag.Key)] = convert.ToValue(tag.Value)
		}
	}

	return tagsMap
}
