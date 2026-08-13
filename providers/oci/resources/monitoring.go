// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/monitoring"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOciMonitoring) id() (string, error) {
	return "oci.monitoring", nil
}

func (o *mqlOciMonitoring) alarms() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	return ociCollect(o.MqlRuntime, ociScopeTenancyRoot,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			log.Debug().Msgf("calling oci monitoring with region %s", region)

			svc, err := conn.MonitoringClient(region)
			if err != nil {
				return nil, err
			}

			alarms, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]monitoring.AlarmSummary, *string, error) {
				response, err := svc.ListAlarms(ctx, monitoring.ListAlarmsRequest{
					CompartmentId: common.String(compartmentID),
					// Alarms are normally created in a workload compartment,
					// not the tenancy root.
					CompartmentIdInSubtree: common.Bool(true),
					Page:                   page,
				})
				if err != nil {
					return nil, nil, err
				}
				return response.Items, response.OpcNextPage, nil
			})
			if err != nil {
				return nil, err
			}

			var res []any
			for i := range alarms {
				alarm := alarms[i]

				destinations := make([]any, 0, len(alarm.Destinations))
				for _, d := range alarm.Destinations {
					destinations = append(destinations, d)
				}

				mqlInstance, err := createOciResourceInCompartment(o.MqlRuntime, "oci.monitoring.alarm", stringValue(alarm.CompartmentId), map[string]*llx.RawData{
					"id":                  llx.StringDataPtr(alarm.Id),
					"name":                llx.StringDataPtr(alarm.DisplayName),
					"metricCompartmentId": llx.StringDataPtr(alarm.MetricCompartmentId),
					"namespace":           llx.StringDataPtr(alarm.Namespace),
					"query":               llx.StringDataPtr(alarm.Query),
					"severity":            llx.StringData(string(alarm.Severity)),
					"isEnabled":           llx.BoolDataPtr(alarm.IsEnabled),
					"state":               llx.StringData(string(alarm.LifecycleState)),
					"freeformTags":        llx.MapData(strMapToAny(alarm.FreeformTags), types.String),
					"definedTags":         llx.MapData(definedTagsToAny(alarm.DefinedTags), types.Any),
				})
				if err != nil {
					return nil, err
				}
				mqlInstance.(*mqlOciMonitoringAlarm).cacheDestinations = destinations
				res = append(res, mqlInstance)
			}

			return res, nil
		})
}

func (o *mqlOciMonitoringAlarm) id() (string, error) {
	return "oci.monitoring.alarm/" + o.Id.Data, nil
}
