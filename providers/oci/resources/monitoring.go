// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/monitoring"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOciMonitoring) id() (string, error) {
	return "oci.monitoring", nil
}

func (o *mqlOciMonitoring) alarms() ([]any, error) {
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
	poolOfJobs := jobpool.CreatePool(o.getAlarms(conn, list.Data), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (o *mqlOciMonitoring) getAlarms(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci monitoring with region %s", regionResource.Id.Data)

			svc, err := conn.MonitoringClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			alarms := []monitoring.AlarmSummary{}
			var page *string
			for {
				response, err := svc.ListAlarms(ctx, monitoring.ListAlarmsRequest{
					CompartmentId: common.String(conn.TenantID()),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}

				alarms = append(alarms, response.Items...)

				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			var res []any
			for i := range alarms {
				alarm := alarms[i]

				destinations := make([]any, 0, len(alarm.Destinations))
				for _, d := range alarm.Destinations {
					destinations = append(destinations, d)
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.monitoring.alarm", map[string]*llx.RawData{
					"id":                  llx.StringDataPtr(alarm.Id),
					"name":                llx.StringDataPtr(alarm.DisplayName),
					"compartmentID":       llx.StringDataPtr(alarm.CompartmentId),
					"metricCompartmentId": llx.StringDataPtr(alarm.MetricCompartmentId),
					"namespace":           llx.StringDataPtr(alarm.Namespace),
					"query":               llx.StringDataPtr(alarm.Query),
					"severity":            llx.StringData(string(alarm.Severity)),
					"destinations":        llx.ArrayData(destinations, types.String),
					"isEnabled":           llx.BoolDataPtr(alarm.IsEnabled),
					"state":               llx.StringData(string(alarm.LifecycleState)),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlInstance)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (o *mqlOciMonitoringAlarm) id() (string, error) {
	return "oci.monitoring.alarm/" + o.Id.Data, nil
}
