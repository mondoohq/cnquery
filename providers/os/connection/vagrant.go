package connection

import (
	"errors"
	"strings"

	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/providers/os/connection/shared"
	"go.mondoo.com/cnquery/providers/os/connection/vagrant"
)

const (
	Vagrant shared.ConnectionType = "vagrant"
)

var _ shared.Connection = &VagrantConnection{}

type VagrantConnection struct {
	SshConnection
}

func NewVagrantConnection(id uint32, conf *inventory.Config, asset *inventory.Asset) (*VagrantConnection, error) {
	// expect unix shell by default
	conn, err := resolveVagrantSshConf(id, conf, asset)
	if err != nil {
		return nil, err
	}
	res := VagrantConnection{
		SshConnection: *conn,
	}

	return &res, nil
}

func (p *VagrantConnection) ID() uint32 {
	return p.id
}

func (p *VagrantConnection) Name() string {
	return string(Vagrant)
}

func (p *VagrantConnection) Type() shared.ConnectionType {
	return Vagrant
}

func resolveVagrantSshConf(id uint32, conf *inventory.Config, root *inventory.Asset) (*SshConnection, error) {
	localProvider := NewLocalConnection(id, root)

	// we run status first, since vagrant ssh-config does not return a proper state
	// if in a multi-vm setup not all vms are running
	cmd, err := localProvider.RunCommand("vagrant status")
	if err != nil {
		return nil, err
	}

	vmStatus, err := vagrant.ParseVagrantStatus(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	k := conf.Host
	vm, ok := vmStatus[k]
	if !ok {
		return nil, errors.New("could not find vagrant host: " + k)
	}

	if !vm {
		return nil, errors.New("vm is not ready: " + k)
	}

	cmd, err = localProvider.RunCommand("vagrant ssh-config " + k)
	if err != nil {
		return nil, err
	}

	vmSshConfig, err := vagrant.ParseVagrantSshConfig(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	a, err := newVagrantAsset(id, vmSshConfig[k], conf)
	if err != nil {
		return nil, err
	}
	return NewSshConnection(id, a.Connections[0], a)
}

func newVagrantAsset(id uint32, sshConfig *vagrant.VagrantVmSSHConfig, rootTransportConfig *inventory.Config) (*inventory.Asset, error) {
	if sshConfig == nil {
		return nil, errors.New("missing vagrant ssh config")
	}

	cc := &inventory.Config{
		// TODO: do we need to support winrm?
		Backend:  "ssh",
		Type:     "ssh",
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
	cc.Credentials = append(cc.Credentials, credential)

	assetObj := &inventory.Asset{
		Name:        sshConfig.Host,
		PlatformIds: []string{},
		Connections: []*inventory.Config{cc},
		Platform: &inventory.Platform{
			// FIXME: use const like before?
			Kind: "vm",
		},
	}

	return assetObj, nil
}
