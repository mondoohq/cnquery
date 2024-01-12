// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/aws/connection"
	"go.mondoo.com/cnquery/v10/types"
)

func (a *mqlAwsCloudfront) id() (string, error) {
	return "aws.cloudfront", nil
}

func (a *mqlAwsCloudfrontDistribution) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsCloudfrontDistributionOrigin) id() (string, error) {
	account := a.Account.Data
	id := a.Id.Data
	return account + "/" + id, nil
}

func (a *mqlAwsCloudfront) distributions() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Cloudfront("") // global service
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
			origins := []interface{}{}
			if or := d.Origins; or != nil {
				for i := range d.Origins.Items {
					o := d.Origins.Items[i]
					mqlAwsCloudfrontOrigin, err := CreateResource(a.MqlRuntime, "aws.cloudfront.distribution.origin",
						map[string]*llx.RawData{
							"domainName":         llx.StringDataPtr(o.DomainName),
							"id":                 llx.StringDataPtr(o.Id),
							"connectionAttempts": llx.IntData(convert.ToInt64From32(o.ConnectionAttempts)),
							"connectionTimeout":  llx.IntData(convert.ToInt64From32(o.ConnectionTimeout)),
							"originPath":         llx.StringDataPtr(o.OriginPath),
							"account":            llx.StringData(conn.AccountId()),
						})
					if err != nil {
						return nil, err
					}
					origins = append(origins, mqlAwsCloudfrontOrigin)
				}
			}
			cacheBehaviors := []interface{}{}
			if cb := d.CacheBehaviors; cb != nil {
				cacheBehaviors, err = convert.JsonToDictSlice(d.CacheBehaviors.Items)
				if err != nil {
					return nil, err
				}
			}
			defaultCacheBehavior, err := convert.JsonToDict(d.DefaultCacheBehavior)
			if err != nil {
				return nil, err
			}
			args := map[string]*llx.RawData{
				"arn":                  llx.StringDataPtr(d.ARN),
				"status":               llx.StringDataPtr(d.Status),
				"domainName":           llx.StringDataPtr(d.DomainName),
				"origins":              llx.ArrayData(origins, types.Resource("aws.cloudfront.distribution.origin")),
				"defaultCacheBehavior": llx.MapData(defaultCacheBehavior, types.Any),
				"cacheBehaviors":       llx.ArrayData(cacheBehaviors, types.Any),
				"httpVersion":          llx.StringData(string(d.HttpVersion)),
				"isIPV6Enabled":        llx.BoolDataPtr(d.IsIPV6Enabled),
				"enabled":              llx.BoolDataPtr(d.Enabled),
				"priceClass":           llx.StringData(string(d.PriceClass)),
			}

			mqlAwsCloudfrontDist, err := CreateResource(a.MqlRuntime, "aws.cloudfront.distribution", args)
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

func (a *mqlAwsCloudfrontFunction) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsCloudfront) functions() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Cloudfront("") // global service
	ctx := context.Background()
	res := []interface{}{}

	var marker *string
	for {
		functions, err := svc.ListFunctions(ctx, &cloudfront.ListFunctionsInput{Marker: marker})
		if err != nil {
			return nil, errors.Wrap(err, "could not gather aws cloudfront functions")
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
				comment = convert.ToString(config.Comment)
				runtime = string(config.Runtime)
			}

			args := map[string]*llx.RawData{
				"name":             llx.StringDataPtr(funct.Name),
				"status":           llx.StringDataPtr(funct.Status),
				"lastModifiedTime": llx.TimeDataPtr(lmTime),
				"createdTime":      llx.TimeDataPtr(crTime),
				"stage":            llx.StringData(stage),
				"comment":          llx.StringData(comment),
				"runtime":          llx.StringData(runtime),
				"arn":              llx.StringData(fmt.Sprintf(cloudfrontFunctionPattern, "global", conn.AccountId(), convert.ToString(funct.Name))),
			}

			mqlAwsCloudfrontDist, err := CreateResource(a.MqlRuntime, "aws.cloudfront.function", args)
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
