// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"archive/tar"
	"bytes"
	"io"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ulikunitz/xz"
)

// tarXz builds an in-memory .tar.xz containing a single regular-file entry
// with the given name and contents.
func tarXz(t *testing.T, name, content string) io.ReadCloser {
	t.Helper()
	var buf bytes.Buffer
	xzw, err := xz.NewWriter(&buf)
	require.NoError(t, err)
	tw := tar.NewWriter(xzw)
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     name,
		Mode:     0o644,
		Size:     int64(len(content)),
	}))
	_, err = tw.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, xzw.Close())
	return io.NopCloser(&buf)
}

func TestInstallIO_RejectsPathTraversal(t *testing.T) {
	dst := t.TempDir()
	// InstallIO unpacks into a temp dir created *inside* dst, so an entry of
	// "../escapee.txt" resolves to dst/escapee.txt — outside the unpack dir.
	escapee := filepath.Join(dst, "escapee.txt")
	require.NoFileExists(t, escapee)

	_, err := InstallIO(tarXz(t, "../escapee.txt", "pwned"), InstallConf{Dst: dst})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid path in provider archive")

	// Nothing was written outside the unpack directory.
	assert.NoFileExists(t, escapee)
}
