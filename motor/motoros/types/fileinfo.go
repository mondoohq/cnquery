package types

import (
	"os"
	"time"
)

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
	return uint32(mode.FileMode)&00400 != 0
}
func (mode FileModeDetails) UserWriteable() bool {
	return uint32(mode.FileMode)&00200 != 0
}
func (mode FileModeDetails) UserExecutable() bool {
	return uint32(mode.FileMode)&00100 != 0
}
func (mode FileModeDetails) GroupReadable() bool {
	return uint32(mode.FileMode)&00040 != 0
}
func (mode FileModeDetails) GroupWriteable() bool {
	return uint32(mode.FileMode)&00020 != 0
}
func (mode FileModeDetails) GroupExecutable() bool {
	return uint32(mode.FileMode)&00010 != 0
}
func (mode FileModeDetails) OtherReadable() bool {
	return uint32(mode.FileMode)&00004 != 0
}
func (mode FileModeDetails) OtherWriteable() bool {
	return uint32(mode.FileMode)&00002 != 0
}
func (mode FileModeDetails) OtherExecutable() bool {
	return uint32(mode.FileMode)&00001 != 0
}
func (mode FileModeDetails) Suid() bool {
	return uint32(mode.FileMode)&04000 != 0
}
func (mode FileModeDetails) Sgid() bool {
	return uint32(mode.FileMode)&02000 != 0
}
func (mode FileModeDetails) Sticky() bool {
	return uint32(mode.FileMode)&01000 != 0
}

type FileInfoDetails struct {
	Size int64
	Mode FileModeDetails
	Uid  int64
	Gid  int64
}
