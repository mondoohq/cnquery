package fsutil

import (
	"archive/tar"
	"bufio"
	"bytes"
	"io"
)

// TODO: check size of file to ensure we do not crash the process
func ReadFileFromTarStream(r io.Reader) ([]byte, error) {
	var fileBuffer bytes.Buffer
	fileWriter := bufio.NewWriter(&fileBuffer)

	// read stream tar, extract on the fly and put it on stdout
	tr := tar.NewReader(r)
	for {
		_, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if _, err := io.Copy(fileWriter, tr); err != nil {
			return nil, err
		}
	}
	fileWriter.Flush()

	return fileBuffer.Bytes(), nil
}
