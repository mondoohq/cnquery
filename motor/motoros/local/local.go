package local

import (
	"runtime"
	"syscall"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/motoros/capabilities"
	"go.mondoo.io/mondoo/motor/motoros/types"
	"go.mondoo.io/mondoo/nexus/assets"
)

func New() (*LocalTransport, error) {

	// expect unix shell by default
	shell := []string{"sh", "-c"}

	if runtime.GOOS == "windows" {
		// It does not make any sense to use cmd as default shell
		// shell = []string{"cmd", "/C"}
		shell = []string{"powershell", "-c"}
	}

	return &LocalTransport{
		shell: shell,
	}, nil
}

type LocalTransport struct {
	shell []string
	fs    afero.Fs
}

func (t *LocalTransport) RunCommand(command string) (*types.Command, error) {
	log.Debug().Msgf("local> run command %s", command)
	c := &Command{shell: t.shell}
	args := []string{}

	res, err := c.Exec(command, args)
	return res, err
}

func (t *LocalTransport) FS() afero.Fs {
	if t.fs == nil {
		t.fs = afero.NewOsFs()
	}
	return t.fs
}

func (t *LocalTransport) File(path string) (afero.File, error) {
	return t.FS().Open(path)
}

func (t *LocalTransport) FileInfo(path string) (types.FileInfoDetails, error) {
	fs := t.FS()
	afs := &afero.Afero{Fs: fs}
	stat, err := afs.Stat(path)
	if err != nil {
		return types.FileInfoDetails{}, err
	}

	uid := int64(-1)
	gid := int64(-1)
	if stat, ok := stat.Sys().(*syscall.Stat_t); ok {
		uid = int64(stat.Uid)
		gid = int64(stat.Gid)
	}
	mode := stat.Mode()

	return types.FileInfoDetails{
		Mode: types.FileModeDetails{mode},
		Size: stat.Size(),
		Uid:  uid,
		Gid:  gid,
	}, nil
}

func (t *LocalTransport) Close() {
	// TODO: we need to close all commands and file handles
}

func (t *LocalTransport) Capabilities() []capabilities.Capability {
	return []capabilities.Capability{
		capabilities.RunCommand,
		capabilities.File,
	}
}

func (t *LocalTransport) Kind() assets.Kind {
	return assets.Kind_KIND_BARE_METAL
}

func (t *LocalTransport) Runtime() string {
	return ""
}
