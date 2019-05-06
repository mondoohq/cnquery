package docker_engine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"archive/tar"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/fsutil"
	"go.mondoo.io/mondoo/motor/types"
)

func FileOpen(dockerClient *client.Client, path string, container string, transport *Transport) (*File, error) {
	f := &File{
		path:         path,
		dockerClient: dockerClient,
		container:    container,
		transport:    transport,
	}
	err := f.Open()
	return f, err
}

type File struct {
	path         string
	container    string
	dockerClient *client.Client
	transport    *Transport
	reader       *bytes.Reader
}

func (f *File) Open() error {
	r, _, err := f.getFileReader(f.path)
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
	r, dstat, err := f.getFileReader(f.path)
	if err != nil {
		return nil, err
	}
	r.Close()

	return &types.FileInfo{
		FMode:    dstat.Mode,
		FSize:    dstat.Size,
		FName:    dstat.Name,
		FModTime: dstat.Mtime,
	}, nil
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
	c, err := f.transport.RunCommand(fmt.Sprintf("find %s -maxdepth 1 -type d", f.path))
	if err != nil {
		return []string{}, err
	}

	content, err := ioutil.ReadAll(c.Stdout)
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

func (f *File) getFileReader(path string) (io.ReadCloser, dockertypes.ContainerPathStat, error) {
	// special handling for /proc file system, since you cannot copy them via
	// the docker api
	if strings.HasPrefix(f.path, "/proc") {
		r, stat, err := f.getFileCatReader(f.path)
		return r, stat, err
	} else {
		r, stat, err := f.getFileDockerReader(f.path)
		return r, stat, err
	}
}

func (f *File) getFileDockerReader(path string) (io.ReadCloser, dockertypes.ContainerPathStat, error) {
	r, stat, err := f.dockerClient.CopyFromContainer(context.Background(), f.container, path)

	// follow symlink if stat.LinkTarget is set
	if len(stat.LinkTarget) > 0 {
		return f.getFileDockerReader(stat.LinkTarget)
	}

	return r, stat, err
}

// getFileCatReader is used for /proc/* on docker containers
func (f *File) getFileCatReader(path string) (io.ReadCloser, dockertypes.ContainerPathStat, error) {
	c, err := f.transport.RunCommand("find " + f.path + " -type f")
	if err != nil {
		return nil, dockertypes.ContainerPathStat{}, err
	}
	content, err := ioutil.ReadAll(c.Stdout)
	if err != nil {
		return nil, dockertypes.ContainerPathStat{}, err
	}

	// all files
	entries := strings.Split(string(content), "\n")

	// pipe content to a tar stream
	tarReader, tarWriter := io.Pipe()

	// stream content into the pipe
	tw := tar.NewWriter(tarWriter)

	go func() {
		defer tw.Close()
		// create a tar stream by reading all the content via cat
		for i := range entries {
			path := entries[i]
			ec, err := f.transport.RunCommand("cat " + entries[i])
			if err != nil {
				log.Error().Str("file", path).Err(err).Msg("docker> could read file content")
				continue
			}

			econtent, err := ioutil.ReadAll(ec.Stdout)
			if err != nil {
				log.Error().Str("file", path).Err(err).Msg("docker> could read file content")
				continue
			}

			// send tar header
			hdr := &tar.Header{
				Name: path,
				// Mode: int64(fileinfo.Mode()),
				Size: int64(len(econtent)),
			}

			if err := tw.WriteHeader(hdr); err != nil {
				log.Error().Str("file", path).Err(err).Msg("docker> could not write tar header")
			}

			_, err = tw.Write(econtent)
			if err != nil {
				log.Error().Str("file", path).Err(err).Msg("docker> could not write tar stream")
			}
		}
	}()

	return tarReader, dockertypes.ContainerPathStat{}, nil
}

// returns a TarReader stream the caller is responsible for closing the stream
func (f *File) Tar() (io.ReadCloser, error) {
	// special handling for /proc file system, since you cannot copy them via
	// the docker api
	if strings.HasPrefix(f.path, "/proc") {
		r, _, err := f.getFileCatReader(f.path)
		return r, err
	}

	r, _, err := f.getFileReader(f.path)
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
	c, err := f.transport.RunCommand("find /proc")
	if err != nil {
		return []string{}
	}
	content, err := ioutil.ReadAll(c.Stdout)
	if err != nil {
		return []string{}
	}

	// all files
	return strings.Split(string(content), "\n")
}
