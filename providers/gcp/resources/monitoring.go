// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"

	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"

	kms "cloud.google.com/go/kms/apiv1"
	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	monitoringpb "cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"go.mondoo.com/mql/v13/llx"
	"google.golang.org/api/iterator"
	monitoringv3 "google.golang.org/api/monitoring/v3"
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

	conn, ok := runtime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not a GCP connection")
	}
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

func (g *mqlGcpProjectMonitoringService) alertPolicies() ([]any, error) {
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

	var res []any
	it := c.ListAlertPolicies(ctx, &monitoringpb.ListAlertPoliciesRequest{Name: fmt.Sprintf("projects/%s", projectId)})
	for {
		p, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var mqlDoc any
		if p.Documentation != nil {
			mqlDoc = map[string]any{
				"content":  p.Documentation.Content,
				"mimeType": p.Documentation.MimeType,
			}
		}

		mqlConditions := make([]any, 0, len(p.Conditions))
		for _, c := range p.Conditions {
			var mqlThreshold any
			if thresh := c.GetConditionThreshold(); thresh != nil {
				mqlThreshold = map[string]any{
					"filter":                thresh.Filter,
					"denominatorFilter":     thresh.DenominatorFilter,
					"comparison":            thresh.Comparison.String(),
					"thresholdValue":        thresh.ThresholdValue,
					"duration":              thresh.Duration.Seconds,
					"evaluationMissingData": thresh.EvaluationMissingData.String(),
				}
			}

			var mqlAbsent any
			if absent := c.GetConditionAbsent(); absent != nil {
				mqlAbsent = map[string]any{
					"filter":   absent.Filter,
					"duration": llx.DurationToTime(absent.Duration.Seconds),
				}
			}

			var mqlMatchedLog any
			if matchedLog := c.GetConditionMatchedLog(); matchedLog != nil {
				mqlMatchedLog = map[string]any{
					"filter":          matchedLog.Filter,
					"labelExtractors": matchedLog.LabelExtractors,
				}
			}

			var mqlMonitoringQueryLanguage any
			if monitoringQLanguage := c.GetConditionMonitoringQueryLanguage(); monitoringQLanguage != nil {
				mqlMonitoringQueryLanguage = map[string]any{
					"query":                 monitoringQLanguage.Query,
					"duration":              int64(monitoringQLanguage.Duration.Seconds),
					"evaluationMissingData": monitoringQLanguage.EvaluationMissingData.String(),
				}
			}

			mqlConditions = append(mqlConditions, map[string]any{
				"name":                    c.Name,
				"displayName":             c.DisplayName,
				"threshold":               mqlThreshold,
				"absent":                  mqlAbsent,
				"matchedLog":              mqlMatchedLog,
				"monitoringQueryLanguage": mqlMonitoringQueryLanguage,
			})
		}

		var mqlValidity any
		if p.Validity != nil {
			mqlValidity = map[string]any{
				"code":    p.Validity.Code,
				"message": p.Validity.Message,
			}
		}

		var mqlAlertStrategy any
		if p.AlertStrategy != nil {
			var mqlNotifRateLimit any
			if p.AlertStrategy.NotificationRateLimit != nil {
				mqlNotifRateLimit = map[string]any{
					"period": llx.TimeData(llx.DurationToTime(p.AlertStrategy.NotificationRateLimit.Period.Seconds)),
				}
			}
			mqlAlertStrategy = map[string]any{
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

// ---------------------------------------------------------------
// Uptime Check Configs
// ---------------------------------------------------------------

func (g *mqlGcpProjectMonitoringServiceUptimeCheckConfig) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectMonitoringService) uptimeCheckConfigs() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	// Use REST API (not gRPC) because the gRPC proto doesn't expose the
	// Disabled field, which is only available via the REST API.
	client, err := conn.Client(monitoringv3.MonitoringReadScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	monitoringSvc, err := monitoringv3.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	var res []any
	req := monitoringSvc.Projects.UptimeCheckConfigs.List(fmt.Sprintf("projects/%s", projectId))
	if err := req.Pages(ctx, func(page *monitoringv3.ListUptimeCheckConfigsResponse) error {
		for _, cfg := range page.UptimeCheckConfigs {
			httpCheck, err := convert.JsonToDict(cfg.HttpCheck)
			if err != nil {
				return err
			}
			tcpCheck, err := convert.JsonToDict(cfg.TcpCheck)
			if err != nil {
				return err
			}
			monitoredResource, err := convert.JsonToDict(cfg.MonitoredResource)
			if err != nil {
				return err
			}
			resourceGroup, err := convert.JsonToDict(cfg.ResourceGroup)
			if err != nil {
				return err
			}

			contentMatchers := make([]any, 0, len(cfg.ContentMatchers))
			for _, cm := range cfg.ContentMatchers {
				d, err := convert.JsonToDict(cm)
				if err != nil {
					return err
				}
				contentMatchers = append(contentMatchers, d)
			}

			selectedRegions := convert.SliceAnyToInterface(cfg.SelectedRegions)

			var userLabels map[string]any
			if cfg.UserLabels != nil {
				userLabels = convert.MapToInterfaceMap(cfg.UserLabels)
			}

			mqlCfg, err := CreateResource(g.MqlRuntime, "gcp.project.monitoringService.uptimeCheckConfig", map[string]*llx.RawData{
				"name":              llx.StringData(cfg.Name),
				"displayName":       llx.StringData(cfg.DisplayName),
				"disabled":          llx.BoolData(cfg.Disabled),
				"checkerType":       llx.StringData(cfg.CheckerType),
				"period":            llx.StringData(cfg.Period),
				"timeout":           llx.StringData(cfg.Timeout),
				"selectedRegions":   llx.ArrayData(selectedRegions, types.String),
				"httpCheck":         llx.DictData(httpCheck),
				"tcpCheck":          llx.DictData(tcpCheck),
				"contentMatchers":   llx.ArrayData(contentMatchers, types.Dict),
				"monitoredResource": llx.DictData(monitoredResource),
				"resourceGroup":     llx.DictData(resourceGroup),
				"userLabels":        llx.MapData(userLabels, types.String),
			})
			if err != nil {
				return err
			}
			res = append(res, mqlCfg)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return res, nil
}

// ---------------------------------------------------------------
// Notification Channels
// ---------------------------------------------------------------

func (g *mqlGcpProjectMonitoringServiceNotificationChannel) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectMonitoringService) notificationChannels() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(monitoring.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	c, err := monitoring.NewNotificationChannelClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer c.Close()

	var res []any
	it := c.ListNotificationChannels(ctx, &monitoringpb.ListNotificationChannelsRequest{
		Name: fmt.Sprintf("projects/%s", projectId),
	})
	for {
		ch, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		mqlCh, err := CreateResource(g.MqlRuntime, "gcp.project.monitoringService.notificationChannel", map[string]*llx.RawData{
			"name":               llx.StringData(ch.Name),
			"displayName":        llx.StringData(ch.DisplayName),
			"description":        llx.StringData(ch.Description),
			"type":               llx.StringData(ch.Type),
			"enabled":            llx.BoolData(ch.Enabled.GetValue()),
			"labels":             llx.MapData(convert.MapToInterfaceMap(ch.Labels), types.String),
			"userLabels":         llx.MapData(convert.MapToInterfaceMap(ch.UserLabels), types.String),
			"verificationStatus": llx.StringData(ch.VerificationStatus.String()),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlCh)
	}
	return res, nil
}

// ---------------------------------------------------------------
// Groups
// ---------------------------------------------------------------

func (g *mqlGcpProjectMonitoringServiceGroup) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectMonitoringService) groups() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(monitoring.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	c, err := monitoring.NewGroupClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer c.Close()

	var res []any
	it := c.ListGroups(ctx, &monitoringpb.ListGroupsRequest{
		Name: fmt.Sprintf("projects/%s", projectId),
	})
	for {
		grp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		mqlGrp, err := CreateResource(g.MqlRuntime, "gcp.project.monitoringService.group", map[string]*llx.RawData{
			"name":        llx.StringData(grp.Name),
			"displayName": llx.StringData(grp.DisplayName),
			"parentName":  llx.StringData(grp.ParentName),
			"filter":      llx.StringData(grp.Filter),
			"isCluster":   llx.BoolData(grp.IsCluster),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlGrp)
	}
	return res, nil
}
