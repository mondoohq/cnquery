package cache

import (
	"io"
	"os"

	"go.mondoo.io/mondoo/motor/motorutil"
)

func RandomFile() string {
	filename := ".tmp.mondoo.container." + motorutil.NextRandom()
	return filename
}

// This streams a binary stream into a file. The user of this method
// is responsible for deleting the file late
func StreamToTmpFile(r io.ReadCloser, filename string) (string, error) {
	outFile, err := os.Create(filename)
	if err != nil {
		return "", err
	}

	defer outFile.Close()
	_, err = io.Copy(outFile, r)
	if err != nil {
		return "", err
	}

	r.Close()
	return filename, nil
}
