package vsphere

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/vsphere"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "VMware vSphere Resolver"
}

func (r *Resolver) ParseConnectionURL(url string, opts ...transports.TransportConfigOption) (*transports.TransportConfig, error) {
	return transports.NewTransportFromUrl(url, opts...)
	// t := &transports.TransportConfig{}
	// err := t.ParseFromURI(url)
	// if err != nil {
	// 	err := errors.Wrapf(err, "cannot connect to %s", url)
	// 	return nil, err
	// }

	// // copy password from opts asset if it was not encoded in url
	// if len(t.Password) == 0 && len(in.Password) > 0 {
	// 	t.Password = in.Password
	// }

	// return t, nil
}

func (r *Resolver) Resolve(t *transports.TransportConfig, opts map[string]string) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	// we leverage the vpshere transport to establish a connection
	trans, err := vsphere.New(t)
	if err != nil {
		return nil, err
	}

	client := trans.Client()
	discoveryClient := New(client)

	identifier, err := trans.Identifier()
	if err != nil {
		return nil, err
	}

	// detect platform info for the asset
	detector := platform.NewDetector(trans)
	pf, err := detector.Platform()
	if err != nil {
		return nil, err
	}

	// add asset for the api itself
	info := trans.Info()

	name := info.Name
	if info.InstanceUuid != "" {
		name = fmt.Sprintf("%s (%s)", info.Name, info.InstanceUuid)
	}

	resolved = append(resolved, &asset.Asset{
		PlatformIDs: []string{identifier},
		Name:        name,
		Platform:    pf,
		Connections: []*transports.TransportConfig{t}, // pass-in the current config
	})

	if _, ok := opts["host-machines"]; ok {
		// resolve esxi hosts
		hosts, err := discoveryClient.ListEsxiHosts()
		if err != nil {
			return nil, err
		}

		// add transport config for each host
		for i := range hosts {
			host := hosts[i]
			ht := t.Clone()
			// pass-through "vsphere.vmware.com/reference-type" and "vsphere.vmware.com/inventorypath"
			ht.Options = host.Annotations
			host.Connections = append(host.Connections, ht)

			pf, err := platform.VspherePlatform(trans, host.PlatformIDs[0])
			if err == nil {
				host.Platform = pf
			} else {
				log.Error().Err(err).Msg("could not determine platform information for esxi host")
			}

			resolved = append(resolved, host)
		}
	}

	if _, ok := opts["instances"]; ok {
		// resolve vms
		vms, err := discoveryClient.ListVirtualMachines()
		if err != nil {
			return nil, err
		}

		// add transport config for each vm
		for i := range vms {
			vm := vms[i]
			vt := t.Clone()
			// pass-through "vsphere.vmware.com/reference-type" and "vsphere.vmware.com/inventorypath"
			vt.Options = vm.Annotations
			vm.Connections = append(vm.Connections, vt)

			pf, err := platform.VspherePlatform(trans, vm.PlatformIDs[0])
			if err == nil {
				vm.Platform = pf
			} else {
				log.Error().Err(err).Msg("could not determine platform information for esxi vm")
			}

			resolved = append(resolved, vm)
		}
	}

	return resolved, nil
}
