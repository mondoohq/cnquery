package local

import (
	"runtime"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/transports"
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

func (t *LocalTransport) RunCommand(command string) (*transports.Command, error) {
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

func (t *LocalTransport) FileInfo(path string) (transports.FileInfoDetails, error) {
	fs := t.FS()
	afs := &afero.Afero{Fs: fs}
	stat, err := afs.Stat(path)
	if err != nil {
		return transports.FileInfoDetails{}, err
	}

	uid, gid := t.fileowner(stat)

	mode := stat.Mode()
	return transports.FileInfoDetails{
		Mode: transports.FileModeDetails{mode},
		Size: stat.Size(),
		Uid:  uid,
		Gid:  gid,
	}, nil
}

func (t *LocalTransport) Close() {
	// TODO: we need to close all commands and file handles
}

func (t *LocalTransport) Capabilities() transports.Capabilities {
	return transports.Capabilities{
		transports.Cabability_RunCommand,
		transports.Cabability_File,
	}
}

func (t *LocalTransport) Kind() assets.Kind {
	return assets.Kind_KIND_BARE_METAL
}

func (t *LocalTransport) Runtime() string {
	return ""
}
