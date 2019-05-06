package fsutil

import (
	"archive/tar"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/rs/zerolog/log"
)

// StreamFileAsTar task t
func StreamFileAsTar(
	path string, // file path
	stat os.FileInfo, // stat of the file
	fileReader io.ReadCloser, // raw file byte stream
	writer io.WriteCloser, // tar output stream
) {
	// close all open connection
	defer fileReader.Close()

	// stream content into the pipe
	tw := tar.NewWriter(writer)
	bufReader := bufio.NewReader(fileReader)
	defer tw.Close()
	defer writer.Close()

	// send tar header
	hdr := &tar.Header{
		Name: path,
		Mode: int64(stat.Mode()),
		Size: stat.Size(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		fmt.Print(err)
		writer.Close()
	}

	// copy file content
	if _, err := io.Copy(writer, bufReader); err != nil {
		fmt.Print(err)
		writer.Close()
	}
}

func ExtractFileFromTarStream(path string, tarReader io.Reader) (*bufio.Reader, error) {
	log.Debug().Str("path", path).Msg("fsutil> extract file from tar")
	var fileBuffer bytes.Buffer
	bufWriter := bufio.NewWriter(&fileBuffer)

	// read stream tar, extract on the fly and put it on stdout
	tr := tar.NewReader(tarReader)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		// log.Debug().Msgf("File %s, Size: %d", h.Name, h.Size)
		if h.Name == path {
			log.Debug().Str("path", path).Msg("fsutil> found file")
			if _, err := io.CopyN(bufWriter, tr, h.Size); err != nil {
				return nil, err
			}
		}
	}

	bufWriter.Flush()
	return bufio.NewReader(&fileBuffer), nil
}
