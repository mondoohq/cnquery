// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/sch"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
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
//
// detailFetched is an atomic.Bool rather than a plain bool because getDetail
// reads it once before taking the lock. A plain bool read outside the mutex
// races with the write inside it - the same reason cloudguard.go uses
// atomic.Bool for its configFetched and homeRegionSet flags.
type mqlOciServiceConnectorHubConnectorInternal struct {
	cacheCompartmentId string
	cacheRegion        string

	detailLock    sync.Mutex
	detailFetched atomic.Bool
	detail        *sch.ServiceConnector
}

func (o *mqlOciServiceConnectorHubConnector) id() (string, error) {
	return "oci.serviceConnectorHub.connector/" + o.Id.Data, nil
}

func (o *mqlOciServiceConnectorHubConnector) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.cacheCompartmentId, &o.Compartment)
}

func (o *mqlOciServiceConnectorHubConnector) getDetail() (*sch.ServiceConnector, error) {
	// Fast path: five computed fields share this fetch, so after the first one
	// the rest should not queue on the mutex just to read a cached pointer.
	if o.detailFetched.Load() {
		return o.detail, nil
	}

	o.detailLock.Lock()
	defer o.detailLock.Unlock()
	if o.detailFetched.Load() {
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
	o.detailFetched.Store(true)
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

func (o *mqlOciServiceConnectorHubConnector) targetBucket() (*mqlOciObjectStorageBucket, error) {
	detail, err := o.getDetail()
	if err != nil {
		return nil, err
	}

	// OCI keys a bucket on namespace+name rather than on its OCID, so both
	// have to come from the target block; an OCID alone cannot be resolved.
	// Both are therefore required: resolving with an empty namespace would
	// build the cache key "oci.objectStorage.bucket//<name>", which every
	// namespace-less bucket would share, so a malformed target would collide
	// rather than report nothing.
	target, ok := detail.Target.(sch.ObjectStorageTargetDetailsResponse)
	if !ok || stringValue(target.BucketName) == "" || stringValue(target.Namespace) == "" {
		o.TargetBucket.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	// The bucket's own init needs a region: it fetches the bucket detail to
	// populate everything ListBuckets omits, and GetBucket is a regional call.
	// Passing only namespace and name left that region unset, so every
	// objectStorage-target connector failed with "region is required to fetch
	// bucket details". A Connector Hub target bucket is in the connector's own
	// region, which is already cached here.
	regionRes, err := NewResource(o.MqlRuntime, "oci.region", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheRegion),
	})
	if err != nil {
		return nil, err
	}

	res, err := NewResource(o.MqlRuntime, "oci.objectStorage.bucket", map[string]*llx.RawData{
		"namespace": llx.StringData(stringValue(target.Namespace)),
		"name":      llx.StringData(stringValue(target.BucketName)),
		"region":    llx.ResourceData(regionRes, "oci.region"),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOciObjectStorageBucket), nil
}

func (o *mqlOciServiceConnectorHubConnector) targetTopic() (*mqlOciOnsTopic, error) {
	detail, err := o.getDetail()
	if err != nil {
		return nil, err
	}

	target, ok := detail.Target.(sch.NotificationsTargetDetailsResponse)
	if !ok || stringValue(target.TopicId) == "" {
		o.TargetTopic.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	res, err := NewResource(o.MqlRuntime, "oci.ons.topic", map[string]*llx.RawData{
		"id": llx.StringData(stringValue(target.TopicId)),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOciOnsTopic), nil
}

// ociAuditLogGroup is the reserved name Connector Hub uses for the tenancy
// audit log. A logging source names its log group either by OCID or with this
// sentinel, which is not an OCID and cannot be resolved to an
// oci.logging.logGroup - it is the audit stream itself, not a customer log
// group.
const ociAuditLogGroup = "_Audit"

// ociSplitLogGroupRefs separates the log-group references on a Connector Hub
// logging source into the OCIDs that name a real log group and a flag for the
// tenancy audit log.
//
// Passing every reference through to a log-group lookup meant the audit
// sentinel failed to resolve and was dropped, so a connector exporting the
// audit log - the reason most tenancies run Connector Hub at all - reported no
// sources. A reference that is neither an OCID nor the sentinel is dropped:
// only a compartment-scoped source, which names no log group.
func ociSplitLogGroupRefs(refs []string) (ocids []string, exportsAuditLog bool) {
	ocids = make([]string, 0, len(refs))
	for _, ref := range refs {
		switch {
		case ref == ociAuditLogGroup:
			exportsAuditLog = true
		case isOcid(ref):
			ocids = append(ocids, ref)
		}
	}
	return ocids, exportsAuditLog
}

func (o *mqlOciServiceConnectorHubConnector) exportsAuditLog() (bool, error) {
	detail, err := o.getDetail()
	if err != nil {
		return false, err
	}

	source, ok := detail.Source.(sch.LoggingSourceDetailsResponse)
	if !ok {
		return false, nil
	}

	refs := make([]string, 0, len(source.LogSources))
	for i := range source.LogSources {
		refs = append(refs, stringValue(source.LogSources[i].LogGroupId))
	}
	_, exportsAuditLog := ociSplitLogGroupRefs(refs)
	return exportsAuditLog, nil
}

func (o *mqlOciServiceConnectorHubConnector) sourceLogGroups() ([]any, error) {
	detail, err := o.getDetail()
	if err != nil {
		return nil, err
	}

	source, ok := detail.Source.(sch.LoggingSourceDetailsResponse)
	if !ok {
		return []any{}, nil
	}

	refs := make([]string, 0, len(source.LogSources))
	for i := range source.LogSources {
		refs = append(refs, stringValue(source.LogSources[i].LogGroupId))
	}
	logGroupIds, _ := ociSplitLogGroupRefs(refs)

	res := make([]any, 0, len(logGroupIds))
	for _, logGroupId := range logGroupIds {
		group, err := NewResource(o.MqlRuntime, "oci.logging.logGroup", map[string]*llx.RawData{
			"id": llx.StringData(logGroupId),
		})
		if err != nil {
			// One unresolvable group must not discard the rest of the export
			// path, matching resolveOciSecurityGroups.
			log.Debug().Err(err).Str("logGroup", logGroupId).Msg("skipping unresolvable oci log group")
			continue
		}
		res = append(res, group)
	}

	return res, nil
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
