// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/sch"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOciServiceConnectorHub) id() (string, error) {
	return "oci.serviceConnectorHub", nil
}

func (o *mqlOciServiceConnectorHub) connectors() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	ociResource, err := CreateResource(o.MqlRuntime, "oci", nil)
	if err != nil {
		return nil, err
	}
	regions := ociResource.(*mqlOci).GetRegions()
	if regions.Error != nil {
		return nil, regions.Error
	}

	return ociRunCompartmentRegionPool(conn, regions.Data,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			client, err := conn.ServiceConnectorClient(region)
			if err != nil {
				return nil, err
			}

			connectors := []sch.ServiceConnectorSummary{}
			var page *string
			for {
				response, err := client.ListServiceConnectors(ctx, sch.ListServiceConnectorsRequest{
					CompartmentId: common.String(compartmentID),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}

				connectors = append(connectors, response.Items...)

				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			res := make([]any, 0, len(connectors))
			for i := range connectors {
				c := connectors[i]

				mqlConnector, err := CreateResource(o.MqlRuntime, "oci.serviceConnectorHub.connector", map[string]*llx.RawData{
					"id":           llx.StringDataPtr(c.Id),
					"name":         llx.StringDataPtr(c.DisplayName),
					"description":  llx.StringDataPtr(c.Description),
					"state":        llx.StringData(string(c.LifecycleState)),
					"stateDetails": llx.StringDataPtr(c.LifecycleDetails),
					"region":       llx.StringData(region),
					"created":      sdkTimeData(c.TimeCreated),
					"updated":      sdkTimeData(c.TimeUpdated),
					"freeformTags": llx.MapData(strMapToAny(c.FreeformTags), types.String),
					"definedTags":  llx.MapData(definedTagsToAny(c.DefinedTags), types.Any),
					"systemTags":   llx.MapData(definedTagsToAny(c.SystemTags), types.Dict),
				})
				if err != nil {
					return nil, err
				}
				mqlConnectorTyped := mqlConnector.(*mqlOciServiceConnectorHubConnector)
				mqlConnectorTyped.cacheCompartmentId = stringValue(c.CompartmentId)
				mqlConnectorTyped.cacheRegion = region
				res = append(res, mqlConnectorTyped)
			}

			return res, nil
		})
}

// The connector listing returns only identity and lifecycle fields. What the
// connector actually moves, and where to, comes from the per-connector detail
// call, so source, target and tasks share one lazy fetch rather than issuing
// three.
type mqlOciServiceConnectorHubConnectorInternal struct {
	cacheCompartmentId string
	cacheRegion        string

	detailLock    sync.Mutex
	detailFetched bool
	detail        *sch.ServiceConnector
}

func (o *mqlOciServiceConnectorHubConnector) id() (string, error) {
	return "oci.serviceConnectorHub.connector/" + o.Id.Data, nil
}

func (o *mqlOciServiceConnectorHubConnector) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.cacheCompartmentId, &o.Compartment)
}

func (o *mqlOciServiceConnectorHubConnector) getDetail() (*sch.ServiceConnector, error) {
	o.detailLock.Lock()
	defer o.detailLock.Unlock()
	if o.detailFetched {
		return o.detail, nil
	}

	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	client, err := conn.ServiceConnectorClient(o.cacheRegion)
	if err != nil {
		return nil, err
	}

	response, err := client.GetServiceConnector(context.Background(), sch.GetServiceConnectorRequest{
		ServiceConnectorId: common.String(o.Id.Data),
	})
	if err != nil {
		return nil, err
	}

	o.detail = &response.ServiceConnector
	o.detailFetched = true
	return o.detail, nil
}

func (o *mqlOciServiceConnectorHubConnector) sourceKind() (string, error) {
	detail, err := o.getDetail()
	if err != nil {
		return "", err
	}
	return ociConnectorSourceKind(detail.Source), nil
}

func (o *mqlOciServiceConnectorHubConnector) source() (any, error) {
	detail, err := o.getDetail()
	if err != nil {
		return nil, err
	}
	if detail.Source == nil {
		return nil, nil
	}
	return convert.JsonToDict(detail.Source)
}

func (o *mqlOciServiceConnectorHubConnector) targetKind() (string, error) {
	detail, err := o.getDetail()
	if err != nil {
		return "", err
	}
	return ociConnectorTargetKind(detail.Target), nil
}

func (o *mqlOciServiceConnectorHubConnector) target() (any, error) {
	detail, err := o.getDetail()
	if err != nil {
		return nil, err
	}
	if detail.Target == nil {
		return nil, nil
	}
	return convert.JsonToDict(detail.Target)
}

func (o *mqlOciServiceConnectorHubConnector) tasks() ([]any, error) {
	detail, err := o.getDetail()
	if err != nil {
		return nil, err
	}
	if len(detail.Tasks) == 0 {
		return []any{}, nil
	}
	return convert.JsonToDictSlice(detail.Tasks)
}

// ociConnectorSourceKind names the source end of a connector.
//
// The discriminator is derived from the concrete Go type rather than read back
// out of the marshalled JSON: the SDK models these as an interface, and a type
// switch cannot silently disagree with what was actually deserialized the way a
// string lookup could.
func ociConnectorSourceKind(source sch.SourceDetailsResponse) string {
	switch source.(type) {
	case sch.LoggingSourceDetailsResponse:
		return "logging"
	case sch.MonitoringSourceDetailsResponse:
		return "monitoring"
	case sch.StreamingSourceDetailsResponse:
		return "streaming"
	case sch.PluginSourceDetailsResponse:
		return "plugin"
	case nil:
		return ""
	default:
		// A source kind added to the service after this build. Reporting it as
		// unknown is honest; returning "" would read as "no source".
		return "unknown"
	}
}

// ociConnectorTargetKind names the target end of a connector.
func ociConnectorTargetKind(target sch.TargetDetailsResponse) string {
	switch target.(type) {
	case sch.ObjectStorageTargetDetailsResponse:
		return "objectStorage"
	case sch.StreamingTargetDetailsResponse:
		return "streaming"
	case sch.FunctionsTargetDetailsResponse:
		return "functions"
	case sch.LoggingAnalyticsTargetDetailsResponse:
		return "loggingAnalytics"
	case sch.MonitoringTargetDetailsResponse:
		return "monitoring"
	case sch.NotificationsTargetDetailsResponse:
		return "notifications"
	case nil:
		return ""
	default:
		return "unknown"
	}
}
