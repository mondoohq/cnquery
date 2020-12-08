package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
)

func (k *lumiAwsKms) id() (string, error) {
	return "aws.kms", nil
}

func (k *lumiAwsKms) GetKeys() ([]interface{}, error) {
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(k.getKeys(), 5)
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

func (k *lumiAwsKms) getKeys() []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
	at, err := awstransport(k.Runtime.Motor.Transport)
	if err != nil {
		return []*jobpool.Job{&jobpool.Job{Err: err}}
	}
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{&jobpool.Job{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := at.Kms(regionVal)
			ctx := context.Background()
			res := []interface{}{}
			var marker *string
			for {
				keyList, err := svc.ListKeysRequest(&kms.ListKeysInput{Marker: marker}).Send(ctx)
				if err != nil {
					return nil, err
				}

				for _, key := range keyList.Keys {
					// need to call key rotation status api with key id to get status of key
					status, err := k.getKeyStatus(ctx, svc, key.KeyId)
					if err != nil {
						return nil, err
					}
					lumiRecorder, err := k.Runtime.CreateResource("aws.kms.key",
						"id", toString(key.KeyId),
						"arn", toString(key.KeyArn),
						"region", regionVal,
						"keyRotationEnabled", toBool(status),
					)
					if err != nil {
						return nil, err
					}
					res = append(res, lumiRecorder)
				}
				if keyList.Truncated == nil || *keyList.Truncated == false {
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

func (k *lumiAwsKms) getKeyStatus(ctx context.Context, svc *kms.Client, keyID *string) (*bool, error) {
	params := &kms.GetKeyRotationStatusInput{KeyId: keyID}
	key, err := svc.GetKeyRotationStatusRequest(params).Send(ctx)
	if err != nil {
		return nil, err
	}
	return key.KeyRotationEnabled, nil
}

func (k *lumiAwsKmsKey) id() (string, error) {
	return k.Arn()
}
