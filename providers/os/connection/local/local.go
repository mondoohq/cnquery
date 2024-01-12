// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"bytes"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v10/providers/os/connection/ssh/cat"
)

type LocalConnection struct {
	shell   []string
	fs      afero.Fs
	Sudo    *inventory.Sudo
	runtime string
	id      uint32
	asset   *inventory.Asset
}

func NewConnection(id uint32, conf *inventory.Config, asset *inventory.Asset) *LocalConnection {
	// expect unix shell by default
	res := LocalConnection{
		id:    id,
		asset: asset,
	}
	if conf != nil {
		res.Sudo = conf.Sudo
	}

	if runtime.GOOS == "windows" {
		// It does not make any sense to use cmd as default shell
		// shell = []string{"cmd", "/C"}
		res.shell = []string{"powershell", "-c"}
	} else {
		res.shell = []string{"sh", "-c"}
	}

	return &res
}

func (p *LocalConnection) ID() uint32 {
	return p.id
}

func (p *LocalConnection) Name() string {
	return "local"
}

func (p *LocalConnection) Type() shared.ConnectionType {
	return shared.Type_Local
}

func (p *LocalConnection) Asset() *inventory.Asset {
	return p.asset
}

func (p *LocalConnection) Capabilities() shared.Capabilities {
	return shared.Capability_File | shared.Capability_RunCommand
}

func (p *LocalConnection) RunCommand(command string) (*shared.Command, error) {
	if p.Sudo != nil {
		command = shared.BuildSudoCommand(p.Sudo, command)
	}
	log.Debug().Msgf("local> run command %s", command)
	c := &CommandRunner{Shell: p.shell}
	args := []string{}

	res, err := c.Exec(command, args)
	return res, err
}

func (p *LocalConnection) FileSystem() afero.Fs {
	if p.fs != nil {
		return p.fs
	}

	if p.Sudo != nil && p.Sudo.Active {
		p.fs = cat.New(p)
	} else {
		p.fs = afero.NewOsFs()
	}

	return p.fs
}

func (p *LocalConnection) FileInfo(path string) (shared.FileInfoDetails, error) {
	fs := p.FileSystem()
	afs := &afero.Afero{Fs: fs}
	stat, err := afs.Stat(path)
	if err != nil {
		return shared.FileInfoDetails{}, err
	}

	uid, gid := p.fileowner(stat)

	mode := stat.Mode()
	return shared.FileInfoDetails{
		Mode: shared.FileModeDetails{mode},
		Size: stat.Size(),
		Uid:  uid,
		Gid:  gid,
	}, nil
}

func (p *LocalConnection) Close() {
	// TODO: we need to close all commands and file handles
}

type CommandRunner struct {
	shared.Command
	cmdExecutor *exec.Cmd
	Shell       []string
}

func (c *CommandRunner) Exec(usercmd string, args []string) (*shared.Command, error) {
	c.Command.Stats.Start = time.Now()

	var cmd string
	cmdArgs := []string{}

	if len(c.Shell) > 0 {
		shellCommand, shellArgs := c.Shell[0], c.Shell[1:]
		cmd = shellCommand
		cmdArgs = append(cmdArgs, shellArgs...)
		cmdArgs = append(cmdArgs, usercmd)
	} else {
		cmd = usercmd
	}
	cmdArgs = append(cmdArgs, args...)

	// this only stores the user command, not the shell
	c.Command.Command = usercmd + " " + strings.Join(args, " ")
	c.cmdExecutor = exec.Command(cmd, cmdArgs...)

	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer

	// create buffered stream
	c.Command.Stdout = &stdoutBuffer
	c.Command.Stderr = &stderrBuffer

	c.cmdExecutor.Stdout = c.Command.Stdout
	c.cmdExecutor.Stderr = c.Command.Stderr

	err := c.cmdExecutor.Run()
	c.Command.Stats.Duration = time.Since(c.Command.Stats.Start)

	// command completed successfully, great :-)
	if err == nil {
		return &c.Command, nil
	}

	// if the program failed, we do not return err but its exit code
	if exiterr, ok := err.(*exec.ExitError); ok {
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			c.Command.ExitStatus = status.ExitStatus()
		}
		return &c.Command, nil
	}

	// all other errors are real errors and not expected
	return &c.Command, err
}
