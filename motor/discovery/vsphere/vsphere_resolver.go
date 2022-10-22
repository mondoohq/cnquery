package vsphere

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/motorid"
	"go.mondoo.com/cnquery/motor/platform/detector"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/resolver"
	"go.mondoo.com/cnquery/motor/providers/vsphere"
)

const (
	DiscoveryApi          = "api"
	DiscoveryInstances    = "instances"
	DiscoveryHostMachines = "host-machines"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "VMware vSphere Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{common.DiscoveryAuto, common.DiscoveryAll, DiscoveryInstances, DiscoveryHostMachines}
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, pCfg *providers.Config, cfn common.CredentialFn, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	// we leverage the vpshere transport to establish a connection
	m, err := resolver.NewMotorConnection(ctx, pCfg, cfn)
	if err != nil {
		return nil, err
	}
	defer m.Close()

	trans, ok := m.Provider.(*vsphere.Provider)
	if !ok {
		return nil, errors.New("could not initialize vsphere transport")
	}

	// detect platform info for the asset
	pf, err := m.Platform()
	if err != nil {
		return nil, err
	}

	if pCfg.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryApi) {
		// add asset for the api itself
		info := trans.Info()
		assetObj := &asset.Asset{
			Name:        fmt.Sprintf("%s (%s)", pCfg.Host, info.Name),
			Platform:    pf,
			Connections: []*providers.Config{pCfg}, // pass-in the current config
			Labels: map[string]string{
				"vsphere.vmware.com/name": info.Name,
				"vsphere.vmware.com/uuid": info.InstanceUuid,
			},
		}
		fingerprint, err := motorid.IdentifyPlatform(m.Provider, pf, nil)
		if err != nil {
			return nil, err
		}
		assetObj.PlatformIds = fingerprint.PlatformIDs
		if fingerprint.Name != "" {
			assetObj.Name = fingerprint.Name
		}

		log.Debug().Strs("identifier", assetObj.PlatformIds).Msg("motor connection")

		resolved = append(resolved, assetObj)
	}

	client := trans.Client()
	discoveryClient := New(client)

	if pCfg.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryHostMachines) {
		// resolve esxi hosts
		hosts, err := discoveryClient.ListEsxiHosts()
		if err != nil {
			return nil, err
		}

		// add transport config for each host
		for i := range hosts {
			host := hosts[i]
			ht := pCfg.Clone()
			// pass-through "vsphere.vmware.com/reference-type" and "vsphere.vmware.com/inventorypath"
			ht.Options = host.Annotations
			host.Connections = append(host.Connections, ht)

			pf, err := detector.VspherePlatform(trans, host.PlatformIds[0])
			if err == nil {
				host.Platform = pf
			} else {
				log.Error().Err(err).Msg("could not determine platform information for esxi host")
			}

			resolved = append(resolved, host)
		}
	}

	if pCfg.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryInstances) {
		// resolve vms
		vms, err := discoveryClient.ListVirtualMachines(pCfg)
		if err != nil {
			return nil, err
		}

		// add transport config for each vm
		for i := range vms {
			vm := vms[i]

			pf, err := detector.VspherePlatform(trans, vm.PlatformIds[0])
			if err == nil {
				vm.Platform = pf
			} else {
				log.Error().Err(err).Msg("could not determine platform information for esxi vm")
			}

			// find the secret reference for the asset
			EnrichVsphereToolsConnWithSecrets(vm, cfn, sfn)

			resolved = append(resolved, vm)
		}
	}

	// filter assets
	discoverFilter := map[string]string{}
	if pCfg.Discover != nil {
		discoverFilter = pCfg.Discover.Filter
	}

	if namesFilter, ok := discoverFilter["names"]; ok {
		names := strings.Split(namesFilter, ",")
		resolved = filter(resolved, func(a *asset.Asset) bool {
			return contains(names, a.Name)
		})
	}

	if moidsFilter, ok := discoverFilter["moids"]; ok {
		moids := strings.Split(moidsFilter, ",")
		resolved = filter(resolved, func(a *asset.Asset) bool {
			label, ok := a.Labels["vsphere.vmware.com/moid"]
			log.Debug().Strs("moids", moids).Str("search", label).Msg("check if moid is included")
			if !ok {
				return false
			}
			return contains(moids, label)
		})
	}

	return resolved, nil
}

func filter(a []*asset.Asset, keep func(asset *asset.Asset) bool) []*asset.Asset {
	n := 0
	for _, x := range a {
		if keep(x) {
			a[n] = x
			n++
		}
	}
	a = a[:n]
	return a
}

func contains(slice []string, entry string) bool {
	sanitizedEntry := strings.ToLower(strings.TrimSpace(entry))

	for i := range slice {
		if strings.ToLower(strings.TrimSpace(slice[i])) == sanitizedEntry {
			return true
		}
	}
	return false
}
