package os

import (
	"errors"
	"regexp"
	"strings"

	"go.mondoo.io/mondoo/motor/providers"
	arista_provider "go.mondoo.io/mondoo/motor/providers/arista"
	"go.mondoo.io/mondoo/resources/packs/core"
	"go.mondoo.io/mondoo/resources/packs/os/arista"
)

func aristaClientInstance(t providers.Transport) (*arista.Eos, *arista_provider.Provider, error) {
	provider, ok := t.(*arista_provider.Provider)
	if !ok {
		return nil, nil, errors.New("arista.eos resource is not supported on this transport")
	}

	eos := arista.NewEos(provider.Client())
	return eos, provider, nil
}

func (a *mqlAristaEos) id() (string, error) {
	return "arista.eos", nil
}

func (v *mqlAristaEosIpInterface) id() (string, error) {
	return v.Name()
}

func (v *mqlAristaEosUser) id() (string, error) {
	return v.Name()
}

func (v *mqlAristaEosRole) id() (string, error) {
	return v.Name()
}

func (v *mqlAristaEosRunningConfig) id() (string, error) {
	return "arista.eos.runningConfig", nil
}

func (a *mqlAristaEosRunningConfig) GetContent() (string, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return "", err
	}

	return eos.RunningConfig(), nil
}

func (a *mqlAristaEosRunningConfigSection) id() (string, error) {
	name, err := a.Name()
	if err != nil {
		return "", err
	}
	return "arista.eos.runningConfig.section " + name, nil
}

func (a *mqlAristaEosRunningConfigSection) GetContent() (string, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Provider)
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

func (a *mqlAristaEos) GetSystemConfig() (map[string]interface{}, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Provider)
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

func (a *mqlAristaEos) GetUsers() ([]interface{}, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	users := eos.Users()

	mqlUsers := make([]interface{}, len(users))
	for i, user := range users {
		mqlUser, err := a.MotorRuntime.CreateResource("arista.eos.user",
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
		mqlUsers[i] = mqlUser
	}

	return mqlUsers, nil
}

func (a *mqlAristaEos) GetRoles() ([]interface{}, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	roles, err := eos.Roles()
	if err != nil {
		return nil, err
	}

	lumRoles := make([]interface{}, len(roles))
	for i, role := range roles {

		rules, err := core.JsonToDictSlice(role.Rules)
		if err != nil {
			return nil, err
		}

		mqlRole, err := a.MotorRuntime.CreateResource("arista.eos.role",
			"name", role.Name,
			"default", role.Default,
			"rules", rules,
		)
		if err != nil {
			return nil, err
		}
		lumRoles[i] = mqlRole
	}
	return lumRoles, nil
}

func (a *mqlAristaEos) GetNtp() (interface{}, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Provider)
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

func (v *mqlAristaEosNtpSetting) id() (string, error) {
	return "arista.eos.ntpSetting", nil
}

func (a *mqlAristaEos) GetSnmp() (interface{}, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Provider)
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

func (v *mqlAristaEosSnmpSetting) id() (string, error) {
	return "arista.eos.snmpSetting", nil
}

func (a *mqlAristaEosSnmpSetting) GetNotifications() ([]interface{}, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	notifications, err := eos.SnmpNotifications()
	if err != nil {
		return nil, err
	}

	return core.JsonToDictSlice(notifications)
}

func (a *mqlAristaEos) GetIpInterfaces() ([]interface{}, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	ifaces := eos.IPInterfaces()

	mqlIfaces := make([]interface{}, len(ifaces))
	for i, iface := range ifaces {
		mqlService, err := a.MotorRuntime.CreateResource("arista.eos.ipInterface",
			"name", iface.Name(),
			"address", iface.Address(),
			"mtu", iface.Mtu(),
		)
		if err != nil {
			return nil, err
		}
		mqlIfaces[i] = mqlService
	}

	return mqlIfaces, nil
}

func (a *mqlAristaEos) GetVersion() (map[string]interface{}, error) {
	_, provider, err := aristaClientInstance(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	version, err := provider.GetVersion()
	if err != nil {
		return nil, err
	}
	return core.JsonToDict(version)
}

func (a *mqlAristaEos) GetHostname() (string, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return "", err
	}

	hostname, err := eos.ShowHostname()
	if err != nil {
		return "", err
	}

	return hostname.Hostname, nil
}

func (a *mqlAristaEos) GetFqdn() (string, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return "", err
	}

	hostname, err := eos.ShowHostname()
	if err != nil {
		return "", err
	}

	return hostname.Fqdn, nil
}

func (a *mqlAristaEos) GetInterfaces() ([]interface{}, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	ifaces := eos.ShowInterface()

	mqlIfaces := []interface{}{}
	for k := range ifaces.Interfaces {
		iface := ifaces.Interfaces[k]

		address := []interface{}{}
		for i := range iface.InterfaceAddress {
			ifaceAddress, err := core.JsonToDict(iface.InterfaceAddress[i])
			if err != nil {
				return nil, err
			}
			address = append(address, ifaceAddress)
		}

		counters, err := core.JsonToDict(iface.InterfaceCounters)
		if err != nil {
			return nil, err
		}

		statistics, err := core.JsonToDict(iface.InterfaceStatistics)
		if err != nil {
			return nil, err
		}

		mqlIface, err := a.MotorRuntime.CreateResource("arista.eos.interface",
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
		mqlIfaces = append(mqlIfaces, mqlIface)

	}
	return mqlIfaces, nil
}

func (a *mqlAristaEosInterface) id() (string, error) {
	return a.Name()
}

func (a *mqlAristaEosInterface) GetStatus() (map[string]interface{}, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Provider)
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

	return core.JsonToDict(entry)
}

func (a *mqlAristaEosStp) id() (string, error) {
	return "arista.eos.stp", nil
}

var aristaMstInstanceID = regexp.MustCompile(`(\d+)$`)

func (a *mqlAristaEosStp) GetMstInstances() ([]interface{}, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Provider)
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

		bridge, err := core.JsonToDict(mstInstance.Bridge)
		if err != nil {
			return nil, err
		}

		rootBridge, err := core.JsonToDict(mstInstance.RootBridge)
		if err != nil {
			return nil, err
		}

		regionalRootBridge, err := core.JsonToDict(mstInstance.RegionalRootBridge)
		if err != nil {
			return nil, err
		}

		sptmstInterfaces := []interface{}{}
		for ifacek := range mstInstance.Interfaces {
			iface := mstInstance.Interfaces[ifacek]

			inconsistentFeatures, err := core.JsonToDict(iface.InconsistentFeatures)
			if err != nil {
				return nil, err
			}

			detail, err := core.JsonToDict(iface.Detail)
			if err != nil {
				return nil, err
			}

			mqlArista, err := a.MotorRuntime.CreateResource("arista.eos.spt.mstInterface",
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
			sptmstInterfaces = append(sptmstInterfaces, mqlArista)
		}

		mqlArista, err := a.MotorRuntime.CreateResource("arista.eos.stp.mst",
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
		res = append(res, mqlArista)
	}
	return res, nil
}

func (a *mqlAristaEosStpMst) id() (string, error) {
	return a.Name()
}

func (a *mqlAristaEosSptMstInterface) id() (string, error) {
	return a.Id()
}

func (a *mqlAristaEosSptMstInterface) GetCounters() (map[string]interface{}, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Provider)
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

	return core.JsonToDict(mstInstanceDetails.Counters)
}

func (a *mqlAristaEosSptMstInterface) GetFeatures() (map[string]interface{}, error) {
	eos, _, err := aristaClientInstance(a.MotorRuntime.Motor.Provider)
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

	return core.JsonToDict(mstInstanceDetails.Features)
}
