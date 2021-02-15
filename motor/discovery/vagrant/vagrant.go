package vagrant

import (
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/local"
)

type Resolver struct{}

func (k *Resolver) Name() string {
	return "Vagrant Resolver"
}

type vagrantContext struct {
	Host string
}

func (r *Resolver) ParseConnectionURL(url string, opts ...transports.TransportConfigOption) (*transports.TransportConfig, error) {
	host := strings.TrimPrefix(url, "vagrant://")
	tc := &transports.TransportConfig{
		Host: host,
	}

	for i := range opts {
		opts[i](tc)
	}

	return tc, nil
}

func (v *Resolver) Resolve(t *transports.TransportConfig, opts map[string]string) ([]*asset.Asset, error) {
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
	if len(t.Host) > 0 {
		k := t.Host
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

		resolved = append(resolved, vagrantToAsset(vmSshConfig[k], t))

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
			resolved = append(resolved, vagrantToAsset(vagrantVms[i], t))
		}
	}

	return resolved, nil
}

func vagrantToAsset(sshConfig *VagrantVmSSHConfig, rootTransportConfig *transports.TransportConfig) *asset.Asset {
	if sshConfig == nil {
		return nil
	}

	return &asset.Asset{
		Name: sshConfig.Host,
		Connections: []*transports.TransportConfig{{
			// TODO: do we need to support winrm?
			Backend:       transports.TransportBackend_CONNECTION_SSH,
			Host:          sshConfig.HostName,
			IdentityFiles: []string{sshConfig.IdentityFile},
			Insecure:      strings.ToLower(sshConfig.StrictHostKeyChecking) == "no",
			User:          sshConfig.User,
			Port:          strconv.Itoa(sshConfig.Port),
			Sudo:          rootTransportConfig.Sudo,
		}},
		Platform: &platform.Platform{
			Kind: transports.Kind_KIND_VIRTUAL_MACHINE,
		},
	}
}
