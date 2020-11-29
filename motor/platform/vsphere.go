package platform

import (
	"errors"

	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/vsphere"
	vsphere_transport "go.mondoo.io/mondoo/motor/transports/vsphere"
)

func VspherePlatform(t *vsphere.Transport, identifier string) (*Platform, error) {
	if vsphere_transport.IsVsphereResourceID(identifier) {
		moid, err := vsphere_transport.ParseVsphereResourceID(identifier)
		if err != nil {
			return nil, err
		}

		switch moid.Type {
		case "HostSystem":
			// TODO: check that we can gather a host by its moid
			host, err := t.Host(moid)
			if err != nil {
				return nil, err
			}

			// TODO: Determine full platform information eg. esxi
			esxi_version := ""
			esxi_build := ""
			// we do not abort in case of error because the simulator does not support esxi interface for the host
			ver, err := vsphere_transport.EsxiVersion(host)
			if err == nil {
				esxi_version = ver.Version
				esxi_build = ver.Build
			}

			// host
			return &Platform{
				Name:    "vmware-esxi",
				Title:   "VMware ESXi",
				Release: esxi_version,
				Build:   esxi_build,
				Runtime: transports.RUNTIME_VSPHERE_HOSTS,
				Kind:    transports.Kind_KIND_BARE_METAL,
			}, nil

		case "VirtualMachine":
			// TODO: we should detect more details here
			// vm
			return &Platform{
				Runtime: transports.RUNTIME_VSPHERE_VM,
				Kind:    transports.Kind_KIND_VIRTUAL_MACHINE,
			}, nil
		default:
			return nil, errors.New("unsupported platform identifier " + identifier)
		}
	}

	info := t.Info()
	return &Platform{
		Name:    "vmware-vsphere",
		Title:   info.FullName,
		Release: info.Version,
		Build:   info.Build,
		Kind:    transports.Kind_KIND_API,
		Runtime: transports.RUNTIME_VSPHERE,
	}, nil
}
