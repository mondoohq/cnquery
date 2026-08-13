// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/functions"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOciFunctions) id() (string, error) {
	return "oci.functions", nil
}

func (o *mqlOciFunctions) applications() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	return ociCollect(o.MqlRuntime, ociScopeTenancyRoot,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			log.Debug().Msgf("calling oci functions with region %s", region)

			svc, err := conn.FunctionsManagementClient(region)
			if err != nil {
				return nil, err
			}

			items, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]functions.ApplicationSummary, *string, error) {
				response, err := svc.ListApplications(ctx, functions.ListApplicationsRequest{
					CompartmentId: common.String(compartmentID),
					Page:          page,
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

				mqlInstance, err := createOciResourceInCompartment(o.MqlRuntime, "oci.functions.application", stringValue(app.CompartmentId), map[string]*llx.RawData{
					"id":                llx.StringDataPtr(app.Id),
					"name":              llx.StringDataPtr(app.DisplayName),
					"state":             llx.StringData(string(app.LifecycleState)),
					"shape":             llx.StringData(string(app.Shape)),
					"traceConfig":       llx.DictData(traceConfig),
					"imagePolicyConfig": llx.DictData(imagePolicyConfig),
					"created":           llx.TimeDataPtr(created),
					"timeUpdated":       llx.TimeDataPtr(timeUpdated),
					"freeformTags":      llx.MapData(strMapToAny(app.FreeformTags), types.String),
					"definedTags":       llx.MapData(definedTagsToAny(app.DefinedTags), types.Any),
				})
				if err != nil {
					return nil, err
				}
				mqlApp := mqlInstance.(*mqlOciFunctionsApplication)
				mqlApp.cacheRegion = region
				mqlApp.cacheSubnetIDs = app.SubnetIds
				mqlApp.cacheNsgIDs = app.NetworkSecurityGroupIds
				res = append(res, mqlApp)
			}

			return res, nil
		})
}

type mqlOciFunctionsApplicationInternal struct {
	ociCompartmentRef
	app            ociRetryLazy[*functions.Application]
	cacheRegion    string
	cacheSubnetIDs []string
	cacheNsgIDs    []string
}

func (o *mqlOciFunctionsApplication) id() (string, error) {
	return "oci.functions.application/" + o.Id.Data, nil
}

func (o *mqlOciFunctionsApplication) fetchApplication() (*functions.Application, error) {
	return o.app.get(func() (*functions.Application, error) {
		conn := o.MqlRuntime.Connection.(*connection.OciConnection)

		svc, err := conn.FunctionsManagementClient(o.cacheRegion)
		if err != nil {
			return nil, err
		}

		resp, err := svc.GetApplication(context.Background(), functions.GetApplicationRequest{
			ApplicationId: common.String(o.Id.Data),
		})
		if err != nil {
			return nil, err
		}
		return &resp.Application, nil
	})
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
	res := make([]any, 0, len(o.cacheSubnetIDs))
	for _, id := range o.cacheSubnetIDs {
		mqlSubnet, err := NewResource(o.MqlRuntime, "oci.network.subnet", map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
		if err != nil {
			// Skip an element we cannot resolve rather than failing the
			// whole list and losing the ones that did resolve.
			log.Debug().Err(err).Str("subnet", id).Msg("skipping unresolvable oci reference")
			continue
		}
		res = append(res, mqlSubnet)
	}
	return res, nil
}

func (o *mqlOciFunctionsApplication) networkSecurityGroups() ([]any, error) {
	res := make([]any, 0, len(o.cacheNsgIDs))
	for _, id := range o.cacheNsgIDs {
		mqlNsg, err := NewResource(o.MqlRuntime, "oci.network.networkSecurityGroup", map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
		if err != nil {
			// Skip an element we cannot resolve rather than failing the
			// whole list and losing the ones that did resolve.
			log.Debug().Err(err).Str("nsg", id).Msg("skipping unresolvable oci reference")
			continue
		}
		res = append(res, mqlNsg)
	}
	return res, nil
}

func (o *mqlOciFunctionsApplication) functions() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	ctx := context.Background()

	svc, err := conn.FunctionsManagementClient(o.cacheRegion)
	if err != nil {
		return nil, err
	}

	items, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]functions.FunctionSummary, *string, error) {
		response, err := svc.ListFunctions(ctx, functions.ListFunctionsRequest{
			ApplicationId: common.String(o.Id.Data),
			Page:          page,
		})
		if err != nil {
			return nil, nil, err
		}
		return response.Items, response.OpcNextPage, nil
	})
	if err != nil {
		return nil, err
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

		mqlInstance, err := createOciResourceInCompartment(o.MqlRuntime, "oci.functions.function", stringValue(fn.CompartmentId), map[string]*llx.RawData{
			"id":               llx.StringDataPtr(fn.Id),
			"name":             llx.StringDataPtr(fn.DisplayName),
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
			"freeformTags":     llx.MapData(strMapToAny(fn.FreeformTags), types.String),
			"definedTags":      llx.MapData(definedTagsToAny(fn.DefinedTags), types.Any),
		})
		if err != nil {
			return nil, err
		}
		mqlFn := mqlInstance.(*mqlOciFunctionsFunction)
		mqlFn.cacheRegion = o.cacheRegion
		res = append(res, mqlFn)
	}

	return res, nil
}

type mqlOciFunctionsFunctionInternal struct {
	ociCompartmentRef
	fn          ociRetryLazy[*functions.Function]
	cacheRegion string
}

func (o *mqlOciFunctionsFunction) id() (string, error) {
	return "oci.functions.function/" + o.Id.Data, nil
}

func (o *mqlOciFunctionsFunction) fetchFunction() (*functions.Function, error) {
	return o.fn.get(func() (*functions.Function, error) {
		conn := o.MqlRuntime.Connection.(*connection.OciConnection)

		svc, err := conn.FunctionsManagementClient(o.cacheRegion)
		if err != nil {
			return nil, err
		}

		resp, err := svc.GetFunction(context.Background(), functions.GetFunctionRequest{
			FunctionId: common.String(o.Id.Data),
		})
		if err != nil {
			return nil, err
		}
		return &resp.Function, nil
	})
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
