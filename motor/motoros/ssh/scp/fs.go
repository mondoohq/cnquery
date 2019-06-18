package scp

import (
	"errors"
	"io/ioutil"
	"os"
	"time"

	scp_client "github.com/hnakamur/go-scp"
	"github.com/spf13/afero"
	"golang.org/x/crypto/ssh"
)

func NewFs(client *ssh.Client) *Fs {
	return &Fs{
		sshClient: client,
		scpClient: scp_client.NewSCP(client),
	}
}

type Fs struct {
	sshClient *ssh.Client
	scpClient *scp_client.SCP
}

func (s Fs) Name() string { return "scpfs" }

func (s Fs) Create(name string) (afero.File, error) {
	return nil, errors.New("not implemented")
}

func (s Fs) Mkdir(name string, perm os.FileMode) error {
	return errors.New("not implemented")
}

func (s Fs) MkdirAll(path string, perm os.FileMode) error {
	return errors.New("not implemented")
}

func (s Fs) Open(path string) (afero.File, error) {
	return FileOpen(s.scpClient, path)
}

func (s Fs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return nil, errors.New("not implemented")
}

func (s Fs) Remove(name string) error {
	return errors.New("not implemented")
}

func (s Fs) RemoveAll(path string) error {
	return errors.New("not implemented")
}

func (s Fs) Rename(oldname, newname string) error {
	return errors.New("not implemented")
}

func (s Fs) Stat(path string) (os.FileInfo, error) {
	return s.scpClient.Receive(path, ioutil.Discard)
}

func (s Fs) Lstat(p string) (os.FileInfo, error) {
	return nil, errors.New("not implemented")
}

func (s Fs) Chmod(name string, mode os.FileMode) error {
	return errors.New("not implemented")
}

func (s Fs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return errors.New("not implemented")
}
