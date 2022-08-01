package resources

import (
	"errors"
	"regexp"
	"strings"

	"go.mondoo.io/mondoo/lumi/resources/arista"
	"go.mondoo.io/mondoo/motor/providers"
	arista_transport "go.mondoo.io/mondoo/motor/providers/arista"
)

func aristaClientInstance(t providers.Transport) (*arista.Eos, *arista_transport.Transport, error) {
	at, ok := t.(*arista_transport.Transport)
	if !ok {
		return nil, nil, errors.New("arista.eos resource is not supported on this transport")
	}

	eos := arista.NewEos(at.Client())
	return eos, at, nil
}

func (a *lumiAristaEos) id() (string, error) {
	return "arista.eos", nil
}

func (v *lumiAristaEosIpInterface) id() (string, error) {
	return v.Name()
}

func (v *lumiAristaEosUser) id() (string, error) {
	return v.Name()
}

func (v *lumiAristaEosRole) id() (string, error) {
	return v.Name()
}

func (v *lumiAristaEosRunningConfig) id() (string, error) {
	return "arista.eos.runningConfig", nil
}

func (a *lumiAristaEosRunningConfig) GetContent() (string, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Transport)
	if err != nil {
		return "", err
	}

	return eos.RunningConfig(), nil
}

func (a *lumiAristaEosRunningConfigSection) id() (string, error) {
	name, err := a.Name()
	if err != nil {
		return "", err
	}
	return "arista.eos.runningConfig.section " + name, nil
}

func (a *lumiAristaEosRunningConfigSection) GetContent() (string, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Transport)
	if err != nil {
		return "", err
	}

	name, err := a.Name()
	if err != nil {
		return "", err
	}

	// todo: use content from arista.eos.runningconfig
	content := eos.RunningConfig()

	return arista.GetSection(strings.NewReader(content), name), nil
}

func (a *lumiAristaEos) GetSystemConfig() (map[string]interface{}, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	config := eos.SystemConfig()

	res := map[string]interface{}{}
	for k := range config {
		res[k] = config[k]
	}

	return res, nil
}

func (a *lumiAristaEos) GetUsers() ([]interface{}, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	users := eos.Users()

	lumiUsers := make([]interface{}, len(users))
	for i, user := range users {
		lumiUser, err := a.MotorRuntime.CreateResource("arista.eos.user",
			"name", user.UserName(),
			"privilege", user.Privilege(),
			"role", user.Role(),
			"nopassword", user.Nopassword(),
			"format", user.Format(),
			"secret", user.Secret(),
			"sshkey", user.SSHKey(),
		)
		if err != nil {
			return nil, err
		}
		lumiUsers[i] = lumiUser
	}

	return lumiUsers, nil
}

func (a *lumiAristaEos) GetRoles() ([]interface{}, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	roles, err := eos.Roles()
	if err != nil {
		return nil, err
	}

	lumRoles := make([]interface{}, len(roles))
	for i, role := range roles {

		rules, err := jsonToDictSlice(role.Rules)
		if err != nil {
			return nil, err
		}

		lumiRole, err := a.MotorRuntime.CreateResource("arista.eos.role",
			"name", role.Name,
			"default", role.Default,
			"rules", rules,
		)
		if err != nil {
			return nil, err
		}
		lumRoles[i] = lumiRole
	}
	return lumRoles, nil
}

func (a *lumiAristaEos) GetNtp() (interface{}, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	ntp, err := eos.NtpStatus()
	if err != nil {
		return nil, err
	}

	return a.MotorRuntime.CreateResource("arista.eos.ntpSetting",
		"status", ntp.Status,
	)
}

func (v *lumiAristaEosNtpSetting) id() (string, error) {
	return "arista.eos.ntpSetting", nil
}

func (a *lumiAristaEos) GetSnmp() (interface{}, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	snmp, err := eos.Snmp()
	if err != nil {
		return nil, err
	}

	return a.MotorRuntime.CreateResource("arista.eos.snmpSetting",
		"enabled", snmp.Enabled,
	)
}

func (v *lumiAristaEosSnmpSetting) id() (string, error) {
	return "arista.eos.snmpSetting", nil
}

func (a *lumiAristaEosSnmpSetting) GetNotifications() ([]interface{}, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	notifications, err := eos.SnmpNotifications()
	if err != nil {
		return nil, err
	}

	return jsonToDictSlice(notifications)
}

func (a *lumiAristaEos) GetIpInterfaces() ([]interface{}, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	ifaces := eos.IPInterfaces()

	lumiIfaces := make([]interface{}, len(ifaces))
	for i, iface := range ifaces {
		lumiService, err := a.MotorRuntime.CreateResource("arista.eos.ipInterface",
			"name", iface.Name(),
			"address", iface.Address(),
			"mtu", iface.Mtu(),
		)
		if err != nil {
			return nil, err
		}
		lumiIfaces[i] = lumiService
	}

	return lumiIfaces, nil
}

func (a *lumiAristaEos) GetVersion() (map[string]interface{}, error) {
	_, at, err := aristaClientInstance(a.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	version, err := at.GetVersion()
	if err != nil {
		return nil, err
	}
	return jsonToDict(version)
}

func (a *lumiAristaEos) GetHostname() (string, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Transport)
	if err != nil {
		return "", err
	}

	hostname, err := eos.ShowHostname()
	if err != nil {
		return "", err
	}

	return hostname.Hostname, nil
}

func (a *lumiAristaEos) GetFqdn() (string, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Transport)
	if err != nil {
		return "", err
	}

	hostname, err := eos.ShowHostname()
	if err != nil {
		return "", err
	}

	return hostname.Fqdn, nil
}

func (a *lumiAristaEos) GetInterfaces() ([]interface{}, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	ifaces := eos.ShowInterface()

	lumiIfaces := []interface{}{}
	for k := range ifaces.Interfaces {
		iface := ifaces.Interfaces[k]

		address := []interface{}{}
		for i := range iface.InterfaceAddress {
			ifaceAddress, err := jsonToDict(iface.InterfaceAddress[i])
			if err != nil {
				return nil, err
			}
			address = append(address, ifaceAddress)
		}

		counters, err := jsonToDict(iface.InterfaceCounters)
		if err != nil {
			return nil, err
		}

		statistics, err := jsonToDict(iface.InterfaceStatistics)
		if err != nil {
			return nil, err
		}

		lumiIface, err := a.MotorRuntime.CreateResource("arista.eos.interface",
			"name", iface.Name,
			"bandwidth", int64(iface.Bandwidth),
			"burnedInAddress", iface.BurnedInAddress,
			"description", iface.Description,
			"forwardingModel", iface.ForwardingModel,
			"hardware", iface.Hardware,
			"interfaceAddress", address,
			"interfaceCounters", counters,
			"interfaceMembership", iface.InterfaceMembership,
			"interfaceStatistics", statistics,
			"interfaceStatus", iface.InterfaceStatus,
			"l2Mtu", int64(iface.L2Mtu),
			"lastStatusChangeTimestamp", int64(iface.LastStatusChangeTimestamp),
			"lineProtocolStatus", iface.LineProtocolStatus,
			"mtu", int64(iface.Mtu),
			"physicalAddress", iface.PhysicalAddress,
		)
		if err != nil {
			return nil, err
		}
		lumiIfaces = append(lumiIfaces, lumiIface)

	}
	return lumiIfaces, nil
}

func (a *lumiAristaEosInterface) id() (string, error) {
	return a.Name()
}

func (a *lumiAristaEosInterface) GetStatus() (map[string]interface{}, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	ifaceName, err := a.Name()
	if err != nil {
		return nil, err
	}

	status, err := eos.ShowInterfacesStatus()
	if err != nil {
		return nil, err
	}

	entry, ok := status[ifaceName]
	if !ok {
		return nil, nil
	}

	return jsonToDict(entry)
}

func (a *lumiAristaEosStp) id() (string, error) {
	return "arista.eos.stp", nil
}

var aristaMstInstanceID = regexp.MustCompile(`(\d+)$`)

func (a *lumiAristaEosStp) GetMstInstances() ([]interface{}, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	mstInstances, err := eos.Stp()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}

	for mstk := range mstInstances {
		mstInstance := mstInstances[mstk]

		m := aristaMstInstanceID.FindStringSubmatch(mstk)

		bridge, err := jsonToDict(mstInstance.Bridge)
		if err != nil {
			return nil, err
		}

		rootBridge, err := jsonToDict(mstInstance.RootBridge)
		if err != nil {
			return nil, err
		}

		regionalRootBridge, err := jsonToDict(mstInstance.RegionalRootBridge)
		if err != nil {
			return nil, err
		}

		sptmstInterfaces := []interface{}{}
		for ifacek := range mstInstance.Interfaces {
			iface := mstInstance.Interfaces[ifacek]

			inconsistentFeatures, err := jsonToDict(iface.InconsistentFeatures)
			if err != nil {
				return nil, err
			}

			detail, err := jsonToDict(iface.Detail)
			if err != nil {
				return nil, err
			}

			lumiArista, err := a.MotorRuntime.CreateResource("arista.eos.spt.mstInterface",
				"id", mstk+"/"+ifacek,
				"mstInstanceId", m[1],
				"name", ifacek,
				"priority", iface.Priority,
				"linkType", iface.LinkType,
				"state", iface.State,
				"cost", int64(iface.Cost),
				"role", iface.Role,
				"inconsistentFeatures", inconsistentFeatures,
				"portNumber", int64(iface.PortNumber),
				"isEdgePort", iface.IsEdgePort,
				"detail", detail,
				"boundaryType", iface.State,
			)
			if err != nil {
				return nil, err
			}
			sptmstInterfaces = append(sptmstInterfaces, lumiArista)
		}

		lumiArista, err := a.MotorRuntime.CreateResource("arista.eos.stp.mst",
			"instanceId", m[1],
			"name", mstk,
			"protocol", mstInstance.Protocol,
			"bridge", bridge,
			"rootBridge", rootBridge,
			"regionalRootBridge", regionalRootBridge,
			"interfaces", sptmstInterfaces,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiArista)
	}
	return res, nil
}

func (a *lumiAristaEosStpMst) id() (string, error) {
	return a.Name()
}

func (a *lumiAristaEosSptMstInterface) id() (string, error) {
	return a.Id()
}

func (a *lumiAristaEosSptMstInterface) GetCounters() (map[string]interface{}, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	mstInstanceId, err := a.MstInstanceId()
	if err != nil {
		return nil, err
	}

	name, err := a.Name()
	if err != nil {
		return nil, err
	}

	mstInstanceDetails, err := eos.StpInterfaceDetails(mstInstanceId, name)
	if err != nil {
		return nil, err
	}

	return jsonToDict(mstInstanceDetails.Counters)
}

func (a *lumiAristaEosSptMstInterface) GetFeatures() (map[string]interface{}, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	mstInstanceId, err := a.MstInstanceId()
	if err != nil {
		return nil, err
	}

	name, err := a.Name()
	if err != nil {
		return nil, err
	}

	mstInstanceDetails, err := eos.StpInterfaceDetails(mstInstanceId, name)
	if err != nil {
		return nil, err
	}

	return jsonToDict(mstInstanceDetails.Features)
}
