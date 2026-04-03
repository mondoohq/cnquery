// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/operationalinsights/armoperationalinsights"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

type mqlAzureSubscriptionMonitorServiceWorkspaceInternal struct {
	cacheCapping                    *armoperationalinsights.WorkspaceCapping
	cacheFeatures                   *armoperationalinsights.WorkspaceFeatures
	cachePrivateLinkScopedResources []*armoperationalinsights.PrivateLinkScopedResource
}

type mqlAzureSubscriptionMonitorServiceApplicationInsightInternal struct {
	cacheWorkspaceResourceId string
}

func (a *mqlAzureSubscriptionMonitorServiceWorkspace) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionMonitorService) workspaces() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := armoperationalinsights.NewWorkspacesClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(nil)
	var res []any
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, ws := range page.Value {
			if ws == nil {
				continue
			}
			mqlWs, err := createWorkspaceResource(a.MqlRuntime, ws)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlWs)
		}
	}
	return res, nil
}

// parseAzureDateString parses an Azure date string to *llx.RawData containing a time value.
func parseAzureDateString(s *string) *llx.RawData {
	if s == nil || *s == "" {
		return llx.NilData
	}
	// Azure returns dates in RFC3339 format
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		// Try RFC3339Nano as fallback
		t, err = time.Parse(time.RFC3339Nano, *s)
		if err != nil {
			log.Warn().Str("value", *s).Msg("failed to parse Azure date string")
			return llx.NilData
		}
	}
	return llx.TimeData(t)
}

func createWorkspaceResource(runtime *plugin.Runtime, ws *armoperationalinsights.Workspace) (*mqlAzureSubscriptionMonitorServiceWorkspace, error) {
	props := ws.Properties
	if props == nil {
		props = &armoperationalinsights.WorkspaceProperties{}
	}

	var skuName string
	var skuCapacityReservationLevel int64
	if props.SKU != nil {
		if props.SKU.Name != nil {
			skuName = string(*props.SKU.Name)
		}
		if props.SKU.CapacityReservationLevel != nil {
			skuCapacityReservationLevel = int64(*props.SKU.CapacityReservationLevel)
		}
	}

	var retentionInDays int64
	if props.RetentionInDays != nil {
		retentionInDays = int64(*props.RetentionInDays)
	}

	var publicNetworkAccessForIngestion, publicNetworkAccessForQuery string
	if props.PublicNetworkAccessForIngestion != nil {
		publicNetworkAccessForIngestion = string(*props.PublicNetworkAccessForIngestion)
	}
	if props.PublicNetworkAccessForQuery != nil {
		publicNetworkAccessForQuery = string(*props.PublicNetworkAccessForQuery)
	}

	var provisioningState string
	if props.ProvisioningState != nil {
		provisioningState = string(*props.ProvisioningState)
	}

	resource, err := CreateResource(runtime, ResourceAzureSubscriptionMonitorServiceWorkspace,
		map[string]*llx.RawData{
			"id":                              llx.StringDataPtr(ws.ID),
			"name":                            llx.StringDataPtr(ws.Name),
			"location":                        llx.StringDataPtr(ws.Location),
			"type":                            llx.StringDataPtr(ws.Type),
			"tags":                            llx.MapData(convert.PtrMapStrToInterface(ws.Tags), types.String),
			"skuName":                         llx.StringData(skuName),
			"skuCapacityReservationLevel":     llx.IntData(skuCapacityReservationLevel),
			"retentionInDays":                 llx.IntData(retentionInDays),
			"publicNetworkAccessForIngestion": llx.StringData(publicNetworkAccessForIngestion),
			"publicNetworkAccessForQuery":     llx.StringData(publicNetworkAccessForQuery),
			"forceCmkForQuery":                llx.BoolDataPtr(props.ForceCmkForQuery),
			"createdDate":                     parseAzureDateString(props.CreatedDate),
			"modifiedDate":                    parseAzureDateString(props.ModifiedDate),
			"provisioningState":               llx.StringData(provisioningState),
			"customerId":                      llx.StringDataPtr(props.CustomerID),
		})
	if err != nil {
		return nil, err
	}

	mqlWs := resource.(*mqlAzureSubscriptionMonitorServiceWorkspace)
	mqlWs.cacheCapping = props.WorkspaceCapping
	mqlWs.cacheFeatures = props.Features
	mqlWs.cachePrivateLinkScopedResources = props.PrivateLinkScopedResources

	return mqlWs, nil
}

// initAzureSubscriptionMonitorServiceWorkspace fetches a workspace by ID for cross-ref resolution.
func initAzureSubscriptionMonitorServiceWorkspace(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	idRaw, ok := args["id"]
	if !ok || idRaw == nil {
		return args, nil, nil
	}
	id, ok := idRaw.Value.(string)
	if !ok || id == "" {
		return args, nil, nil
	}

	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, fmt.Errorf("invalid connection provided, it is not an Azure connection")
	}

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, nil, err
	}
	workspaceName, err := resourceID.Component("workspaces")
	if err != nil {
		return nil, nil, err
	}

	ctx := context.Background()
	client, err := armoperationalinsights.NewWorkspacesClient(resourceID.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, nil, err
	}

	resp, err := client.Get(ctx, resourceID.ResourceGroup, workspaceName, nil)
	if err != nil {
		return nil, nil, err
	}

	mqlWs, err := createWorkspaceResource(runtime, &resp.Workspace)
	if err != nil {
		return nil, nil, err
	}
	return nil, mqlWs, nil
}

// capping builds the capping sub-resource from cached data.
func (a *mqlAzureSubscriptionMonitorServiceWorkspace) capping() (*mqlAzureSubscriptionMonitorServiceWorkspaceCapping, error) {
	cap := a.cacheCapping
	if cap == nil {
		a.Capping.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	var dailyQuotaGb float64
	if cap.DailyQuotaGb != nil {
		dailyQuotaGb = *cap.DailyQuotaGb
	}

	var dataIngestionStatus string
	if cap.DataIngestionStatus != nil {
		dataIngestionStatus = string(*cap.DataIngestionStatus)
	}

	res, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionMonitorServiceWorkspaceCapping,
		map[string]*llx.RawData{
			"id":                  llx.StringData(a.Id.Data + "/capping"),
			"dailyQuotaGb":        llx.FloatData(dailyQuotaGb),
			"dataIngestionStatus": llx.StringData(dataIngestionStatus),
			"quotaNextResetTime":  llx.StringDataPtr(cap.QuotaNextResetTime),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionMonitorServiceWorkspaceCapping), nil
}

func (a *mqlAzureSubscriptionMonitorServiceWorkspaceCapping) id() (string, error) {
	return a.Id.Data, nil
}

// features builds the features sub-resource from cached data.
func (a *mqlAzureSubscriptionMonitorServiceWorkspace) features() (*mqlAzureSubscriptionMonitorServiceWorkspaceFeatures, error) {
	feat := a.cacheFeatures
	if feat == nil {
		a.Features.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	res, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionMonitorServiceWorkspaceFeatures,
		map[string]*llx.RawData{
			"id":               llx.StringData(a.Id.Data + "/features"),
			"disableLocalAuth": llx.BoolDataPtr(feat.DisableLocalAuth),
			"enableDataExport": llx.BoolDataPtr(feat.EnableDataExport),
			"enableLogAccessUsingOnlyResourcePermissions": llx.BoolDataPtr(feat.EnableLogAccessUsingOnlyResourcePermissions),
			"immediatePurgeDataOn30Days":                  llx.BoolDataPtr(feat.ImmediatePurgeDataOn30Days),
			"clusterResourceId":                           llx.StringDataPtr(feat.ClusterResourceID),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionMonitorServiceWorkspaceFeatures), nil
}

func (a *mqlAzureSubscriptionMonitorServiceWorkspaceFeatures) id() (string, error) {
	return a.Id.Data, nil
}

// privateLinkScopedResources returns the private link scoped resources as dicts.
func (a *mqlAzureSubscriptionMonitorServiceWorkspace) privateLinkScopedResources() ([]any, error) {
	var res []any
	for _, plsr := range a.cachePrivateLinkScopedResources {
		if plsr == nil {
			continue
		}
		d, err := convert.JsonToDict(plsr)
		if err != nil {
			return nil, err
		}
		res = append(res, d)
	}
	return res, nil
}

// dataExports fetches data export rules for the workspace.
func (a *mqlAzureSubscriptionMonitorServiceWorkspace) dataExports() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}
	workspaceName, err := resourceID.Component("workspaces")
	if err != nil {
		return nil, err
	}

	client, err := armoperationalinsights.NewDataExportsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListByWorkspacePager(resourceID.ResourceGroup, workspaceName, nil)
	var res []any
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, de := range page.Value {
			if de == nil {
				continue
			}

			var enabled bool
			var destinationResourceId string
			var tableNames []any
			var createdDate, lastModifiedDate *string
			if de.Properties != nil {
				if de.Properties.Enable != nil {
					enabled = *de.Properties.Enable
				}
				if de.Properties.Destination != nil && de.Properties.Destination.ResourceID != nil {
					destinationResourceId = *de.Properties.Destination.ResourceID
				}
				createdDate = de.Properties.CreatedDate
				lastModifiedDate = de.Properties.LastModifiedDate
				for _, tn := range de.Properties.TableNames {
					if tn != nil {
						tableNames = append(tableNames, *tn)
					}
				}
			}

			mqlDe, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionMonitorServiceWorkspaceDataExport,
				map[string]*llx.RawData{
					"id":                    llx.StringDataPtr(de.ID),
					"name":                  llx.StringDataPtr(de.Name),
					"type":                  llx.StringDataPtr(de.Type),
					"enabled":               llx.BoolData(enabled),
					"tableNames":            llx.ArrayData(tableNames, types.String),
					"destinationResourceId": llx.StringData(destinationResourceId),
					"createdDate":           parseAzureDateString(createdDate),
					"lastModifiedDate":      parseAzureDateString(lastModifiedDate),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlDe)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionMonitorServiceWorkspaceDataExport) id() (string, error) {
	return a.Id.Data, nil
}

// linkedServices fetches linked services for the workspace.
func (a *mqlAzureSubscriptionMonitorServiceWorkspace) linkedServices() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}
	workspaceName, err := resourceID.Component("workspaces")
	if err != nil {
		return nil, err
	}

	client, err := armoperationalinsights.NewLinkedServicesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListByWorkspacePager(resourceID.ResourceGroup, workspaceName, nil)
	var res []any
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, ls := range page.Value {
			if ls == nil {
				continue
			}

			var resourceId, writeAccessResourceId, provisioningState string
			if ls.Properties != nil {
				if ls.Properties.ResourceID != nil {
					resourceId = *ls.Properties.ResourceID
				}
				if ls.Properties.WriteAccessResourceID != nil {
					writeAccessResourceId = *ls.Properties.WriteAccessResourceID
				}
				if ls.Properties.ProvisioningState != nil {
					provisioningState = string(*ls.Properties.ProvisioningState)
				}
			}

			mqlLs, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionMonitorServiceWorkspaceLinkedService,
				map[string]*llx.RawData{
					"id":                    llx.StringDataPtr(ls.ID),
					"name":                  llx.StringDataPtr(ls.Name),
					"type":                  llx.StringDataPtr(ls.Type),
					"resourceId":            llx.StringData(resourceId),
					"writeAccessResourceId": llx.StringData(writeAccessResourceId),
					"provisioningState":     llx.StringData(provisioningState),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlLs)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionMonitorServiceWorkspaceLinkedService) id() (string, error) {
	return a.Id.Data, nil
}

// workspace returns a typed reference to the Log Analytics workspace from an Application Insight.
func (a *mqlAzureSubscriptionMonitorServiceApplicationInsight) workspace() (*mqlAzureSubscriptionMonitorServiceWorkspace, error) {
	if a.cacheWorkspaceResourceId == "" {
		a.Workspace.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	res, err := NewResource(a.MqlRuntime, ResourceAzureSubscriptionMonitorServiceWorkspace,
		map[string]*llx.RawData{
			"id": llx.StringData(a.cacheWorkspaceResourceId),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionMonitorServiceWorkspace), nil
}
