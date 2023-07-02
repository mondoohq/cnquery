package vagrant

import (
	"context"
	"strings"

	"errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/motorid"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/local"
	"go.mondoo.com/cnquery/motor/providers/resolver"
	"go.mondoo.com/cnquery/motor/vault"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Vagrant Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{common.DiscoveryAuto, common.DiscoveryAll}
}

func (v *Resolver) Resolve(ctx context.Context, root *asset.Asset, pCfg *providers.Config, credsResolver vault.Resolver, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	localProvider, err := local.New()
	if err != nil {
		return nil, err
	}

	// we run status first, since vagrant ssh-config does not return a proper state
	// if in a multi-vm setup not all vms are running
	cmd, err := localProvider.RunCommand("vagrant status")
	if err != nil {
		return nil, err
	}

	vmStatus, err := ParseVagrantStatus(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	// filter vms if a context was provided
	if len(pCfg.Host) > 0 {
		k := pCfg.Host
		vm, ok := vmStatus[k]
		if !ok {
			return nil, errors.New("could not find vagrant host: " + k)
		}

		if !vm {
			return nil, errors.New("vm is not ready: " + k)
		}

		cmd, err := localProvider.RunCommand("vagrant ssh-config " + k)
		if err != nil {
			return nil, err
		}

		vmSshConfig, err := ParseVagrantSshConfig(cmd.Stdout)
		if err != nil {
			return nil, err
		}

		a, err := newVagrantAsset(ctx, vmSshConfig[k], pCfg)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, a)

	} else {
		vagrantVms := map[string]*VagrantVmSSHConfig{}
		for k := range vmStatus {
			if vmStatus[k] == false {
				log.Debug().Str("vm", k).Msg("skip vm since it is not running")
				continue
			}

			log.Debug().Str("vm", k).Msg("gather ssh config")
			cmd, err := localProvider.RunCommand("vagrant ssh-config " + k)
			if err != nil {
				return nil, err
			}

			vmSshConfig, err := ParseVagrantSshConfig(cmd.Stdout)
			if err != nil {
				return nil, err
			}

			for k := range vmSshConfig {
				vagrantVms[k] = vmSshConfig[k]
			}
		}

		for i := range vagrantVms {
			a, err := newVagrantAsset(ctx, vagrantVms[i], pCfg)
			if err != nil {
				return nil, err
			}
			resolved = append(resolved, a)
		}
	}

	return resolved, nil
}

func newVagrantAsset(ctx context.Context, sshConfig *VagrantVmSSHConfig, rootTransportConfig *providers.Config) (*asset.Asset, error) {
	if sshConfig == nil {
		return nil, errors.New("missing vagrant ssh config")
	}

	cc := &providers.Config{
		// TODO: do we need to support winrm?
		Backend:  providers.ProviderType_SSH,
		Host:     sshConfig.HostName,
		Insecure: strings.ToLower(sshConfig.StrictHostKeyChecking) == "no",

		Port: int32(sshConfig.Port),
		Sudo: rootTransportConfig.Sudo,
	}

	// load secret
	credential, err := vault.NewPrivateKeyCredentialFromPath(sshConfig.User, sshConfig.IdentityFile, "")
	if err != nil {
		return nil, err
	}
	cc.AddCredential(credential)

	assetObj := &asset.Asset{
		Name:        sshConfig.Host,
		PlatformIds: []string{},
		Connections: []*providers.Config{cc},
		Platform: &platform.Platform{
			Kind: providers.Kind_KIND_VIRTUAL_MACHINE,
		},
	}

	m, err := resolver.NewMotorConnection(ctx, cc, nil)
	if err != nil {
		return nil, err
	}
	defer m.Close()

	p, err := m.Platform()
	if err != nil {
		return nil, err
	}

	fingerprint, err := motorid.IdentifyPlatform(m.Provider, p, nil)
	if err != nil {
		return nil, err
	}
	assetObj.PlatformIds = fingerprint.PlatformIDs
	if fingerprint.Name != "" {
		assetObj.Name = fingerprint.Name
	}

	log.Debug().Strs("identifier", assetObj.PlatformIds).Msg("motor connection")

	return assetObj, nil
}
