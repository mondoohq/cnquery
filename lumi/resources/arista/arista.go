package arista

import (
	"github.com/aristanetworks/goeapi"
	"github.com/aristanetworks/goeapi/module"
)

func NewEos(node *goeapi.Node) *Eos {
	return &Eos{node: node}
}

type Eos struct {
	node *goeapi.Node
}

func (eos *Eos) RunningConfig() string {
	return eos.node.RunningConfig()
}

func (eos *Eos) SystemConfig() map[string]string {
	// get api system module
	sys := module.System(eos.node)
	return sys.Get()
}

func (eos *Eos) IPInterfaces() []module.IPInterfaceConfig {
	ifaceModule := module.IPInterface(eos.node)
	interfaces := ifaceModule.GetAll()

	res := []module.IPInterfaceConfig{}
	for i := range interfaces {
		iface := interfaces[i]
		res = append(res, iface)
	}
	return res
}
