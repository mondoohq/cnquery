package platform

import (
	"errors"

	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/vsphere"
	vsphere_transport "go.mondoo.io/mondoo/motor/transports/vsphere"
)

func VspherePlatform(t *vsphere.Transport, identifier string) (*Platform, error) {
	if vsphere_transport.IsVsphereResourceID(identifier) {
		typ, inventoryPath, err := vsphere_transport.ParseVsphereResourceID(identifier)
		if err != nil {
			return nil, err
		}

		switch typ {
		case "HostSystem":
			host, err := t.Host(inventoryPath)
			if err != nil {
				return nil, err
			}

			// TODO: Determine full platform information eg. esxi
			esxi_version := ""
			// we do not abort in case of error because the simulator does not support esxi interface for the host
			ver, err := vsphere_transport.EsxiVersion(host)
			if err == nil {
				esxi_version = ver.Version
			}

			// host
			return &Platform{
				Name:    "vmware-esxi",
				Title:   "VMware ESXi",
				Release: esxi_version,
				Runtime: RUNTIME_VSPHERE_HOSTS,
				Kind:    transports.Kind_KIND_BARE_METAL,
			}, nil

		case "VirtualMachine":
			// vm
			return &Platform{
				Runtime: RUNTIME_VSPHERE_VM,
				Kind:    transports.Kind_KIND_VIRTUAL_MACHINE,
			}, nil
		default:
			return nil, errors.New("unsupported platform identier " + identifier)
		}
	}

	info := t.Info()
	return &Platform{
		Name:    "vmware-vsphere",
		Title:   info.FullName,
		Release: info.Version,
		Kind:    transports.Kind_KIND_API,
		Runtime: RUNTIME_VSPHERE,
	}, nil
}
