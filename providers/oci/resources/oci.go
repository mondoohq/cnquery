// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/oracle/oci-go-sdk/v65/audit"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/oci/connection"
)

func (o *mqlOci) id() (string, error) {
	return "oci", nil
}

func (o *mqlOci) regions() ([]interface{}, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	regions, err := conn.GetRegions(context.Background())
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range regions {
		region := regions[i]

		homeRegion := false
		if region.IsHomeRegion != nil {
			homeRegion = *region.IsHomeRegion
		}

		mqlRegion, err := CreateResource(o.MqlRuntime, "oci.region", map[string]*llx.RawData{
			"id":           llx.StringDataPtr(region.RegionKey),
			"name":         llx.StringDataPtr(region.RegionName),
			"isHomeRegion": llx.BoolData(homeRegion),
			"status":       llx.StringData(string(region.Status)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlRegion)
	}

	return res, nil
}

func (o *mqlOciRegion) id() (string, error) {
	return "oci.region/" + o.Id.Data, nil
}

func (o *mqlOci) compartments() ([]interface{}, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	compartments, err := conn.GetCompartments(context.Background())
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range compartments {
		compartment := compartments[i]

		var created *time.Time
		if compartment.TimeCreated != nil {
			created = &compartment.TimeCreated.Time
		}

		mqlCompartment, err := CreateResource(o.MqlRuntime, "oci.compartment", map[string]*llx.RawData{
			"id":          llx.StringDataPtr(compartment.Id),
			"name":        llx.StringDataPtr(compartment.Name),
			"description": llx.StringDataPtr(compartment.Description),
			"created":     llx.TimeDataPtr(created),
			"state":       llx.StringData(string(compartment.LifecycleState)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlCompartment)
	}

	return res, nil
}

func (o *mqlOciCompartment) id() (string, error) {
	return "oci.compartment/" + o.Id.Data, nil
}

func (o *mqlOciTenancy) id() (string, error) {
	return "oci.tenancy", nil
}

func initOciTenancy(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn := runtime.Connection.(*connection.OciConnection)

	tenancy, err := conn.Tenant(context.Background())
	if err != nil {
		return nil, nil, err
	}

	args["id"] = llx.StringDataPtr(tenancy.Id)
	args["name"] = llx.StringDataPtr(tenancy.Name)
	args["description"] = llx.StringDataPtr(tenancy.Description)

	return args, nil, nil
}

func (o *mqlOciTenancy) retentionPeriod() (*time.Time, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	ctx := context.Background()
	tenancy, err := conn.Tenant(ctx)
	if err != nil {
		return nil, err
	}

	if tenancy.HomeRegionKey == nil {
		return nil, errors.New("no home region set")
	}

	client, err := conn.AuditClient(*tenancy.HomeRegionKey)
	if err != nil {
		return nil, err
	}
	response, err := client.GetConfiguration(ctx, audit.GetConfigurationRequest{
		CompartmentId: tenancy.Id,
	})
	if err != nil {
		return nil, err
	}

	// retention period is in days
	if response.Configuration.RetentionPeriodDays == nil {
		return nil, nil
	}

	days := time.Duration(*response.Configuration.RetentionPeriodDays) * 24 * time.Hour

	ts := llx.DurationToTime(int64(days.Seconds()))
	return &ts, nil
}
