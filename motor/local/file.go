package local

import (
	"archive/tar"
	"crypto/md5"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/types"
)

type File struct {
	filePath string
}

func (f *File) Name() string {
	return f.filePath
}

func (f *File) Stat() (os.FileInfo, error) {
	return os.Stat(f.filePath)
}

func (f *File) Readdir(n int) ([]os.FileInfo, error) {
	file, err := os.Open(f.filePath)
	if err != nil {
		return nil, err
	}
	return file.Readdir(n)
}

func (f *File) Readdirnames(n int) ([]string, error) {
	file, err := os.Open(f.filePath)
	if err != nil {
		return nil, err
	}
	return file.Readdirnames(n)
}

func (f *File) Tar() (io.ReadCloser, error) {
	stat, err := f.Stat()
	if err != nil {
		return nil, errors.New("could not retrieve file stats")
	}

	// determine all files that we need to transfer
	fileList := map[string]os.FileInfo{}
	if stat.IsDir() == true {

		err = filepath.Walk(f.filePath, func(path string, f os.FileInfo, err error) error {
			fileList[path] = f
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		fileList[f.filePath] = stat
	}

	// pipe content to a tar stream
	tarReader, tarWriter := io.Pipe()

	// stream content into the pipe
	tw := tar.NewWriter(tarWriter)

	// copy file content in the background
	go func() {
		for path, fileinfo := range fileList {
			// we ignore the error for now but log them
			fReader, err := os.Open(path)
			if err == nil {
				// send tar header
				hdr := &tar.Header{
					Name: path,
					Mode: int64(fileinfo.Mode()),
					Size: fileinfo.Size(),
				}

				if err := tw.WriteHeader(hdr); err != nil {
					log.Error().Str("file", path).Err(err).Msg("local> could not write tar header")
				}

				_, err := io.Copy(tw, fReader)
				if err != nil {

				}
			} else {
				log.Error().Str("file", path).Err(err).Msg("local> could not stream file")
			}
		}
		tarWriter.Close()
	}()

	return tarReader, nil
}

func (f *File) Open() (types.FileStream, error) {
	stat, err := f.Stat()
	if err != nil {
		return nil, errors.New("could not retrieve file stats")
	}

	if stat.IsDir() == true {
		return nil, errors.New("cannot stream directories")
	}

	return os.Open(f.filePath)
}

func (f *File) HashMd5() (string, error) {

	file, err := os.Open(f.filePath)
	defer file.Close()
	if err != nil {
		return "", err
	}

	h := md5.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}

	return string(h.Sum(nil)), nil

}
func (f *File) HahsSha256() (string, error) {
	file, err := os.Open(f.filePath)
	defer file.Close()
	if err != nil {
		return "", err
	}

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}

	return string(h.Sum(nil)), nil
}

func (f *File) Exists() bool {
	_, err := os.Stat(f.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func (f *File) Delete() error {
	return os.Remove(f.filePath)
}
