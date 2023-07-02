package aws

import (
	"context"
	"fmt"
	"time"

	"errors"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (f *mqlAwsCloudfront) id() (string, error) {
	return "aws.cloudfront", nil
}

func (f *mqlAwsCloudfrontDistribution) id() (string, error) {
	return f.Arn()
}

func (f *mqlAwsCloudfrontDistributionOrigin) id() (string, error) {
	account, err := f.Account()
	if err != nil {
		return "", err
	}
	id, err := f.Id()
	if err != nil {
		return "", err
	}
	return account + "/" + id, nil
}

func (f *mqlAwsCloudfront) GetDistributions() ([]interface{}, error) {
	provider, err := awsProvider(f.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	account, err := provider.Account()
	if err != nil {
		return nil, err
	}
	svc := provider.Cloudfront("") // global service
	ctx := context.Background()
	res := []interface{}{}

	var marker *string
	for {
		distributions, err := svc.ListDistributions(ctx, &cloudfront.ListDistributionsInput{Marker: marker})
		if err != nil {
			return nil, errors.Join(err, errors.New("could not gather aws cloudfront distributions"))
		}

		for i := range distributions.DistributionList.Items {
			d := distributions.DistributionList.Items[i]
			origins := []interface{}{}
			if or := d.Origins; or != nil {
				for i := range d.Origins.Items {
					o := d.Origins.Items[i]
					mqlAwsCloudfrontOrigin, err := f.MotorRuntime.CreateResource("aws.cloudfront.distribution.origin",
						"domainName", core.ToString(o.DomainName),
						"id", core.ToString(o.Id),
						"connectionAttempts", core.ToInt64From32(o.ConnectionAttempts),
						"connectionTimeout", core.ToInt64From32(o.ConnectionTimeout),
						"originPath", core.ToString(o.OriginPath),
						"account", account.ID,
					)
					if err != nil {
						return nil, err
					}
					origins = append(origins, mqlAwsCloudfrontOrigin)
				}
			}
			cacheBehaviors := []interface{}{}
			if cb := d.CacheBehaviors; cb != nil {
				cacheBehaviors, err = core.JsonToDictSlice(d.CacheBehaviors.Items)
				if err != nil {
					return nil, err
				}
			}
			defaultCacheBehavior, err := core.JsonToDict(d.DefaultCacheBehavior)
			if err != nil {
				return nil, err
			}
			args := []interface{}{
				"arn", core.ToString(d.ARN),
				"status", core.ToString(d.Status),
				"domainName", core.ToString(d.DomainName),
				"origins", origins,
				"defaultCacheBehavior", defaultCacheBehavior,
				"cacheBehaviors", cacheBehaviors,
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

func (f *mqlAwsCloudfrontFunction) id() (string, error) {
	return f.Arn()
}

func (f *mqlAwsCloudfront) GetFunctions() ([]interface{}, error) {
	provider, err := awsProvider(f.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	svc := provider.Cloudfront("") // global service
	ctx := context.Background()
	res := []interface{}{}
	account, err := provider.Account()
	if err != nil {
		return nil, err
	}
	var marker *string
	for {
		functions, err := svc.ListFunctions(ctx, &cloudfront.ListFunctionsInput{Marker: marker})
		if err != nil {
			return nil, errors.Join(err, errors.New("could not gather aws cloudfront functions"))
		}

		for i := range functions.FunctionList.Items {
			funct := functions.FunctionList.Items[i]
			var stage, comment, runtime string
			var lmTime, crTime *time.Time
			if metadata := funct.FunctionMetadata; metadata != nil {
				lmTime = metadata.LastModifiedTime
				crTime = metadata.CreatedTime
				stage = string(metadata.Stage)
			}
			if config := funct.FunctionConfig; config != nil {
				comment = core.ToString(config.Comment)
				runtime = string(config.Runtime)
			}

			args := []interface{}{
				"name", core.ToString(funct.Name),
				"status", core.ToString(funct.Status),
				"lastModifiedTime", lmTime,
				"createdTime", crTime,
				"stage", stage,
				"comment", comment,
				"runtime", runtime,
				"arn", fmt.Sprintf(cloudfrontFunctionPattern, "global", account.ID, core.ToString(funct.Name)),
			}

			mqlAwsCloudfrontDist, err := f.MotorRuntime.CreateResource("aws.cloudfront.function", args...)
			if err != nil {
				return nil, err
			}

			res = append(res, mqlAwsCloudfrontDist)
		}
		if functions.FunctionList.NextMarker == nil {
			break
		}
		marker = functions.FunctionList.NextMarker
	}

	return res, nil
}

const cloudfrontFunctionPattern = "arn:aws:cloudfront:%s:%s::/functions/%s"
