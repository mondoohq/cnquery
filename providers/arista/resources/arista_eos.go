// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/aristanetworks/goeapi/module"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/arista/connection"
	"go.mondoo.com/mql/v13/providers/arista/resources/eos"
	"go.mondoo.com/mql/v13/types"
)

func aristaClient(runtime *plugin.Runtime) *eos.Eos {
	conn := runtime.Connection.(*connection.AristaConnection)
	return eos.NewEos(conn.Client())
}

func (a *mqlAristaEos) id() (string, error) {
	return "arista.eos", nil
}

func (v *mqlAristaEosIpInterface) id() (string, error) {
	return v.Name.Data, v.Name.Error
}

func (v *mqlAristaEosUser) id() (string, error) {
	return v.Name.Data, v.Name.Error
}

func (v *mqlAristaEosRole) id() (string, error) {
	return v.Name.Data, v.Name.Error
}

func (v *mqlAristaEosRunningConfig) id() (string, error) {
	return "arista.eos.runningConfig", nil
}

type mqlAristaEosRunningConfigInternal struct {
	contentFetched bool
	contentCache   string
	lock           sync.Mutex
}

func (a *mqlAristaEosRunningConfig) fetchContent() string {
	if a.contentFetched {
		return a.contentCache
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.contentFetched {
		return a.contentCache
	}
	eosClient := aristaClient(a.MqlRuntime)
	a.contentCache = eosClient.RunningConfig()
	a.contentFetched = true
	return a.contentCache
}

func (a *mqlAristaEosRunningConfig) content() (string, error) {
	return a.fetchContent(), nil
}

func (a *mqlAristaEosRunningConfigSection) id() (string, error) {
	if a.Name.Error != nil {
		return "", a.Name.Error
	}
	return "arista.eos.runningConfig.section " + a.Name.Data, nil
}

type mqlAristaEosRunningConfigSectionInternal struct {
	runningConfig string
}

func (a *mqlAristaEosRunningConfigSection) content() (string, error) {
	if a.Name.Error != nil {
		return "", a.Name.Error
	}
	name := a.Name.Data

	// Use cached running config passed from parent, or fetch directly
	content := a.runningConfig
	if content == "" {
		eosClient := aristaClient(a.MqlRuntime)
		content = eosClient.RunningConfig()
	}

	return eos.GetSection(strings.NewReader(content), name), nil
}

func (a *mqlAristaEos) systemConfig() (map[string]any, error) {
	eos := aristaClient(a.MqlRuntime)
	config := eos.SystemConfig()

	res := map[string]any{}
	for k := range config {
		res[k] = config[k]
	}

	return res, nil
}

func (a *mqlAristaEos) users() ([]any, error) {
	eos := aristaClient(a.MqlRuntime)
	users := eos.Users()

	mqlUsers := make([]any, len(users))
	for i, user := range users {
		mqlUser, err := CreateResource(a.MqlRuntime, "arista.eos.user", map[string]*llx.RawData{
			"name":       llx.StringData(user.UserName()),
			"privilege":  llx.StringData(user.Privilege()),
			"role":       llx.StringData(user.Role()),
			"nopassword": llx.StringData(user.Nopassword()),
			"format":     llx.StringData(user.Format()),
			"secret":     llx.StringData(user.Secret()),
			"sshkey":     llx.StringData(user.SSHKey()),
		})
		if err != nil {
			return nil, err
		}
		mqlUsers[i] = mqlUser
	}

	return mqlUsers, nil
}

func (a *mqlAristaEosUser) locked() (bool, error) {
	// A user is considered locked if they have no password and no SSH key
	if a.Nopassword.Error != nil {
		return false, a.Nopassword.Error
	}
	if a.Secret.Error != nil {
		return false, a.Secret.Error
	}
	if a.Sshkey.Error != nil {
		return false, a.Sshkey.Error
	}

	hasNoPassword := a.Nopassword.Data == "nopassword"
	hasSecret := a.Secret.Data != ""
	hasSSHKey := a.Sshkey.Data != ""

	// User is locked if they explicitly have nopassword and no SSH key
	return hasNoPassword && !hasSecret && !hasSSHKey, nil
}

func (a *mqlAristaEos) roles() ([]any, error) {
	eos := aristaClient(a.MqlRuntime)
	roles, err := eos.Roles()
	if err != nil {
		return nil, err
	}

	lumRoles := make([]any, len(roles))
	for i, role := range roles {

		rules, err := convert.JsonToDictSlice(role.Rules)
		if err != nil {
			return nil, err
		}

		mqlRole, err := CreateResource(a.MqlRuntime, "arista.eos.role", map[string]*llx.RawData{
			"name":    llx.StringData(role.Name),
			"default": llx.BoolData(role.Default),
			"rules":   llx.DictData(rules),
		})
		if err != nil {
			return nil, err
		}
		lumRoles[i] = mqlRole
	}
	return lumRoles, nil
}

func (a *mqlAristaEos) ntp() (*mqlAristaEosNtpSetting, error) {
	eos := aristaClient(a.MqlRuntime)

	ntp, err := eos.NtpStatus()
	if err != nil {
		return nil, err
	}

	res, err := CreateResource(a.MqlRuntime, "arista.eos.ntpSetting", map[string]*llx.RawData{
		"status": llx.StringData(ntp.Status),
	})
	if err != nil {
		return nil, err
	}

	return res.(*mqlAristaEosNtpSetting), nil
}

func (v *mqlAristaEosNtpSetting) id() (string, error) {
	return "arista.eos.ntpSetting", nil
}

func (a *mqlAristaEos) snmp() (*mqlAristaEosSnmpSetting, error) {
	eos := aristaClient(a.MqlRuntime)

	snmp, err := eos.Snmp()
	if err != nil {
		return nil, err
	}

	res, err := CreateResource(a.MqlRuntime, "arista.eos.snmpSetting", map[string]*llx.RawData{
		"enabled": llx.BoolData(snmp.Enabled),
	})
	if err != nil {
		return nil, err
	}

	return res.(*mqlAristaEosSnmpSetting), nil
}

func (v *mqlAristaEosSnmpSetting) id() (string, error) {
	return "arista.eos.snmpSetting", nil
}

func (a *mqlAristaEosSnmpSetting) notifications() ([]any, error) {
	eos := aristaClient(a.MqlRuntime)
	notifications, err := eos.SnmpNotifications()
	if err != nil {
		return nil, err
	}

	return convert.JsonToDictSlice(notifications)
}

func (a *mqlAristaEos) ipInterfaces() ([]any, error) {
	eos := aristaClient(a.MqlRuntime)
	ifaces := eos.IPInterfaces()

	mqlIfaces := make([]any, len(ifaces))
	for i, iface := range ifaces {
		mqlService, err := CreateResource(a.MqlRuntime, "arista.eos.ipInterface", map[string]*llx.RawData{
			"name":    llx.StringData(iface.Name()),
			"address": llx.StringData(iface.Address()),
			"mtu":     llx.StringData(iface.Mtu()),
		})
		if err != nil {
			return nil, err
		}
		mqlIfaces[i] = mqlService
	}

	return mqlIfaces, nil
}

func (a *mqlAristaEos) version() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AristaConnection)
	version, err := conn.GetVersion()
	if err != nil {
		return nil, err
	}
	return convert.JsonToDict(version)
}

func (a *mqlAristaEos) hostname() (string, error) {
	eos := aristaClient(a.MqlRuntime)

	hostname, err := eos.ShowHostname()
	if err != nil {
		return "", err
	}

	return hostname.Hostname, nil
}

func (a *mqlAristaEos) fqdn() (string, error) {
	eos := aristaClient(a.MqlRuntime)

	hostname, err := eos.ShowHostname()
	if err != nil {
		return "", err
	}

	return hostname.Fqdn, nil
}

type mqlAristaEosInterfaceInternal struct {
	cachedStatus map[string]any
}

func (a *mqlAristaEos) interfaces() ([]any, error) {
	eosClient := aristaClient(a.MqlRuntime)
	ifaces := eosClient.ShowInterface()

	// Fetch interface status once for all interfaces
	allStatus, err := eosClient.ShowInterfacesStatus()
	if err != nil {
		allStatus = nil
	}

	mqlIfaces := []any{}
	for k := range ifaces.Interfaces {
		iface := ifaces.Interfaces[k]

		address := []any{}
		for i := range iface.InterfaceAddress {
			ifaceAddress, err := convert.JsonToDict(iface.InterfaceAddress[i])
			if err != nil {
				return nil, err
			}
			address = append(address, ifaceAddress)
		}

		counters, err := convert.JsonToDict(iface.InterfaceCounters)
		if err != nil {
			return nil, err
		}

		statistics, err := convert.JsonToDict(iface.InterfaceStatistics)
		if err != nil {
			return nil, err
		}

		mqlIface, err := CreateResource(a.MqlRuntime, "arista.eos.interface", map[string]*llx.RawData{
			"name":                      llx.StringData(iface.Name),
			"bandwidth":                 llx.IntData(int64(iface.Bandwidth)),
			"burnedInAddress":           llx.StringData(iface.BurnedInAddress),
			"description":               llx.StringData(iface.Description),
			"forwardingModel":           llx.StringData(iface.ForwardingModel),
			"hardware":                  llx.StringData(iface.Hardware),
			"interfaceAddress":          llx.ArrayData(address, types.Dict),
			"interfaceCounters":         llx.DictData(counters),
			"interfaceMembership":       llx.StringData(iface.InterfaceMembership),
			"interfaceStatistics":       llx.DictData(statistics),
			"interfaceStatus":           llx.StringData(iface.InterfaceStatus),
			"l2Mtu":                     llx.IntData(int64(iface.L2Mtu)),
			"lastStatusChangeTimestamp": llx.IntData(int64(iface.LastStatusChangeTimestamp)),
			"lineProtocolStatus":        llx.StringData(iface.LineProtocolStatus),
			"mtu":                       llx.IntData(int64(iface.Mtu)),
			"physicalAddress":           llx.StringData(iface.PhysicalAddress),
		})
		if err != nil {
			return nil, err
		}

		// Cache the pre-fetched status for this interface
		if allStatus != nil {
			if entry, ok := allStatus[iface.Name]; ok {
				statusDict, err := convert.JsonToDict(entry)
				if err == nil {
					mqlIface.(*mqlAristaEosInterface).cachedStatus = statusDict
				}
			}
		}

		mqlIfaces = append(mqlIfaces, mqlIface)

	}
	return mqlIfaces, nil
}

func (a *mqlAristaEosInterface) id() (string, error) {
	return a.Name.Data, a.Name.Error
}

func (a *mqlAristaEosInterface) status() (map[string]any, error) {
	// Return cached status if available (pre-fetched in interfaces())
	if a.cachedStatus != nil {
		return a.cachedStatus, nil
	}

	// Fallback for standalone interface lookups
	eosClient := aristaClient(a.MqlRuntime)

	if a.Name.Error != nil {
		return nil, a.Name.Error
	}
	ifaceName := a.Name.Data

	status, err := eosClient.ShowInterfacesStatus()
	if err != nil {
		return nil, err
	}

	entry, ok := status[ifaceName]
	if !ok {
		return nil, nil
	}

	return convert.JsonToDict(entry)
}

func (a *mqlAristaEosInterface) enabled() (bool, error) {
	if a.InterfaceStatus.Error != nil {
		return false, a.InterfaceStatus.Error
	}
	// Interface is enabled if status is not "disabled"
	return a.InterfaceStatus.Data != "disabled", nil
}

func (a *mqlAristaEosInterface) duplex() (string, error) {
	tv := a.GetStatus()
	if tv.Error != nil {
		return "", tv.Error
	}
	if tv.Data == nil {
		return "", nil
	}
	if duplex, ok := tv.Data.(map[string]any)["duplex"].(string); ok {
		return duplex, nil
	}
	return "", nil
}

func (a *mqlAristaEosInterface) autoNegotiate() (bool, error) {
	tv := a.GetStatus()
	if tv.Error != nil {
		return false, tv.Error
	}
	if tv.Data == nil {
		return false, nil
	}
	if autoNeg, ok := tv.Data.(map[string]any)["autoNegotiateActive"].(bool); ok {
		return autoNeg, nil
	}
	return false, nil
}

func (a *mqlAristaEosStp) id() (string, error) {
	return "arista.eos.stp", nil
}

var aristaMstInstanceID = regexp.MustCompile(`(\d+)$`)

func (a *mqlAristaEosStp) mstInstances() ([]any, error) {
	eosClient := aristaClient(a.MqlRuntime)

	mstInstances, err := eosClient.Stp()
	if err != nil {
		return nil, err
	}

	// Pre-fetch all STP interface details to avoid N+1 API calls.
	// StpInterfaceDetails is called per-interface for counters() and features();
	// by fetching once per (instance, interface) pair here, we cache the results.
	type stpDetailKey struct {
		instanceID string
		ifaceName  string
	}
	stpDetails := map[stpDetailKey]*eos.SptMestInterfaceDetail{}
	for mstk := range mstInstances {
		m := aristaMstInstanceID.FindStringSubmatch(mstk)
		if m == nil {
			continue
		}
		for ifacek := range mstInstances[mstk].Interfaces {
			detail, err := eosClient.StpInterfaceDetails(m[1], ifacek)
			if err == nil {
				stpDetails[stpDetailKey{m[1], ifacek}] = &detail
			}
		}
	}

	res := []any{}

	for mstk := range mstInstances {
		mstInstance := mstInstances[mstk]

		m := aristaMstInstanceID.FindStringSubmatch(mstk)

		bridge, err := convert.JsonToDict(mstInstance.Bridge)
		if err != nil {
			return nil, err
		}

		rootBridge, err := convert.JsonToDict(mstInstance.RootBridge)
		if err != nil {
			return nil, err
		}

		regionalRootBridge, err := convert.JsonToDict(mstInstance.RegionalRootBridge)
		if err != nil {
			return nil, err
		}

		sptmstInterfaces := []any{}
		for ifacek := range mstInstance.Interfaces {
			iface := mstInstance.Interfaces[ifacek]

			inconsistentFeatures, err := convert.JsonToDict(iface.InconsistentFeatures)
			if err != nil {
				return nil, err
			}

			detail, err := convert.JsonToDict(iface.Detail)
			if err != nil {
				return nil, err
			}

			mqlArista, err := CreateResource(a.MqlRuntime, "arista.eos.spt.mstInterface", map[string]*llx.RawData{
				"id":                   llx.StringData(mstk + "/" + ifacek),
				"mstInstanceId":        llx.StringData(m[1]),
				"name":                 llx.StringData(ifacek),
				"priority":             llx.IntData(iface.Priority),
				"linkType":             llx.StringData(iface.LinkType),
				"state":                llx.StringData(iface.State),
				"cost":                 llx.IntData(int64(iface.Cost)),
				"role":                 llx.StringData(iface.Role),
				"inconsistentFeatures": llx.DictData(inconsistentFeatures),
				"portNumber":           llx.IntData(int64(iface.PortNumber)),
				"isEdgePort":           llx.BoolData(iface.IsEdgePort),
				"detail":               llx.DictData(detail),
				"boundaryType":         llx.StringData(iface.State),
			})
			if err != nil {
				return nil, err
			}

			// Cache pre-fetched counters and features on the interface resource
			if d, ok := stpDetails[stpDetailKey{m[1], ifacek}]; ok {
				mqlIface := mqlArista.(*mqlAristaEosSptMstInterface)
				if counters, err := convert.JsonToDict(d.Counters); err == nil {
					mqlIface.cachedCounters = counters
				}
				if features, err := convert.JsonToDict(d.Features); err == nil {
					mqlIface.cachedFeatures = features
				}
			}

			sptmstInterfaces = append(sptmstInterfaces, mqlArista)
		}

		mqlArista, err := CreateResource(a.MqlRuntime, "arista.eos.stp.mst", map[string]*llx.RawData{
			"instanceId":         llx.StringData(m[1]),
			"name":               llx.StringData(mstk),
			"protocol":           llx.StringData(mstInstance.Protocol),
			"bridge":             llx.DictData(bridge),
			"rootBridge":         llx.DictData(rootBridge),
			"regionalRootBridge": llx.DictData(regionalRootBridge),
			"interfaces":         llx.ArrayData(sptmstInterfaces, types.Resource("arista.eos.spt.mstInterface")),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlArista)
	}
	return res, nil
}

func (a *mqlAristaEosStpMst) id() (string, error) {
	return a.Name.Data, a.Name.Error
}

func (a *mqlAristaEosSptMstInterface) id() (string, error) {
	return a.Id.Data, a.Id.Error
}

type mqlAristaEosSptMstInterfaceInternal struct {
	cachedCounters map[string]any
	cachedFeatures map[string]any
}

func (a *mqlAristaEosSptMstInterface) counters() (map[string]any, error) {
	// Return cached counters if available (pre-fetched in mstInstances())
	if a.cachedCounters != nil {
		return a.cachedCounters, nil
	}

	// Fallback for standalone lookups
	eosClient := aristaClient(a.MqlRuntime)

	if a.MstInstanceId.Error != nil {
		return nil, a.MstInstanceId.Error
	}
	mstInstanceId := a.MstInstanceId.Data

	if a.Name.Error != nil {
		return nil, a.Name.Error
	}
	name := a.Name.Data

	mstInstanceDetails, err := eosClient.StpInterfaceDetails(mstInstanceId, name)
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(mstInstanceDetails.Counters)
}

func (a *mqlAristaEosSptMstInterface) features() (map[string]any, error) {
	// Return cached features if available (pre-fetched in mstInstances())
	if a.cachedFeatures != nil {
		return a.cachedFeatures, nil
	}

	// Fallback for standalone lookups
	eosClient := aristaClient(a.MqlRuntime)

	if a.MstInstanceId.Error != nil {
		return nil, a.MstInstanceId.Error
	}
	mstInstanceId := a.MstInstanceId.Data

	if a.Name.Error != nil {
		return nil, a.Name.Error
	}
	name := a.Name.Data

	mstInstanceDetails, err := eosClient.StpInterfaceDetails(mstInstanceId, name)
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(mstInstanceDetails.Features)
}

type mqlAristaEosVlanInternal struct {
	cachedInterfaces map[string]eos.VlanInterface
}

func (v *mqlAristaEosVlan) id() (string, error) {
	return "arista.eos.vlan/" + v.Id.Data, v.Id.Error
}

func (a *mqlAristaEos) vlans() ([]any, error) {
	eosClient := aristaClient(a.MqlRuntime)
	vlans := eosClient.Vlans()

	// Fetch show vlan data once for dynamic field and interfaces
	showVlans, err := eosClient.ShowVlans()
	if err != nil {
		showVlans = nil
	}

	res := make([]any, 0, len(vlans))
	for vid, vlanCfg := range vlans {
		trunkGroups := vlanCfg.TrunkGroups()
		trunkGroupsData := make([]any, len(trunkGroups))
		for i, tg := range trunkGroups {
			trunkGroupsData[i] = tg
		}

		dynamic := false
		if showVlans != nil {
			if sv, ok := showVlans[vid]; ok {
				dynamic = sv.Dynamic
			}
		}

		mqlVlan, err := CreateResource(a.MqlRuntime, "arista.eos.vlan", map[string]*llx.RawData{
			"id":          llx.StringData(vid),
			"name":        llx.StringData(vlanCfg.Name()),
			"state":       llx.StringData(vlanCfg.State()),
			"trunkGroups": llx.ArrayData(trunkGroupsData, types.String),
			"dynamic":     llx.BoolData(dynamic),
		})
		if err != nil {
			return nil, err
		}

		// Cache the show vlan interfaces for this VLAN to avoid re-fetching
		if showVlans != nil {
			if sv, ok := showVlans[vid]; ok {
				mqlVlan.(*mqlAristaEosVlan).cachedInterfaces = sv.Interfaces
			}
		}

		res = append(res, mqlVlan)
	}

	return res, nil
}

func (a *mqlAristaEosVlan) interfaces() ([]any, error) {
	// Use cached interfaces if available (pre-fetched in vlans())
	if a.cachedInterfaces != nil {
		interfaces := make([]any, 0, len(a.cachedInterfaces))
		for name := range a.cachedInterfaces {
			interfaces = append(interfaces, name)
		}
		sort.Slice(interfaces, func(i, j int) bool {
			return interfaces[i].(string) < interfaces[j].(string)
		})
		return interfaces, nil
	}

	// Fallback for standalone VLAN lookups
	eosClient := aristaClient(a.MqlRuntime)

	if a.Id.Error != nil {
		return nil, a.Id.Error
	}
	vlanId := a.Id.Data

	showVlans, err := eosClient.ShowVlans()
	if err != nil {
		return nil, err
	}

	sv, ok := showVlans[vlanId]
	if !ok {
		return []any{}, nil
	}

	interfaces := make([]any, 0, len(sv.Interfaces))
	for name := range sv.Interfaces {
		interfaces = append(interfaces, name)
	}
	sort.Slice(interfaces, func(i, j int) bool {
		return interfaces[i].(string) < interfaces[j].(string)
	})

	return interfaces, nil
}

func (v *mqlAristaEosRoute) id() (string, error) {
	return "arista.eos.route/" + v.Vrf.Data + "/" + v.Destination.Data, v.Destination.Error
}

func (a *mqlAristaEos) routes() ([]any, error) {
	eosClient := aristaClient(a.MqlRuntime)
	routeTable, err := eosClient.ShowIPRoute()
	if err != nil {
		return nil, err
	}

	res := []any{}
	for vrfName, vrf := range routeTable.VRFs {
		for dest, route := range vrf.Routes {
			nextHops := make([]any, len(route.Vias))
			for i, via := range route.Vias {
				hop, err := convert.JsonToDict(via)
				if err != nil {
					return nil, err
				}
				nextHops[i] = hop
			}

			mqlRoute, err := CreateResource(a.MqlRuntime, "arista.eos.route", map[string]*llx.RawData{
				"destination":        llx.StringData(dest),
				"vrf":                llx.StringData(vrfName),
				"routeType":          llx.StringData(route.RouteType),
				"preference":         llx.IntData(int64(route.Preference)),
				"metric":             llx.IntData(int64(route.Metric)),
				"hardwareProgrammed": llx.BoolData(route.HardwareProgrammed),
				"kernelProgrammed":   llx.BoolData(route.KernelProgrammed),
				"routeAction":        llx.StringData(route.RouteAction),
				"nextHops":           llx.ArrayData(nextHops, types.Dict),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRoute)
		}
	}

	return res, nil
}

func (a *mqlAristaEosRoute) active() (bool, error) {
	// A route is considered active if it's programmed in hardware or kernel
	if a.HardwareProgrammed.Error != nil {
		log.Warn().Err(a.HardwareProgrammed.Error).Str("route", a.Destination.Data).Msg("could not determine hardware programming status")
		return false, nil
	}
	if a.KernelProgrammed.Error != nil {
		log.Warn().Err(a.KernelProgrammed.Error).Str("route", a.Destination.Data).Msg("could not determine kernel programming status")
		return false, nil
	}
	return a.HardwareProgrammed.Data || a.KernelProgrammed.Data, nil
}

func (v *mqlAristaEosSwitchport) id() (string, error) {
	return "arista.eos.switchport/" + v.Name.Data, v.Name.Error
}

func (a *mqlAristaEos) switchports() ([]any, error) {
	eosClient := aristaClient(a.MqlRuntime)
	switchports := eosClient.Switchports()

	res := make([]any, 0, len(switchports))
	for name, spCfg := range switchports {
		trunkGroups := spCfg.TrunkGroups()
		trunkGroupsData := make([]any, len(trunkGroups))
		for i, tg := range trunkGroups {
			trunkGroupsData[i] = tg
		}

		mqlSwitchport, err := CreateResource(a.MqlRuntime, "arista.eos.switchport", map[string]*llx.RawData{
			"name":              llx.StringData(name),
			"mode":              llx.StringData(spCfg.Mode()),
			"accessVlan":        llx.StringData(spCfg.AccessVlan()),
			"trunkNativeVlan":   llx.StringData(spCfg.TrunkNativeVlan()),
			"trunkAllowedVlans": llx.StringData(spCfg.TrunkAllowedVlans()),
			"trunkGroups":       llx.ArrayData(trunkGroupsData, types.String),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlSwitchport)
	}

	return res, nil
}

func (v *mqlAristaEosBgp) id() (string, error) {
	return "arista.eos.bgp", nil
}

type mqlAristaEosBgpInternal struct {
	configFetched bool
	bgpConfig     *module.BgpConfig
	lock          sync.Mutex
}

func (a *mqlAristaEosBgp) fetchConfig() *module.BgpConfig {
	if a.configFetched {
		return a.bgpConfig
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.configFetched {
		return a.bgpConfig
	}
	eosClient := aristaClient(a.MqlRuntime)
	a.bgpConfig = eosClient.BGPConfig()
	a.configFetched = true
	return a.bgpConfig
}

func (a *mqlAristaEosBgp) enabled() (bool, error) {
	cfg := a.fetchConfig()
	if cfg == nil {
		return false, nil
	}
	return cfg.Shutdown() != "true", nil
}

func (a *mqlAristaEosBgp) asNumber() (string, error) {
	cfg := a.fetchConfig()
	if cfg == nil {
		return "", nil
	}
	return cfg.BgpAs(), nil
}

func (a *mqlAristaEosBgp) routerId() (string, error) {
	cfg := a.fetchConfig()
	if cfg == nil {
		return "", nil
	}
	return cfg.RouterID(), nil
}

func (a *mqlAristaEos) bgp() (*mqlAristaEosBgp, error) {
	res, err := CreateResource(a.MqlRuntime, "arista.eos.bgp", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAristaEosBgp), nil
}

func (a *mqlAristaEosBgp) vrfs() ([]any, error) {
	eosClient := aristaClient(a.MqlRuntime)

	summary, err := eosClient.BGPSummary()
	if err != nil {
		return nil, err
	}

	// Fetch neighbor details once for all VRFs
	neighbors, err := eosClient.BGPNeighbors()
	if err != nil {
		neighbors = nil
	}

	// Pre-build neighbor lookup maps to avoid O(n²) linear searches
	type neighborInfo struct {
		Description      string
		InboundRouteMap  string
		OutboundRouteMap string
	}
	// neighborsByVrf maps vrfName -> peerAddress -> neighborInfo
	neighborsByVrf := map[string]map[string]neighborInfo{}
	if neighbors != nil {
		for vrfName, vrfNeighbors := range neighbors.VRFs {
			peerMap := make(map[string]neighborInfo, len(vrfNeighbors.PeerList))
			for _, neighbor := range vrfNeighbors.PeerList {
				peerMap[neighbor.PeerAddress] = neighborInfo{
					Description:      neighbor.Description,
					InboundRouteMap:  neighbor.InboundRouteMap,
					OutboundRouteMap: neighbor.OutboundRouteMap,
				}
			}
			neighborsByVrf[vrfName] = peerMap
		}
	}

	res := []any{}
	for vrfName, vrfData := range summary.VRFs {
		// Build peers for this VRF
		peers := []any{}
		for peerAddr, peerData := range vrfData.Peers {
			var description, inRouteMap, outRouteMap string
			if peerMap, ok := neighborsByVrf[vrfName]; ok {
				if info, ok := peerMap[peerAddr]; ok {
					description = info.Description
					inRouteMap = info.InboundRouteMap
					outRouteMap = info.OutboundRouteMap
				}
			}

			mqlPeer, err := CreateResource(a.MqlRuntime, "arista.eos.bgp.peer", map[string]*llx.RawData{
				"vrfName":          llx.StringData(vrfName),
				"peerAddress":      llx.StringData(peerAddr),
				"remoteAs":         llx.StringData(peerData.ASN),
				"state":            llx.StringData(peerData.PeerState),
				"uptime":           llx.IntData(int64(peerData.UpDownTime)), // EOS reports uptime in whole seconds; sub-second precision is not meaningful
				"prefixesReceived": llx.IntData(peerData.PrefixReceived),
				"prefixesAccepted": llx.IntData(peerData.PrefixAccepted),
				"inboundRouteMap":  llx.StringData(inRouteMap),
				"outboundRouteMap": llx.StringData(outRouteMap),
				"description":      llx.StringData(description),
			})
			if err != nil {
				return nil, err
			}
			peers = append(peers, mqlPeer)
		}

		mqlVrf, err := CreateResource(a.MqlRuntime, "arista.eos.bgp.vrf", map[string]*llx.RawData{
			"name":     llx.StringData(vrfName),
			"routerId": llx.StringData(vrfData.RouterID),
			"asNumber": llx.StringData(strconv.FormatInt(vrfData.ASN, 10)),
			"peers":    llx.ArrayData(peers, types.Resource("arista.eos.bgp.peer")),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlVrf)
	}

	return res, nil
}

func (v *mqlAristaEosBgpVrf) id() (string, error) {
	return "arista.eos.bgp.vrf/" + v.Name.Data, v.Name.Error
}

func (v *mqlAristaEosBgpPeer) id() (string, error) {
	if v.VrfName.Error != nil {
		return "", v.VrfName.Error
	}
	return "arista.eos.bgp.peer/" + v.VrfName.Data + "/" + v.PeerAddress.Data, v.PeerAddress.Error
}

// MLAG resource implementations

func (v *mqlAristaEosMlag) id() (string, error) {
	return "arista.eos.mlag", nil
}

type mqlAristaEosMlagInternal struct {
	configFetched bool
	mlagConfig    *module.MlagConfig
	lock          sync.Mutex
}

func (a *mqlAristaEosMlag) fetchConfig() *module.MlagConfig {
	if a.configFetched {
		return a.mlagConfig
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.configFetched {
		return a.mlagConfig
	}
	eosClient := aristaClient(a.MqlRuntime)
	a.mlagConfig = eosClient.MlagConfig()
	a.configFetched = true
	return a.mlagConfig
}

func (a *mqlAristaEosMlag) domainId() (string, error) {
	cfg := a.fetchConfig()
	if cfg == nil {
		return "", nil
	}
	return cfg.DomainID(), nil
}

func (a *mqlAristaEosMlag) localInterface() (string, error) {
	cfg := a.fetchConfig()
	if cfg == nil {
		return "", nil
	}
	return cfg.LocalInterface(), nil
}

func (a *mqlAristaEosMlag) peerAddress() (string, error) {
	cfg := a.fetchConfig()
	if cfg == nil {
		return "", nil
	}
	return cfg.PeerAddress(), nil
}

func (a *mqlAristaEosMlag) peerLink() (string, error) {
	cfg := a.fetchConfig()
	if cfg == nil {
		return "", nil
	}
	return cfg.PeerLink(), nil
}

func (a *mqlAristaEosMlag) shutdown() (bool, error) {
	cfg := a.fetchConfig()
	if cfg == nil {
		return false, nil
	}
	return cfg.Shutdown() == "true", nil
}

func (a *mqlAristaEos) mlag() (*mqlAristaEosMlag, error) {
	res, err := CreateResource(a.MqlRuntime, "arista.eos.mlag", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAristaEosMlag), nil
}

func (a *mqlAristaEosMlag) interfaces() ([]any, error) {
	eosClient := aristaClient(a.MqlRuntime)
	mlagConfig := eosClient.MlagConfig()

	if mlagConfig == nil {
		return []any{}, nil
	}

	// Use the cached running config from the runningConfig resource if available,
	// otherwise fetch directly
	var runningConfig string
	rcRes, err := CreateResource(a.MqlRuntime, "arista.eos.runningConfig", map[string]*llx.RawData{})
	if err == nil {
		runningConfig = rcRes.(*mqlAristaEosRunningConfig).fetchContent()
	} else {
		runningConfig = eosClient.RunningConfig()
	}
	mlagInterfaces := eos.ParseMlagInterfaces(runningConfig)

	res := make([]any, 0, len(mlagInterfaces))
	for _, intf := range mlagInterfaces {
		mqlIntf, err := CreateResource(a.MqlRuntime, "arista.eos.mlag.interface", map[string]*llx.RawData{
			"name":   llx.StringData(intf.Name),
			"mlagId": llx.StringData(intf.MlagID),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlIntf)
	}

	return res, nil
}

func (v *mqlAristaEosMlagInterface) id() (string, error) {
	return "arista.eos.mlag.interface/" + v.Name.Data, v.Name.Error
}

// ACL resource implementations

func (v *mqlAristaEosAcl) id() (string, error) {
	return "arista.eos.acl/" + v.Name.Data, v.Name.Error
}

type mqlAristaEosAclInternal struct {
	cachedEntries module.AclEntryMap
}

func (a *mqlAristaEos) acls() ([]any, error) {
	eosClient := aristaClient(a.MqlRuntime)
	aclConfigs := eosClient.AclConfigs()

	res := []any{}
	for name, aclConfig := range aclConfigs {
		mqlAcl, err := CreateResource(a.MqlRuntime, "arista.eos.acl", map[string]*llx.RawData{
			"name": llx.StringData(name),
			"type": llx.StringData(aclConfig.Type()),
		})
		if err != nil {
			return nil, err
		}

		// Cache the entries to avoid re-fetching all ACLs in entries()
		mqlAcl.(*mqlAristaEosAcl).cachedEntries = aclConfig.Entries()

		res = append(res, mqlAcl)
	}

	return res, nil
}

func (a *mqlAristaEosAcl) entries() ([]any, error) {
	if a.Name.Error != nil {
		return nil, a.Name.Error
	}
	aclName := a.Name.Data

	// Use cached entries if available (pre-fetched in acls())
	rawEntries := a.cachedEntries
	if rawEntries == nil {
		// Fallback for standalone ACL lookups
		eosClient := aristaClient(a.MqlRuntime)
		aclConfigs := eosClient.AclConfigs()

		aclConfig, ok := aclConfigs[aclName]
		if !ok {
			return []any{}, nil
		}
		rawEntries = aclConfig.Entries()
	}

	// Parse and sort entries by sequence number
	parsed := make([]eos.AclEntryParsed, 0, len(rawEntries))
	for seqNum, entry := range rawEntries {
		p, err := eos.ParseAclEntry(seqNum, entry.Action(), entry.SrcAddr(), entry.SrcLen(), entry.Log())
		if err != nil {
			log.Warn().Err(err).Str("acl", aclName).Msg("skipping invalid ACL entry")
			continue
		}
		parsed = append(parsed, p)
	}
	eos.SortAclEntries(parsed)

	res := make([]any, 0, len(parsed))
	for _, p := range parsed {
		mqlEntry, err := CreateResource(a.MqlRuntime, "arista.eos.acl.entry", map[string]*llx.RawData{
			"aclName":        llx.StringData(aclName),
			"sequenceNumber": llx.IntData(int64(p.SequenceNumber)),
			"action":         llx.StringData(p.Action),
			"srcAddress":     llx.StringData(p.SrcAddress),
			"srcPrefixLen":   llx.IntData(int64(p.SrcPrefixLen)),
			"log":            llx.BoolData(p.Log),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlEntry)
	}

	return res, nil
}

func (v *mqlAristaEosAclEntry) id() (string, error) {
	if v.AclName.Error != nil {
		return "", v.AclName.Error
	}
	return "arista.eos.acl.entry/" + v.AclName.Data + "/" + strconv.FormatInt(v.SequenceNumber.Data, 10), v.SequenceNumber.Error
}

// Hardware resource implementations

func (v *mqlAristaEosHardware) id() (string, error) {
	return "arista.eos.hardware", nil
}

func (a *mqlAristaEos) hardware() (*mqlAristaEosHardware, error) {
	res, err := CreateResource(a.MqlRuntime, "arista.eos.hardware", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAristaEosHardware), nil
}

func (a *mqlAristaEosHardware) powerSupplies() ([]any, error) {
	eosClient := aristaClient(a.MqlRuntime)

	envPower, err := eosClient.ShowEnvironmentPower()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(envPower.PowerSupplies))
	for name, ps := range envPower.PowerSupplies {
		// Build temp sensor dicts
		tempSensors := make([]any, 0, len(ps.TempSensors))
		for sensorName, sensor := range ps.TempSensors {
			tempSensors = append(tempSensors, map[string]any{
				"name":        sensorName,
				"status":      sensor.Status,
				"temperature": sensor.Temperature,
			})
		}

		// Build fan dicts
		fans := make([]any, 0, len(ps.Fans))
		for fanName, fan := range ps.Fans {
			fans = append(fans, map[string]any{
				"name":   fanName,
				"status": fan.Status,
				"speed":  fan.Speed,
			})
		}

		mqlPSU, err := CreateResource(a.MqlRuntime, "arista.eos.hardware.powerSupply", map[string]*llx.RawData{
			"name":          llx.StringData(name),
			"state":         llx.StringData(ps.State),
			"modelName":     llx.StringData(ps.ModelName),
			"capacity":      llx.IntData(int64(ps.Capacity)),
			"outputPower":   llx.FloatData(ps.OutputPower),
			"inputCurrent":  llx.FloatData(ps.InputCurrent),
			"outputCurrent": llx.FloatData(ps.OutputCurrent),
			"uptime":        llx.FloatData(ps.Uptime),
			"managed":       llx.BoolData(ps.Managed),
			"tempSensors":   llx.ArrayData(tempSensors, types.Dict),
			"fans":          llx.ArrayData(fans, types.Dict),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlPSU)
	}

	return res, nil
}

func (v *mqlAristaEosHardwarePowerSupply) id() (string, error) {
	return "arista.eos.hardware.powerSupply/" + v.Name.Data, v.Name.Error
}

func (a *mqlAristaEosHardware) fans() ([]any, error) {
	eosClient := aristaClient(a.MqlRuntime)

	cooling, err := eosClient.ShowEnvironmentCooling()
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, tray := range cooling.FanTraySlots {
		for _, fan := range tray.Fans {
			mqlFan, err := CreateResource(a.MqlRuntime, "arista.eos.hardware.fan", map[string]*llx.RawData{
				"name":            llx.StringData(tray.Label + "/" + fan.Label),
				"trayLabel":       llx.StringData(tray.Label),
				"status":          llx.StringData(fan.Status),
				"speed":           llx.IntData(int64(fan.Speed)),
				"configuredSpeed": llx.IntData(int64(fan.ConfiguredSpeed)),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlFan)
		}
	}

	return res, nil
}

func (v *mqlAristaEosHardwareFan) id() (string, error) {
	return "arista.eos.hardware.fan/" + v.Name.Data, v.Name.Error
}

func (a *mqlAristaEosHardware) inventory() ([]any, error) {
	eosClient := aristaClient(a.MqlRuntime)

	inv, err := eosClient.ShowInventory()
	if err != nil {
		return nil, err
	}

	res := []any{}

	// Add the system/chassis as an inventory item
	if inv.SystemInformation.Name != "" || inv.SystemInformation.SerialNum != "" {
		mqlItem, err := CreateResource(a.MqlRuntime, "arista.eos.hardware.inventoryItem", map[string]*llx.RawData{
			"name":             llx.StringData(inv.SystemInformation.Name),
			"description":      llx.StringData(inv.SystemInformation.Description),
			"serialNumber":     llx.StringData(inv.SystemInformation.SerialNum),
			"manufacturerDate": llx.StringData(inv.SystemInformation.MfgDate),
			"hardwareRevision": llx.StringData(inv.SystemInformation.HardwareRev),
			"category":         llx.StringData("system"),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlItem)
	}

	// Add all hardware slots (PSUs, fan trays, transceivers, line cards)
	type slotCategory struct {
		slots    map[string]eos.InventoryEntry
		category string
	}
	slotMaps := []slotCategory{
		{inv.PowerSupplySlots, "powerSupply"},
		{inv.FanTraySlots, "fanTray"},
		{inv.XcvrSlots, "transceiver"},
		{inv.CardSlots, "card"},
	}
	for _, sc := range slotMaps {
		for slotName, entry := range sc.slots {
			name := entry.Name
			if name == "" {
				name = slotName
			}
			mqlItem, err := CreateResource(a.MqlRuntime, "arista.eos.hardware.inventoryItem", map[string]*llx.RawData{
				"name":             llx.StringData(name),
				"description":      llx.StringData(entry.Description),
				"serialNumber":     llx.StringData(entry.SerialNum),
				"manufacturerDate": llx.StringData(entry.MfgDate),
				"hardwareRevision": llx.StringData(entry.HardwareRev),
				"category":         llx.StringData(sc.category),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlItem)
		}
	}

	return res, nil
}

func (v *mqlAristaEosHardwareInventoryItem) id() (string, error) {
	if v.Category.Error != nil {
		return "", v.Category.Error
	}
	if v.SerialNumber.Error != nil {
		return "", v.SerialNumber.Error
	}
	return "arista.eos.hardware.inventoryItem/" + v.Category.Data + "/" + v.Name.Data + "/" + v.SerialNumber.Data, v.Name.Error
}
