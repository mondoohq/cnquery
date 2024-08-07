// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package tar

import (
	"io"
	"os"
)

// StreamToTmpFile streams a binary stream into a file. The user of this method
// is responsible for deleting the file later
func StreamToTmpFile(r io.ReadCloser, outFile *os.File) error {
	defer outFile.Close()
	_, err := io.Copy(outFile, r)
	if err != nil {
		return err
	}

	return r.Close()
}
