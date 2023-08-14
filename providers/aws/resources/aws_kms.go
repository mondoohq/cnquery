package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/providers/aws/connection"
	"go.mondoo.com/cnquery/providers/aws/resources/jobpool"
)

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

// func (p *mqlAwsKmsKey) init(args *resources.Args) (*resources.Args, AwsKmsKey, error) {
// 	if len(*args) > 2 {
// 		return args, nil, nil
// 	}

// 	if len(*args) == 0 {
// 		if ids := getAssetIdentifier(p.MqlResource().MotorRuntime); ids != nil {
// 			(*args)["name"] = ids.name
// 			(*args)["arn"] = ids.arn
// 		}
// 	}

// 	if (*args)["arn"] == nil {
// 		return nil, nil, errors.New("arn required to fetch aws kms key")
// 	}

// 	// load all keys
// 	obj, err := p.MotorRuntime.CreateResource("aws.kms")
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	aws := obj.(AwsKms)

// 	rawResources, err := aws.Keys()
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	arnVal := (*args)["arn"].(string)
// 	for i := range rawResources {
// 		key := rawResources[i].(AwsKmsKey)
// 		mqlKeyArn, err := key.Arn()
// 		if err != nil {
// 			return nil, nil, errors.New("kms key does not exist")
// 		}
// 		if mqlKeyArn == arnVal {
// 			return args, key, nil
// 		}
// 	}
// 	return nil, nil, errors.New("kms key does not exist")
// }
