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

	handle, err := eos.node.GetHandle("json")
	if err != nil {
		return nil, err
	}
	err = handle.AddCommand(shIntRsp)
	if err != nil {
		return nil, err
	}

	if err := handle.Call(); err != nil {
		return nil, err
	}

	return shIntRsp.InterfaceStatuses, nil
}

type showSpanningTree struct {
	SpanningTreeInstances map[string]SptMstInstance
}

func (s *showSpanningTree) GetCmd() string {
	return "show spanning-tree"
}

type showSpanningTreeMst struct {
	SpanningTreeMstInstances map[string]SptMstInstance
}

func (s *showSpanningTreeMst) GetCmd() string {
	return "show spanning-tree mst detail"
}

type SptMstInstance struct {
	Protocol           string                     `json:"protocol"` // NOTE: only returned for `show spanning-tree` but not `show spanning-tree mst`
	Bridge             sptMstBridge               `json:"bridge"`
	RootBridge         sptMstBridge               `json:"rootBridge"`
	RegionalRootBridge sptMstBridge               `json:"regionalRootBridge"`
	Interfaces         map[string]sptMstInterface `json:"interfaces"`
}

type sptMstBridge struct {
	Priority          int64   `json:"priority"`
	MacAddress        string  `json:"macAddress"`
	SystemIdExtension float64 `json:"systemIdExtension"`

	// NOTE: only returned for `show spanning-tree` but not `show spanning-tree mst`
	ForwardDelay int64 `json:"forwardDelay,omitempty"`
	HelloTime    int64 `json:"helloTime,omitempty"`
	MaxAge       int64 `json:"maxAge,omitempty"`
}

type sptMstInterface struct {
	Priority             int64  `json:"priority"`
	LinkType             string `json:"linkType"`
	State                string `json:"state"`
	Cost                 int64  `json:"cost"`
	Role                 string `json:"role"`
	InconsistentFeatures struct {
		LoopGuard       bool `json:"loopGuard"`
		RootGuard       bool `json:"rootGuard"`
		MstPvstBorder   bool `json:"mstPvstBorder"`
		BridgeAssurance bool `json:"bridgeAssurance"`
	} `json:"inconsistentFeatures"`
	PortNumber int64 `json:"portNumber"`
	IsEdgePort bool  `json:"isEdgePort"`
	Detail     struct {
		DesignatedBridgePriority int64  `json:"designatedBridgePriority"`
		RegionalRootPriority     int64  `json:"regionalRootPriority"`
		RegionalRootAddress      string `json:"regionalRootAddress"`
		DesignatedRootPriority   int64  `json:"designatedRootPriority"`
		DesignatedPortNumber     int64  `json:"designatedPortNumber"`
		DesignatedBridgeAddress  string `json:"designatedBridgeAddress"`
		DesignatedPortPriority   int64  `json:"designatedPortPriority"`
		DesignatedRootAddress    string `json:"designatedRootAddress"`
	} `json:"detail"`
}

// runs both
// show spanning-tree
// show spanning-tree mst
func (eos *Eos) Stp() (map[string]SptMstInstance, error) {
	shRsp := &showSpanningTree{}
	shMstRsp := &showSpanningTreeMst{}

	handle, err := eos.node.GetHandle("json")
	if err != nil {
		return nil, err
	}
	err = handle.AddCommand(shRsp)
	if err != nil {
		return nil, err
	}
	err = handle.AddCommand(shMstRsp)
	if err != nil {
		return nil, err
	}

	if err := handle.Call(); err != nil {
		return nil, err
	}

	handle.Close()

	// merge the information
	merge := shMstRsp.SpanningTreeMstInstances

	for k := range merge {
		entry := merge[k]

		// check if we found the protocol stuff
		tobemerged, ok := shRsp.SpanningTreeInstances[k]
		if ok {
			entry.Protocol = tobemerged.Protocol
			entry.Bridge.HelloTime = tobemerged.Bridge.HelloTime
			entry.Bridge.MaxAge = tobemerged.Bridge.MaxAge
			entry.Bridge.ForwardDelay = tobemerged.Bridge.ForwardDelay
		}
		merge[k] = entry
	}

	return merge, nil
}

type showSpanningTreeMstInstanceDetail struct {
	SpanningTreeMstInterface SptMestInterfaceDetail
}

func (s *showSpanningTreeMstInstanceDetail) GetCmd() string {
	return "show spanning-tree mst 0 interface Ethernet1 detail"
}

type SptMestInterfaceDetail struct {
	IsEdgePort bool `json:"isEdgePort"`
	Features   struct {
		LinkType struct {
			Default bool   `json:"default"`
			Value   string `json:"value"`
		} `json:"linkType"`
		BpduGuard struct {
			Default bool   `json:"default"`
			Value   string `json:"value"`
		} `json:"bpduGuard"`
		BpduFilter struct {
			Default bool   `json:"default"`
			Value   string `json:"value"`
		} `json:"BpduFilter"`
	} `json:"features"`
	InterfaceName string `json:"interfaceName"`
	Counters      struct {
		BpduRateLimitCount int64 `json:"bpduRateLimitCount"`
		BpduOtherError     int64 `json:"bpduOtherError"`
		BpduReceived       int64 `json:"bpduReceived"`
		BpduTaggedError    int64 `json:"bpduTaggedError"`
		BpduSent           int64 `json:"bpduSent"`
	} `json:"counters"`
}

// show spanning-tree mst 0 interface Ethernet1 detail
func (eos *Eos) StpInterfaceDetails(mstInstanceID string, iface string) (SptMestInterfaceDetail, error) {
	shRsp := &showSpanningTreeMstInstanceDetail{}

	handle, err := eos.node.GetHandle("json")
	if err != nil {
		return SptMestInterfaceDetail{}, err
	}
	err = handle.AddCommand(shRsp)
	if err != nil {
		return SptMestInterfaceDetail{}, err
	}

	if err := handle.Call(); err != nil {
		return SptMestInterfaceDetail{}, err
	}

	handle.Close()

	return shRsp.SpanningTreeMstInterface, nil
}
