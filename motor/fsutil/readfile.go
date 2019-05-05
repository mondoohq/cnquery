package motorutil

// Download is a helper method to load file content into memory
import (
	"archive/tar"
	"bufio"
	"bytes"
	"io"

	"go.mondoo.io/mondoo/motor/types"
)

// TODO: check size of file to ensure we do not crash the process
func ReadFile(f types.File) ([]byte, error) {
	var fileBuffer bytes.Buffer
	fileWriter := bufio.NewWriter(&fileBuffer)

	r, err := f.Tar()
	if err != nil {
		return nil, err
	}
	defer r.Close()

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
