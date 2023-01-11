package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (f *mqlAwsCloudfront) id() (string, error) {
	return "aws.cloudfront", nil
}

func (f *mqlAwsCloudfrontDistribution) id() (string, error) {
	return f.Arn()
}

func (f *mqlAwsCloudfront) GetDistributions() ([]interface{}, error) {
	provider, err := awsProvider(f.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	// global service
	svc := provider.Cloudfront("")
	ctx := context.Background()
	res := []interface{}{}

	var marker *string
	for {
		distributions, err := svc.ListDistributions(ctx, &cloudfront.ListDistributionsInput{Marker: marker})
		if err != nil {
			return nil, errors.Wrap(err, "could not gather aws cloudfront distributions")
		}

		for i := range distributions.DistributionList.Items {
			d := distributions.DistributionList.Items[i]
			origins, err := core.JsonToDict(d.Origins)
			if err != nil {
				return nil, err
			}
			cacheBehavior, err := core.JsonToDict(d.CacheBehaviors)
			if err != nil {
				return nil, err
			}
			args := []interface{}{
				"arn", core.ToString(d.ARN),
				"status", core.ToString(d.Status),
				"domainName", core.ToString(d.DomainName),
				"origins", origins,
				"defaultCacheBehavior", cacheBehavior,
			}

			mqlAwsCloudfrontDist, err := f.MotorRuntime.CreateResource("aws.cloudfront.distribution", args...)
			if err != nil {
				return nil, err
			}

			res = append(res, mqlAwsCloudfrontDist)
		}
		if distributions.DistributionList.NextMarker == nil {
			break
		}
		marker = distributions.DistributionList.NextMarker
	}

	return res, nil
}
