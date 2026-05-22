// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "go.mondoo.com/mql/v13/providers-sdk/v1/plugin"

// region/zone typed accessors. The matching `regionUrl`/`zoneUrl` string
// fields remain populated for now and are marked @maturity("deprecated") in
// the .lr schema. New audits should use the typed accessors so they can
// traverse the region or zone resource directly.

func (g *mqlGcpProjectComputeServiceAddress) region() (*mqlGcpProjectComputeServiceRegion, error) {
	if g.RegionUrl.Error != nil {
		return nil, g.RegionUrl.Error
	}
	r, err := getRegionByUrl(g.RegionUrl.Data, g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if r == nil {
		g.Region.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return r, nil
}

func (g *mqlGcpProjectComputeServiceForwardingRule) region() (*mqlGcpProjectComputeServiceRegion, error) {
	if g.RegionUrl.Error != nil {
		return nil, g.RegionUrl.Error
	}
	r, err := getRegionByUrl(g.RegionUrl.Data, g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if r == nil {
		g.Region.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return r, nil
}

func (g *mqlGcpProjectComputeServiceBackendService) region() (*mqlGcpProjectComputeServiceRegion, error) {
	if g.RegionUrl.Error != nil {
		return nil, g.RegionUrl.Error
	}
	r, err := getRegionByUrl(g.RegionUrl.Data, g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if r == nil {
		g.Region.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return r, nil
}

func (g *mqlGcpProjectComputeServiceSecurityPolicy) region() (*mqlGcpProjectComputeServiceRegion, error) {
	if g.RegionUrl.Error != nil {
		return nil, g.RegionUrl.Error
	}
	r, err := getRegionByUrl(g.RegionUrl.Data, g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if r == nil {
		g.Region.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return r, nil
}

func (g *mqlGcpProjectComputeServiceSslPolicy) region() (*mqlGcpProjectComputeServiceRegion, error) {
	if g.RegionUrl.Error != nil {
		return nil, g.RegionUrl.Error
	}
	r, err := getRegionByUrl(g.RegionUrl.Data, g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if r == nil {
		g.Region.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return r, nil
}

func (g *mqlGcpProjectComputeServiceSslCertificate) region() (*mqlGcpProjectComputeServiceRegion, error) {
	if g.RegionUrl.Error != nil {
		return nil, g.RegionUrl.Error
	}
	r, err := getRegionByUrl(g.RegionUrl.Data, g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if r == nil {
		g.Region.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return r, nil
}

func (g *mqlGcpProjectComputeServiceVpnGateway) region() (*mqlGcpProjectComputeServiceRegion, error) {
	if g.RegionUrl.Error != nil {
		return nil, g.RegionUrl.Error
	}
	r, err := getRegionByUrl(g.RegionUrl.Data, g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if r == nil {
		g.Region.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return r, nil
}

func (g *mqlGcpProjectComputeServiceVpnTunnel) region() (*mqlGcpProjectComputeServiceRegion, error) {
	if g.RegionUrl.Error != nil {
		return nil, g.RegionUrl.Error
	}
	r, err := getRegionByUrl(g.RegionUrl.Data, g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if r == nil {
		g.Region.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return r, nil
}

func (g *mqlGcpProjectComputeServiceInstanceGroup) zone() (*mqlGcpProjectComputeServiceZone, error) {
	if g.ZoneUrl.Error != nil {
		return nil, g.ZoneUrl.Error
	}
	z, err := getZoneByUrl(g.ZoneUrl.Data, g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if z == nil {
		g.Zone.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return z, nil
}

func (g *mqlGcpProjectComputeServiceInstanceGroupManager) region() (*mqlGcpProjectComputeServiceRegion, error) {
	if g.RegionUrl.Error != nil {
		return nil, g.RegionUrl.Error
	}
	r, err := getRegionByUrl(g.RegionUrl.Data, g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if r == nil {
		g.Region.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return r, nil
}

func (g *mqlGcpProjectComputeServiceInstanceGroupManager) zone() (*mqlGcpProjectComputeServiceZone, error) {
	if g.ZoneUrl.Error != nil {
		return nil, g.ZoneUrl.Error
	}
	z, err := getZoneByUrl(g.ZoneUrl.Data, g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if z == nil {
		g.Zone.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return z, nil
}

func (g *mqlGcpProjectComputeServiceFirewallPolicy) region() (*mqlGcpProjectComputeServiceRegion, error) {
	if g.RegionUrl.Error != nil {
		return nil, g.RegionUrl.Error
	}
	r, err := getRegionByUrl(g.RegionUrl.Data, g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if r == nil {
		g.Region.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return r, nil
}

func (g *mqlGcpProjectComputeServiceHealthCheck) region() (*mqlGcpProjectComputeServiceRegion, error) {
	if g.RegionUrl.Error != nil {
		return nil, g.RegionUrl.Error
	}
	r, err := getRegionByUrl(g.RegionUrl.Data, g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if r == nil {
		g.Region.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return r, nil
}

func (g *mqlGcpProjectComputeServiceUrlMap) region() (*mqlGcpProjectComputeServiceRegion, error) {
	if g.RegionUrl.Error != nil {
		return nil, g.RegionUrl.Error
	}
	r, err := getRegionByUrl(g.RegionUrl.Data, g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if r == nil {
		g.Region.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return r, nil
}

func (g *mqlGcpProjectComputeServiceTargetHttpProxy) region() (*mqlGcpProjectComputeServiceRegion, error) {
	if g.RegionUrl.Error != nil {
		return nil, g.RegionUrl.Error
	}
	r, err := getRegionByUrl(g.RegionUrl.Data, g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if r == nil {
		g.Region.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return r, nil
}

func (g *mqlGcpProjectComputeServiceTargetHttpsProxy) region() (*mqlGcpProjectComputeServiceRegion, error) {
	if g.RegionUrl.Error != nil {
		return nil, g.RegionUrl.Error
	}
	r, err := getRegionByUrl(g.RegionUrl.Data, g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if r == nil {
		g.Region.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return r, nil
}

func (g *mqlGcpProjectComputeServiceServiceAttachment) region() (*mqlGcpProjectComputeServiceRegion, error) {
	if g.RegionUrl.Error != nil {
		return nil, g.RegionUrl.Error
	}
	r, err := getRegionByUrl(g.RegionUrl.Data, g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if r == nil {
		g.Region.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return r, nil
}

func (g *mqlGcpProjectComputeServiceNetworkEndpointGroup) region() (*mqlGcpProjectComputeServiceRegion, error) {
	if g.RegionUrl.Error != nil {
		return nil, g.RegionUrl.Error
	}
	r, err := getRegionByUrl(g.RegionUrl.Data, g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if r == nil {
		g.Region.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return r, nil
}

func (g *mqlGcpProjectComputeServiceNetworkEndpointGroup) zone() (*mqlGcpProjectComputeServiceZone, error) {
	if g.ZoneUrl.Error != nil {
		return nil, g.ZoneUrl.Error
	}
	z, err := getZoneByUrl(g.ZoneUrl.Data, g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if z == nil {
		g.Zone.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return z, nil
}

func (g *mqlGcpProjectComputeServiceInterconnectAttachment) region() (*mqlGcpProjectComputeServiceRegion, error) {
	if g.RegionUrl.Error != nil {
		return nil, g.RegionUrl.Error
	}
	r, err := getRegionByUrl(g.RegionUrl.Data, g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if r == nil {
		g.Region.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return r, nil
}

func (g *mqlGcpProjectComputeServiceTargetTcpProxy) region() (*mqlGcpProjectComputeServiceRegion, error) {
	if g.RegionUrl.Error != nil {
		return nil, g.RegionUrl.Error
	}
	r, err := getRegionByUrl(g.RegionUrl.Data, g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if r == nil {
		g.Region.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return r, nil
}

func (g *mqlGcpProjectComputeServicePacketMirroring) region() (*mqlGcpProjectComputeServiceRegion, error) {
	if g.RegionUrl.Error != nil {
		return nil, g.RegionUrl.Error
	}
	r, err := getRegionByUrl(g.RegionUrl.Data, g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if r == nil {
		g.Region.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return r, nil
}

func (g *mqlGcpProjectComputeServiceTargetPool) region() (*mqlGcpProjectComputeServiceRegion, error) {
	if g.RegionUrl.Error != nil {
		return nil, g.RegionUrl.Error
	}
	r, err := getRegionByUrl(g.RegionUrl.Data, g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if r == nil {
		g.Region.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return r, nil
}
