package ssh

import (
	"archive/tar"
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/pkg/sftp"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/types"
	"golang.org/x/crypto/ssh"
)

type File struct {
	filePath  string
	SSHClient *ssh.Client
}

func (f *File) Name() string {
	return f.filePath
}

func (f *File) Stat() (os.FileInfo, error) {
	c, err := sftpClient(f.SSHClient)
	if err != nil {
		return nil, fmt.Errorf("ssh> could not open client: %v", err)
	}
	// we close the SFTP client, not the ssh client
	defer c.Close()

	r, err := c.Open(f.filePath)
	if err != nil {
		return nil, fmt.Errorf("ssh> could not open file: %v", err)
	}
	defer r.Close()

	fstat, err := r.Stat()
	if err != nil {
		return nil, fmt.Errorf("ssh> could not retrieve file stat: %v", err)
	}

	stat := types.FileInfo{FSize: fstat.Size(), FModTime: time.Now(), FMode: fstat.Mode()}
	return &stat, nil
}

func (f *File) Tar() (io.ReadCloser, error) {
	// we need to make sure this only is closed when the go routine with data copy is done
	c, err := sftpClient(f.SSHClient)
	if err != nil {
		return nil, fmt.Errorf("ssh> could not open client: %v", err)
	}

	file, err := c.Open(f.filePath)
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("ssh> could open file: %v", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		c.Close()
		return nil, errors.New("ssh> could not retrieve file stats")
	}

	// determine all files that we need to transfer
	fileList := map[string]os.FileInfo{}
	if stat.IsDir() == true {
		walker := c.Walk(f.filePath)
		for walker.Step() {
			if err := walker.Err(); err != nil {
				log.Error().Err(err).Msg("ssh> could not access file")
				continue
			}

			// I think we need to copy the data, since references may change
			stat := walker.Stat()
			log.Debug().Str("file", walker.Path()).Int64("size", stat.Size()).Msg("ssh> found file")
			fileList[walker.Path()] = walker.Stat()
		}
	} else {
		fileList[f.filePath] = stat
	}

	// pipe content to a tar stream
	tarReader, tarWriter := io.Pipe()

	// copy file content in the background
	go func() {
		// stream content into the pipe
		tw := tar.NewWriter(tarWriter)
		for path, stat := range fileList {
			if stat.IsDir() {
				// handle directories
				hdr := &tar.Header{
					Name: path,
					Mode: int64(stat.Mode()),
					Size: 0,
				}

				if err := tw.WriteHeader(hdr); err != nil {
					log.Error().Str("file", path).Err(err).Msg("ssh> could not write tar header")
				}
			} else {
				// handle files

				// we ignore the error for now but log them
				fReader, err := c.Open(path)
				if err != nil {
					log.Error().Str("file", path).Err(err).Msg("ssh> could not open tar file stream")
					continue
				}

				// special handling for /proc & /sys filesystem on linux, read file in cache since
				// we do not get the file size for files in those directories
				// abort when buffer is getting greater
				var fileBuffer bytes.Buffer
				fileBufferWriter := bufio.NewWriter(&fileBuffer)
				_, err = io.Copy(fileBufferWriter, fReader)
				fileBufferWriter.Flush()
				fReader.Close()

				if err != nil {
					log.Error().Str("file", path).Err(err).Msg("ssh> could not stream tar file stream")
				}

				// send tar header
				hdr := &tar.Header{
					Name: path,
					Mode: int64(stat.Mode()),
					Size: int64(fileBuffer.Len()),
				}

				if err := tw.WriteHeader(hdr); err != nil {
					log.Error().Str("file", path).Err(err).Msg("ssh> could not write tar header")
				}

				// write file content
				tw.Write(fileBuffer.Bytes())
			}
		}

		// close streams
		tw.Close()
		tarWriter.Close()
		// close sftp client
		c.Close()
	}()
	return tarReader, nil
}

func (f *File) Readdir(n int) ([]os.FileInfo, error) {
	c, err := sftpClient(f.SSHClient)
	if err != nil {
		return nil, fmt.Errorf("ssh> could not open client: %v", err)
	}
	defer c.Close()
	return c.ReadDir(f.filePath)
}

func (f *File) Readdirnames(n int) ([]string, error) {
	c, err := sftpClient(f.SSHClient)
	if err != nil {
		return nil, fmt.Errorf("ssh> could not open client: %v", err)
	}
	defer c.Close()

	dirFileInfos, err := c.ReadDir(f.filePath)
	if err != nil {
		return nil, fmt.Errorf("ssh> could not read dirnames: %v", err)
	}

	dir := make([]string, len(dirFileInfos))
	for i := range dirFileInfos {
		dir[i] = dirFileInfos[i].Name()
	}
	return dir, nil
}

// sftpStreamCloser ensure that the sftp connection is closed once the
// file stream is closed by the user
type sftpStreamCloser struct {
	io.ReadCloser
	sftpClient *sftp.Client
}

func (s *sftpStreamCloser) Close() error {
	errFile := s.ReadCloser.Close()
	errSftp := s.sftpClient.Close()

	if errFile != nil {
		return errFile
	}
	if errSftp != nil {
		return errSftp
	}
	return nil
}

// opens a byte stream to a single file, the reader is responsible for closing the stream
func (f *File) Open() (types.FileStream, error) {
	c, err := sftpClient(f.SSHClient)
	if err != nil {
		return nil, fmt.Errorf("ssh> could not open client: %v", err)
	}

	file, err := c.Open(f.filePath)
	if err != nil {
		return nil, fmt.Errorf("ssh> could not open file %v", err)
	}
	return &sftpStreamCloser{ReadCloser: file, sftpClient: c}, nil
}

func (f *File) Exists() bool {
	_, err := f.Stat()
	if err != nil {
		return false
	}
	return true
}
