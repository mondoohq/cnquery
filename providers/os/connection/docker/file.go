// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package docker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v11/providers/os/connection/ssh/cat"
	"go.mondoo.com/cnquery/v11/providers/os/fsutil"
)

func FileOpen(dockerClient *client.Client, path string, container string, conn *ContainerConnection, catFs *cat.Fs) (afero.File, error) {
	f := &File{
		path:         path,
		dockerClient: dockerClient,
		container:    container,
		connection:   conn,
		catFs:        catFs,
	}
	err := f.Open()
	return f, err
}

type File struct {
	path         string
	container    string
	dockerClient *client.Client
	connection   *ContainerConnection
	reader       *bytes.Reader
	catFs        *cat.Fs
}

func (f *File) Open() error {
	r, _, err := f.getFileDockerReader(f.path)
	if err != nil {
		return os.ErrNotExist
	}
	defer r.Close()
	data, err := fsutil.ReadFileFromTarStream(r)
	if err != nil {
		return err
	}
	f.reader = bytes.NewReader(data)
	return nil
}

func (f *File) Close() error {
	return nil
}

func (f *File) Name() string {
	return f.path
}

func (f *File) Stat() (os.FileInfo, error) {
	return f.catFs.Stat(f.path)
}

func (f *File) Sync() error {
	return errors.New("not implemented")
}

func (f *File) Truncate(size int64) error {
	return errors.New("not implemented")
}

func (f *File) Read(b []byte) (n int, err error) {
	return f.reader.Read(b)
}

func (f *File) ReadAt(b []byte, off int64) (n int, err error) {
	return f.reader.ReadAt(b, off)
}

func (f *File) Readdir(count int) (res []os.FileInfo, err error) {
	return nil, errors.New("not implemented")
}

func (f *File) Readdirnames(n int) ([]string, error) {
	c, err := f.connection.RunCommand(fmt.Sprintf("find %s -maxdepth 1 -type d", f.path))
	if err != nil {
		return []string{}, err
	}

	content, err := io.ReadAll(c.Stdout)
	if err != nil {
		return []string{}, err
	}

	directories := strings.Split(string(content), "\n")

	// first result is always self
	if len(directories) > 0 {
		directories = directories[1:]
	}

	// extract names
	basenames := make([]string, len(directories))
	for i := range directories {
		basenames[i] = filepath.Base(directories[i])
	}
	return basenames, nil
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	return 0, errors.New("not implemented")
}

func (f *File) Write(b []byte) (n int, err error) {
	return 0, errors.New("not implemented")
}

func (f *File) WriteAt(b []byte, off int64) (n int, err error) {
	return 0, errors.New("not implemented")
}

func (f *File) WriteString(s string) (ret int, err error) {
	return 0, errors.New("not implemented")
}

func (f *File) getFileDockerReader(path string) (io.ReadCloser, dockertypes.ContainerPathStat, error) {
	r, stat, err := f.dockerClient.CopyFromContainer(context.Background(), f.container, path)

	// follow symlink if stat.LinkTarget is set
	if len(stat.LinkTarget) > 0 {
		return f.getFileDockerReader(stat.LinkTarget)
	}

	return r, stat, err
}

// returns a TarReader stream the caller is responsible for closing the stream
func (f *File) Tar() (io.ReadCloser, error) {
	r, _, err := f.getFileDockerReader(f.path)
	return r, err
}

// func (f *File) Exists() bool {
// 	if strings.HasPrefix(f.path, "/proc") {
// 		entries := f.procls()
// 		for i := range entries {
// 			if entries[i] == f.path {
// 				return true
// 			}
// 		}
// 		return false
// 	}

// 	r, _, err := f.getFileReader(f.path)
// 	if err != nil {
// 		return false
// 	}
// 	r.Close()
// 	return true
// }

// returns all directories and files under /proc
func (f *File) procls() []string {
	c, err := f.connection.RunCommand("find /proc")
	if err != nil {
		return []string{}
	}
	content, err := io.ReadAll(c.Stdout)
	if err != nil {
		return []string{}
	}

	// all files
	return strings.Split(string(content), "\n")
}
