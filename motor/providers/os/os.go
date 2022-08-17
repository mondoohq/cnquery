package os

import (
	"io"
	"io/fs"
	"os"
	"regexp"
	"time"

	"go.mondoo.io/mondoo/motor/providers"

	"github.com/spf13/afero"
)

type OperatingSystemProvider interface {
	providers.Transport
	// RunCommand executes a command on the target system
	RunCommand(command string) (*Command, error)
	// returns file permissions and ownership
	FileInfo(path string) (FileInfoDetails, error)
	// FS provides access to the file system of the target system
	FS() afero.Fs
}

type PerfStats struct {
	Start    time.Time     `json:"start"`
	Duration time.Duration `json:"duration"`
}

type Command struct {
	Command    string
	Stats      PerfStats
	Stdout     io.ReadWriter
	Stderr     io.ReadWriter
	ExitStatus int
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

type FileInfoDetails struct {
	Size int64
	Mode FileModeDetails
	Uid  int64
	Gid  int64
}

type FileSearch interface {
	Find(from string, r *regexp.Regexp, typ string) ([]string, error)
}
