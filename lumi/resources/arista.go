package resources

import (
	"errors"

	"go.mondoo.io/mondoo/lumi/resources/arista"
	arista_transport "go.mondoo.io/mondoo/motor/transports/arista"
)

func (a *lumiAristaeos) id() (string, error) {
	return "aristaeos", nil // TODO: consider encoding the host to ensure we can handle multiple instances
}

func (v *lumiAristaeosIpinterface) id() (string, error) {
	return v.Name()
}

func (a *lumiAristaeos) getClientInstance() (*arista.Eos, error) {
	at, ok := a.Runtime.Motor.Transport.(*arista_transport.Transport)
	if !ok {
		return nil, errors.New("aristaeos resource is not supported on this transport")
	}

	eos := arista.NewEos(at.Client())
	return eos, nil
}

func (a *lumiAristaeos) GetRunningConfig() (string, error) {
	eos, err := a.getClientInstance()
	if err != nil {
		return "", err
	}
	return eos.RunningConfig(), nil
}

func (a *lumiAristaeos) GetSystemConfig() (map[string]interface{}, error) {
	eos, err := a.getClientInstance()
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

func (a *lumiAristaeos) GetIpInterfaces() ([]interface{}, error) {
	eos, err := a.getClientInstance()
	if err != nil {
		return nil, err
	}
	ifaces := eos.IPInterfaces()

	lumiIfaces := make([]interface{}, len(ifaces))
	for i, iface := range ifaces {
		lumiService, err := a.Runtime.CreateResource("aristaeos.ipinterface",
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
