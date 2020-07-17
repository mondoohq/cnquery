package vsphere

import (
	"errors"

	"github.com/rs/zerolog/log"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/govc/host/esxcli"
	"github.com/vmware/govmomi/object"
)

func NewEsxiClient(c *govmomi.Client, host *object.HostSystem) *Esxi {
	return &Esxi{
		c:    c,
		host: host,
	}
}

type Esxi struct {
	c    *govmomi.Client
	host *object.HostSystem
}

type VSwitch map[string]interface{}

// (Get - EsxCli).network.vswitch.standard.list()
func (esxi *Esxi) VswitchStandard() ([]VSwitch, error) {
	e, err := esxcli.NewExecutor(esxi.c.Client, esxi.host)
	if err != nil {
		return nil, err
	}

	res, err := e.Run([]string{"network", "vswitch", "standard", "list"})
	if err != nil {
		return nil, err
	}

	vswitches := make([]VSwitch, len(res.Values))
	for i, val := range res.Values {
		vswitch := VSwitch{}
		for k := range val {
			if len(val[k]) == 1 {
				vswitch[k] = val[k][0]
			} else {
				log.Error().Str("key", k).Msg("EsxiVswitchStandard> unsupported key")
			}
		}
		vswitches[i] = vswitch
	}
	return vswitches, nil
}

// (Get-EsxCli).network.vswitch.dvs.vmware.list()
func (esxi *Esxi) VswitchDvs() ([]VSwitch, error) {
	e, err := esxcli.NewExecutor(esxi.c.Client, esxi.host)
	if err != nil {
		return nil, err
	}

	res, err := e.Run([]string{"network", "vswitch", "dvs", "vmware", "list"})
	if err != nil {
		return nil, err
	}

	vswitches := make([]VSwitch, len(res.Values))
	for i, val := range res.Values {
		vswitch := VSwitch{}
		for k := range val {
			if len(val[k]) == 1 {
				vswitch[k] = val[k][0]
			} else {
				log.Error().Str("key", k).Msg("EsxiVswitchDvs> unsupported key")
			}
		}
		vswitches[i] = vswitch
	}
	return vswitches, nil
}

type Adapter map[string]interface{}

// (Get-EsxCli).network.nic.list.Invoke()
func (esxi *Esxi) Adapters() ([]Adapter, error) {
	e, err := esxcli.NewExecutor(esxi.c.Client, esxi.host)
	if err != nil {
		return nil, err
	}

	res, err := e.Run([]string{"network", "nic", "list"})
	if err != nil {
		return nil, err
	}

	nics := make([]Adapter, len(res.Values))
	for i, val := range res.Values {
		nic := Adapter{}
		for k := range val {
			if len(val[k]) == 1 {
				nic[k] = val[k][0]
			} else {
				log.Error().Str("key", k).Msg("EsxiAdapters> unsupported key")
			}
		}
		nics[i] = nic
	}
	return nics, nil
}

type VmKernelNic struct {
	Properties map[string]interface{}
	Ipv4       map[string]interface{}
	Ipv6       map[string]interface{}
}

// (Get-EsxCli).network.ip.interface.list()
func (esxi *Esxi) Vmknics() ([]VmKernelNic, error) {
	e, err := esxcli.NewExecutor(esxi.c.Client, esxi.host)
	if err != nil {
		return nil, err
	}

	res, err := e.Run([]string{"network", "ip", "interface", "list"})
	if err != nil {
		return nil, err
	}

	vmknics := make([]VmKernelNic, len(res.Values))
	for i, val := range res.Values {
		nic := VmKernelNic{
			Properties: map[string]interface{}{},
		}
		for k := range val {
			if len(val[k]) == 1 {
				nic.Properties[k] = val[k][0]
			} else {
				log.Error().Str("key", k).Msg("EsxiVmknics> unsupported key")
			}
		}

		name := val["Name"][0]
		netstack := val["NetstackInstance"][0]

		// gather ipv4 information
		ipv4Params, err := esxi.VmknixIp(name, netstack, "ipv4")
		if err != nil {
			return nil, err
		}
		nic.Ipv4 = ipv4Params

		// gather ipv6 information
		ipv6Params, err := esxi.VmknixIp(name, netstack, "ipv6")
		if err != nil {
			return nil, err
		}
		nic.Ipv4 = ipv6Params

		vmknics[i] = nic
	}
	return vmknics, nil
}

// (Get-EsxCli).network.ip.interface.ipv4.get('vmk0', 'defaultTcpipStack')
func (esxi *Esxi) VmknixIp(interfacename string, netstack string, ipprotocol string) (map[string]interface{}, error) {
	e, err := esxcli.NewExecutor(esxi.c.Client, esxi.host)
	if err != nil {
		return nil, err
	}

	res, err := e.Run([]string{"network", "ip", "interface", ipprotocol, "get"})
	if err != nil {
		return nil, err
	}

	properties := map[string]interface{}{}
	for _, val := range res.Values {
		for k := range val {
			if len(val[k]) == 1 {
				properties[k] = val[k][0]
			} else {
				log.Error().Str("key", k).Msg("EsxiVmknixIp> unsupported key")
			}
		}
	}
	return properties, nil
}
