// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/shield"
	shieldtypes "github.com/aws/aws-sdk-go-v2/service/shield/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/aws/connection"
)

func (a *mqlAwsShield) id() (string, error) {
	return "aws.shield", nil
}

func (a *mqlAwsShield) subscriptionState() (string, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Shield("us-east-1") // Shield is a global service, must use us-east-1
	ctx := context.Background()

	resp, err := svc.GetSubscriptionState(ctx, &shield.GetSubscriptionStateInput{})
	if err != nil {
		if Is400AccessDeniedError(err) {
			log.Warn().Msg("access denied querying Shield subscription state; returning UNKNOWN")
			return "UNKNOWN", nil
		}
		return "", err
	}
	return string(resp.SubscriptionState), nil
}

func (a *mqlAwsShield) subscription() (*mqlAwsShieldSubscription, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Shield("us-east-1")
	ctx := context.Background()

	resp, err := svc.DescribeSubscription(ctx, &shield.DescribeSubscriptionInput{})
	if err != nil {
		if Is400AccessDeniedError(err) {
			a.Subscription.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, nil
		}
		var notFoundErr *shieldtypes.ResourceNotFoundException
		if errors.As(err, &notFoundErr) {
			a.Subscription.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, nil
		}
		return nil, err
	}

	sub := resp.Subscription
	limits, _ := convert.JsonToDictSlice(sub.Limits)

	mqlSub, err := CreateResource(a.MqlRuntime, "aws.shield.subscription",
		map[string]*llx.RawData{
			"arn":                        llx.StringDataPtr(sub.SubscriptionArn),
			"startTime":                  llx.TimeDataPtr(sub.StartTime),
			"endTime":                    llx.TimeDataPtr(sub.EndTime),
			"timeCommitmentInDays":       llx.IntData(sub.TimeCommitmentInSeconds / 86400),
			"autoRenew":                  llx.StringData(string(sub.AutoRenew)),
			"limits":                     llx.ArrayData(limits, "dict"),
			"proactiveEngagementStatus":  llx.StringData(string(sub.ProactiveEngagementStatus)),
		})
	if err != nil {
		return nil, err
	}
	return mqlSub.(*mqlAwsShieldSubscription), nil
}

func (a *mqlAwsShieldSubscription) id() (string, error) {
	return a.Arn.Data, a.Arn.Error
}

func (a *mqlAwsShield) protections() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Shield("us-east-1")
	ctx := context.Background()

	res := []any{}
	paginator := shield.NewListProtectionsPaginator(svc, &shield.ListProtectionsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if Is400AccessDeniedError(err) {
				return res, nil
			}
			var notFoundErr *shieldtypes.ResourceNotFoundException
			if errors.As(err, &notFoundErr) {
				return res, nil
			}
			return nil, err
		}
		for _, p := range page.Protections {
			var appLayerConfig any
			if p.ApplicationLayerAutomaticResponseConfiguration != nil {
				appLayerConfig, _ = convert.JsonToDict(p.ApplicationLayerAutomaticResponseConfiguration)
			}
			mqlProtection, err := CreateResource(a.MqlRuntime, "aws.shield.protection",
				map[string]*llx.RawData{
					"id":           llx.StringDataPtr(p.Id),
					"arn":          llx.StringDataPtr(p.ProtectionArn),
					"name":         llx.StringDataPtr(p.Name),
					"resourceArn":  llx.StringDataPtr(p.ResourceArn),
					"healthCheckIds": llx.ArrayData(llx.TArr2Raw(p.HealthCheckIds), "string"),
					"applicationLayerAutomaticResponseConfiguration": llx.DictData(appLayerConfig),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlProtection)
		}
	}
	return res, nil
}

func (a *mqlAwsShieldProtection) id() (string, error) {
	return a.Arn.Data, a.Arn.Error
}

func (a *mqlAwsShield) protectionGroups() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Shield("us-east-1")
	ctx := context.Background()

	res := []any{}
	paginator := shield.NewListProtectionGroupsPaginator(svc, &shield.ListProtectionGroupsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if Is400AccessDeniedError(err) {
				return res, nil
			}
			var notFoundErr *shieldtypes.ResourceNotFoundException
			if errors.As(err, &notFoundErr) {
				return res, nil
			}
			return nil, err
		}
		for _, pg := range page.ProtectionGroups {
			mqlGroup, err := CreateResource(a.MqlRuntime, "aws.shield.protectionGroup",
				map[string]*llx.RawData{
					"id":           llx.StringDataPtr(pg.ProtectionGroupId),
					"arn":          llx.StringDataPtr(pg.ProtectionGroupArn),
					"aggregation":  llx.StringData(string(pg.Aggregation)),
					"pattern":      llx.StringData(string(pg.Pattern)),
					"resourceType": llx.StringData(string(pg.ResourceType)),
					"members":      llx.ArrayData(llx.TArr2Raw(pg.Members), "string"),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlGroup)
		}
	}
	return res, nil
}

func (a *mqlAwsShieldProtectionGroup) id() (string, error) {
	if a.Arn.Error != nil {
		return "", a.Arn.Error
	}
	if a.Arn.Data != "" {
		return a.Arn.Data, nil
	}
	return "aws.shield.protectionGroup/" + a.Id.Data, a.Id.Error
}
