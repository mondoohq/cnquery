package cat

import (
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"os"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/kballard/go-shellquote"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/statutil"
)

type CommandRunner interface {
	RunCommand(command string) (*transports.Command, error)
}

func New(cmdRunner CommandRunner) *CatFs {
	return &CatFs{
		commandRunner: cmdRunner,
	}
}

type CatFs struct {
	commandRunner CommandRunner
	base64        *bool
}

func (cat *CatFs) Name() string {
	return "Cat FS"
}

func (cat *CatFs) useBase64encoding() bool {
	if cat.base64 != nil {
		return *cat.base64
	}

	b := cat.base64available()
	cat.base64 = &b
	return b
}

func (cat *CatFs) base64available() bool {
	cmd, err := cat.commandRunner.RunCommand("command -v base64")
	if err != nil {
		log.Debug().Msg("base64 command not found on target system")
		return false
	}
	log.Debug().Msg("use base64 encoding for data transfer")
	return cmd.ExitStatus == 0
}

func (cat *CatFs) Open(name string) (afero.File, error) {
	// we need shellquote to escape filenames with spaces
	catCmd := shellquote.Join("cat", name)
	if cat.useBase64encoding() {
		catCmd = catCmd + " | base64"
	}

	cmd, err := cat.commandRunner.RunCommand(catCmd)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	if cat.useBase64encoding() {
		log.Debug().Msg(string(data))

		data, err = base64.StdEncoding.DecodeString(string(data))
		if err != nil {
			return nil, errors.Wrap(err, "could not decode base64 data stream")
		}
	}

	return NewFile(cat, name, bytes.NewBuffer(data)), nil
}

func (cat *CatFs) Stat(name string) (os.FileInfo, error) {
	return statutil.New(cat.commandRunner).Stat(name)
}

func (cat *CatFs) Create(name string) (afero.File, error) {
	return nil, errors.New("not implemented")
}

func (cat *CatFs) Mkdir(name string, perm os.FileMode) error {
	return errors.New("not implemented")
}

func (cat *CatFs) MkdirAll(path string, perm os.FileMode) error {
	return errors.New("not implemented")
}

func (cat *CatFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return nil, errors.New("not implemented")
}

func (cat *CatFs) Remove(name string) error {
	return errors.New("not implemented")
}

func (cat *CatFs) RemoveAll(path string) error {
	return errors.New("not implemented")
}

func (cat *CatFs) Rename(oldname, newname string) error {
	return errors.New("not implemented")
}

func (cat *CatFs) Chmod(name string, mode os.FileMode) error {
	return errors.New("not implemented")
}

func (cat *CatFs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return errors.New("not implemented")
}
