package docker_engine

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"go.mondoo.io/mondoo/motor/motorutil"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/types"
)

type File struct {
	filePath     string
	Container    string
	dockerClient *client.Client
	Transport    *DockerTransport
}

func (f *File) Name() string {
	return f.filePath
}

func (f *File) Stat() (os.FileInfo, error) {
	r, dstat, err := f.getFileReader(f.filePath)
	if err != nil {
		return nil, err
	}
	r.Close()

	stat := types.FileInfo{FMode: dstat.Mode, FSize: dstat.Size, FName: dstat.Name, FModTime: dstat.Mtime}
	return &stat, nil
}

// returns a TarReader stream the caller is responsible for closing the stream
func (f *File) Tar() (io.ReadCloser, error) {
	// special handling for /proc file system, since you cannot copy them via
	// the docker api
	if strings.HasPrefix(f.filePath, "/proc") {
		r, _, err := f.getFileCatReader(f.filePath)
		return r, err
	}

	r, _, err := f.getFileReader(f.filePath)
	return r, err
}

func (f *File) Open() (types.FileStream, error) {
	data, err := motorutil.ReadFile(f)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader(data)
	return ioutil.NopCloser(reader), nil
}

func (f *File) Exists() bool {
	if strings.HasPrefix(f.filePath, "/proc") {
		entries := f.procls()
		for i := range entries {
			if entries[i] == f.filePath {
				return true
			}
		}
		return false
	}

	r, _, err := f.getFileReader(f.filePath)
	if err != nil {
		return false
	}
	r.Close()
	return true
}

func (f *File) getFileReader(path string) (io.ReadCloser, dockertypes.ContainerPathStat, error) {
	r, stat, err := f.dockerClient.CopyFromContainer(context.Background(), f.Container, path)

	// follow symlink if stat.LinkTarget is set
	if len(stat.LinkTarget) > 0 {
		return f.getFileReader(stat.LinkTarget)
	}

	return r, stat, err
}

// returns all directories and files under /proc
func (f *File) procls() []string {
	c, err := f.Transport.RunCommand("find /proc")
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

// getFileCatReader is used for /proc/* on docker containers
func (f *File) getFileCatReader(path string) (io.ReadCloser, dockertypes.ContainerPathStat, error) {
	c, err := f.Transport.RunCommand("find " + f.filePath + " -type f")
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
			ec, err := f.Transport.RunCommand("cat " + entries[i])
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

func (f *File) Readdir(n int) ([]os.FileInfo, error) {
	return nil, errors.New("not implemented yet")
}

func (f *File) Readdirnames(n int) ([]string, error) {
	c, err := f.Transport.RunCommand(fmt.Sprintf("find %s -maxdepth 1 -type d", f.filePath))
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

	fmt.Printf("dirs %s", basenames)
	return basenames, nil
}
