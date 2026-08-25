// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package tar

import (
	archivetar "archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// imageTarBytes builds a tar whose entry names look like a flattened container image.
func imageTarBytes(t testing.TB, n int) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := archivetar.NewWriter(&buf)
	dirs := []string{"usr/share/doc", "usr/lib/python3.11/site-packages", "usr/bin", "etc", "var/lib/dpkg/info"}
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("%s/package-%06d/module_%d.py", dirs[i%len(dirs)], i/7, i)
		require.NoError(t, tw.WriteHeader(&archivetar.Header{
			Name: name, Typeflag: archivetar.TypeReg, Mode: 0o644, Uname: "root", Gname: "root",
		}))
	}
	require.NoError(t, tw.Close())
	return buf.Bytes()
}

// TestLoadKeepsHeaderNames checks that sharing the backing array of the map key does not
// change the value of Header.Name. fs.open matches that name against the tar stream.
func TestLoadKeepsHeaderNames(t *testing.T) {
	names := []string{
		"usr/bin/ls",
		"./etc/hosts",
		"/var/log/syslog",
		"usr//share/doc",
		"opt/app/../app/config.yaml",
		"srv/data/",
	}

	var buf bytes.Buffer
	tw := archivetar.NewWriter(&buf)
	for _, name := range names {
		typ := byte(archivetar.TypeReg)
		if strings.HasSuffix(name, "/") {
			typ = archivetar.TypeDir
		}
		require.NoError(t, tw.WriteHeader(&archivetar.Header{Name: name, Typeflag: typ, Mode: 0o644}))
	}
	require.NoError(t, tw.Close())

	// Read the names back the way archive/tar reports them, so the expectation does not
	// depend on how the writer normalizes them.
	expected := map[string]string{}
	tr := archivetar.NewReader(bytes.NewReader(buf.Bytes()))
	for {
		h, err := tr.Next()
		if err != nil {
			break
		}
		expected[Abs(h.Name)] = h.Name
	}
	require.NotEmpty(t, expected)

	c := &Connection{fs: NewFs("")}
	require.NoError(t, c.Load(bytes.NewReader(buf.Bytes())))

	require.Len(t, c.fs.FileMap, len(expected))
	for path, want := range expected {
		h, ok := c.fs.FileMap[path]
		require.True(t, ok, "missing entry for %q", path)
		assert.Equal(t, want, h.Name, "header name changed for %q", path)
	}
}

// BenchmarkLoad loads a 50000 entry image tar. It also reports the heap that the file map
// holds after the load, as retained-B/entry.
func BenchmarkLoad(b *testing.B) {
	const entries = 50000
	data := imageTarBytes(b, entries)

	b.ReportAllocs()
	b.ResetTimer()
	var c *Connection
	for i := 0; i < b.N; i++ {
		c = &Connection{fs: NewFs("")}
		require.NoError(b, c.Load(bytes.NewReader(data)))
	}
	b.StopTimer()

	runtime.GC()
	var withMap runtime.MemStats
	runtime.ReadMemStats(&withMap)
	held := withMap.HeapAlloc
	c.fs.FileMap = nil
	runtime.GC()
	var withoutMap runtime.MemStats
	runtime.ReadMemStats(&withoutMap)
	b.ReportMetric(float64(held-withoutMap.HeapAlloc)/float64(entries), "retained-B/entry")
}

// OSTree-based images (Fedora CoreOS, and the rpm-ostree family generally) do
// not store /usr/lib/os-release as a regular file. It is a hard link into the
// ostree object store, reached through a symlink from /etc:
//
//	etc/os-release      -> ../usr/lib/os-release                 (symlink)
//	usr/lib/os-release  link to sysroot/ostree/repo/objects/...   (hard link)
//
// A tar hard-link entry carries no content of its own, so without following it
// the file reads as empty and the platform cannot be identified at all.
func TestTarFollowsHardLinks(t *testing.T) {
	const osRelease = "NAME=\"Fedora Linux\"\nID=fedora\nVERSION_ID=44\n"
	const objectPath = "sysroot/ostree/repo/objects/a2/69745d02ee36b7709513ce323b1cf401aa2745f.file"

	var buf bytes.Buffer
	tw := archivetar.NewWriter(&buf)
	require.NoError(t, tw.WriteHeader(&archivetar.Header{
		Name: objectPath, Typeflag: archivetar.TypeReg, Mode: 0o644, Size: int64(len(osRelease)),
	}))
	_, err := tw.Write([]byte(osRelease))
	require.NoError(t, err)
	require.NoError(t, tw.WriteHeader(&archivetar.Header{
		Name: "usr/lib/os-release", Typeflag: archivetar.TypeLink, Linkname: objectPath, Mode: 0o644,
	}))
	require.NoError(t, tw.WriteHeader(&archivetar.Header{
		Name: "etc/os-release", Typeflag: archivetar.TypeSymlink, Linkname: "../usr/lib/os-release", Mode: 0o777,
	}))
	require.NoError(t, tw.Close())

	tmp := filepath.Join(t.TempDir(), "image.tar")
	require.NoError(t, os.WriteFile(tmp, buf.Bytes(), 0o644))

	c := &Connection{fs: NewFs(tmp)}
	require.NoError(t, c.Load(bytes.NewReader(buf.Bytes())))

	t.Run("reads through a hard link", func(t *testing.T) {
		f, err := c.fs.Open("/usr/lib/os-release")
		require.NoError(t, err)
		content, err := io.ReadAll(f)
		require.NoError(t, err)
		assert.Equal(t, osRelease, string(content))
	})

	t.Run("reads through a symlink onto a hard link", func(t *testing.T) {
		f, err := c.fs.Open("/etc/os-release")
		require.NoError(t, err)
		content, err := io.ReadAll(f)
		require.NoError(t, err)
		assert.Equal(t, osRelease, string(content), "this is how Fedora CoreOS reports its platform")
	})

	t.Run("a relative symlink behind a hard link resolves from where we came in", func(t *testing.T) {
		// etc/redhat-release is a hard link to an ostree object that is itself a
		// symlink to "fedora-release". That target is relative to /etc, the path
		// we entered by, not to the object store the hard link pointed into.
		const releaseText = "Fedora release 44 (Coming Soon)\n"
		const objSymlink = "sysroot/ostree/repo/objects/e8/1400dd73cc60112e3213386396401d5c50e2.file"

		var b bytes.Buffer
		w := archivetar.NewWriter(&b)
		require.NoError(t, w.WriteHeader(&archivetar.Header{
			Name: "etc/fedora-release", Typeflag: archivetar.TypeReg, Mode: 0o644, Size: int64(len(releaseText)),
		}))
		_, err := w.Write([]byte(releaseText))
		require.NoError(t, err)
		require.NoError(t, w.WriteHeader(&archivetar.Header{
			Name: objSymlink, Typeflag: archivetar.TypeSymlink, Linkname: "fedora-release", Mode: 0o777,
		}))
		require.NoError(t, w.WriteHeader(&archivetar.Header{
			Name: "etc/redhat-release", Typeflag: archivetar.TypeLink, Linkname: objSymlink, Mode: 0o644,
		}))
		require.NoError(t, w.Close())

		tmp2 := filepath.Join(t.TempDir(), "ostree.tar")
		require.NoError(t, os.WriteFile(tmp2, b.Bytes(), 0o644))
		c2 := &Connection{fs: NewFs(tmp2)}
		require.NoError(t, c2.Load(bytes.NewReader(b.Bytes())))

		f, err := c2.fs.Open("/etc/redhat-release")
		require.NoError(t, err)
		content, err := io.ReadAll(f)
		require.NoError(t, err)
		assert.Equal(t, releaseText, string(content))
	})

	t.Run("stat reports the target size", func(t *testing.T) {
		f, err := c.fs.Open("/etc/os-release")
		require.NoError(t, err)
		st, err := f.Stat()
		require.NoError(t, err)
		assert.Equal(t, int64(len(osRelease)), st.Size())
	})
}
