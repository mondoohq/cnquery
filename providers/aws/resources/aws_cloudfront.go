// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/cockroachdb/errors"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/aws/connection"
	"go.mondoo.com/mql/v13/types"
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

func (a *mqlAwsCloudfront) distributions() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Cloudfront("") // global service
	ctx := context.Background()
	res := []any{}

	params := &cloudfront.ListDistributionsInput{}
	paginator := cloudfront.NewListDistributionsPaginator(svc, params)
	for paginator.HasMorePages() {
		distributions, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not gather aws cloudfront distributions")
		}

		for _, distribution := range distributions.DistributionList.Items {
			origins := []any{}
			if or := distribution.Origins; or != nil {
				for _, origin := range distribution.Origins.Items {
					mqlAwsCloudfrontOrigin, err := CreateResource(a.MqlRuntime, "aws.cloudfront.distribution.origin",
						map[string]*llx.RawData{
							"domainName":         llx.StringDataPtr(origin.DomainName),
							"id":                 llx.StringDataPtr(origin.Id),
							"connectionAttempts": llx.IntDataDefault(origin.ConnectionAttempts, 0),
							"connectionTimeout":  llx.IntDataDefault(origin.ConnectionTimeout, 0),
							"originPath":         llx.StringDataPtr(origin.OriginPath),
							"account":            llx.StringData(conn.AccountId()),
						})
					if err != nil {
						return nil, err
					}
					origins = append(origins, mqlAwsCloudfrontOrigin)
				}
			}
			cacheBehaviors := []any{}
			if cb := distribution.CacheBehaviors; cb != nil {
				cacheBehaviors, err = convert.JsonToDictSlice(distribution.CacheBehaviors.Items)
				if err != nil {
					return nil, err
				}
			}
			defaultCacheBehavior, err := convert.JsonToDict(distribution.DefaultCacheBehavior)
			if err != nil {
				return nil, err
			}

			cnames := []any{}
			for _, alias := range distribution.Aliases.Items {
				cnames = append(cnames, alias)
			}

			var viewerProtocolPolicy string
			if distribution.DefaultCacheBehavior != nil {
				viewerProtocolPolicy = string(distribution.DefaultCacheBehavior.ViewerProtocolPolicy)
			}
			var minimumProtocolVersion string
			if distribution.ViewerCertificate != nil {
				minimumProtocolVersion = string(distribution.ViewerCertificate.MinimumProtocolVersion)
			}
			var geoRestrictionType string
			if distribution.Restrictions != nil && distribution.Restrictions.GeoRestriction != nil {
				geoRestrictionType = string(distribution.Restrictions.GeoRestriction.RestrictionType)
			}

			args := map[string]*llx.RawData{
				"arn":                    llx.StringDataPtr(distribution.ARN),
				"cacheBehaviors":         llx.ArrayData(cacheBehaviors, types.Any),
				"cnames":                 llx.ArrayData(cnames, types.String),
				"defaultCacheBehavior":   llx.MapData(defaultCacheBehavior, types.Any),
				"domainName":             llx.StringDataPtr(distribution.DomainName),
				"enabled":                llx.BoolDataPtr(distribution.Enabled),
				"httpVersion":            llx.StringData(string(distribution.HttpVersion)),
				"isIPV6Enabled":          llx.BoolDataPtr(distribution.IsIPV6Enabled),
				"origins":                llx.ArrayData(origins, types.Resource("aws.cloudfront.distribution.origin")),
				"priceClass":             llx.StringData(string(distribution.PriceClass)),
				"status":                 llx.StringDataPtr(distribution.Status),
				"viewerProtocolPolicy":   llx.StringData(viewerProtocolPolicy),
				"minimumProtocolVersion": llx.StringData(minimumProtocolVersion),
				"webAclId":               llx.StringDataPtr(distribution.WebACLId),
				"geoRestrictionType":     llx.StringData(geoRestrictionType),
				"lastModifiedAt":         llx.TimeDataPtr(distribution.LastModifiedTime),
			}

			mqlAwsCloudfrontDist, err := CreateResource(a.MqlRuntime, "aws.cloudfront.distribution", args)
			if err != nil {
				return nil, err
			}

			res = append(res, mqlAwsCloudfrontDist)
		}
	}

	return res, nil
}

func (a *mqlAwsCloudfrontFunction) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsCloudfront) functions() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Cloudfront("") // global service
	ctx := context.Background()
	res := []any{}

	// the AWS SDK does not have a paginator for this function
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
				comment = convert.ToValue(config.Comment)
				runtime = string(config.Runtime)
			}

			args := map[string]*llx.RawData{
				"name":             llx.StringDataPtr(funct.Name),
				"status":           llx.StringDataPtr(funct.Status),
				"lastModifiedTime": llx.TimeDataPtr(lmTime),
				"createdTime":      llx.TimeDataPtr(crTime),
				"createdAt":        llx.TimeDataPtr(crTime),
				"stage":            llx.StringData(stage),
				"comment":          llx.StringData(comment),
				"runtime":          llx.StringData(runtime),
				"arn":              llx.StringData(fmt.Sprintf(cloudfrontFunctionPattern, "global", conn.AccountId(), convert.ToValue(funct.Name))),
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
