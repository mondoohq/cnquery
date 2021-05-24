package fsutil

import (
	"archive/tar"
	"errors"
	"io"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
)

func Tar(fs afero.Fs, f afero.File) (io.ReadCloser, error) {
	stat, err := f.Stat()
	if err != nil {
		return nil, errors.New("could not retrieve file stats")
	}

	afutil := afero.Afero{Fs: fs}

	// determine all files that we need to transfer
	fileList := map[string]os.FileInfo{}
	if stat.IsDir() == true {

		err = afutil.Walk(f.Name(), func(path string, f os.FileInfo, err error) error {
			fileList[path] = f
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		fileList[f.Name()] = stat
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
					log.Error().Str("file", path).Err(err).Msg("local> could not write tar data")
				}
			} else {
				log.Error().Str("file", path).Err(err).Msg("local> could not stream file")
			}
		}
		tarWriter.Close()
	}()

	return tarReader, nil
}
