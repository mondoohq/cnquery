// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package fsutil

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

func TestStreamFileAsTar(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.txt")
	// a body whose length is not a multiple of 512, so the tar block padding
	// is observable
	content := []byte("hello tar stream\nsecond line\n")
	require.NoError(t, os.WriteFile(path, content, 0o644))

	stat, err := os.Stat(path)
	require.NoError(t, err)
	f, err := os.Open(path)
	require.NoError(t, err)

	var buf bytes.Buffer
	StreamFileAsTar(path, stat, f, nopWriteCloser{Writer: &buf})

	// The body must be written through the tar writer so it is framed with
	// the rest of the archive (block padding + trailer). With the previous
	// bug the body was copied straight to the underlying writer, bypassing
	// the tar framing, which produced a malformed archive: header + raw body
	// + trailer with no padding of the data record to a 512-byte boundary.
	roundUp := func(n, mult int) int { return ((n + mult - 1) / mult) * mult }
	wantLen := 512 /* header */ + roundUp(len(content), 512) /* padded data */ + 2*512 /* trailer */
	assert.Equal(t, wantLen, buf.Len(), "archive must be framed in 512-byte tar records")

	// And the archive must read back as a well-formed tar yielding the
	// original bytes, ending in a clean EOF.
	tr := tar.NewReader(bytes.NewReader(buf.Bytes()))
	hdr, err := tr.Next()
	require.NoError(t, err)
	assert.Equal(t, path, hdr.Name)
	assert.Equal(t, int64(len(content)), hdr.Size)
	got, err := io.ReadAll(tr)
	require.NoError(t, err)
	assert.Equal(t, content, got)
	_, err = tr.Next()
	assert.Equal(t, io.EOF, err, "archive must terminate with a valid tar trailer")
}
