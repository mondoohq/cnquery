package resources

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/providers/aws/connection"
	"go.mondoo.com/cnquery/providers/aws/resources/jobpool"
)

func (m *mqlAwsKms) id() (string, error) {
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
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := conn.Kms(regionVal)
			ctx := context.Background()
			res := []interface{}{}
			var marker *string
			for {
				keyList, err := svc.ListKeys(ctx, &kms.ListKeysInput{Marker: marker})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, key := range keyList.Keys {
					mqlRecorder, err := a.MqlRuntime.CreateResource(a.MqlRuntime, "aws.kms.key",
						map[string]*llx.RawData{
							"id":     llx.StringData(toString(key.KeyId)),
							"arn":    llx.StringData(toString(key.KeyArn)),
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
	r := args["arn"]
	if r == nil {
		return nil, nil, errors.New("arn required to fetch aws kms key")
	}
	arn, ok := r.Value.(string)
	if !ok {
		return args, nil, nil
	}

	obj, err := runtime.CreateResource(runtime, "aws.kms", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	kms := obj.(*mqlAwsKms)

	rawResources, err := kms.keys()
	if err != nil {
		return nil, nil, err
	}

	for i := range rawResources {
		key := rawResources[i].(*mqlAwsKmsKey)
		if key.Arn.Data == arn {
			return args, key, nil
		}
	}
	return args, nil, nil
}
