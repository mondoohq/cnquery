package docker_engine

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/client"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/motor/providers/ssh/cat"
)

type FS struct {
	Container    string
	dockerClient *client.Client
	Transport    *Provider
	catFS        *cat.Fs
}

func (fs *FS) Name() string {
	return "dockerfs"
}

func (fs *FS) Create(name string) (afero.File, error) {
	return nil, errors.New("create not implemented")
}

func (fs *FS) Mkdir(name string, perm os.FileMode) error {
	return errors.New("mkdir not implemented")
}

func (fs *FS) MkdirAll(path string, perm os.FileMode) error {
	return errors.New("mkdirall not implemented")
}

func isDockerClientSupported(path string) bool {
	// This is incomplete. There are other things that are
	// unsupported like tmpfs and paths the user mounted
	// in the container.
	// See https://docs.docker.com/engine/reference/commandline/cp/#corner-cases
	unsupported := []string{"/proc", "/dev", "/sys"}
	for _, v := range unsupported {
		if v == path || strings.HasPrefix(path, v) {
			return false
		}
	}
	return true
}

func (fs *FS) Open(name string) (afero.File, error) {
	if isDockerClientSupported(name) {
		return FileOpen(fs.dockerClient, name, fs.Container, fs.Transport)
	} else {
		return fs.catFS.Open(name)
	}
}

func (fs *FS) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return nil, errors.New("openfile not implemented")
}

func (fs *FS) Remove(name string) error {
	return errors.New("remove not implemented")
}

func (fs *FS) RemoveAll(path string) error {
	return errors.New("removeall not implemented")
}

func (fs *FS) Rename(oldname, newname string) error {
	return errors.New("rename not implemented")
}

func (fs *FS) Stat(name string) (os.FileInfo, error) {
	f, err := fs.Open(name)
	if err != nil {
		return nil, err
	}
	return f.Stat()
}

func (fs *FS) Chmod(name string, mode os.FileMode) error {
	return errors.New("chmod not implemented")
}

func (fs *FS) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return errors.New("chtimes not implemented")
}

func (fs *FS) Chown(name string, uid, gid int) error {
	return errors.New("chown not implemented")
}
