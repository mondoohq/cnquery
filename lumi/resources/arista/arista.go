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

func (eos *Eos) ShowVersion() module.ShowVersion {
	show := module.Show(eos.node)
	return show.ShowVersion()
}

func (eos *Eos) ShowInterface() module.ShowInterface {
	show := module.Show(eos.node)
	return show.ShowInterfaces()
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

type showInterfacesStatus struct {
	InterfaceStatuses map[string]InterfaceStatus
}

func (s *showInterfacesStatus) GetCmd() string {
	return "show interfaces status"
}

type InterfaceStatus struct {
	Bandwidth           int64    `json:"bandwidth"`
	InterfaceType       string   `json:"interfaceType"`
	Description         string   `json:"description"`
	AutoNegotiateActive bool     `json:"autoNegotiateActive"`
	Duplex              string   `json:"duplex"`
	LinkStatus          string   `json:"linkStatus"`
	LineProtocolStatus  string   `json:"lineProtocolStatus"`
	VlanInformation     vlanInfo `json:"vlanInformation"`
}

type vlanInfo struct {
	InterfaceMode            string `json:"interfaceMode"`
	VlanID                   int64  `json:"vlanId"`
	InterfaceForwardingModel string `json:"interfaceForwardingModel"`
}

func (eos *Eos) ShowInterfacesStatus() (map[string]InterfaceStatus, error) {
	shIntRsp := &showInterfacesStatus{}

	handle, _ := eos.node.GetHandle("json")
	handle.AddCommand(shIntRsp)

	if err := handle.Call(); err != nil {
		return nil, err
	}

	return shIntRsp.InterfaceStatuses, nil
}
