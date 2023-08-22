// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package vsphere

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/resolver"
	"go.mondoo.com/cnquery/motor/providers/vmwareguestapi"
	"go.mondoo.com/cnquery/motor/vault"
)

type VMGuestResolver struct{}

func (k *VMGuestResolver) Name() string {
	return "VMware vSphere VM Guest Resolver"
}

func (r *VMGuestResolver) AvailableDiscoveryTargets() []string {
	return []string{common.DiscoveryAuto}
}

func (k *VMGuestResolver) Resolve(ctx context.Context, root *asset.Asset, pCfg *providers.Config, credsResolver vault.Resolver, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	// we leverage the vpshere provider to establish a connection
	m, err := resolver.NewMotorConnection(ctx, pCfg, credsResolver)
	if err != nil {
		return nil, err
	}
	defer m.Close()

	trans, ok := m.Provider.(*vmwareguestapi.Provider)
	if !ok {
		return nil, errors.New("could not initialize vsphere guest provider")
	}

	client := trans.Client()
	discoveryClient := New(client)

	// resolve vms
	vms, err := discoveryClient.ListVirtualMachines(pCfg)
	if err != nil {
		return nil, err
	}

	// add provider config for each vm
	for i := range vms {
		vm := vms[i]
		resolved = append(resolved, vm)
	}

	// filter the vms by inventoryPath
	inventoryPaths := []string{}
	inventoryPathFilter, ok := pCfg.Options["inventoryPath"]
	if ok {
		inventoryPaths = []string{inventoryPathFilter}
	}

	resolved = filter(resolved, func(a *asset.Asset) bool {
		inventoryPathLabel := a.Labels["vsphere.vmware.com/inventory-path"]

		return contains(inventoryPaths, inventoryPathLabel)
	})

	if len(resolved) == 1 {
		a := resolved[0]
		a.Connections = []*providers.Config{pCfg}

		// find the secret reference for the asset
		EnrichVsphereToolsConnWithSecrets(a, credsResolver, sfn)

		return []*asset.Asset{a}, nil
	} else {
		return nil, errors.New("could not resolve vSphere VM")
	}
}

func EnrichVsphereToolsConnWithSecrets(a *asset.Asset, credsResolver vault.Resolver, sfn common.QuerySecretFn) {
	// search secret for vm
	// NOTE: we do not use `common.EnrichAssetWithSecrets(a, sfn)` here since vmware requires two secrets at the same time
	for j := range a.Connections {
		conn := a.Connections[j]

		// special handling for vsphere vm config
		if conn.Backend == providers.ProviderType_VSPHERE_VM {
			var creds *vault.Credential

			secretRefCred, err := sfn(a)
			if err == nil && secretRefCred != nil {
				creds, err = credsResolver.GetCredential(secretRefCred)
			}

			if err == nil && creds != nil {
				if conn.Options == nil {
					conn.Options = map[string]string{}
				}
				conn.Options["guestUser"] = creds.User
				conn.Options["guestPassword"] = string(creds.Secret)
			}
		} else {
			log.Warn().Str("name", a.Name).Msg("could not determine credentials for asset")
		}
	}
}
