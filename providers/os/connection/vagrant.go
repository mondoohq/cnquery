// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"strings"

	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/providers/os/connection/shared"
	"go.mondoo.com/cnquery/providers/os/connection/vagrant"
	"go.mondoo.com/cnquery/providers/os/id/ids"
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
	// For now, we do not provide the conf to the local connection
	// conf might include sudo, which is only intended for the actual vagrant connection
	// local currently does not need it. Quite the contrary, it cause issues.
	localProvider := NewLocalConnection(id, nil, root)

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

	err = migrateVagrantAssetToSsh(id, vmSshConfig[k], conf, root)
	if err != nil {
		return nil, err
	}
	return NewSshConnection(id, root.Connections[0], root)
}

func migrateVagrantAssetToSsh(id uint32, sshConfig *vagrant.VagrantVmSSHConfig, rootTransportConfig *inventory.Config, asset *inventory.Asset) error {
	if sshConfig == nil {
		return errors.New("missing vagrant ssh config")
	}

	cc := &inventory.Config{
		// TODO: do we need to support winrm?
		Type:     "ssh",
		Runtime:  "vagrant",
		Host:     sshConfig.HostName,
		Insecure: strings.ToLower(sshConfig.StrictHostKeyChecking) == "no",

		Port: int32(sshConfig.Port),
		Sudo: rootTransportConfig.Sudo,
	}

	// load secret
	credential, err := vault.NewPrivateKeyCredentialFromPath(sshConfig.User, sshConfig.IdentityFile, "")
	if err != nil {
		return err
	}
	cc.Credentials = append(cc.Credentials, credential)

	asset.Name = sshConfig.Host
	asset.Connections = []*inventory.Config{cc}
	asset.IdDetector = []string{ids.IdDetector_Hostname, ids.IdDetector_SshHostkey}

	return nil
}
