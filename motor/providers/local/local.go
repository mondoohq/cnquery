package local

import (
	"runtime"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/shared"
)

var _ providers.Transport = (*LocalTransport)(nil)

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
		fs:    afero.NewOsFs(),
		// kind:    endpoint.Kind,
		// runtime: endpoint.Runtime,
	}, nil
}

type LocalTransport struct {
	shell   []string
	fs      afero.Fs
	kind    providers.Kind
	runtime string
}

func (t *LocalTransport) RunCommand(command string) (*providers.Command, error) {
	log.Debug().Msgf("local> run command %s", command)
	c := &shared.Command{Shell: t.shell}
	args := []string{}

	res, err := c.Exec(command, args)
	return res, err
}

func (t *LocalTransport) FS() afero.Fs {
	return t.fs
}

func (t *LocalTransport) FileInfo(path string) (providers.FileInfoDetails, error) {
	fs := t.FS()
	afs := &afero.Afero{Fs: fs}
	stat, err := afs.Stat(path)
	if err != nil {
		return providers.FileInfoDetails{}, err
	}

	uid, gid := t.fileowner(stat)

	mode := stat.Mode()
	return providers.FileInfoDetails{
		Mode: providers.FileModeDetails{mode},
		Size: stat.Size(),
		Uid:  uid,
		Gid:  gid,
	}, nil
}

func (t *LocalTransport) Close() {
	// TODO: we need to close all commands and file handles
}

func (t *LocalTransport) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		providers.Capability_RunCommand,
		providers.Capability_File,
	}
}

func (t *LocalTransport) Kind() providers.Kind {
	return t.kind
}

func (t *LocalTransport) Runtime() string {
	return t.runtime
}

func (t *LocalTransport) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.HostnameDetector,
		providers.CloudDetector,
	}
}
