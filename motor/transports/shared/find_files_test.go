package shared

import (
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindFilesMatcher(t *testing.T) {
	possibleTypes := []byte{'b', 'c', 'd', 'p', 'f', 'l'}
	possibleModes := []fs.FileMode{
		fs.ModeDevice,
		fs.ModeDevice | fs.ModeCharDevice,
		fs.ModeDir,
		fs.ModeNamedPipe,
		fs.ModePerm,
		fs.ModeSymlink,
	}
	t.Run("type matching", func(t *testing.T) {
		testCases := []struct {
			typ     fs.FileMode
			matches byte
		}{
			{
				typ:     fs.ModeDir,
				matches: 'd',
			},
			{
				typ:     fs.ModePerm,
				matches: 'f',
			},
			{
				typ:     fs.ModeDevice | fs.ModeCharDevice,
				matches: 'c',
			},
			{
				typ:     fs.ModeDevice,
				matches: 'b',
			},
			{
				typ:     fs.ModeSymlink,
				matches: 'l',
			},
			{
				typ:     fs.ModeNamedPipe,
				matches: 'p',
			},
		}

		for _, tc := range testCases {
			excludeTypes := []string{}
			for _, b := range possibleTypes {
				if b != tc.matches {
					excludeTypes = append(excludeTypes, string(b))
				}
			}
			t.Run(fmt.Sprintf("%s matcher", string(tc.matches)), func(t *testing.T) {
				exclusionMatcher := createFindFilesMatcher(strings.Join(excludeTypes, ","), nil)
				exactMatcher := createFindFilesMatcher(string(tc.matches), nil)
				assert.True(t, exactMatcher.Match("/foo", tc.typ), "exact matcher failed to match")
				assert.False(t, exclusionMatcher.Match("/foo", tc.typ), "exclusion matcher matched")
			})
		}
	})

	t.Run("regex", func(t *testing.T) {
		t.Run("any type", func(t *testing.T) {
			exactMatcher := createFindFilesMatcher("", regexp.MustCompile("foo.*"))

			for _, m := range possibleModes {
				t.Run(fmt.Sprintf("mode %s", m.String()), func(t *testing.T) {
					assert.True(t, exactMatcher.Match("foobar", m))
					assert.True(t, exactMatcher.Match("foofoobar", m))
					assert.False(t, exactMatcher.Match("barfoo", m))
				})
			}
		})

		t.Run("specific type", func(t *testing.T) {
			exactMatcher := createFindFilesMatcher("f", regexp.MustCompile("foo.*"))

			assert.False(t, exactMatcher.Match("foobar", fs.ModeDir))
			assert.True(t, exactMatcher.Match("foobar", fs.ModePerm))
		})
	})

	t.Run("no type or regex", func(t *testing.T) {
		testCases := []struct {
			typ     fs.FileMode
			matches byte
		}{
			{
				typ:     fs.ModeDir,
				matches: 'd',
			},
			{
				typ:     fs.ModePerm,
				matches: 'f',
			},
			{
				typ:     fs.ModeDevice | fs.ModeCharDevice,
				matches: 'c',
			},
			{
				typ:     fs.ModeDevice,
				matches: 'b',
			},
			{
				typ:     fs.ModeSymlink,
				matches: 'l',
			},
			{
				typ:     fs.ModeNamedPipe,
				matches: 'p',
			},
		}
		for _, tc := range testCases {
			excludeTypes := []string{}
			for _, b := range possibleTypes {
				if b != tc.matches {
					excludeTypes = append(excludeTypes, string(b))
				}
			}
			t.Run(fmt.Sprintf("%s matcher", string(tc.matches)), func(t *testing.T) {
				exactMatcher := createFindFilesMatcher("", nil)
				assert.True(t, exactMatcher.Match("/foo", tc.typ), "matcher failed to match")
			})
		}
	})
}

func TestFindFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	mkDir(t, fs, "root/a")
	mkDir(t, fs, "root/b")
	mkDir(t, fs, "root/c")
	mkFile(t, fs, "root/a/file1")
	mkFile(t, fs, "root/a/file2")
	mkFile(t, fs, "root/b/file1")

	rootAFiles, err := FindFiles(afero.NewIOFS(fs), "root/a", nil, "f")
	require.NoError(t, err)
	assert.ElementsMatch(t, rootAFiles, []string{"root/a/file1", "root/a/file2"})

	rootAFilesAndDir, err := FindFiles(afero.NewIOFS(fs), "root/a", nil, "f,d")
	require.NoError(t, err)
	assert.ElementsMatch(t, rootAFilesAndDir, []string{"root/a", "root/a/file1", "root/a/file2"})

	rootBFiles, err := FindFiles(afero.NewIOFS(fs), "root", regexp.MustCompile("root/b.*"), "f")
	assert.ElementsMatch(t, rootBFiles, []string{"root/b/file1"})
}

func mkFile(t *testing.T, fs afero.Fs, name string) {
	t.Helper()
	f, err := fs.Create(name)
	require.NoError(t, err)
	err = f.Close()
	require.NoError(t, err)
}

func mkDir(t *testing.T, fs afero.Fs, name string) {
	t.Helper()
	err := fs.MkdirAll(name, os.ModePerm)
	require.NoError(t, err)
}
