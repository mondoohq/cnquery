// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/functions"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOciFunctions) id() (string, error) {
	return "oci.functions", nil
}

func (o *mqlOciFunctions) applications() ([]any, error) {
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

	return ociRunRegionPool(o.getApplications(conn, list.Data))
}

func (o *mqlOciFunctions) getApplications(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci functions with region %s", regionResource.Id.Data)

			svc, err := conn.FunctionsManagementClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			var items []functions.ApplicationSummary
			var page *string
			for {
				response, err := svc.ListApplications(ctx, functions.ListApplicationsRequest{
					CompartmentId: common.String(conn.TenantID()),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}

				items = append(items, response.Items...)

				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			var res []any
			for i := range items {
				app := items[i]

				var created *time.Time
				if app.TimeCreated != nil {
					created = &app.TimeCreated.Time
				}
				var timeUpdated *time.Time
				if app.TimeUpdated != nil {
					timeUpdated = &app.TimeUpdated.Time
				}

				traceConfig, err := convert.JsonToDict(app.TraceConfig)
				if err != nil {
					return nil, err
				}

				imagePolicyConfig, err := convert.JsonToDict(app.ImagePolicyConfig)
				if err != nil {
					return nil, err
				}

				freeformTags := make(map[string]interface{}, len(app.FreeformTags))
				for k, v := range app.FreeformTags {
					freeformTags[k] = v
				}
				definedTags := make(map[string]interface{}, len(app.DefinedTags))
				for k, v := range app.DefinedTags {
					definedTags[k] = v
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.functions.application", map[string]*llx.RawData{
					"id":                llx.StringDataPtr(app.Id),
					"name":              llx.StringDataPtr(app.DisplayName),
					"compartmentID":     llx.StringDataPtr(app.CompartmentId),
					"state":             llx.StringData(string(app.LifecycleState)),
					"shape":             llx.StringData(string(app.Shape)),
					"traceConfig":       llx.DictData(traceConfig),
					"imagePolicyConfig": llx.DictData(imagePolicyConfig),
					"created":           llx.TimeDataPtr(created),
					"timeUpdated":       llx.TimeDataPtr(timeUpdated),
					"freeformTags":      llx.MapData(freeformTags, types.String),
					"definedTags":       llx.MapData(definedTags, types.Any),
				})
				if err != nil {
					return nil, err
				}
				mqlApp := mqlInstance.(*mqlOciFunctionsApplication)
				mqlApp.region = regionResource.Id.Data
				mqlApp.cacheSubnetIds = app.SubnetIds
				mqlApp.cacheNsgIds = app.NetworkSecurityGroupIds
				res = append(res, mqlApp)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlOciFunctionsApplicationInternal struct {
	lock           sync.Mutex
	fetched        atomic.Bool
	app            *functions.Application
	region         string
	cacheSubnetIds []string
	cacheNsgIds    []string
}

func (o *mqlOciFunctionsApplication) id() (string, error) {
	return "oci.functions.application/" + o.Id.Data, nil
}

func (o *mqlOciFunctionsApplication) fetchApplication() (*functions.Application, error) {
	if o.fetched.Load() {
		return o.app, nil
	}
	o.lock.Lock()
	defer o.lock.Unlock()
	if o.fetched.Load() {
		return o.app, nil
	}

	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	svc, err := conn.FunctionsManagementClient(o.region)
	if err != nil {
		return nil, err
	}

	resp, err := svc.GetApplication(context.Background(), functions.GetApplicationRequest{
		ApplicationId: common.String(o.Id.Data),
	})
	if err != nil {
		return nil, err
	}

	o.app = &resp.Application
	o.fetched.Store(true)
	return o.app, nil
}

func (o *mqlOciFunctionsApplication) config() (map[string]interface{}, error) {
	app, err := o.fetchApplication()
	if err != nil {
		return nil, err
	}

	config := make(map[string]interface{}, len(app.Config))
	for k, v := range app.Config {
		config[k] = v
	}
	return config, nil
}

func (o *mqlOciFunctionsApplication) syslogUrl() (string, error) {
	app, err := o.fetchApplication()
	if err != nil {
		return "", err
	}
	return stringValue(app.SyslogUrl), nil
}

func (o *mqlOciFunctionsApplication) subnets() ([]any, error) {
	res := make([]any, 0, len(o.cacheSubnetIds))
	for _, id := range o.cacheSubnetIds {
		mqlSubnet, err := NewResource(o.MqlRuntime, "oci.network.subnet", map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlSubnet)
	}
	return res, nil
}

func (o *mqlOciFunctionsApplication) networkSecurityGroups() ([]any, error) {
	res := make([]any, 0, len(o.cacheNsgIds))
	for _, id := range o.cacheNsgIds {
		mqlNsg, err := NewResource(o.MqlRuntime, "oci.network.networkSecurityGroup", map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlNsg)
	}
	return res, nil
}

func (o *mqlOciFunctionsApplication) functions() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	ctx := context.Background()

	svc, err := conn.FunctionsManagementClient(o.region)
	if err != nil {
		return nil, err
	}

	var items []functions.FunctionSummary
	var page *string
	for {
		response, err := svc.ListFunctions(ctx, functions.ListFunctionsRequest{
			ApplicationId: common.String(o.Id.Data),
			Page:          page,
		})
		if err != nil {
			return nil, err
		}

		items = append(items, response.Items...)

		if response.OpcNextPage == nil {
			break
		}
		page = response.OpcNextPage
	}

	res := make([]any, 0, len(items))
	for i := range items {
		fn := items[i]

		var created *time.Time
		if fn.TimeCreated != nil {
			created = &fn.TimeCreated.Time
		}
		var timeUpdated *time.Time
		if fn.TimeUpdated != nil {
			timeUpdated = &fn.TimeUpdated.Time
		}

		traceConfig, err := convert.JsonToDict(fn.TraceConfig)
		if err != nil {
			return nil, err
		}

		freeformTags := make(map[string]interface{}, len(fn.FreeformTags))
		for k, v := range fn.FreeformTags {
			freeformTags[k] = v
		}
		definedTags := make(map[string]interface{}, len(fn.DefinedTags))
		for k, v := range fn.DefinedTags {
			definedTags[k] = v
		}

		mqlInstance, err := CreateResource(o.MqlRuntime, "oci.functions.function", map[string]*llx.RawData{
			"id":               llx.StringDataPtr(fn.Id),
			"name":             llx.StringDataPtr(fn.DisplayName),
			"compartmentID":    llx.StringDataPtr(fn.CompartmentId),
			"applicationId":    llx.StringDataPtr(fn.ApplicationId),
			"state":            llx.StringData(string(fn.LifecycleState)),
			"image":            llx.StringDataPtr(fn.Image),
			"imageDigest":      llx.StringDataPtr(fn.ImageDigest),
			"shape":            llx.StringData(string(fn.Shape)),
			"memoryInMBs":      llx.IntData(int64Value(fn.MemoryInMBs)),
			"timeoutInSeconds": llx.IntData(intValue(fn.TimeoutInSeconds)),
			"invokeEndpoint":   llx.StringDataPtr(fn.InvokeEndpoint),
			"traceConfig":      llx.DictData(traceConfig),
			"created":          llx.TimeDataPtr(created),
			"timeUpdated":      llx.TimeDataPtr(timeUpdated),
			"freeformTags":     llx.MapData(freeformTags, types.String),
			"definedTags":      llx.MapData(definedTags, types.Any),
		})
		if err != nil {
			return nil, err
		}
		mqlFn := mqlInstance.(*mqlOciFunctionsFunction)
		mqlFn.region = o.region
		res = append(res, mqlFn)
	}

	return res, nil
}

type mqlOciFunctionsFunctionInternal struct {
	lock    sync.Mutex
	fetched atomic.Bool
	fn      *functions.Function
	region  string
}

func (o *mqlOciFunctionsFunction) id() (string, error) {
	return "oci.functions.function/" + o.Id.Data, nil
}

func (o *mqlOciFunctionsFunction) fetchFunction() (*functions.Function, error) {
	if o.fetched.Load() {
		return o.fn, nil
	}
	o.lock.Lock()
	defer o.lock.Unlock()
	if o.fetched.Load() {
		return o.fn, nil
	}

	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	svc, err := conn.FunctionsManagementClient(o.region)
	if err != nil {
		return nil, err
	}

	resp, err := svc.GetFunction(context.Background(), functions.GetFunctionRequest{
		FunctionId: common.String(o.Id.Data),
	})
	if err != nil {
		return nil, err
	}

	o.fn = &resp.Function
	o.fetched.Store(true)
	return o.fn, nil
}

func (o *mqlOciFunctionsFunction) config() (map[string]interface{}, error) {
	fn, err := o.fetchFunction()
	if err != nil {
		return nil, err
	}

	config := make(map[string]interface{}, len(fn.Config))
	for k, v := range fn.Config {
		config[k] = v
	}
	return config, nil
}
