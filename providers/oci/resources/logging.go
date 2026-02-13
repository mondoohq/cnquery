// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/logging"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v12/providers/oci/connection"
)

func (o *mqlOciLogging) id() (string, error) {
	return "oci.logging", nil
}

func (o *mqlOciLogging) logGroups() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(o.getLogGroups(conn), 5)
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

func (o *mqlOciLogging) getLogGroups(conn *connection.OciConnection) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.GetRegions(ctx)
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci logging with region %s", *region.RegionKey)

			svc, err := conn.LoggingClient(*region.RegionKey)
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
				})
				if err != nil {
					return nil, err
				}
				// Store the region internally so logs() knows which region to query
				mqlInstance.(*mqlOciLoggingLogGroup).region = *region.RegionKey
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

		isEnabled := false
		if l.IsEnabled != nil {
			isEnabled = *l.IsEnabled
		}

		var retentionDuration int64
		if l.RetentionDuration != nil {
			retentionDuration = int64(*l.RetentionDuration)
		}

		config, err := convertLogConfiguration(l.Configuration)
		if err != nil {
			return nil, err
		}

		mqlInstance, err := CreateResource(o.MqlRuntime, "oci.logging.log", map[string]*llx.RawData{
			"id":                llx.StringDataPtr(l.Id),
			"name":              llx.StringDataPtr(l.DisplayName),
			"logType":           llx.StringData(string(l.LogType)),
			"logGroupId":        llx.StringDataPtr(l.LogGroupId),
			"isEnabled":         llx.BoolData(isEnabled),
			"state":             llx.StringData(string(l.LifecycleState)),
			"retentionDuration": llx.IntData(retentionDuration),
			"configuration":     llx.DictData(config),
		})
		if err != nil {
			return nil, err
		}
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

func (o *mqlOciLoggingLog) id() (string, error) {
	return "oci.logging.log/" + o.Id.Data, nil
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
