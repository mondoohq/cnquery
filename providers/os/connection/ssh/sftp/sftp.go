// Copyright © 2015 Jerry Jacobs <jerry.jacobs@xor-gate.org>.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sftp

import (
	"os"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/pkg/sftp"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v9/providers/os/connection/ssh/cat"
	"golang.org/x/crypto/ssh"
)

// Fs is a afero.Fs implementation that uses functions provided by the sftp package.
//
// For details in any method, check the documentation of the sftp package
// (github.com/pkg/sftp).
type Fs struct {
	client *sftp.Client
	catFs  *cat.Fs
}

func New(commandRunner cat.CommandRunner, client *ssh.Client) (afero.Fs, error) {
	ftpClient, err := sftpClient(client)
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize sftp backend")
	}

	return &Fs{
		client: ftpClient,
		catFs:  cat.New(commandRunner),
	}, nil
}

func sftpClient(sshClient *ssh.Client) (*sftp.Client, error) {
	c, err := sftp.NewClient(sshClient, sftp.MaxPacket(1<<15))
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s Fs) Name() string { return "sftpfs" }

func (s Fs) Create(name string) (afero.File, error) {
	return FileCreate(s.client, name)
}

func (s Fs) Mkdir(name string, perm os.FileMode) error {
	err := s.client.Mkdir(name)
	if err != nil {
		return err
	}
	return s.client.Chmod(name, perm)
}

func (s Fs) MkdirAll(path string, perm os.FileMode) error {
	// Fast path: if we can tell whether path is a directory or file, stop with success or error.
	dir, err := s.Stat(path)
	if err == nil {
		if dir.IsDir() {
			return nil
		}
		return err
	}

	// Slow path: make sure parent exists and then call Mkdir for path.
	i := len(path)
	for i > 0 && os.IsPathSeparator(path[i-1]) { // Skip trailing path separator.
		i--
	}

	j := i
	for j > 0 && !os.IsPathSeparator(path[j-1]) { // Scan backward over element.
		j--
	}

	if j > 1 {
		// Create parent
		err = s.MkdirAll(path[0:j-1], perm)
		if err != nil {
			return err
		}
	}

	// Parent now exists; invoke Mkdir and use its result.
	err = s.Mkdir(path, perm)
	if err != nil {
		// Handle arguments like "foo/." by
		// double-checking that directory doesn't exist.
		dir, err1 := s.Lstat(path)
		if err1 == nil && dir.IsDir() {
			return nil
		}
		return err
	}
	return nil
}

func (s Fs) Open(path string) (afero.File, error) {
	// NOTE: procfs cannot be read via scp, so we fall-back to catfs all paths there
	if strings.HasPrefix(path, "/proc") {
		return s.catFs.Open(path)
	}

	return FileOpen(s.client, path)
}

func (s Fs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	// sftp client does not support mode
	sshfsFile, err := s.client.OpenFile(name, flag)
	if err != nil {
		return nil, err
	}
	return &File{fd: sshfsFile}, nil
}

func (s Fs) Remove(name string) error {
	return s.client.Remove(name)
}

func (s Fs) RemoveAll(path string) error {
	// TODO have a look at os.RemoveAll
	// https://github.com/golang/go/blob/master/src/os/path.go#L66
	return errors.New("removeall not implemented")
}

func (s Fs) Rename(oldname, newname string) error {
	return s.client.Rename(oldname, newname)
}

func (s Fs) Stat(name string) (os.FileInfo, error) {
	return s.client.Stat(name)
}

func (s Fs) Lstat(p string) (os.FileInfo, error) {
	return s.client.Lstat(p)
}

func (s Fs) Chmod(name string, mode os.FileMode) error {
	return s.client.Chmod(name, mode)
}

func (s Fs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return s.client.Chtimes(name, atime, mtime)
}

func (s Fs) Chown(name string, uid, gid int) error {
	return s.client.Chown(name, uid, gid)
}
