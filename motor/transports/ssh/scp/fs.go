package scp

import (
	"errors"
	"os"
	"time"

	scp_client "github.com/hnakamur/go-scp"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/transports/ssh/cat"
	"go.mondoo.io/mondoo/motor/transports/statutil"
	"golang.org/x/crypto/ssh"
)

func NewFs(commandRunner cat.CommandRunner, client *ssh.Client) *Fs {
	return &Fs{
		sshClient:     client,
		scpClient:     scp_client.NewSCP(client),
		commandRunner: commandRunner,
	}
}

type Fs struct {
	sshClient     *ssh.Client
	scpClient     *scp_client.SCP
	commandRunner cat.CommandRunner
}

func (s Fs) Name() string { return "scpfs" }

func (s Fs) Create(name string) (afero.File, error) {
	return nil, errors.New("create not implemented")
}

func (s Fs) Mkdir(name string, perm os.FileMode) error {
	return errors.New("mkdir not implemented")
}

func (s Fs) MkdirAll(path string, perm os.FileMode) error {
	return errors.New("mkdirall not implemented")
}

func (s Fs) Open(path string) (afero.File, error) {
	return FileOpen(s.scpClient, path)
}

func (s Fs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return nil, errors.New("openfile not implemented")
}

func (s Fs) Remove(name string) error {
	return errors.New("remove not implemented")
}

func (s Fs) RemoveAll(path string) error {
	return errors.New("removeall not implemented")
}

func (s Fs) Rename(oldname, newname string) error {
	return errors.New("rename not implemented")
}

func (s Fs) Stat(path string) (os.FileInfo, error) {
	// NOTE we cannot use s.scpClient.Receive(path, ioutil.Discard) since it would not work with directories
	return statutil.New(s.commandRunner).Stat(path)
}

func (s Fs) Lstat(p string) (os.FileInfo, error) {
	return nil, errors.New("lstat not implemented")
}

func (s Fs) Chmod(name string, mode os.FileMode) error {
	return errors.New("chmod not implemented")
}

func (s Fs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return errors.New("chtimes not implemented")
}

func (s Fs) Chown(name string, uid, gid int) error {
	return errors.New("chown not implemented")
}
