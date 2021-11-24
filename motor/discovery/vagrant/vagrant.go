package vagrant

import (
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/common"
	"go.mondoo.io/mondoo/motor/motorid"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/local"
	"go.mondoo.io/mondoo/motor/transports/resolver"
	"go.mondoo.io/mondoo/motor/vault"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Vagrant Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{}
}

func (v *Resolver) Resolve(tc *transports.TransportConfig, cfn common.CredentialFn, sfn common.QuerySecretFn, userIdDetectors ...transports.PlatformIdDetector) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	localTransport, err := local.New()
	if err != nil {
		return nil, err
	}
	m, err := motor.New(localTransport)
	if err != nil {
		return nil, err
	}

	// we run status first, since vagrant ssh-config does not return a proper state
	// if in a multi-vm setup not all vms are running
	cmd, err := m.Transport.RunCommand("vagrant status")
	if err != nil {
		return nil, err
	}

	vmStatus, err := ParseVagrantStatus(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	// filter vms if a context was provided
	if len(tc.Host) > 0 {
		k := tc.Host
		vm, ok := vmStatus[k]
		if !ok {
			return nil, errors.New("could not find vagrant host: " + k)
		}

		if !vm {
			return nil, errors.New("vm is not ready: " + k)
		}

		cmd, err := m.Transport.RunCommand("vagrant ssh-config " + k)
		if err != nil {
			return nil, err
		}

		vmSshConfig, err := ParseVagrantSshConfig(cmd.Stdout)
		if err != nil {
			return nil, err
		}

		a, err := newVagrantAsset(vmSshConfig[k], tc)
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
			cmd, err := m.Transport.RunCommand("vagrant ssh-config " + k)
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
			a, err := newVagrantAsset(vagrantVms[i], tc)
			if err != nil {
				return nil, err
			}
			resolved = append(resolved, a)
		}
	}

	return resolved, nil
}

func newVagrantAsset(sshConfig *VagrantVmSSHConfig, rootTransportConfig *transports.TransportConfig) (*asset.Asset, error) {
	if sshConfig == nil {
		return nil, errors.New("missing vagrant ssh config")
	}

	cc := &transports.TransportConfig{
		// TODO: do we need to support winrm?
		Backend:  transports.TransportBackend_CONNECTION_SSH,
		Host:     sshConfig.HostName,
		Insecure: strings.ToLower(sshConfig.StrictHostKeyChecking) == "no",

		Port: strconv.Itoa(sshConfig.Port),
		Sudo: rootTransportConfig.Sudo,
	}

	// load secret
	credential, err := vault.NewPrivateKeyCredentialFromPath(sshConfig.User, sshConfig.IdentityFile, "")
	if err != nil {
		return nil, err
	}
	cc.AddCredential(credential)

	assetInfo := &asset.Asset{
		Name:        sshConfig.Host,
		PlatformIds: []string{},
		Connections: []*transports.TransportConfig{cc},
		Platform: &platform.Platform{
			Kind: transports.Kind_KIND_VIRTUAL_MACHINE,
		},
	}

	m, err := resolver.NewMotorConnection(cc, nil)
	if err != nil {
		return nil, err
	}
	defer m.Close()

	p, err := m.Platform()
	if err != nil {
		return nil, err
	}

	platformIds, err := motorid.GatherIDs(m.Transport, p, nil)
	if err != nil {
		return nil, err
	}
	assetInfo.PlatformIds = platformIds
	log.Debug().Strs("identifier", assetInfo.PlatformIds).Msg("motor connection")

	return assetInfo, nil
}
