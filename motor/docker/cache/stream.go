package cache

import (
	"io"
	"io/ioutil"
	"os"
)

func RandomFile() (*os.File, error) {
	return ioutil.TempFile("", "mondoo.inspection")
}

// This streams a binary stream into a file. The user of this method
// is responsible for deleting the file late
func StreamToTmpFile(r io.ReadCloser, outFile *os.File) error {
	defer outFile.Close()
	_, err := io.Copy(outFile, r)
	if err != nil {
		return err
	}

	return r.Close()
}
