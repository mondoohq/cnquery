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
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

// Customer-premises equipment

func (o *mqlOciNetwork) cpes() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	return ociCollect(o.MqlRuntime, ociScopeAllCompartments,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			regionKey := region
			log.Debug().Msgf("calling oci CPEs with region %s", regionKey)

			svc, err := conn.NetworkClient(regionKey)
			if err != nil {
				return nil, err
			}

			cpes, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]core.Cpe, *string, error) {
				response, err := svc.ListCpes(ctx, core.ListCpesRequest{
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

			return res, nil
		})
}

func initOciNetworkCpe(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	idVal := ociArgString(args, "id")
	if idVal == "" {
		return nil, nil, errors.New("id required to fetch oci.network.cpe")
	}

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
	cacheRegion string
	cacheCpeID  string
	cacheDrgID  string
}

func (o *mqlOciNetwork) ipsecConnections() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	return ociCollect(o.MqlRuntime, ociScopeAllCompartments,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			regionKey := region
			log.Debug().Msgf("calling oci IPSec connections with region %s", regionKey)

			svc, err := conn.NetworkClient(regionKey)
			if err != nil {
				return nil, err
			}

			conns, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]core.IpSecConnection, *string, error) {
				response, err := svc.ListIPSecConnections(ctx, core.ListIPSecConnectionsRequest{
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
				mqlIpsc.cacheRegion = regionKey
				mqlIpsc.cacheCpeID = stringValue(ipsc.CpeId)
				mqlIpsc.cacheDrgID = stringValue(ipsc.DrgId)
				res = append(res, mqlIpsc)
			}

			return res, nil
		})
}

func initOciNetworkIpsecConnection(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	idVal := ociArgString(args, "id")
	if idVal == "" {
		return nil, nil, errors.New("id required to fetch oci.network.ipsecConnection")
	}

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
	if !isOcid(o.cacheCpeID) {
		o.Cpe.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.network.cpe", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheCpeID),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciNetworkCpe), nil
}

func (o *mqlOciNetworkIpsecConnection) drg() (*mqlOciNetworkDrg, error) {
	if !isOcid(o.cacheDrgID) {
		o.Drg.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.network.drg", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheDrgID),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciNetworkDrg), nil
}

func (o *mqlOciNetworkIpsecConnection) tunnels() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	region := o.cacheRegion
	if region == "" {
		region = ociRegionFromOCID(o.Id.Data)
	}
	svc, err := conn.NetworkClient(region)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()

	tunnels, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]core.IpSecConnectionTunnel, *string, error) {
		response, err := svc.ListIPSecConnectionTunnels(ctx, core.ListIPSecConnectionTunnelsRequest{
			IpscId: common.String(o.Id.Data),
			Page:   page,
		})
		if err != nil {
			return nil, nil, err
		}
		return response.Items, response.OpcNextPage, nil
	})
	if err != nil {
		return nil, err
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
		tun := mqlInstance.(*mqlOciNetworkIpsecConnectionTunnel)
		tun.cacheIpscID = o.Id.Data
		tun.cacheRegion = region
		res = append(res, tun)
	}
	return res, nil
}

func (o *mqlOciNetworkIpsecConnectionTunnel) id() (string, error) {
	return "oci.network.ipsecConnectionTunnel/" + o.Id.Data, nil
}

type mqlOciNetworkIpsecConnectionTunnelInternal struct {
	cacheIpscID string
	cacheRegion string
	details     ociOnce
	phaseOne    *core.TunnelPhaseOneDetails
	phaseTwo    *core.TunnelPhaseTwoDetails
}

// fetchDetails lazily loads the tunnel detail, which carries the negotiated
// phase 1/2 crypto that the list call does not populate. The result is cached
// so the phase accessors share a single API call. A transient failure is not
// cached, so a later access retries rather than returning the stale error
// forever.
func (o *mqlOciNetworkIpsecConnectionTunnel) fetchDetails() error {
	return o.details.do(func() error {
		if o.cacheIpscID == "" {
			return nil
		}
		conn := o.MqlRuntime.Connection.(*connection.OciConnection)
		region := o.cacheRegion
		if region == "" {
			region = ociRegionFromOCID(o.Id.Data)
		}
		svc, err := conn.NetworkClient(region)
		if err != nil {
			return err
		}
		resp, err := svc.GetIPSecConnectionTunnel(context.Background(), core.GetIPSecConnectionTunnelRequest{
			IpscId:   common.String(o.cacheIpscID),
			TunnelId: common.String(o.Id.Data),
		})
		if err != nil {
			return err
		}
		o.phaseOne = resp.PhaseOneDetails
		o.phaseTwo = resp.PhaseTwoDetails
		return nil
	})
}

// ociTunnelCryptoValue reports a tunnel crypto parameter, marking the field
// explicitly null when OCI has no value for it.
//
// The negotiated parameters exist only once the tunnel establishes its security
// association, so a DOWN tunnel - the state a mismatched-proposal tunnel sits in,
// and precisely the one an audit needs to catch - has none. Returning "" for that
// made the natural denylist assertion (`phase1DhGroup != "GROUP2" && ...`) pass
// trivially on an unmeasured tunnel. Null makes those comparisons null instead.
func ociTunnelCryptoValue(field *plugin.TValue[string], v *string) (string, error) {
	if v == nil || *v == "" {
		field.State = plugin.StateIsSet | plugin.StateIsNull
		return "", nil
	}
	return *v, nil
}

// ociTunnelCryptoFlag is the boolean counterpart. Unlike the strings it resolves
// an absent value to false rather than null, because false is the failing
// direction for every flag here (PFS off, IKE/ESP not established).
func ociTunnelCryptoFlag(v *bool) bool {
	return v != nil && *v
}

func (o *mqlOciNetworkIpsecConnectionTunnel) phase1EncryptionAlgorithm() (string, error) {
	if err := o.fetchDetails(); err != nil {
		return "", err
	}
	var v *string
	if o.phaseOne != nil {
		v = o.phaseOne.NegotiatedEncryptionAlgorithm
	}
	return ociTunnelCryptoValue(&o.Phase1EncryptionAlgorithm, v)
}

func (o *mqlOciNetworkIpsecConnectionTunnel) phase1AuthenticationAlgorithm() (string, error) {
	if err := o.fetchDetails(); err != nil {
		return "", err
	}
	var v *string
	if o.phaseOne != nil {
		v = o.phaseOne.NegotiatedAuthenticationAlgorithm
	}
	return ociTunnelCryptoValue(&o.Phase1AuthenticationAlgorithm, v)
}

func (o *mqlOciNetworkIpsecConnectionTunnel) phase1DhGroup() (string, error) {
	if err := o.fetchDetails(); err != nil {
		return "", err
	}
	var v *string
	if o.phaseOne != nil {
		v = o.phaseOne.NegotiatedDhGroup
	}
	return ociTunnelCryptoValue(&o.Phase1DhGroup, v)
}

func (o *mqlOciNetworkIpsecConnectionTunnel) phase2EncryptionAlgorithm() (string, error) {
	if err := o.fetchDetails(); err != nil {
		return "", err
	}
	var v *string
	if o.phaseTwo != nil {
		v = o.phaseTwo.NegotiatedEncryptionAlgorithm
	}
	return ociTunnelCryptoValue(&o.Phase2EncryptionAlgorithm, v)
}

func (o *mqlOciNetworkIpsecConnectionTunnel) phase2AuthenticationAlgorithm() (string, error) {
	if err := o.fetchDetails(); err != nil {
		return "", err
	}
	var v *string
	if o.phaseTwo != nil {
		v = o.phaseTwo.NegotiatedAuthenticationAlgorithm
	}
	return ociTunnelCryptoValue(&o.Phase2AuthenticationAlgorithm, v)
}

func (o *mqlOciNetworkIpsecConnectionTunnel) phase2PfsEnabled() (bool, error) {
	if err := o.fetchDetails(); err != nil {
		return false, err
	}
	if o.phaseTwo == nil {
		return false, nil
	}
	return ociTunnelCryptoFlag(o.phaseTwo.IsPfsEnabled), nil
}

func (o *mqlOciNetworkIpsecConnectionTunnel) phase2DhGroup() (string, error) {
	if err := o.fetchDetails(); err != nil {
		return "", err
	}
	var v *string
	if o.phaseTwo != nil {
		v = o.phaseTwo.NegotiatedDhGroup
	}
	return ociTunnelCryptoValue(&o.Phase2DhGroup, v)
}

// The configured parameters, unlike the negotiated ones above, are present even
// while the tunnel is down, so they are the only way to audit the crypto of a
// tunnel that never came up.

func (o *mqlOciNetworkIpsecConnectionTunnel) phase1ConfiguredEncryptionAlgorithm() (string, error) {
	if err := o.fetchDetails(); err != nil {
		return "", err
	}
	var v *string
	if o.phaseOne != nil {
		v = o.phaseOne.CustomEncryptionAlgorithm
	}
	return ociTunnelCryptoValue(&o.Phase1ConfiguredEncryptionAlgorithm, v)
}

func (o *mqlOciNetworkIpsecConnectionTunnel) phase1ConfiguredAuthenticationAlgorithm() (string, error) {
	if err := o.fetchDetails(); err != nil {
		return "", err
	}
	var v *string
	if o.phaseOne != nil {
		v = o.phaseOne.CustomAuthenticationAlgorithm
	}
	return ociTunnelCryptoValue(&o.Phase1ConfiguredAuthenticationAlgorithm, v)
}

func (o *mqlOciNetworkIpsecConnectionTunnel) phase1ConfiguredDhGroup() (string, error) {
	if err := o.fetchDetails(); err != nil {
		return "", err
	}
	var v *string
	if o.phaseOne != nil {
		v = o.phaseOne.CustomDhGroup
	}
	return ociTunnelCryptoValue(&o.Phase1ConfiguredDhGroup, v)
}

func (o *mqlOciNetworkIpsecConnectionTunnel) phase2ConfiguredEncryptionAlgorithm() (string, error) {
	if err := o.fetchDetails(); err != nil {
		return "", err
	}
	var v *string
	if o.phaseTwo != nil {
		v = o.phaseTwo.CustomEncryptionAlgorithm
	}
	return ociTunnelCryptoValue(&o.Phase2ConfiguredEncryptionAlgorithm, v)
}

func (o *mqlOciNetworkIpsecConnectionTunnel) phase2ConfiguredAuthenticationAlgorithm() (string, error) {
	if err := o.fetchDetails(); err != nil {
		return "", err
	}
	var v *string
	if o.phaseTwo != nil {
		v = o.phaseTwo.CustomAuthenticationAlgorithm
	}
	return ociTunnelCryptoValue(&o.Phase2ConfiguredAuthenticationAlgorithm, v)
}

func (o *mqlOciNetworkIpsecConnectionTunnel) phase2ConfiguredDhGroup() (string, error) {
	if err := o.fetchDetails(); err != nil {
		return "", err
	}
	var v *string
	if o.phaseTwo != nil {
		v = o.phaseTwo.DhGroup
	}
	return ociTunnelCryptoValue(&o.Phase2ConfiguredDhGroup, v)
}

func (o *mqlOciNetworkIpsecConnectionTunnel) isCustomPhase1Config() (bool, error) {
	if err := o.fetchDetails(); err != nil {
		return false, err
	}
	if o.phaseOne == nil {
		return false, nil
	}
	return ociTunnelCryptoFlag(o.phaseOne.IsCustomPhaseOneConfig), nil
}

func (o *mqlOciNetworkIpsecConnectionTunnel) isCustomPhase2Config() (bool, error) {
	if err := o.fetchDetails(); err != nil {
		return false, err
	}
	if o.phaseTwo == nil {
		return false, nil
	}
	return ociTunnelCryptoFlag(o.phaseTwo.IsCustomPhaseTwoConfig), nil
}

func (o *mqlOciNetworkIpsecConnectionTunnel) isIkeEstablished() (bool, error) {
	if err := o.fetchDetails(); err != nil {
		return false, err
	}
	if o.phaseOne == nil {
		return false, nil
	}
	return ociTunnelCryptoFlag(o.phaseOne.IsIkeEstablished), nil
}

func (o *mqlOciNetworkIpsecConnectionTunnel) isEspEstablished() (bool, error) {
	if err := o.fetchDetails(); err != nil {
		return false, err
	}
	if o.phaseTwo == nil {
		return false, nil
	}
	return ociTunnelCryptoFlag(o.phaseTwo.IsEspEstablished), nil
}

func (o *mqlOciNetworkIpsecConnectionTunnel) phase1Lifetime() (int64, error) {
	if err := o.fetchDetails(); err != nil {
		return 0, err
	}
	if o.phaseOne == nil {
		o.Phase1Lifetime.State = plugin.StateIsSet | plugin.StateIsNull
		return 0, nil
	}
	return ociTunnelLifetime(&o.Phase1Lifetime, o.phaseOne.Lifetime)
}

func (o *mqlOciNetworkIpsecConnectionTunnel) phase2Lifetime() (int64, error) {
	if err := o.fetchDetails(); err != nil {
		return 0, err
	}
	if o.phaseTwo == nil {
		o.Phase2Lifetime.State = plugin.StateIsSet | plugin.StateIsNull
		return 0, nil
	}
	return ociTunnelLifetime(&o.Phase2Lifetime, o.phaseTwo.Lifetime)
}

// ociTunnelLifetime marks the field null rather than reporting 0 seconds, which
// would read as an absurdly short - and therefore passing - rekey interval.
func ociTunnelLifetime(field *plugin.TValue[int64], v *int64) (int64, error) {
	if v == nil {
		field.State = plugin.StateIsSet | plugin.StateIsNull
		return 0, nil
	}
	return *v, nil
}

func (o *mqlOciNetworkIpsecConnectionTunnel) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

// FastConnect virtual circuits

type mqlOciNetworkVirtualCircuitInternal struct {
	cacheDrgID string
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
	return ociCollect(o.MqlRuntime, ociScopeAllCompartments,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			regionKey := region
			log.Debug().Msgf("calling oci virtual circuits with region %s", regionKey)

			svc, err := conn.NetworkClient(regionKey)
			if err != nil {
				return nil, err
			}

			vcs, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]core.VirtualCircuit, *string, error) {
				response, err := svc.ListVirtualCircuits(ctx, core.ListVirtualCircuitsRequest{
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
					mqlVc.cacheDrgID = gw
				}
				res = append(res, mqlVc)
			}

			return res, nil
		})
}

func initOciNetworkVirtualCircuit(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	idVal := ociArgString(args, "id")
	if idVal == "" {
		return nil, nil, errors.New("id required to fetch oci.network.virtualCircuit")
	}

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
	if !isOcid(o.cacheDrgID) {
		o.Drg.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.network.drg", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheDrgID),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciNetworkDrg), nil
}

// FastConnect cross-connects

func (o *mqlOciNetwork) crossConnects() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	return ociCollect(o.MqlRuntime, ociScopeAllCompartments,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			regionKey := region
			log.Debug().Msgf("calling oci cross-connects with region %s", regionKey)

			svc, err := conn.NetworkClient(regionKey)
			if err != nil {
				return nil, err
			}

			xcs, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]core.CrossConnect, *string, error) {
				response, err := svc.ListCrossConnects(ctx, core.ListCrossConnectsRequest{
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

			return res, nil
		})
}

func (o *mqlOciNetworkCrossConnect) id() (string, error) {
	return "oci.network.crossConnect/" + o.Id.Data, nil
}

func (o *mqlOciNetworkCrossConnect) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}
