package resources

import (
	"github.com/aristanetworks/goeapi"
	"go.mondoo.io/mondoo/lumi/resources/arista"
)

func (a *lumiAristaeos) id() (string, error) {
	return "aristaeos", nil // TODO: consider encoding the host to ensure we can handle multiple instances
}

func (v *lumiAristaeosIpinterface) id() (string, error) {
	return v.Name()
}

func (a *lumiAristaeos) getInstance() (*arista.Eos, error) {
	node, err := goeapi.Connect("http", "localhost", "admin", "", 8080)
	if err != nil {
		return nil, err
	}
	eos := arista.NewEos(node)
	return eos, nil
}

func (a *lumiAristaeos) GetRunningconfig() (string, error) {
	eos, err := a.getInstance()
	if err != nil {
		return "", err
	}
	return eos.RunningConfig(), nil
}

func (a *lumiAristaeos) GetSystemconfig() (map[string]interface{}, error) {
	eos, err := a.getInstance()
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

func (a *lumiAristaeos) GetIpinterfaces() ([]interface{}, error) {
	eos, err := a.getInstance()
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
