// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/networkloadbalancer"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/providers/oci/connection"
	"go.mondoo.com/mql/types"
)

func (o *mqlOciNetworkLoadBalancer) id() (string, error) {
	return "oci.networkLoadBalancer", nil
}

func (o *mqlOciNetworkLoadBalancer) loadBalancers() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	return ociCollect(o.MqlRuntime, ociScopeAllCompartments,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			client, err := conn.NetworkLoadBalancerClient(region)
			if err != nil {
				return nil, err
			}

			nlbs, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]networkloadbalancer.NetworkLoadBalancerSummary, *string, error) {
				response, err := client.ListNetworkLoadBalancers(ctx, networkloadbalancer.ListNetworkLoadBalancersRequest{
					CompartmentId: common.String(compartmentID),
					Page:          page,
				})
				if err != nil {
					return nil, nil, err
				}
				return response.NetworkLoadBalancerCollection.Items, response.OpcNextPage, nil
			})
			if err != nil {
				return nil, err
			}

			return o.newNetworkLoadBalancers(nlbs)
		})
}

func (o *mqlOciNetworkLoadBalancer) newNetworkLoadBalancers(nlbs []networkloadbalancer.NetworkLoadBalancerSummary) ([]any, error) {
	res := make([]any, 0, len(nlbs))
	for i := range nlbs {
		nlb := nlbs[i]

		// Built by hand rather than marshalled from the SDK slice: isPublic is
		// optional on the model, and exposure() reads the key back to decide
		// internet reachability. A marshalled nil arrives as JSON null, which
		// reads as "not public" and clears a genuinely internet-facing
		// balancer. This mirrors the classic load balancer.
		ipAddresses := make([]any, 0, len(nlb.IpAddresses))
		for j := range nlb.IpAddresses {
			ip := nlb.IpAddresses[j]
			entry := map[string]any{
				"ipAddress": stringValue(ip.IpAddress),
				"isPublic":  ociIpIsPublic(ip.IsPublic, boolValue(nlb.IsPrivate)),
			}
			if ip.ReservedIp != nil {
				entry["reservedIpId"] = stringValue(ip.ReservedIp.Id)
			}
			ipAddresses = append(ipAddresses, entry)
		}

		listeners, err := o.newListeners(stringValue(nlb.Id), nlb.Listeners)
		if err != nil {
			return nil, err
		}

		backendSets, err := o.newBackendSets(stringValue(nlb.Id), nlb.BackendSets)
		if err != nil {
			return nil, err
		}

		mqlNlb, err := CreateResource(o.MqlRuntime, "oci.networkLoadBalancer.loadBalancer", map[string]*llx.RawData{
			"id":                          llx.StringDataPtr(nlb.Id),
			"name":                        llx.StringDataPtr(nlb.DisplayName),
			"isPrivate":                   llx.BoolData(boolValue(nlb.IsPrivate)),
			"ipAddresses":                 llx.ArrayData(ipAddresses, types.Dict),
			"ipVersion":                   llx.StringData(string(nlb.NlbIpVersion)),
			"listeners":                   llx.ArrayData(listeners, types.Resource("oci.networkLoadBalancer.listener")),
			"backendSets":                 llx.ArrayData(backendSets, types.Resource("oci.networkLoadBalancer.backendSet")),
			"isPreserveSourceDestination": llx.BoolData(boolValue(nlb.IsPreserveSourceDestination)),
			"isSymmetricHashEnabled":      llx.BoolData(boolValue(nlb.IsSymmetricHashEnabled)),
			"securityAttributes":          llx.MapData(definedTagsToAny(nlb.SecurityAttributes), types.Dict),
			"state":                       llx.StringData(string(nlb.LifecycleState)),
			"stateDetails":                llx.StringDataPtr(nlb.LifecycleDetails),
			"created":                     sdkTimeData(nlb.TimeCreated),
			"updated":                     sdkTimeData(nlb.TimeUpdated),
			"freeformTags":                llx.MapData(strMapToAny(nlb.FreeformTags), types.String),
			"definedTags":                 llx.MapData(definedTagsToAny(nlb.DefinedTags), types.Any),
			"systemTags":                  llx.MapData(definedTagsToAny(nlb.SystemTags), types.Dict),
		})
		if err != nil {
			return nil, err
		}
		mqlNlbTyped := mqlNlb.(*mqlOciNetworkLoadBalancerLoadBalancer)
		mqlNlbTyped.cacheCompartmentID = stringValue(nlb.CompartmentId)
		mqlNlbTyped.cacheSubnetID = stringValue(nlb.SubnetId)
		mqlNlbTyped.cacheNsgIDs = nlb.NetworkSecurityGroupIds
		res = append(res, mqlNlbTyped)
	}

	return res, nil
}

// newListeners builds the listener resources for one network load balancer.
//
// The SDK returns listeners as a map keyed by name. Go map iteration order is
// randomized, so the keys are sorted before building resources: without it the
// listener list would come back in a different order on every query, which
// makes result diffing and any index-based assertion unstable.
func (o *mqlOciNetworkLoadBalancer) newListeners(nlbID string, listeners map[string]networkloadbalancer.Listener) ([]any, error) {
	names := make([]string, 0, len(listeners))
	for name := range listeners {
		names = append(names, name)
	}
	sort.Strings(names)

	res := make([]any, 0, len(names))
	for _, name := range names {
		listener := listeners[name]

		mqlListener, err := CreateResource(o.MqlRuntime, "oci.networkLoadBalancer.listener", map[string]*llx.RawData{
			"__id":                  llx.StringData(nlbID + "/listener/" + name),
			"name":                  llx.StringDataPtr(listener.Name),
			"port":                  llx.IntDataDefault(listener.Port, 0),
			"protocol":              llx.StringData(string(listener.Protocol)),
			"defaultBackendSetName": llx.StringDataPtr(listener.DefaultBackendSetName),
			"ipVersion":             llx.StringData(string(listener.IpVersion)),
			"tcpIdleTimeout":        llx.IntDataDefault(listener.TcpIdleTimeout, 0),
			"udpIdleTimeout":        llx.IntDataDefault(listener.UdpIdleTimeout, 0),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlListener)
	}

	return res, nil
}

func (o *mqlOciNetworkLoadBalancer) newBackendSets(nlbID string, backendSets map[string]networkloadbalancer.BackendSet) ([]any, error) {
	names := make([]string, 0, len(backendSets))
	for name := range backendSets {
		names = append(names, name)
	}
	sort.Strings(names)

	res := make([]any, 0, len(names))
	for _, name := range names {
		backendSet := backendSets[name]

		var healthChecker map[string]any
		if backendSet.HealthChecker != nil {
			var err error
			healthChecker, err = convert.JsonToDict(backendSet.HealthChecker)
			if err != nil {
				return nil, err
			}
		}

		backends, err := o.newBackends(nlbID+"/backendSet/"+name, backendSet.Backends)
		if err != nil {
			return nil, err
		}

		mqlBackendSet, err := CreateResource(o.MqlRuntime, "oci.networkLoadBalancer.backendSet", map[string]*llx.RawData{
			"__id":                     llx.StringData(nlbID + "/backendSet/" + name),
			"name":                     llx.StringDataPtr(backendSet.Name),
			"policy":                   llx.StringData(string(backendSet.Policy)),
			"ipVersion":                llx.StringData(string(backendSet.IpVersion)),
			"isPreserveSource":         llx.BoolData(boolValue(backendSet.IsPreserveSource)),
			"isFailOpen":               llx.BoolData(boolValue(backendSet.IsFailOpen)),
			"isInstantFailoverEnabled": llx.BoolData(boolValue(backendSet.IsInstantFailoverEnabled)),
			"healthChecker":            llx.DictData(healthChecker),
			"backends":                 llx.ArrayData(backends, types.Resource("oci.networkLoadBalancer.backend")),
			"backendCount":             llx.IntData(int64(len(backendSet.Backends))),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlBackendSet)
	}

	return res, nil
}

func (o *mqlOciNetworkLoadBalancer) newBackends(backendSetID string, backends []networkloadbalancer.Backend) ([]any, error) {
	res := make([]any, 0, len(backends))
	for i := range backends {
		backend := backends[i]

		// A backend is identified by name where the service supplies one, and
		// by address and port otherwise. Falling back to the index alone would
		// make the cache key unstable across list calls that reorder.
		key := stringValue(backend.Name)
		if key == "" {
			key = stringValue(backend.IpAddress) + ":" + strconv.FormatInt(intValue(backend.Port), 10)
		}

		mqlBackend, err := CreateResource(o.MqlRuntime, "oci.networkLoadBalancer.backend", map[string]*llx.RawData{
			"__id":      llx.StringData(backendSetID + "/backend/" + key),
			"name":      llx.StringDataPtr(backend.Name),
			"ipAddress": llx.StringDataPtr(backend.IpAddress),
			"port":      llx.IntDataDefault(backend.Port, 0),
			"weight":    llx.IntDataDefault(backend.Weight, 0),
			"isDrain":   llx.BoolData(boolValue(backend.IsDrain)),
			"isBackup":  llx.BoolData(boolValue(backend.IsBackup)),
			"isOffline": llx.BoolData(boolValue(backend.IsOffline)),
		})
		if err != nil {
			return nil, err
		}
		mqlBackendTyped := mqlBackend.(*mqlOciNetworkLoadBalancerBackend)
		mqlBackendTyped.cacheTargetID = stringValue(backend.TargetId)
		res = append(res, mqlBackendTyped)
	}

	return res, nil
}

type mqlOciNetworkLoadBalancerLoadBalancerInternal struct {
	cacheCompartmentID string
	cacheSubnetID      string
	cacheNsgIDs        []string
}

func (o *mqlOciNetworkLoadBalancerLoadBalancer) id() (string, error) {
	return "oci.networkLoadBalancer.loadBalancer/" + o.Id.Data, nil
}

func (o *mqlOciNetworkLoadBalancerLoadBalancer) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.cacheCompartmentID, &o.Compartment)
}

func (o *mqlOciNetworkLoadBalancerLoadBalancer) subnet() (*mqlOciNetworkSubnet, error) {
	if o.cacheSubnetID == "" {
		o.Subnet.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(o.MqlRuntime, "oci.network.subnet", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheSubnetID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOciNetworkSubnet), nil
}

func (o *mqlOciNetworkLoadBalancerLoadBalancer) securityGroups() ([]any, error) {
	return resolveOciSecurityGroups(o.MqlRuntime, stringsToAny(o.cacheNsgIDs))
}

type mqlOciNetworkLoadBalancerBackendInternal struct {
	cacheTargetID string
}

func (o *mqlOciNetworkLoadBalancerBackend) instance() (*mqlOciComputeInstance, error) {
	// A backend addressed by IP has no target OCID, and one addressed by OCID
	// may point at something other than a compute instance. Only resolve what
	// is actually an instance; anything else is reported as null rather than
	// forced through a lookup that would fail.
	if !strings.HasPrefix(o.cacheTargetID, "ocid1.instance.") {
		o.Instance.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(o.MqlRuntime, "oci.compute.instance", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheTargetID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOciComputeInstance), nil
}
