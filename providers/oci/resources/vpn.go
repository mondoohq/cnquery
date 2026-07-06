// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

// Customer-premises equipment

func (o *mqlOciNetwork) cpes() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	regions, err := o.regionResources()
	if err != nil {
		return nil, err
	}
	return runNetworkPool(o.getCpes(conn, regions))
}

func (o *mqlOciNetwork) getCpes(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			regionKey := regionResource.Id.Data
			log.Debug().Msgf("calling oci CPEs with region %s", regionKey)

			svc, err := conn.NetworkClient(regionKey)
			if err != nil {
				return nil, err
			}

			cpes := []core.Cpe{}
			var page *string
			for {
				response, err := svc.ListCpes(ctx, core.ListCpesRequest{
					CompartmentId: common.String(conn.TenantID()),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}
				cpes = append(cpes, response.Items...)
				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			var res []any
			for i := range cpes {
				cpe := cpes[i]

				var created *time.Time
				if cpe.TimeCreated != nil {
					created = &cpe.TimeCreated.Time
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.network.cpe", map[string]*llx.RawData{
					"id":               llx.StringDataPtr(cpe.Id),
					"name":             llx.StringDataPtr(cpe.DisplayName),
					"compartmentID":    llx.StringDataPtr(cpe.CompartmentId),
					"ipAddress":        llx.StringDataPtr(cpe.IpAddress),
					"cpeDeviceShapeId": llx.StringDataPtr(cpe.CpeDeviceShapeId),
					"isPrivate":        llx.BoolDataPtr(cpe.IsPrivate),
					"created":          llx.TimeDataPtr(created),
					"freeformTags":     llx.MapData(strMapToAny(cpe.FreeformTags), types.String),
					"definedTags":      llx.MapData(definedTagsToAny(cpe.DefinedTags), types.Any),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlInstance)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func initOciNetworkCpe(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch oci.network.cpe")
	}
	idVal := args["id"].Value.(string)

	obj, err := CreateResource(runtime, "oci.network", nil)
	if err != nil {
		return nil, nil, err
	}
	network := obj.(*mqlOciNetwork)

	raws := network.GetCpes()
	if raws.Error != nil {
		return nil, nil, raws.Error
	}
	for _, raw := range raws.Data {
		cpe := raw.(*mqlOciNetworkCpe)
		if cpe.Id.Data == idVal {
			return args, cpe, nil
		}
	}
	return nil, nil, errors.New("oci.network.cpe not found: " + idVal)
}

func (o *mqlOciNetworkCpe) id() (string, error) {
	return "oci.network.cpe/" + o.Id.Data, nil
}

func (o *mqlOciNetworkCpe) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

// IPSec connections

type mqlOciNetworkIpsecConnectionInternal struct {
	region     string
	cacheCpeId string
	cacheDrgId string
}

func (o *mqlOciNetwork) ipsecConnections() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	regions, err := o.regionResources()
	if err != nil {
		return nil, err
	}
	return runNetworkPool(o.getIpsecConnections(conn, regions))
}

func (o *mqlOciNetwork) getIpsecConnections(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			regionKey := regionResource.Id.Data
			log.Debug().Msgf("calling oci IPSec connections with region %s", regionKey)

			svc, err := conn.NetworkClient(regionKey)
			if err != nil {
				return nil, err
			}

			conns := []core.IpSecConnection{}
			var page *string
			for {
				response, err := svc.ListIPSecConnections(ctx, core.ListIPSecConnectionsRequest{
					CompartmentId: common.String(conn.TenantID()),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}
				conns = append(conns, response.Items...)
				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			var res []any
			for i := range conns {
				ipsc := conns[i]

				var created *time.Time
				if ipsc.TimeCreated != nil {
					created = &ipsc.TimeCreated.Time
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.network.ipsecConnection", map[string]*llx.RawData{
					"id":                     llx.StringDataPtr(ipsc.Id),
					"name":                   llx.StringDataPtr(ipsc.DisplayName),
					"compartmentID":          llx.StringDataPtr(ipsc.CompartmentId),
					"staticRoutes":           llx.ArrayData(stringsToAny(ipsc.StaticRoutes), types.String),
					"cpeLocalIdentifier":     llx.StringDataPtr(ipsc.CpeLocalIdentifier),
					"cpeLocalIdentifierType": llx.StringData(string(ipsc.CpeLocalIdentifierType)),
					"transportType":          llx.StringData(string(ipsc.TransportType)),
					"state":                  llx.StringData(string(ipsc.LifecycleState)),
					"created":                llx.TimeDataPtr(created),
					"freeformTags":           llx.MapData(strMapToAny(ipsc.FreeformTags), types.String),
					"definedTags":            llx.MapData(definedTagsToAny(ipsc.DefinedTags), types.Any),
				})
				if err != nil {
					return nil, err
				}
				mqlIpsc := mqlInstance.(*mqlOciNetworkIpsecConnection)
				mqlIpsc.region = regionKey
				mqlIpsc.cacheCpeId = stringValue(ipsc.CpeId)
				mqlIpsc.cacheDrgId = stringValue(ipsc.DrgId)
				res = append(res, mqlIpsc)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func initOciNetworkIpsecConnection(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch oci.network.ipsecConnection")
	}
	idVal := args["id"].Value.(string)

	obj, err := CreateResource(runtime, "oci.network", nil)
	if err != nil {
		return nil, nil, err
	}
	network := obj.(*mqlOciNetwork)

	raws := network.GetIpsecConnections()
	if raws.Error != nil {
		return nil, nil, raws.Error
	}
	for _, raw := range raws.Data {
		ipsc := raw.(*mqlOciNetworkIpsecConnection)
		if ipsc.Id.Data == idVal {
			return args, ipsc, nil
		}
	}
	return nil, nil, errors.New("oci.network.ipsecConnection not found: " + idVal)
}

func (o *mqlOciNetworkIpsecConnection) id() (string, error) {
	return "oci.network.ipsecConnection/" + o.Id.Data, nil
}

func (o *mqlOciNetworkIpsecConnection) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

func (o *mqlOciNetworkIpsecConnection) cpe() (*mqlOciNetworkCpe, error) {
	if !isOcid(o.cacheCpeId) {
		o.Cpe.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.network.cpe", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheCpeId),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciNetworkCpe), nil
}

func (o *mqlOciNetworkIpsecConnection) drg() (*mqlOciNetworkDrg, error) {
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

func (o *mqlOciNetworkIpsecConnection) tunnels() ([]any, error) {
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

	tunnels := []core.IpSecConnectionTunnel{}
	var page *string
	for {
		response, err := svc.ListIPSecConnectionTunnels(ctx, core.ListIPSecConnectionTunnelsRequest{
			IpscId: common.String(o.Id.Data),
			Page:   page,
		})
		if err != nil {
			return nil, err
		}
		tunnels = append(tunnels, response.Items...)
		if response.OpcNextPage == nil {
			break
		}
		page = response.OpcNextPage
	}

	res := make([]any, 0, len(tunnels))
	for i := range tunnels {
		tunnel := tunnels[i]

		var created *time.Time
		if tunnel.TimeCreated != nil {
			created = &tunnel.TimeCreated.Time
		}

		bgpState := ""
		if tunnel.BgpSessionInfo != nil {
			bgpState = string(tunnel.BgpSessionInfo.BgpState)
		}

		mqlInstance, err := CreateResource(o.MqlRuntime, "oci.network.ipsecConnectionTunnel", map[string]*llx.RawData{
			"id":                    llx.StringDataPtr(tunnel.Id),
			"name":                  llx.StringDataPtr(tunnel.DisplayName),
			"compartmentID":         llx.StringDataPtr(tunnel.CompartmentId),
			"status":                llx.StringData(string(tunnel.Status)),
			"ikeVersion":            llx.StringData(string(tunnel.IkeVersion)),
			"routing":               llx.StringData(string(tunnel.Routing)),
			"oracleCanInitiate":     llx.StringData(string(tunnel.OracleCanInitiate)),
			"natTranslationEnabled": llx.StringData(string(tunnel.NatTranslationEnabled)),
			"dpdMode":               llx.StringData(string(tunnel.DpdMode)),
			"vpnIp":                 llx.StringDataPtr(tunnel.VpnIp),
			"cpeIp":                 llx.StringDataPtr(tunnel.CpeIp),
			"bgpState":              llx.StringData(bgpState),
			"state":                 llx.StringData(string(tunnel.LifecycleState)),
			"created":               llx.TimeDataPtr(created),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}
	return res, nil
}

func (o *mqlOciNetworkIpsecConnectionTunnel) id() (string, error) {
	return "oci.network.ipsecConnectionTunnel/" + o.Id.Data, nil
}

func (o *mqlOciNetworkIpsecConnectionTunnel) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

// FastConnect virtual circuits

type mqlOciNetworkVirtualCircuitInternal struct {
	cacheDrgId string
}

// crossConnectMapping is the subset of an OCI virtual-circuit cross-connect
// mapping we surface to dict. It deliberately omits BgpMd5AuthKey (a secret).
type crossConnectMapping struct {
	CrossConnectOrCrossConnectGroupId string `json:"crossConnectOrCrossConnectGroupId,omitempty"`
	Vlan                              *int   `json:"vlan,omitempty"`
	CustomerBgpPeeringIp              string `json:"customerBgpPeeringIp,omitempty"`
	OracleBgpPeeringIp                string `json:"oracleBgpPeeringIp,omitempty"`
	CustomerBgpPeeringIpv6            string `json:"customerBgpPeeringIpv6,omitempty"`
	OracleBgpPeeringIpv6              string `json:"oracleBgpPeeringIpv6,omitempty"`
}

func (o *mqlOciNetwork) virtualCircuits() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	regions, err := o.regionResources()
	if err != nil {
		return nil, err
	}
	return runNetworkPool(o.getVirtualCircuits(conn, regions))
}

func (o *mqlOciNetwork) getVirtualCircuits(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			regionKey := regionResource.Id.Data
			log.Debug().Msgf("calling oci virtual circuits with region %s", regionKey)

			svc, err := conn.NetworkClient(regionKey)
			if err != nil {
				return nil, err
			}

			vcs := []core.VirtualCircuit{}
			var page *string
			for {
				response, err := svc.ListVirtualCircuits(ctx, core.ListVirtualCircuitsRequest{
					CompartmentId: common.String(conn.TenantID()),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}
				vcs = append(vcs, response.Items...)
				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			var res []any
			for i := range vcs {
				vc := vcs[i]

				var created *time.Time
				if vc.TimeCreated != nil {
					created = &vc.TimeCreated.Time
				}

				mappings := make([]crossConnectMapping, 0, len(vc.CrossConnectMappings))
				for j := range vc.CrossConnectMappings {
					m := vc.CrossConnectMappings[j]
					mappings = append(mappings, crossConnectMapping{
						CrossConnectOrCrossConnectGroupId: stringValue(m.CrossConnectOrCrossConnectGroupId),
						Vlan:                              m.Vlan,
						CustomerBgpPeeringIp:              stringValue(m.CustomerBgpPeeringIp),
						OracleBgpPeeringIp:                stringValue(m.OracleBgpPeeringIp),
						CustomerBgpPeeringIpv6:            stringValue(m.CustomerBgpPeeringIpv6),
						OracleBgpPeeringIpv6:              stringValue(m.OracleBgpPeeringIpv6),
					})
				}
				mappingDicts, err := convert.JsonToDictSlice(mappings)
				if err != nil {
					return nil, err
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.network.virtualCircuit", map[string]*llx.RawData{
					"id":                   llx.StringDataPtr(vc.Id),
					"name":                 llx.StringDataPtr(vc.DisplayName),
					"compartmentID":        llx.StringDataPtr(vc.CompartmentId),
					"type":                 llx.StringData(string(vc.Type)),
					"serviceType":          llx.StringData(string(vc.ServiceType)),
					"bandwidthShapeName":   llx.StringDataPtr(vc.BandwidthShapeName),
					"bgpManagement":        llx.StringData(string(vc.BgpManagement)),
					"bgpSessionState":      llx.StringData(string(vc.BgpSessionState)),
					"bgpAdminState":        llx.StringData(string(vc.BgpAdminState)),
					"providerName":         llx.StringDataPtr(vc.ProviderName),
					"providerServiceName":  llx.StringDataPtr(vc.ProviderServiceName),
					"publicPrefixes":       llx.ArrayData(stringsToAny(vc.PublicPrefixes), types.String),
					"crossConnectMappings": llx.ArrayData(mappingDicts, types.Dict),
					"state":                llx.StringData(string(vc.LifecycleState)),
					"created":              llx.TimeDataPtr(created),
					"freeformTags":         llx.MapData(strMapToAny(vc.FreeformTags), types.String),
					"definedTags":          llx.MapData(definedTagsToAny(vc.DefinedTags), types.Any),
				})
				if err != nil {
					return nil, err
				}
				mqlVc := mqlInstance.(*mqlOciNetworkVirtualCircuit)
				if gw := stringValue(vc.GatewayId); ociResourceTypeFromOCID(gw) == "drg" {
					mqlVc.cacheDrgId = gw
				}
				res = append(res, mqlVc)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func initOciNetworkVirtualCircuit(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch oci.network.virtualCircuit")
	}
	idVal := args["id"].Value.(string)

	obj, err := CreateResource(runtime, "oci.network", nil)
	if err != nil {
		return nil, nil, err
	}
	network := obj.(*mqlOciNetwork)

	raws := network.GetVirtualCircuits()
	if raws.Error != nil {
		return nil, nil, raws.Error
	}
	for _, raw := range raws.Data {
		vc := raw.(*mqlOciNetworkVirtualCircuit)
		if vc.Id.Data == idVal {
			return args, vc, nil
		}
	}
	return nil, nil, errors.New("oci.network.virtualCircuit not found: " + idVal)
}

func (o *mqlOciNetworkVirtualCircuit) id() (string, error) {
	return "oci.network.virtualCircuit/" + o.Id.Data, nil
}

func (o *mqlOciNetworkVirtualCircuit) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

func (o *mqlOciNetworkVirtualCircuit) drg() (*mqlOciNetworkDrg, error) {
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

// FastConnect cross-connects

func (o *mqlOciNetwork) crossConnects() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	regions, err := o.regionResources()
	if err != nil {
		return nil, err
	}
	return runNetworkPool(o.getCrossConnects(conn, regions))
}

func (o *mqlOciNetwork) getCrossConnects(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			regionKey := regionResource.Id.Data
			log.Debug().Msgf("calling oci cross-connects with region %s", regionKey)

			svc, err := conn.NetworkClient(regionKey)
			if err != nil {
				return nil, err
			}

			xcs := []core.CrossConnect{}
			var page *string
			for {
				response, err := svc.ListCrossConnects(ctx, core.ListCrossConnectsRequest{
					CompartmentId: common.String(conn.TenantID()),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}
				xcs = append(xcs, response.Items...)
				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			var res []any
			for i := range xcs {
				xc := xcs[i]

				var created *time.Time
				if xc.TimeCreated != nil {
					created = &xc.TimeCreated.Time
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.network.crossConnect", map[string]*llx.RawData{
					"id":                  llx.StringDataPtr(xc.Id),
					"name":                llx.StringDataPtr(xc.DisplayName),
					"compartmentID":       llx.StringDataPtr(xc.CompartmentId),
					"locationName":        llx.StringDataPtr(xc.LocationName),
					"portName":            llx.StringDataPtr(xc.PortName),
					"portSpeedShapeName":  llx.StringDataPtr(xc.PortSpeedShapeName),
					"crossConnectGroupId": llx.StringDataPtr(xc.CrossConnectGroupId),
					"state":               llx.StringData(string(xc.LifecycleState)),
					"created":             llx.TimeDataPtr(created),
					"freeformTags":        llx.MapData(strMapToAny(xc.FreeformTags), types.String),
					"definedTags":         llx.MapData(definedTagsToAny(xc.DefinedTags), types.Any),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlInstance)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (o *mqlOciNetworkCrossConnect) id() (string, error) {
	return "oci.network.crossConnect/" + o.Id.Data, nil
}

func (o *mqlOciNetworkCrossConnect) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}
