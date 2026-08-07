// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/dns"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOciDns) id() (string, error) {
	return "oci.dns", nil
}

// ociDnsZoneScopes are the two scopes a zone can live in.
//
// The Zones API scopes its listing, and omitting the parameter does not return
// the union: private zones - the ones resolving inside a VCN - are only
// returned when PRIVATE is asked for explicitly. Listing one scope would report
// a tenancy's internal DNS as absent, so both are queried and merged.
var ociDnsZoneScopes = []dns.ListZonesScopeEnum{
	dns.ListZonesScopeGlobal,
	dns.ListZonesScopePrivate,
}

// ociDnsSteeringPolicyScopes are the scopes a steering policy can live in.
// The listing scopes the same way zones do, so both have to be asked for.
var ociDnsSteeringPolicyScopes = []dns.ListSteeringPoliciesScopeEnum{
	dns.ListSteeringPoliciesScopeGlobal,
	dns.ListSteeringPoliciesScopePrivate,
}

func (o *mqlOciDns) zones() ([]any, error) {
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
			client, err := conn.DnsClient(region)
			if err != nil {
				return nil, err
			}

			zones := []dns.ZoneSummary{}
			for _, scope := range ociDnsZoneScopes {
				var page *string
				for {
					response, err := client.ListZones(ctx, dns.ListZonesRequest{
						CompartmentId: common.String(compartmentID),
						Scope:         scope,
						Page:          page,
					})
					if err != nil {
						return nil, err
					}

					zones = append(zones, response.Items...)

					if response.OpcNextPage == nil {
						break
					}
					page = response.OpcNextPage
				}
			}

			return o.newZones(region, zones)
		})
}

func (o *mqlOciDns) newZones(region string, zones []dns.ZoneSummary) ([]any, error) {
	res := make([]any, 0, len(zones))
	for i := range zones {
		zone := zones[i]

		var dnssecKeyVersions map[string]any
		if zone.DnssecConfig != nil {
			var err error
			dnssecKeyVersions, err = convert.JsonToDict(zone.DnssecConfig)
			if err != nil {
				return nil, err
			}
		}

		mqlZone, err := CreateResource(o.MqlRuntime, "oci.dns.zone", map[string]*llx.RawData{
			"id":                llx.StringDataPtr(zone.Id),
			"name":              llx.StringDataPtr(zone.Name),
			"zoneType":          llx.StringData(string(zone.ZoneType)),
			"scope":             llx.StringData(string(zone.Scope)),
			"resolutionMode":    llx.StringData(string(zone.ResolutionMode)),
			"dnssecState":       llx.StringData(string(zone.DnssecState)),
			"dnssecKeyVersions": llx.DictData(dnssecKeyVersions),
			"isProtected":       llx.BoolData(boolValue(zone.IsProtected)),
			"serial":            llx.IntDataDefault(zone.Serial, 0),
			"version":           llx.StringDataPtr(zone.Version),
			"viewId":            llx.StringDataPtr(zone.ViewId),
			"state":             llx.StringData(string(zone.LifecycleState)),
			"created":           sdkTimeData(zone.TimeCreated),
			"freeformTags":      llx.MapData(strMapToAny(zone.FreeformTags), types.String),
			"definedTags":       llx.MapData(definedTagsToAny(zone.DefinedTags), types.Any),
		})
		if err != nil {
			return nil, err
		}
		mqlZoneTyped := mqlZone.(*mqlOciDnsZone)
		mqlZoneTyped.cacheCompartmentId = stringValue(zone.CompartmentId)
		mqlZoneTyped.cacheRegion = region
		mqlZoneTyped.cacheScope = zone.Scope
		res = append(res, mqlZoneTyped)
	}

	return res, nil
}

func (o *mqlOciDns) steeringPolicies() ([]any, error) {
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
			client, err := conn.DnsClient(region)
			if err != nil {
				return nil, err
			}

			policies := []dns.SteeringPolicySummary{}
			// Scoped the same way zones are, and for the same reason: the
			// listing returns one scope at a time rather than the union, so
			// asking only for the default reports a tenancy's private traffic
			// management as absent.
			for _, scope := range ociDnsSteeringPolicyScopes {
				var page *string
				for {
					response, err := client.ListSteeringPolicies(ctx, dns.ListSteeringPoliciesRequest{
						CompartmentId: common.String(compartmentID),
						Scope:         scope,
						Page:          page,
					})
					if err != nil {
						return nil, err
					}

					policies = append(policies, response.Items...)

					if response.OpcNextPage == nil {
						break
					}
					page = response.OpcNextPage
				}
			}

			res := make([]any, 0, len(policies))
			for i := range policies {
				policy := policies[i]

				mqlPolicy, err := CreateResource(o.MqlRuntime, "oci.dns.steeringPolicy", map[string]*llx.RawData{
					"id":                   llx.StringDataPtr(policy.Id),
					"name":                 llx.StringDataPtr(policy.DisplayName),
					"template":             llx.StringData(string(policy.Template)),
					"ttl":                  llx.IntDataDefault(policy.Ttl, 0),
					"healthCheckMonitorId": llx.StringDataPtr(policy.HealthCheckMonitorId),
					"state":                llx.StringData(string(policy.LifecycleState)),
					"created":              sdkTimeData(policy.TimeCreated),
					"freeformTags":         llx.MapData(strMapToAny(policy.FreeformTags), types.String),
					"definedTags":          llx.MapData(definedTagsToAny(policy.DefinedTags), types.Any),
				})
				if err != nil {
					return nil, err
				}
				mqlPolicyTyped := mqlPolicy.(*mqlOciDnsSteeringPolicy)
				mqlPolicyTyped.cacheCompartmentId = stringValue(policy.CompartmentId)
				res = append(res, mqlPolicyTyped)
			}

			return res, nil
		})
}

type mqlOciDnsZoneInternal struct {
	cacheCompartmentId string
	cacheRegion        string
	cacheScope         dns.ScopeEnum
}

func (o *mqlOciDnsZone) id() (string, error) {
	return "oci.dns.zone/" + o.Id.Data, nil
}

func (o *mqlOciDnsZone) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.cacheCompartmentId, &o.Compartment)
}

// records fetches the zone's resource records on demand.
//
// This is a per-zone call, so it stays computed rather than being prefetched
// with the listing: a query that only asks which zones have DNSSEC disabled
// should not pull every record in the tenancy to answer it.
func (o *mqlOciDnsZone) records() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	client, err := conn.DnsClient(o.cacheRegion)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	records := []dns.Record{}
	var page *string
	for {
		response, err := client.GetZoneRecords(ctx, dns.GetZoneRecordsRequest{
			ZoneNameOrId: common.String(o.Id.Data),
			// A private zone is only addressable within its scope, and the
			// service rejects the request without it.
			Scope: dns.GetZoneRecordsScopeEnum(o.cacheScope),
			Page:  page,
		})
		if err != nil {
			return nil, err
		}

		records = append(records, response.Items...)

		if response.OpcNextPage == nil {
			break
		}
		page = response.OpcNextPage
	}

	res := make([]any, 0, len(records))
	for i := range records {
		record := records[i]

		// recordHash uniquely identifies a record within a zone. Records have
		// no OCID, and domain+type is not unique because a name can carry
		// several values of the same type.
		key := stringValue(record.RecordHash)
		if key == "" {
			key = stringValue(record.Domain) + "/" + stringValue(record.Rtype) + "/" + stringValue(record.Rdata)
		}

		mqlRecord, err := CreateResource(o.MqlRuntime, "oci.dns.record", map[string]*llx.RawData{
			"__id":         llx.StringData(o.Id.Data + "/record/" + key),
			"domain":       llx.StringDataPtr(record.Domain),
			"rtype":        llx.StringDataPtr(record.Rtype),
			"rdata":        llx.StringDataPtr(record.Rdata),
			"ttl":          llx.IntDataDefault(record.Ttl, 0),
			"isProtected":  llx.BoolData(boolValue(record.IsProtected)),
			"rrsetVersion": llx.StringDataPtr(record.RrsetVersion),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlRecord)
	}

	return res, nil
}

type mqlOciDnsSteeringPolicyInternal struct {
	cacheCompartmentId string
}

func (o *mqlOciDnsSteeringPolicy) id() (string, error) {
	return "oci.dns.steeringPolicy/" + o.Id.Data, nil
}

func (o *mqlOciDnsSteeringPolicy) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.cacheCompartmentId, &o.Compartment)
}
