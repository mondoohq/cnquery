package resources

import (
	"errors"

	"go.mondoo.io/mondoo/lumi/resources/arista"
	"go.mondoo.io/mondoo/motor/transports"
	arista_transport "go.mondoo.io/mondoo/motor/transports/arista"
)

func aristaClientInstance(t transports.Transport) (*arista.Eos, error) {
	at, ok := t.(*arista_transport.Transport)
	if !ok {
		return nil, errors.New("arista.eos resource is not supported on this transport")
	}

	eos := arista.NewEos(at.Client())
	return eos, nil
}

func (a *lumiAristaEos) id() (string, error) {
	return "arista.eos", nil
}

func (v *lumiAristaEosIpinterface) id() (string, error) {
	return v.Name()
}

func (a *lumiAristaEos) GetRunningConfig() (string, error) {
	eos, err := aristaClientInstance(a.Runtime.Motor.Transport)
	if err != nil {
		return "", err
	}
	return eos.RunningConfig(), nil
}

func (a *lumiAristaEos) GetSystemConfig() (map[string]interface{}, error) {
	eos, err := aristaClientInstance(a.Runtime.Motor.Transport)
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

func (a *lumiAristaEos) GetIpInterfaces() ([]interface{}, error) {
	eos, err := aristaClientInstance(a.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	ifaces := eos.IPInterfaces()

	lumiIfaces := make([]interface{}, len(ifaces))
	for i, iface := range ifaces {
		lumiService, err := a.Runtime.CreateResource("arista.eos.ipinterface",
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
	eos, err := aristaClientInstance(a.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	version := eos.ShowVersion()
	return jsonToDict(version)
}

func (a *lumiAristaEos) GetInterfaces() ([]interface{}, error) {
	eos, err := aristaClientInstance(a.Runtime.Motor.Transport)
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

		lumiIface, err := a.Runtime.CreateResource("arista.eos.interface",
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
	eos, err := aristaClientInstance(a.Runtime.Motor.Transport)
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
