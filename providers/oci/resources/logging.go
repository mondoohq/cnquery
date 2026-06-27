// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/logging"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOciLogging) id() (string, error) {
	return "oci.logging", nil
}

func (o *mqlOciLogging) logGroups() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	ociResource, err := CreateResource(o.MqlRuntime, "oci", nil)
	if err != nil {
		return nil, err
	}
	oci := ociResource.(*mqlOci)
	list := oci.GetRegions()
	if list.Error != nil {
		return nil, list.Error
	}

	res := []any{}
	poolOfJobs := jobpool.CreatePool(o.getLogGroups(conn, list.Data), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (o *mqlOciLogging) getLogGroupsForRegion(ctx context.Context, client *logging.LoggingManagementClient, compartmentID string) ([]logging.LogGroupSummary, error) {
	entries := []logging.LogGroupSummary{}
	var page *string
	for {
		request := logging.ListLogGroupsRequest{
			CompartmentId:            common.String(compartmentID),
			IsCompartmentIdInSubtree: common.Bool(true),
			Page:                     page,
		}

		response, err := client.ListLogGroups(ctx, request)
		if err != nil {
			return nil, err
		}

		entries = append(entries, response.Items...)

		if response.OpcNextPage == nil {
			break
		}
		page = response.OpcNextPage
	}

	return entries, nil
}

func (o *mqlOciLogging) getLogGroups(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci logging with region %s", regionResource.Id.Data)

			svc, err := conn.LoggingClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			var res []any
			logGroups, err := o.getLogGroupsForRegion(ctx, svc, conn.TenantID())
			if err != nil {
				return nil, err
			}

			for i := range logGroups {
				lg := logGroups[i]

				var created *time.Time
				if lg.TimeCreated != nil {
					created = &lg.TimeCreated.Time
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.logging.logGroup", map[string]*llx.RawData{
					"id":            llx.StringDataPtr(lg.Id),
					"name":          llx.StringDataPtr(lg.DisplayName),
					"description":   llx.StringDataPtr(lg.Description),
					"compartmentID": llx.StringDataPtr(lg.CompartmentId),
					"state":         llx.StringData(string(lg.LifecycleState)),
					"created":       llx.TimeDataPtr(created),
					"systemTags":    llx.MapData(definedTagsToAny(lg.SystemTags), types.Dict),
				})
				if err != nil {
					return nil, err
				}
				// Store the region internally so logs() knows which region to query
				mqlInstance.(*mqlOciLoggingLogGroup).region = regionResource.Id.Data
				res = append(res, mqlInstance)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlOciLoggingLogGroupInternal struct {
	region string
}

func initOciLoggingLogGroup(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch oci.logging.logGroup")
	}
	idVal := args["id"].Value.(string)

	obj, err := CreateResource(runtime, "oci.logging", nil)
	if err != nil {
		return nil, nil, err
	}
	l := obj.(*mqlOciLogging)

	rawGroups := l.GetLogGroups()
	if rawGroups.Error != nil {
		return nil, nil, rawGroups.Error
	}

	for _, raw := range rawGroups.Data {
		lg := raw.(*mqlOciLoggingLogGroup)
		if lg.Id.Data == idVal {
			return args, lg, nil
		}
	}

	return nil, nil, errors.New("oci.logging.logGroup not found: " + idVal)
}

func (o *mqlOciLoggingLogGroup) id() (string, error) {
	return "oci.logging.logGroup/" + o.Id.Data, nil
}

func (o *mqlOciLoggingLogGroup) logs() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	ctx := context.Background()

	logGroupId := o.Id.Data

	svc, err := conn.LoggingClient(o.region)
	if err != nil {
		return nil, err
	}

	logs, err := o.getLogsForGroup(ctx, svc, logGroupId)
	if err != nil {
		return nil, err
	}

	res := []any{}
	for i := range logs {
		l := logs[i]

		config, err := convertLogConfiguration(l.Configuration)
		if err != nil {
			return nil, err
		}

		var logCreated, logLastModified *time.Time
		if l.TimeCreated != nil {
			logCreated = &l.TimeCreated.Time
		}
		if l.TimeLastModified != nil {
			logLastModified = &l.TimeLastModified.Time
		}

		category, sourceService, sourceResource := extractLogSource(l.Configuration)

		mqlInstance, err := CreateResource(o.MqlRuntime, "oci.logging.log", map[string]*llx.RawData{
			"id":                llx.StringDataPtr(l.Id),
			"name":              llx.StringDataPtr(l.DisplayName),
			"logType":           llx.StringData(string(l.LogType)),
			"isEnabled":         llx.BoolDataPtr(l.IsEnabled),
			"state":             llx.StringData(string(l.LifecycleState)),
			"retentionDuration": llx.IntDataPtr(l.RetentionDuration),
			"configuration":     llx.DictData(config),
			"category":          llx.StringData(category),
			"sourceService":     llx.StringData(sourceService),
			"sourceResource":    llx.StringData(sourceResource),
			"created":           llx.TimeDataPtr(logCreated),
			"timeLastModified":  llx.TimeDataPtr(logLastModified),
			"systemTags":        llx.MapData(definedTagsToAny(l.SystemTags), types.Dict),
		})
		if err != nil {
			return nil, err
		}
		mqlInstance.(*mqlOciLoggingLog).cacheLogGroupId = stringValue(l.LogGroupId)
		res = append(res, mqlInstance)
	}

	return res, nil
}

func (o *mqlOciLoggingLogGroup) getLogsForGroup(ctx context.Context, client *logging.LoggingManagementClient, logGroupId string) ([]logging.LogSummary, error) {
	entries := []logging.LogSummary{}
	var page *string
	for {
		request := logging.ListLogsRequest{
			LogGroupId: common.String(logGroupId),
			Page:       page,
		}

		response, err := client.ListLogs(ctx, request)
		if err != nil {
			return nil, err
		}

		entries = append(entries, response.Items...)

		if response.OpcNextPage == nil {
			break
		}
		page = response.OpcNextPage
	}

	return entries, nil
}

type mqlOciLoggingLogInternal struct {
	cacheLogGroupId string
}

func (o *mqlOciLoggingLog) id() (string, error) {
	return "oci.logging.log/" + o.Id.Data, nil
}

func (o *mqlOciLoggingLog) logGroup() (*mqlOciLoggingLogGroup, error) {
	if o.cacheLogGroupId == "" {
		o.LogGroup.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlLg, err := NewResource(o.MqlRuntime, "oci.logging.logGroup", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheLogGroupId),
	})
	if err != nil {
		return nil, err
	}
	return mqlLg.(*mqlOciLoggingLogGroup), nil
}

// extractLogSource pulls category, service, and resource out of a logging
// configuration's Source union. Any missing layer (nil configuration, nil
// source, unknown source type) yields empty strings.
func extractLogSource(cfg *logging.Configuration) (category, service, resource string) {
	if cfg == nil || cfg.Source == nil {
		return "", "", ""
	}
	svc, ok := cfg.Source.(logging.OciService)
	if !ok {
		return "", "", ""
	}
	return stringValue(svc.Category), stringValue(svc.Service), stringValue(svc.Resource)
}

func convertLogConfiguration(cfg *logging.Configuration) (map[string]interface{}, error) {
	if cfg == nil {
		return nil, nil
	}

	result := map[string]interface{}{}

	if cfg.CompartmentId != nil {
		result["compartmentId"] = *cfg.CompartmentId
	}

	if cfg.Source != nil {
		source, err := convert.JsonToDict(cfg.Source)
		if err != nil {
			return nil, err
		}
		result["source"] = source
	}

	return result, nil
}
