// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

// regionResources returns the tenancy's subscribed regions as []*mqlOciRegion
// wrapped in []any, ready to hand to a jobpool fan-out.
func (o *mqlOciNetwork) regionResources() ([]any, error) {
	ociResource, err := CreateResource(o.MqlRuntime, "oci", nil)
	if err != nil {
		return nil, err
	}
	list := ociResource.(*mqlOci).GetRegions()
	if list.Error != nil {
		return nil, list.Error
	}
	return list.Data, nil
}

// Dynamic Routing Gateways

type mqlOciNetworkDrgInternal struct {
	region string
}

func (o *mqlOciNetwork) drgs() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	regions, err := o.regionResources()
	if err != nil {
		return nil, err
	}
	return ociRunRegionPool(o.getDrgs(conn, regions))
}

func (o *mqlOciNetwork) getDrgs(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			regionKey := regionResource.Id.Data
			log.Debug().Msgf("calling oci DRGs with region %s", regionKey)

			svc, err := conn.NetworkClient(regionKey)
			if err != nil {
				return nil, err
			}

			drgs := []core.Drg{}
			var page *string
			for {
				response, err := svc.ListDrgs(ctx, core.ListDrgsRequest{
					CompartmentId: common.String(conn.TenantID()),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}
				drgs = append(drgs, response.Items...)
				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			var res []any
			for i := range drgs {
				drg := drgs[i]

				var created *time.Time
				if drg.TimeCreated != nil {
					created = &drg.TimeCreated.Time
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.network.drg", map[string]*llx.RawData{
					"id":            llx.StringDataPtr(drg.Id),
					"name":          llx.StringDataPtr(drg.DisplayName),
					"compartmentID": llx.StringDataPtr(drg.CompartmentId),
					"state":         llx.StringData(string(drg.LifecycleState)),
					"created":       llx.TimeDataPtr(created),
					"freeformTags":  llx.MapData(strMapToAny(drg.FreeformTags), types.String),
					"definedTags":   llx.MapData(definedTagsToAny(drg.DefinedTags), types.Any),
				})
				if err != nil {
					return nil, err
				}
				mqlDrg := mqlInstance.(*mqlOciNetworkDrg)
				mqlDrg.region = regionKey
				res = append(res, mqlDrg)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func initOciNetworkDrg(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	idVal := ociArgString(args, "id")
	if idVal == "" {
		return nil, nil, errors.New("id required to fetch oci.network.drg")
	}

	obj, err := CreateResource(runtime, "oci.network", nil)
	if err != nil {
		return nil, nil, err
	}
	network := obj.(*mqlOciNetwork)

	raws := network.GetDrgs()
	if raws.Error != nil {
		return nil, nil, raws.Error
	}
	for _, raw := range raws.Data {
		drg := raw.(*mqlOciNetworkDrg)
		if drg.Id.Data == idVal {
			return args, drg, nil
		}
	}
	return nil, nil, errors.New("oci.network.drg not found: " + idVal)
}

func (o *mqlOciNetworkDrg) id() (string, error) {
	return "oci.network.drg/" + o.Id.Data, nil
}

func (o *mqlOciNetworkDrg) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

func (o *mqlOciNetworkDrg) drgRegion() string {
	if o.region != "" {
		return o.region
	}
	return ociRegionFromOCID(o.Id.Data)
}

func (o *mqlOciNetworkDrg) attachments() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	svc, err := conn.NetworkClient(o.drgRegion())
	if err != nil {
		return nil, err
	}
	ctx := context.Background()

	atts := []core.DrgAttachment{}
	var page *string
	for {
		response, err := svc.ListDrgAttachments(ctx, core.ListDrgAttachmentsRequest{
			CompartmentId: common.String(conn.TenantID()),
			DrgId:         common.String(o.Id.Data),
			Page:          page,
		})
		if err != nil {
			return nil, err
		}
		atts = append(atts, response.Items...)
		if response.OpcNextPage == nil {
			break
		}
		page = response.OpcNextPage
	}

	res := make([]any, 0, len(atts))
	for i := range atts {
		mqlAtt, err := o.newDrgAttachment(atts[i])
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAtt)
	}
	return res, nil
}

func (o *mqlOciNetworkDrg) newDrgAttachment(att core.DrgAttachment) (*mqlOciNetworkDrgAttachment, error) {
	var created *time.Time
	if att.TimeCreated != nil {
		created = &att.TimeCreated.Time
	}

	networkType := ""
	networkID := ""
	ipsecConnID := ""
	virtualCircuitID := ""
	if att.NetworkDetails != nil {
		networkID = stringValue(att.NetworkDetails.GetId())
		switch nd := att.NetworkDetails.(type) {
		case core.VcnDrgAttachmentNetworkDetails:
			networkType = "VCN"
		case core.IpsecTunnelDrgAttachmentNetworkDetails:
			networkType = "IPSEC_TUNNEL"
			// networkID is the tunnel OCID; the connection is the queryable parent.
			ipsecConnID = stringValue(nd.IpsecConnectionId)
		case core.VirtualCircuitDrgAttachmentNetworkDetails:
			networkType = "VIRTUAL_CIRCUIT"
			virtualCircuitID = networkID
		case core.RemotePeeringConnectionDrgAttachmentNetworkDetails:
			networkType = "REMOTE_PEERING_CONNECTION"
		case core.LoopBackDrgAttachmentNetworkDetails:
			networkType = "LOOPBACK"
		}
	}
	// Fall back to the deprecated top-level VcnId when networkDetails is absent.
	if networkID == "" && att.VcnId != nil {
		networkID = *att.VcnId
		networkType = "VCN"
	}

	mqlInstance, err := CreateResource(o.MqlRuntime, "oci.network.drgAttachment", map[string]*llx.RawData{
		"id":             llx.StringDataPtr(att.Id),
		"name":           llx.StringDataPtr(att.DisplayName),
		"compartmentID":  llx.StringDataPtr(att.CompartmentId),
		"networkType":    llx.StringData(networkType),
		"networkId":      llx.StringData(networkID),
		"isCrossTenancy": llx.BoolDataPtr(att.IsCrossTenancy),
		"state":          llx.StringData(string(att.LifecycleState)),
		"created":        llx.TimeDataPtr(created),
	})
	if err != nil {
		return nil, err
	}
	mqlAtt := mqlInstance.(*mqlOciNetworkDrgAttachment)
	mqlAtt.cacheDrgId = stringValue(att.DrgId)
	if networkType == "VCN" {
		mqlAtt.cacheVcnId = networkID
	}
	mqlAtt.cacheIpsecConnectionId = ipsecConnID
	mqlAtt.cacheVirtualCircuitId = virtualCircuitID
	return mqlAtt, nil
}

func (o *mqlOciNetworkDrg) remotePeeringConnections() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	svc, err := conn.NetworkClient(o.drgRegion())
	if err != nil {
		return nil, err
	}
	ctx := context.Background()

	rpcs := []core.RemotePeeringConnection{}
	var page *string
	for {
		response, err := svc.ListRemotePeeringConnections(ctx, core.ListRemotePeeringConnectionsRequest{
			CompartmentId: common.String(conn.TenantID()),
			DrgId:         common.String(o.Id.Data),
			Page:          page,
		})
		if err != nil {
			return nil, err
		}
		rpcs = append(rpcs, response.Items...)
		if response.OpcNextPage == nil {
			break
		}
		page = response.OpcNextPage
	}

	res := make([]any, 0, len(rpcs))
	for i := range rpcs {
		rpc := rpcs[i]

		var created *time.Time
		if rpc.TimeCreated != nil {
			created = &rpc.TimeCreated.Time
		}

		mqlInstance, err := CreateResource(o.MqlRuntime, "oci.network.remotePeeringConnection", map[string]*llx.RawData{
			"id":                    llx.StringDataPtr(rpc.Id),
			"name":                  llx.StringDataPtr(rpc.DisplayName),
			"compartmentID":         llx.StringDataPtr(rpc.CompartmentId),
			"isCrossTenancyPeering": llx.BoolDataPtr(rpc.IsCrossTenancyPeering),
			"peeringStatus":         llx.StringData(string(rpc.PeeringStatus)),
			"peerId":                llx.StringDataPtr(rpc.PeerId),
			"peerRegionName":        llx.StringDataPtr(rpc.PeerRegionName),
			"state":                 llx.StringData(string(rpc.LifecycleState)),
			"created":               llx.TimeDataPtr(created),
			"freeformTags":          llx.MapData(strMapToAny(rpc.FreeformTags), types.String),
			"definedTags":           llx.MapData(definedTagsToAny(rpc.DefinedTags), types.Any),
		})
		if err != nil {
			return nil, err
		}
		mqlRpc := mqlInstance.(*mqlOciNetworkRemotePeeringConnection)
		mqlRpc.cacheDrgId = stringValue(rpc.DrgId)
		res = append(res, mqlRpc)
	}
	return res, nil
}

// DRG attachments

type mqlOciNetworkDrgAttachmentInternal struct {
	cacheDrgId             string
	cacheVcnId             string
	cacheIpsecConnectionId string
	cacheVirtualCircuitId  string
}

func (o *mqlOciNetworkDrgAttachment) id() (string, error) {
	return "oci.network.drgAttachment/" + o.Id.Data, nil
}

func (o *mqlOciNetworkDrgAttachment) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

func (o *mqlOciNetworkDrgAttachment) drg() (*mqlOciNetworkDrg, error) {
	if !isOcid(o.cacheDrgId) {
		o.Drg.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.network.drg", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheDrgId),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciNetworkDrg), nil
}

func (o *mqlOciNetworkDrgAttachment) vcn() (*mqlOciNetworkVcn, error) {
	if !isOcid(o.cacheVcnId) {
		o.Vcn.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.network.vcn", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheVcnId),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciNetworkVcn), nil
}

func (o *mqlOciNetworkDrgAttachment) ipsecConnection() (*mqlOciNetworkIpsecConnection, error) {
	if !isOcid(o.cacheIpsecConnectionId) {
		o.IpsecConnection.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.network.ipsecConnection", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheIpsecConnectionId),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciNetworkIpsecConnection), nil
}

func (o *mqlOciNetworkDrgAttachment) virtualCircuit() (*mqlOciNetworkVirtualCircuit, error) {
	if !isOcid(o.cacheVirtualCircuitId) {
		o.VirtualCircuit.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.network.virtualCircuit", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheVirtualCircuitId),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciNetworkVirtualCircuit), nil
}

// Remote peering connections

type mqlOciNetworkRemotePeeringConnectionInternal struct {
	cacheDrgId string
}

func (o *mqlOciNetworkRemotePeeringConnection) id() (string, error) {
	return "oci.network.remotePeeringConnection/" + o.Id.Data, nil
}

func (o *mqlOciNetworkRemotePeeringConnection) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

func (o *mqlOciNetworkRemotePeeringConnection) drg() (*mqlOciNetworkDrg, error) {
	if !isOcid(o.cacheDrgId) {
		o.Drg.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.network.drg", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheDrgId),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciNetworkDrg), nil
}

// Local peering gateways

type mqlOciNetworkLocalPeeringGatewayInternal struct {
	cacheVcnId        string
	cachePeerId       string
	cacheRouteTableId string
}

func (o *mqlOciNetwork) localPeeringGateways() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	regions, err := o.regionResources()
	if err != nil {
		return nil, err
	}
	return ociRunRegionPool(o.getLocalPeeringGateways(conn, regions))
}

func (o *mqlOciNetwork) getLocalPeeringGateways(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			regionKey := regionResource.Id.Data
			log.Debug().Msgf("calling oci local peering gateways with region %s", regionKey)

			svc, err := conn.NetworkClient(regionKey)
			if err != nil {
				return nil, err
			}

			lpgs := []core.LocalPeeringGateway{}
			var page *string
			for {
				response, err := svc.ListLocalPeeringGateways(ctx, core.ListLocalPeeringGatewaysRequest{
					CompartmentId: common.String(conn.TenantID()),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}
				lpgs = append(lpgs, response.Items...)
				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			var res []any
			for i := range lpgs {
				lpg := lpgs[i]

				var created *time.Time
				if lpg.TimeCreated != nil {
					created = &lpg.TimeCreated.Time
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.network.localPeeringGateway", map[string]*llx.RawData{
					"id":                    llx.StringDataPtr(lpg.Id),
					"name":                  llx.StringDataPtr(lpg.DisplayName),
					"compartmentID":         llx.StringDataPtr(lpg.CompartmentId),
					"isCrossTenancyPeering": llx.BoolDataPtr(lpg.IsCrossTenancyPeering),
					"peeringStatus":         llx.StringData(string(lpg.PeeringStatus)),
					"peerAdvertisedCidr":    llx.StringDataPtr(lpg.PeerAdvertisedCidr),
					"state":                 llx.StringData(string(lpg.LifecycleState)),
					"created":               llx.TimeDataPtr(created),
					"freeformTags":          llx.MapData(strMapToAny(lpg.FreeformTags), types.String),
					"definedTags":           llx.MapData(definedTagsToAny(lpg.DefinedTags), types.Any),
				})
				if err != nil {
					return nil, err
				}
				mqlLpg := mqlInstance.(*mqlOciNetworkLocalPeeringGateway)
				mqlLpg.cacheVcnId = stringValue(lpg.VcnId)
				mqlLpg.cachePeerId = stringValue(lpg.PeerId)
				mqlLpg.cacheRouteTableId = stringValue(lpg.RouteTableId)
				res = append(res, mqlLpg)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func initOciNetworkLocalPeeringGateway(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	idVal := ociArgString(args, "id")
	if idVal == "" {
		return nil, nil, errors.New("id required to fetch oci.network.localPeeringGateway")
	}

	obj, err := CreateResource(runtime, "oci.network", nil)
	if err != nil {
		return nil, nil, err
	}
	network := obj.(*mqlOciNetwork)

	raws := network.GetLocalPeeringGateways()
	if raws.Error != nil {
		return nil, nil, raws.Error
	}
	for _, raw := range raws.Data {
		lpg := raw.(*mqlOciNetworkLocalPeeringGateway)
		if lpg.Id.Data == idVal {
			return args, lpg, nil
		}
	}
	return nil, nil, errors.New("oci.network.localPeeringGateway not found: " + idVal)
}

func (o *mqlOciNetworkLocalPeeringGateway) id() (string, error) {
	return "oci.network.localPeeringGateway/" + o.Id.Data, nil
}

func (o *mqlOciNetworkLocalPeeringGateway) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

func (o *mqlOciNetworkLocalPeeringGateway) vcn() (*mqlOciNetworkVcn, error) {
	if !isOcid(o.cacheVcnId) {
		o.Vcn.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.network.vcn", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheVcnId),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciNetworkVcn), nil
}

func (o *mqlOciNetworkLocalPeeringGateway) peer() (*mqlOciNetworkLocalPeeringGateway, error) {
	// A cross-tenancy peer lives outside this tenancy and cannot be resolved.
	if !isOcid(o.cachePeerId) || o.IsCrossTenancyPeering.Data {
		o.Peer.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.network.localPeeringGateway", map[string]*llx.RawData{
		"id": llx.StringData(o.cachePeerId),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciNetworkLocalPeeringGateway), nil
}

func (o *mqlOciNetworkLocalPeeringGateway) routeTable() (*mqlOciNetworkRouteTable, error) {
	if !isOcid(o.cacheRouteTableId) {
		o.RouteTable.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.network.routeTable", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheRouteTableId),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciNetworkRouteTable), nil
}

// Service gateways

type mqlOciNetworkServiceGatewayInternal struct {
	region            string
	cacheVcnId        string
	cacheRouteTableId string
	cacheServiceIds   []string
}

func (o *mqlOciNetwork) serviceGateways() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	regions, err := o.regionResources()
	if err != nil {
		return nil, err
	}
	return ociRunRegionPool(o.getServiceGateways(conn, regions))
}

func (o *mqlOciNetwork) getServiceGateways(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			regionKey := regionResource.Id.Data
			log.Debug().Msgf("calling oci service gateways with region %s", regionKey)

			svc, err := conn.NetworkClient(regionKey)
			if err != nil {
				return nil, err
			}

			sgws := []core.ServiceGateway{}
			var page *string
			for {
				response, err := svc.ListServiceGateways(ctx, core.ListServiceGatewaysRequest{
					CompartmentId: common.String(conn.TenantID()),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}
				sgws = append(sgws, response.Items...)
				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			var res []any
			for i := range sgws {
				sgw := sgws[i]

				var created *time.Time
				if sgw.TimeCreated != nil {
					created = &sgw.TimeCreated.Time
				}

				serviceIDs := make([]string, 0, len(sgw.Services))
				for j := range sgw.Services {
					serviceIDs = append(serviceIDs, stringValue(sgw.Services[j].ServiceId))
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.network.serviceGateway", map[string]*llx.RawData{
					"id":            llx.StringDataPtr(sgw.Id),
					"name":          llx.StringDataPtr(sgw.DisplayName),
					"compartmentID": llx.StringDataPtr(sgw.CompartmentId),
					"blockTraffic":  llx.BoolDataPtr(sgw.BlockTraffic),
					"state":         llx.StringData(string(sgw.LifecycleState)),
					"created":       llx.TimeDataPtr(created),
					"freeformTags":  llx.MapData(strMapToAny(sgw.FreeformTags), types.String),
					"definedTags":   llx.MapData(definedTagsToAny(sgw.DefinedTags), types.Any),
				})
				if err != nil {
					return nil, err
				}
				mqlSgw := mqlInstance.(*mqlOciNetworkServiceGateway)
				mqlSgw.region = regionKey
				mqlSgw.cacheVcnId = stringValue(sgw.VcnId)
				mqlSgw.cacheRouteTableId = stringValue(sgw.RouteTableId)
				mqlSgw.cacheServiceIds = serviceIDs
				res = append(res, mqlSgw)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func initOciNetworkServiceGateway(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	idVal := ociArgString(args, "id")
	if idVal == "" {
		return nil, nil, errors.New("id required to fetch oci.network.serviceGateway")
	}

	obj, err := CreateResource(runtime, "oci.network", nil)
	if err != nil {
		return nil, nil, err
	}
	network := obj.(*mqlOciNetwork)

	raws := network.GetServiceGateways()
	if raws.Error != nil {
		return nil, nil, raws.Error
	}
	for _, raw := range raws.Data {
		sgw := raw.(*mqlOciNetworkServiceGateway)
		if sgw.Id.Data == idVal {
			return args, sgw, nil
		}
	}
	return nil, nil, errors.New("oci.network.serviceGateway not found: " + idVal)
}

func (o *mqlOciNetworkServiceGateway) id() (string, error) {
	return "oci.network.serviceGateway/" + o.Id.Data, nil
}

func (o *mqlOciNetworkServiceGateway) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

func (o *mqlOciNetworkServiceGateway) vcn() (*mqlOciNetworkVcn, error) {
	if !isOcid(o.cacheVcnId) {
		o.Vcn.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.network.vcn", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheVcnId),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciNetworkVcn), nil
}

func (o *mqlOciNetworkServiceGateway) routeTable() (*mqlOciNetworkRouteTable, error) {
	if !isOcid(o.cacheRouteTableId) {
		o.RouteTable.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.network.routeTable", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheRouteTableId),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciNetworkRouteTable), nil
}

func (o *mqlOciNetworkServiceGateway) services() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	region := o.region
	if region == "" {
		region = ociRegionFromOCID(o.Id.Data)
	}
	svc, err := conn.NetworkClient(region)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()

	res := make([]any, 0, len(o.cacheServiceIds))
	for _, sid := range o.cacheServiceIds {
		if !isOcid(sid) {
			continue
		}
		response, err := svc.GetService(ctx, core.GetServiceRequest{ServiceId: common.String(sid)})
		if err != nil {
			return nil, err
		}
		s := response.Service
		mqlInstance, err := CreateResource(o.MqlRuntime, "oci.network.service", map[string]*llx.RawData{
			"id":          llx.StringDataPtr(s.Id),
			"name":        llx.StringDataPtr(s.Name),
			"cidrBlock":   llx.StringDataPtr(s.CidrBlock),
			"description": llx.StringDataPtr(s.Description),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}
	return res, nil
}

func (o *mqlOciNetworkService) id() (string, error) {
	return "oci.network.service/" + o.Id.Data, nil
}

// Typed route-rule targets

func (o *mqlOciNetworkRouteTable) routes() ([]any, error) {
	rules := o.GetRouteRules()
	if rules.Error != nil {
		return nil, rules.Error
	}

	res := make([]any, 0, len(rules.Data))
	for i, raw := range rules.Data {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		targetID, _ := m["networkEntityId"].(string)
		destination, _ := m["destination"].(string)
		destinationType, _ := m["destinationType"].(string)
		description, _ := m["description"].(string)
		routeType, _ := m["routeType"].(string)

		mqlInstance, err := CreateResource(o.MqlRuntime, "oci.network.routeTable.route", map[string]*llx.RawData{
			"__id":            llx.StringData(fmt.Sprintf("%s/route/%d", o.Id.Data, i)),
			"destination":     llx.StringData(destination),
			"destinationType": llx.StringData(destinationType),
			"routeType":       llx.StringData(routeType),
			"description":     llx.StringData(description),
			"targetType":      llx.StringData(ociRouteTargetType(targetID)),
			"targetId":        llx.StringData(targetID),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}
	return res, nil
}

func (o *mqlOciNetworkRouteTableRoute) internetGateway() (*mqlOciNetworkInternetGateway, error) {
	if o.TargetType.Data != "INTERNET_GATEWAY" || !isOcid(o.TargetId.Data) {
		o.InternetGateway.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.network.internetGateway", map[string]*llx.RawData{
		"id": llx.StringData(o.TargetId.Data),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciNetworkInternetGateway), nil
}

func (o *mqlOciNetworkRouteTableRoute) natGateway() (*mqlOciNetworkNatGateway, error) {
	if o.TargetType.Data != "NAT_GATEWAY" || !isOcid(o.TargetId.Data) {
		o.NatGateway.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.network.natGateway", map[string]*llx.RawData{
		"id": llx.StringData(o.TargetId.Data),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciNetworkNatGateway), nil
}

func (o *mqlOciNetworkRouteTableRoute) serviceGateway() (*mqlOciNetworkServiceGateway, error) {
	if o.TargetType.Data != "SERVICE_GATEWAY" || !isOcid(o.TargetId.Data) {
		o.ServiceGateway.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.network.serviceGateway", map[string]*llx.RawData{
		"id": llx.StringData(o.TargetId.Data),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciNetworkServiceGateway), nil
}

func (o *mqlOciNetworkRouteTableRoute) drg() (*mqlOciNetworkDrg, error) {
	if o.TargetType.Data != "DRG" || !isOcid(o.TargetId.Data) {
		o.Drg.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.network.drg", map[string]*llx.RawData{
		"id": llx.StringData(o.TargetId.Data),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciNetworkDrg), nil
}

func (o *mqlOciNetworkRouteTableRoute) localPeeringGateway() (*mqlOciNetworkLocalPeeringGateway, error) {
	if o.TargetType.Data != "LOCAL_PEERING_GATEWAY" || !isOcid(o.TargetId.Data) {
		o.LocalPeeringGateway.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.network.localPeeringGateway", map[string]*llx.RawData{
		"id": llx.StringData(o.TargetId.Data),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciNetworkLocalPeeringGateway), nil
}

// Init functions enabling route-rule targets (and other references) to resolve
// internet and NAT gateways by OCID via NewResource.

func initOciNetworkInternetGateway(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	idVal := ociArgString(args, "id")
	if idVal == "" {
		return nil, nil, errors.New("id required to fetch oci.network.internetGateway")
	}

	obj, err := CreateResource(runtime, "oci.network", nil)
	if err != nil {
		return nil, nil, err
	}
	network := obj.(*mqlOciNetwork)

	raws := network.GetInternetGateways()
	if raws.Error != nil {
		return nil, nil, raws.Error
	}
	for _, raw := range raws.Data {
		igw := raw.(*mqlOciNetworkInternetGateway)
		if igw.Id.Data == idVal {
			return args, igw, nil
		}
	}
	return nil, nil, errors.New("oci.network.internetGateway not found: " + idVal)
}

func initOciNetworkNatGateway(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	idVal := ociArgString(args, "id")
	if idVal == "" {
		return nil, nil, errors.New("id required to fetch oci.network.natGateway")
	}

	obj, err := CreateResource(runtime, "oci.network", nil)
	if err != nil {
		return nil, nil, err
	}
	network := obj.(*mqlOciNetwork)

	raws := network.GetNatGateways()
	if raws.Error != nil {
		return nil, nil, raws.Error
	}
	for _, raw := range raws.Data {
		ngw := raw.(*mqlOciNetworkNatGateway)
		if ngw.Id.Data == idVal {
			return args, ngw, nil
		}
	}
	return nil, nil, errors.New("oci.network.natGateway not found: " + idVal)
}
