// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cat

import (
	"github.com/spf13/afero"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers/os/connection/mock"
)

func TestCatFs(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/winrm.toml")
	p, err := mock.New(0, &inventory.Asset{}, mock.WithPath(filepath))
	require.NoError(t, err)

	catfs := New(p)

	// fetch file content
	f, err := catfs.Open("C:\\test.txt")
	require.NoError(t, err)

	data, err := io.ReadAll(f)
	require.NoError(t, err)

	expected := "hi\n"
	assert.Equal(t, expected, string(data))

	// get file stats
	fi, err := catfs.Stat("C:\\test.txt")
	require.NoError(t, err)

	assert.Equal(t, int64(2), fi.Size())
	assert.Equal(t, false, fi.IsDir())
	assert.Equal(t, int64(1603529613), fi.ModTime().Unix())
}

func TestCatFsDirectory(t *testing.T) {
	const path = `C:\Users\profile's home\.vscode\extensions`
	for _, listing := range []string{
		`[]`,
		`[{"Name":"only.ext","Length":7,"Attributes":32}]`,
		`[{"Name":"manifest.json","Length":17,"Attributes":32},{"Name":"nested","Attributes":18}]`,
	} {
		t.Run(listing, func(t *testing.T) {
			conn, err := mock.New(0, &inventory.Asset{}, mock.WithData(&mock.TomlData{Commands: map[string]*mock.Command{
				getItemScript(path):      {Stdout: `{"Name":"extensions","Attributes":16}`},
				getDirectoryScript(path): {Stdout: listing},
			}}))
			require.NoError(t, err)
			fs := New(conn)
			entries, err := afero.ReadDir(fs, path)
			require.NoError(t, err)
			if listing == `[]` {
				assert.Empty(t, entries)
			} else if len(entries) == 1 {
				assert.Equal(t, "only.ext", entries[0].Name())
				assert.Equal(t, int64(7), entries[0].Size())
			} else {
				require.Len(t, entries, 2)
				assert.Equal(t, "manifest.json", entries[0].Name())
				assert.Equal(t, int64(17), entries[0].Size())
				assert.True(t, entries[1].IsDir())
			}
			file, err := fs.Open(path)
			require.NoError(t, err)
			t.Cleanup(func() { assert.NoError(t, file.Close()) })
			info, err := file.Stat()
			require.NoError(t, err)
			assert.True(t, info.IsDir())
			_, err = file.Read(make([]byte, 1))
			assert.ErrorIs(t, err, os.ErrInvalid)
			for _, expected := range entries {
				names, err := file.Readdirnames(1)
				require.NoError(t, err)
				assert.Equal(t, []string{expected.Name()}, names)
			}
			_, err = file.Readdir(1)
			assert.ErrorIs(t, err, io.EOF)
			remaining, err := file.Readdir(-1)
			require.NoError(t, err)
			assert.Empty(t, remaining)
		})
	}
}

func TestCatFsDirectoryErrors(t *testing.T) {
	const path = `C:\Users\profile\extensions`
	for _, command := range []*mock.Command{
		{ExitStatus: 1, Stderr: "PermissionDenied"},
		{Stdout: "invalid JSON"},
	} {
		conn, err := mock.New(0, &inventory.Asset{}, mock.WithData(&mock.TomlData{Commands: map[string]*mock.Command{
			getItemScript(path):      {Stdout: `{"Name":"extensions","Attributes":16}`},
			getDirectoryScript(path): command,
		}}))
		require.NoError(t, err)
		_, err = afero.ReadDir(New(conn), path)
		require.Error(t, err)
		assert.False(t, os.IsNotExist(err), "a failed listing cannot be reported as a missing directory")
	}
}
