// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

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

	// copy file content through the tar writer so the body is framed
	// with the rest of the archive (block padding + trailer)
	if _, err := io.Copy(tw, bufReader); err != nil {
		fmt.Print(err)
		writer.Close()
	}
}

// ExtractFileFromTarStream returns the content of path inside the tar stream.
// The tar header states the exact entry size, so the content goes into a slice
// of that size. This avoids the repeated buffer growth of bytes.Buffer, which
// allocates about twice the entry size for large files.
func ExtractFileFromTarStream(path string, tarReader io.Reader) (io.Reader, error) {
	log.Debug().Str("path", path).Msg("fsutil> extract file from tar")
	var content []byte

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
			if h.Size <= 0 {
				continue
			}
			chunk := make([]byte, h.Size)
			if _, err := io.ReadFull(tr, chunk); err != nil {
				return nil, err
			}
			if content == nil {
				content = chunk
			} else {
				content = append(content, chunk...)
			}
		}
	}

	return bytes.NewReader(content), nil
}
