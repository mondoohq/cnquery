// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v10/providers/aws/connection"
)

func (a *mqlAwsKms) id() (string, error) {
	return "aws.kms", nil
}

func (a *mqlAwsKms) keys() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getKeys(conn), 5)
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

func (a *mqlAwsKms) getKeys(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("kms>getKeys>calling aws with region %s", regionVal)

			svc := conn.Kms(regionVal)
			ctx := context.Background()
			res := []interface{}{}
			var marker *string
			for {
				keyList, err := svc.ListKeys(ctx, &kms.ListKeysInput{Marker: marker})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
					}
					continue
				}

				for _, key := range keyList.Keys {
					mqlRecorder, err := CreateResource(a.MqlRuntime, "aws.kms.key",
						map[string]*llx.RawData{
							"id":     llx.StringDataPtr(key.KeyId),
							"arn":    llx.StringDataPtr(key.KeyArn),
							"region": llx.StringData(regionVal),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlRecorder)
				}
				if !keyList.Truncated {
					break
				}
				marker = keyList.NextMarker
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsKmsKey) metadata() (interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	key := a.Arn.Data

	svc := conn.Kms(a.Region.Data)
	ctx := context.Background()

	keyMetadata, err := svc.DescribeKey(ctx, &kms.DescribeKeyInput{KeyId: &key})
	if err != nil {
		return nil, err
	}
	return convert.JsonToDict(keyMetadata.KeyMetadata)
}

func (a *mqlAwsKmsKey) keyRotationEnabled() (bool, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	keyId := a.Id.Data

	svc := conn.Kms(a.Region.Data)
	ctx := context.Background()

	key, err := svc.GetKeyRotationStatus(ctx, &kms.GetKeyRotationStatusInput{KeyId: &keyId})
	if err != nil {
		return false, err
	}
	return key.KeyRotationEnabled, nil
}

func (a *mqlAwsKmsKey) id() (string, error) {
	return a.Arn.Data, nil
}

func initAwsKmsKey(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	r := args["arn"]
	if r == nil {
		return nil, nil, errors.New("arn required to fetch aws kms key")
	}
	a, ok := r.Value.(string)
	if !ok {
		return nil, nil, errors.New("invalid arn")
	}
	arnVal, err := arn.Parse(a)
	if arnVal.AccountID != runtime.Connection.(*connection.AwsConnection).AccountId() {
		// sometimes there are references to keys in other accounts, like the master account of an org
		return nil, nil, errors.New("no access to key")
	}

	obj, err := CreateResource(runtime, "aws.kms", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	kms := obj.(*mqlAwsKms)

	rawResources := kms.GetKeys()
	if rawResources.Error != nil {
		return nil, nil, rawResources.Error
	}
	for i := range rawResources.Data {
		key := rawResources.Data[i].(*mqlAwsKmsKey)
		if key.Arn.Data == a {
			return args, key, nil
		}
	}
	return nil, nil, errors.New("key not found")
}
