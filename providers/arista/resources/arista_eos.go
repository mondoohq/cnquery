// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"regexp"
	"strings"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/arista/connection"
	"go.mondoo.com/cnquery/v11/providers/arista/resources/eos"
	"go.mondoo.com/cnquery/v11/types"
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

func (a *mqlAristaEosRunningConfig) content() (string, error) {
	eos := aristaClient(a.MqlRuntime)
	return eos.RunningConfig(), nil
}

func (a *mqlAristaEosRunningConfigSection) id() (string, error) {
	if a.Name.Error != nil {
		return "", a.Name.Error
	}
	return "arista.eos.runningConfig.section " + a.Name.Data, nil
}

func (a *mqlAristaEosRunningConfigSection) content() (string, error) {
	eosClient := aristaClient(a.MqlRuntime)

	if a.Name.Error != nil {
		return "", a.Name.Error
	}
	name := a.Name.Data

	// todo: use content from arista.eos.runningconfig
	content := eosClient.RunningConfig()

	return eos.GetSection(strings.NewReader(content), name), nil
}

func (a *mqlAristaEos) systemConfig() (map[string]interface{}, error) {
	eos := aristaClient(a.MqlRuntime)
	config := eos.SystemConfig()

	res := map[string]interface{}{}
	for k := range config {
		res[k] = config[k]
	}

	return res, nil
}

func (a *mqlAristaEos) users() ([]interface{}, error) {
	eos := aristaClient(a.MqlRuntime)
	users := eos.Users()

	mqlUsers := make([]interface{}, len(users))
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

func (a *mqlAristaEos) roles() ([]interface{}, error) {
	eos := aristaClient(a.MqlRuntime)
	roles, err := eos.Roles()
	if err != nil {
		return nil, err
	}

	lumRoles := make([]interface{}, len(roles))
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

func (a *mqlAristaEosSnmpSetting) notifications() ([]interface{}, error) {
	eos := aristaClient(a.MqlRuntime)
	notifications, err := eos.SnmpNotifications()
	if err != nil {
		return nil, err
	}

	return convert.JsonToDictSlice(notifications)
}

func (a *mqlAristaEos) ipInterfaces() ([]interface{}, error) {
	eos := aristaClient(a.MqlRuntime)
	ifaces := eos.IPInterfaces()

	mqlIfaces := make([]interface{}, len(ifaces))
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

func (a *mqlAristaEos) version() (map[string]interface{}, error) {
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

func (a *mqlAristaEos) interfaces() ([]interface{}, error) {
	eos := aristaClient(a.MqlRuntime)
	ifaces := eos.ShowInterface()

	mqlIfaces := []interface{}{}
	for k := range ifaces.Interfaces {
		iface := ifaces.Interfaces[k]

		address := []interface{}{}
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
		mqlIfaces = append(mqlIfaces, mqlIface)

	}
	return mqlIfaces, nil
}

func (a *mqlAristaEosInterface) id() (string, error) {
	return a.Name.Data, a.Name.Error
}

func (a *mqlAristaEosInterface) status() (map[string]interface{}, error) {
	eos := aristaClient(a.MqlRuntime)

	if a.Name.Error != nil {
		return nil, a.Name.Error
	}
	ifaceName := a.Name.Data

	status, err := eos.ShowInterfacesStatus()
	if err != nil {
		return nil, err
	}

	entry, ok := status[ifaceName]
	if !ok {
		return nil, nil
	}

	return convert.JsonToDict(entry)
}

func (a *mqlAristaEosStp) id() (string, error) {
	return "arista.eos.stp", nil
}

var aristaMstInstanceID = regexp.MustCompile(`(\d+)$`)

func (a *mqlAristaEosStp) mstInstances() ([]interface{}, error) {
	eos := aristaClient(a.MqlRuntime)

	mstInstances, err := eos.Stp()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}

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

		sptmstInterfaces := []interface{}{}
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

func (a *mqlAristaEosSptMstInterface) counters() (map[string]interface{}, error) {
	eos := aristaClient(a.MqlRuntime)

	if a.MstInstanceId.Error != nil {
		return nil, a.MstInstanceId.Error
	}
	mstInstanceId := a.MstInstanceId.Data

	if a.Name.Error != nil {
		return nil, a.Name.Error
	}
	name := a.Name.Data

	mstInstanceDetails, err := eos.StpInterfaceDetails(mstInstanceId, name)
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(mstInstanceDetails.Counters)
}

func (a *mqlAristaEosSptMstInterface) features() (map[string]interface{}, error) {
	eos := aristaClient(a.MqlRuntime)

	if a.MstInstanceId.Error != nil {
		return nil, a.MstInstanceId.Error
	}
	mstInstanceId := a.MstInstanceId.Data

	if a.Name.Error != nil {
		return nil, a.Name.Error
	}
	name := a.Name.Data

	mstInstanceDetails, err := eos.StpInterfaceDetails(mstInstanceId, name)
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(mstInstanceDetails.Features)
}
