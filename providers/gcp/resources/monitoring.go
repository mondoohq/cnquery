// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/gcp/connection"
	"go.mondoo.com/cnquery/v11/types"

	kms "cloud.google.com/go/kms/apiv1"
	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	monitoringpb "cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"go.mondoo.com/cnquery/v11/llx"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func (g *mqlGcpProjectMonitoringService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data
	return fmt.Sprintf("%s/gcp.project.monitoringService", projectId), nil
}

func initGcpProjectMonitoringService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.GcpConnection)
	projectId := conn.ResourceID()
	args["projectId"] = llx.StringData(projectId)

	return args, nil, nil
}

func (g *mqlGcpProject) monitoring() (*mqlGcpProjectMonitoringService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	res, err := CreateResource(g.MqlRuntime, "gcp.project.monitoringService", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectMonitoringService), nil
}

func (g *mqlGcpProjectMonitoringServiceAlertPolicy) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectMonitoringService) alertPolicies() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	creds, err := conn.Credentials(kms.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	c, err := monitoring.NewAlertPolicyClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer c.Close()

	var res []interface{}
	it := c.ListAlertPolicies(ctx, &monitoringpb.ListAlertPoliciesRequest{Name: fmt.Sprintf("projects/%s", projectId)})
	for {
		p, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var mqlDoc interface{}
		if p.Documentation != nil {
			mqlDoc = map[string]interface{}{
				"content":  p.Documentation.Content,
				"mimeType": p.Documentation.MimeType,
			}
		}

		mqlConditions := make([]interface{}, 0, len(p.Conditions))
		for _, c := range p.Conditions {
			var mqlThreshold interface{}
			if thresh := c.GetConditionThreshold(); thresh != nil {
				mqlThreshold = map[string]interface{}{
					"filter":                thresh.Filter,
					"denominatorFilter":     thresh.DenominatorFilter,
					"comparison":            thresh.Comparison.String(),
					"thresholdValue":        thresh.ThresholdValue,
					"duration":              thresh.Duration.Seconds,
					"evaluationMissingData": thresh.EvaluationMissingData.String(),
				}
			}

			var mqlAbsent interface{}
			if absent := c.GetConditionAbsent(); absent != nil {
				mqlAbsent = map[string]interface{}{
					"filter":   absent.Filter,
					"duration": llx.DurationToTime(absent.Duration.Seconds),
				}
			}

			var mqlMatchedLog interface{}
			if matchedLog := c.GetConditionMatchedLog(); matchedLog != nil {
				mqlMatchedLog = map[string]interface{}{
					"filter":          matchedLog.Filter,
					"labelExtractors": matchedLog.LabelExtractors,
				}
			}

			var mqlMonitoringQueryLanguage interface{}
			if monitoringQLanguage := c.GetConditionMonitoringQueryLanguage(); monitoringQLanguage != nil {
				mqlMonitoringQueryLanguage = map[string]interface{}{
					"query":                 monitoringQLanguage.Query,
					"duration":              int64(monitoringQLanguage.Duration.Seconds),
					"evaluationMissingData": monitoringQLanguage.EvaluationMissingData.String(),
				}
			}

			mqlConditions = append(mqlConditions, map[string]interface{}{
				"name":                    c.Name,
				"displayName":             c.DisplayName,
				"threshold":               mqlThreshold,
				"absent":                  mqlAbsent,
				"matchedLog":              mqlMatchedLog,
				"monitoringQueryLanguage": mqlMonitoringQueryLanguage,
			})
		}

		var mqlValidity interface{}
		if p.Validity != nil {
			mqlValidity = map[string]interface{}{
				"code":    p.Validity.Code,
				"message": p.Validity.Message,
			}
		}

		var mqlAlertStrategy interface{}
		if p.AlertStrategy != nil {
			var mqlNotifRateLimit interface{}
			if p.AlertStrategy.NotificationRateLimit != nil {
				mqlNotifRateLimit = map[string]interface{}{
					"period": llx.TimeData(llx.DurationToTime(p.AlertStrategy.NotificationRateLimit.Period.Seconds)),
				}
			}
			mqlAlertStrategy = map[string]interface{}{
				"notificationRateLimit": mqlNotifRateLimit,
				"autoClose":             llx.TimeData(llx.DurationToTime(p.AlertStrategy.AutoClose.Seconds)),
			}
		}

		mqlPolicy, err := CreateResource(g.MqlRuntime, "gcp.project.monitoringService.alertPolicy", map[string]*llx.RawData{
			"projectId":               llx.StringData(projectId),
			"name":                    llx.StringData(p.Name),
			"displayName":             llx.StringData(p.DisplayName),
			"documentation":           llx.DictData(mqlDoc),
			"labels":                  llx.MapData(convert.MapToInterfaceMap(p.UserLabels), types.String),
			"conditions":              llx.ArrayData(mqlConditions, types.Dict),
			"combiner":                llx.StringData(p.Combiner.String()),
			"enabled":                 llx.BoolData(p.Enabled.Value),
			"validity":                llx.DictData(mqlValidity),
			"notificationChannelUrls": llx.ArrayData(convert.SliceAnyToInterface(p.NotificationChannels), types.String),
			"created":                 llx.TimeData(p.CreationRecord.MutateTime.AsTime()),
			"createdBy":               llx.StringData(p.CreationRecord.MutatedBy),
			"updated":                 llx.TimeData(p.MutationRecord.MutateTime.AsTime()),
			"updatedBy":               llx.StringData(p.MutationRecord.MutatedBy),
			"alertStrategy":           llx.DictData(mqlAlertStrategy),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlPolicy)
	}
	return res, nil
}
