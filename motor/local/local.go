package local

import (
	"runtime"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/types"
)

func New() (*LocalTransport, error) {

	// expect unix shell by default
	shell := []string{"sh", "-c"}

	if runtime.GOOS == "windows" {
		shell = []string{"cmd", "/C"}
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

func (t *LocalTransport) Close() {
	// TODO: we need to close all commands and file handles

}
