package aws

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/rs/zerolog/log"
	aws_provider "go.mondoo.com/cnquery/motor/providers/aws"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/library/jobpool"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (k *mqlAwsKms) id() (string, error) {
	return "aws.kms", nil
}

func (k *mqlAwsKms) GetKeys() ([]interface{}, error) {
	provider, err := awsProvider(k.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(k.getKeys(provider), 5)
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

func (k *mqlAwsKms) getKeys(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := provider.Kms(regionVal)
			ctx := context.Background()
			res := []interface{}{}
			var marker *string
			for {
				keyList, err := svc.ListKeys(ctx, &kms.ListKeysInput{Marker: marker})
				if err != nil {
					return nil, err
				}

				for _, key := range keyList.Keys {
					mqlRecorder, err := k.MotorRuntime.CreateResource("aws.kms.key",
						"id", core.ToString(key.KeyId),
						"arn", core.ToString(key.KeyArn),
						"region", regionVal,
					)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlRecorder)
				}
				if keyList.Truncated == false {
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

func (k *mqlAwsKmsKey) GetMetadata() (interface{}, error) {
	key, err := k.Arn()
	if err != nil {
		return nil, err
	}
	region, err := k.Region()
	if err != nil {
		return nil, err
	}
	provider, err := awsProvider(k.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	svc := provider.Kms(region)
	ctx := context.Background()

	keyMetadata, err := svc.DescribeKey(ctx, &kms.DescribeKeyInput{KeyId: &key})
	if err != nil {
		return nil, err
	}
	return core.JsonToDict(keyMetadata.KeyMetadata)
}

func (k *mqlAwsKmsKey) GetKeyRotationEnabled() (bool, error) {
	keyId, err := k.Id()
	if err != nil {
		return false, err
	}
	region, err := k.Region()
	if err != nil {
		return false, err
	}
	provider, err := awsProvider(k.MotorRuntime.Motor.Provider)
	if err != nil {
		return false, err
	}
	svc := provider.Kms(region)
	ctx := context.Background()

	key, err := svc.GetKeyRotationStatus(ctx, &kms.GetKeyRotationStatusInput{KeyId: &keyId})
	if err != nil {
		return false, err
	}
	return key.KeyRotationEnabled, nil
}

func (k *mqlAwsKmsKey) id() (string, error) {
	return k.Arn()
}

func (p *mqlAwsKmsKey) init(args *resources.Args) (*resources.Args, AwsKmsKey, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if len(*args) == 0 {
		if ids := getAssetIdentifier(p.MqlResource().MotorRuntime); ids != nil {
			(*args)["name"] = ids.name
			(*args)["arn"] = ids.arn
		}
	}

	if (*args)["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch aws kms key")
	}

	// load all keys
	obj, err := p.MotorRuntime.CreateResource("aws.kms")
	if err != nil {
		return nil, nil, err
	}
	aws := obj.(AwsKms)

	rawResources, err := aws.Keys()
	if err != nil {
		return nil, nil, err
	}

	arnVal := (*args)["arn"].(string)
	for i := range rawResources {
		key := rawResources[i].(AwsKmsKey)
		mqlKeyArn, err := key.Arn()
		if err != nil {
			return nil, nil, errors.New("kms key does not exist")
		}
		if mqlKeyArn == arnVal {
			return args, key, nil
		}
	}
	return nil, nil, errors.New("kms key does not exist")
}
