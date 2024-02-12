// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/masterzen/winrm"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
	winrmConn "go.mondoo.com/cnquery/v10/providers/os/connection/winrm"
	"go.mondoo.com/cnquery/v10/providers/os/connection/winrm/cat"
)

var _ shared.Connection = (*WinrmConnection)(nil)

func VerifyConfig(config *inventory.Config) (*winrm.Endpoint, error) {
	if config.Type != string(shared.Type_Winrm) {
		return nil, errors.New("only winrm backend for winrm transport supported")
	}

	winrmEndpoint := &winrm.Endpoint{
		Host: config.Host,
		Port: int(config.Port),
		// everything about winrm is insecure, therefore we always disable TLS verification since
		// only very few actually use valid certificates that are not self-signed
		Insecure: true,
		HTTPS:    true,
		Timeout:  time.Duration(0),
	}

	return winrmEndpoint, nil
}

// NewWinrmConnection creates a winrm client and establishes a connection to verify the connection
func NewWinrmConnection(id uint32, conf *inventory.Config, asset *inventory.Asset) (*WinrmConnection, error) {
	// ensure all required configs are set
	winrmEndpoint, err := VerifyConfig(conf)
	if err != nil {
		return nil, err
	}

	// set default config if required
	winrmEndpoint = winrmConn.DefaultConfig(winrmEndpoint)

	params := winrm.DefaultParameters
	params.TransportDecorator = func() winrm.Transporter { return &winrm.ClientNTLM{} }

	// search for password secret
	c, err := vault.GetPassword(conf.Credentials)
	if err != nil {
		return nil, errors.New("missing password for winrm transport")
	}

	client, err := winrm.NewClientWithParameters(winrmEndpoint, c.User, string(c.Secret), params)
	if err != nil {
		return nil, err
	}

	// test connection
	log.Debug().Str("user", c.User).Str("host", conf.Host).Msg("winrm> connecting to remote shell via WinRM")
	shell, err := client.CreateShell()
	if err != nil {
		return nil, err
	}

	err = shell.Close()
	if err != nil {
		return nil, err
	}

	log.Debug().Msg("winrm> connection established")
	conn := &WinrmConnection{
		id:       id,
		conf:     conf,
		asset:    asset,
		Endpoint: winrmEndpoint,
		Client:   client,
	}
	if len(asset.Connections) > 0 && asset.Connections[0].ParentConnectionId > 0 {
		conn.parentId = &asset.Connections[0].ParentConnectionId
	}
	return conn, nil
}

type WinrmConnection struct {
	id       uint32
	parentId *uint32
	conf     *inventory.Config
	asset    *inventory.Asset

	fs afero.Fs

	Endpoint *winrm.Endpoint
	Client   *winrm.Client
}

func (c *WinrmConnection) ID() uint32 {
	return c.id
}

func (c *WinrmConnection) ParentID() *uint32 {
	return c.parentId
}

func (c *WinrmConnection) Name() string {
	return "ssh"
}

func (c *WinrmConnection) Type() shared.ConnectionType {
	return shared.Type_Winrm
}

func (p *WinrmConnection) Asset() *inventory.Asset {
	return p.asset
}

func (p *WinrmConnection) Capabilities() shared.Capabilities {
	return shared.Capability_File | shared.Capability_RunCommand
}

func (p *WinrmConnection) RunCommand(command string) (*shared.Command, error) {
	log.Debug().Str("command", command).Str("provider", "winrm").Msg("winrm> run command")

	stdoutBuffer := &bytes.Buffer{}
	stderrBuffer := &bytes.Buffer{}

	res := &shared.Command{
		Command: command,
		Stats: shared.PerfStats{
			Start: time.Now(),
		},
		Stdout: stdoutBuffer,
		Stderr: stderrBuffer,
	}
	defer func() {
		res.Stats.Duration = time.Since(res.Stats.Start)
	}()

	// Note: winrm does not return err of the command was executed with a non-zero exit code
	exitCode, err := p.Client.RunWithContext(context.Background(), command, stdoutBuffer, stderrBuffer)
	if err != nil {
		log.Error().Err(err).Str("command", command).Msg("could not execute winrm command")
		return res, err
	}

	res.ExitStatus = exitCode
	return res, nil
}

func (p *WinrmConnection) FileInfo(path string) (shared.FileInfoDetails, error) {
	fs := p.FileSystem()
	afs := &afero.Afero{Fs: fs}
	stat, err := afs.Stat(path)
	if err != nil {
		return shared.FileInfoDetails{}, err
	}

	uid := int64(-1)
	gid := int64(-1)
	mode := stat.Mode()

	return shared.FileInfoDetails{
		Mode: shared.FileModeDetails{mode},
		Size: stat.Size(),
		Uid:  uid,
		Gid:  gid,
	}, nil
}

func (p *WinrmConnection) FileSystem() afero.Fs {
	if p.fs == nil {
		p.fs = cat.New(p)
	}
	return p.fs
}

func (p *WinrmConnection) Close() {
	// nothing to do yet
}
