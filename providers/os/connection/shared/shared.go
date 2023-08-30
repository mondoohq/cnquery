// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package shared

import (
	"io"
	"io/fs"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
)

type ConnectionType string

type Connection interface {
	RunCommand(command string) (*Command, error)
	FileInfo(path string) (FileInfoDetails, error)
	FileSystem() afero.Fs
	ID() uint32
	Name() string
	Type() ConnectionType
	Asset() *inventory.Asset
	Capabilities() Capabilities
}

type Command struct {
	Command    string
	Stats      PerfStats
	Stdout     io.ReadWriter
	Stderr     io.ReadWriter
	ExitStatus int
}

type Capabilities byte

const (
	Capability_RunCommand Capabilities = 1 << iota
	Capability_File
	Capability_FindFile
	Capability_FileSearch
)

func (c Capabilities) Has(other Capabilities) bool {
	return c&other == other
}

func (c Capabilities) String() []string {
	res := []string{}
	if c.Has(Capability_RunCommand) {
		res = append(res, "run-command")
	}
	if c.Has(Capability_File) {
		res = append(res, "file")
	}
	if c.Has(Capability_FindFile) {
		res = append(res, "find-file")
	}
	return res
}

type FileSearch interface {
	Find(from string, r *regexp.Regexp, typ string) ([]string, error)
}

type PerfStats struct {
	Start    time.Time     `json:"start"`
	Duration time.Duration `json:"duration"`
}

type FileInfo struct {
	FName    string
	FSize    int64
	FIsDir   bool
	FModTime time.Time
	FMode    os.FileMode
	Uid      int64
	Gid      int64
}

func (f *FileInfo) Name() string {
	return f.FName
}

func (f *FileInfo) Size() int64 {
	return f.FSize
}

func (f *FileInfo) Mode() os.FileMode {
	return f.FMode
}

func (f *FileInfo) ModTime() time.Time {
	return f.FModTime
}

func (f *FileInfo) IsDir() bool {
	return f.FIsDir
}

func (f *FileInfo) Sys() interface{} {
	return f
}

type FileInfoDetails struct {
	Size int64
	Mode FileModeDetails
	Uid  int64
	Gid  int64
}

type FileModeDetails struct {
	os.FileMode
}

func (mode FileModeDetails) UserReadable() bool {
	return uint32(mode.FileMode)&0o0400 != 0
}

func (mode FileModeDetails) UserWriteable() bool {
	return uint32(mode.FileMode)&0o0200 != 0
}

func (mode FileModeDetails) UserExecutable() bool {
	return uint32(mode.FileMode)&0o0100 != 0
}

func (mode FileModeDetails) GroupReadable() bool {
	return uint32(mode.FileMode)&0o0040 != 0
}

func (mode FileModeDetails) GroupWriteable() bool {
	return uint32(mode.FileMode)&0o0020 != 0
}

func (mode FileModeDetails) GroupExecutable() bool {
	return uint32(mode.FileMode)&0o0010 != 0
}

func (mode FileModeDetails) OtherReadable() bool {
	return uint32(mode.FileMode)&0o0004 != 0
}

func (mode FileModeDetails) OtherWriteable() bool {
	return uint32(mode.FileMode)&0o0002 != 0
}

func (mode FileModeDetails) OtherExecutable() bool {
	return uint32(mode.FileMode)&0o0001 != 0
}

func (mode FileModeDetails) Suid() bool {
	return mode.FileMode&fs.ModeSetuid != 0
}

func (mode FileModeDetails) Sgid() bool {
	return mode.FileMode&fs.ModeSetgid != 0
}

func (mode FileModeDetails) Sticky() bool {
	return mode.FileMode&fs.ModeSticky != 0
}

func (mode FileModeDetails) UnixMode() uint32 {
	m := mode.FileMode & 0o777

	if mode.IsDir() {
	}

	if (mode.FileMode & fs.ModeSetuid) != 0 {
		m |= 0o4000
	}

	if (mode.FileMode & fs.ModeSetgid) != 0 {
		m |= 0o2000
	}

	if (mode.FileMode & fs.ModeSticky) != 0 {
		m |= 0o1000
	}

	return uint32(m)
}

// TODO: unify the internal sudy and this data structure
func ParseSudo(flags map[string]*llx.Primitive) *inventory.Sudo {
	sudo := flags["sudo"]
	if sudo == nil {
		return nil
	}

	active := sudo.RawData().Value.(bool)
	if !active {
		return nil
	}

	return &inventory.Sudo{
		Active: true,
	}
}

func NewSudo() *Sudo {
	return &Sudo{
		user:       "",
		executable: "sudo",
		shell:      "",
	}
}

type Sudo struct {
	user       string
	executable string
	shell      string
}

func (sudo *Sudo) Build(cmd string) string {
	var sb strings.Builder

	sb.WriteString(sudo.executable)

	if len(sudo.user) > 0 {
		sb.WriteString(" -u " + sudo.user)
	}

	if len(sudo.shell) > 0 {
		sb.WriteString(" " + sudo.shell + " -c " + cmd)
	} else {
		sb.WriteString(" ")
		sb.WriteString(cmd)
	}

	return sb.String()
}

type Wrapper interface {
	Build(cmd string) string
}
