// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package gcp

import (
	"context"
	"fmt"

	kms "cloud.google.com/go/kms/apiv1"
	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	monitoringpb "cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func (g *mqlGcpProjectMonitoringService) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/gcp.project.monitoringService", projectId), nil
}

func (g *mqlGcpProjectMonitoringService) init(args *resources.Args) (*resources.Args, GcpProjectMonitoringService, error) {
	if len(*args) > 0 {
		return args, nil, nil
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	projectId := provider.ResourceID()
	(*args)["projectId"] = projectId

	return args, nil, nil
}

func (g *mqlGcpProject) GetMonitoring() (interface{}, error) {
	projectId, err := g.Id()
	if err != nil {
		return nil, err
	}

	return g.MotorRuntime.CreateResource("gcp.project.monitoringService",
		"projectId", projectId,
	)
}

func (g *mqlGcpProjectMonitoringServiceAlertPolicy) id() (string, error) {
	return g.Name()
}

func (g *mqlGcpProjectMonitoringService) GetAlertPolicies() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	creds, err := provider.Credentials(kms.DefaultAuthScopes()...)
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
					"duration":              core.MqlTime(llx.DurationToTime(thresh.Duration.Seconds)),
					"evaluationMissingData": thresh.EvaluationMissingData.String(),
				}
			}

			var mqlAbsent interface{}
			if absent := c.GetConditionAbsent(); absent != nil {
				mqlAbsent = map[string]interface{}{
					"filter":   absent.Filter,
					"duration": core.MqlTime(llx.DurationToTime(absent.Duration.Seconds)),
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
					"duration":              core.MqlTime(llx.DurationToTime(int64(monitoringQLanguage.Duration.Seconds))),
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
					"period": core.MqlTime(llx.DurationToTime(p.AlertStrategy.NotificationRateLimit.Period.Seconds)),
				}
			}
			mqlAlertStrategy = map[string]interface{}{
				"notificationRateLimit": mqlNotifRateLimit,
				"autoClose":             core.MqlTime(llx.DurationToTime(p.AlertStrategy.AutoClose.Seconds)),
			}
		}

		mqlPolicy, err := g.MotorRuntime.CreateResource("gcp.project.monitoringService.alertPolicy",
			"projectId", projectId,
			"name", p.Name,
			"displayName", p.DisplayName,
			"documentation", mqlDoc,
			"labels", core.StrMapToInterface(p.UserLabels),
			"conditions", mqlConditions,
			"combiner", p.Combiner.String(),
			"enabled", p.Enabled.Value,
			"validity", mqlValidity,
			"notificationChannelUrls", core.StrSliceToInterface(p.NotificationChannels),
			"created", core.MqlTime(p.CreationRecord.MutateTime.AsTime()),
			"createdBy", p.CreationRecord.MutatedBy,
			"updated", core.MqlTime(p.MutationRecord.MutateTime.AsTime()),
			"updatedBy", p.MutationRecord.MutatedBy,
			"alertStrategy", mqlAlertStrategy,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlPolicy)
	}
	return res, nil
}
