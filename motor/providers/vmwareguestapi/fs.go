package vmwareguestapi

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/providers/os/statutil"
	"go.mondoo.io/mondoo/motor/providers/ssh/cat"
	"go.mondoo.io/mondoo/motor/providers/vmwareguestapi/toolbox"
)

// NOTE: this is not useable since simple file transfers like
// /etc/os-release are throwing errors
type VmwareGuestFs struct {
	tb            *toolbox.Client
	commandRunner cat.CommandRunner
}

var notImplementedError = errors.New("not implemented")

func (vfs *VmwareGuestFs) Name() string {
	return "Vmware GuestFS"
}

func (vfs *VmwareGuestFs) Create(name string) (afero.File, error) {
	return nil, notImplementedError
}

func (vfs VmwareGuestFs) Mkdir(name string, perm os.FileMode) error {
	return notImplementedError
}

func (vfs *VmwareGuestFs) MkdirAll(path string, perm os.FileMode) error {
	return notImplementedError
}

func (vfs *VmwareGuestFs) Open(name string) (afero.File, error) {
	// for now this methods is not reliable for all paths on the os
	// https://communities.vmware.com/thread/624928
	ctx := context.Background()
	rc, _, err := vfs.tb.Download(ctx, name)
	if err != nil {
		return nil, err
	}

	return NewFile(name, rc), nil
}

func (vfs *VmwareGuestFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return nil, notImplementedError
}

func (vfs *VmwareGuestFs) Remove(name string) error {
	return notImplementedError
}

func (vfs *VmwareGuestFs) RemoveAll(path string) error {
	return notImplementedError
}

func (VmwareGuestFs) Rename(oldname, newname string) error {
	return notImplementedError
}

// needs to be implemented
func (vfs *VmwareGuestFs) Stat(path string) (os.FileInfo, error) {
	return statutil.New(vfs.commandRunner).Stat(path)
}

func (vfs *VmwareGuestFs) Chmod(name string, mode os.FileMode) error {
	return notImplementedError
}

func (vfs *VmwareGuestFs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return notImplementedError
}
