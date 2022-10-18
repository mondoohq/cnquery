package detector

import (
	"errors"

	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/vsphere"
	vsphere_provider "go.mondoo.com/cnquery/motor/providers/vsphere"
)

func VspherePlatform(t *vsphere.Provider, identifier string) (*platform.Platform, error) {
	if vsphere_provider.IsVsphereResourceID(identifier) {
		moid, err := vsphere_provider.ParseVsphereResourceID(identifier)
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
			ver, err := vsphere_provider.EsxiVersion(host)
			if err == nil {
				esxi_version = ver.Version
				esxi_build = ver.Build
			}

			// host
			return &platform.Platform{
				Name:    "vmware-esxi",
				Title:   "VMware ESXi",
				Release: esxi_version,
				Version: esxi_version,
				Build:   esxi_build,
				Runtime: providers.RUNTIME_VSPHERE_HOSTS,
				Kind:    providers.Kind_KIND_BARE_METAL,
			}, nil

		case "VirtualMachine":
			// TODO: we should detect more details here
			// vm
			return &platform.Platform{
				Runtime: providers.RUNTIME_VSPHERE_VM,
				Kind:    providers.Kind_KIND_VIRTUAL_MACHINE,
			}, nil
		default:
			return nil, errors.New("unsupported platform identifier " + identifier)
		}
	}

	info := t.Info()
	return &platform.Platform{
		Name:    "vmware-vsphere",
		Title:   info.FullName,
		Release: info.Version,
		Version: info.Version,
		Build:   info.Build,
		Kind:    providers.Kind_KIND_API,
		Runtime: providers.RUNTIME_VSPHERE,
	}, nil
}
