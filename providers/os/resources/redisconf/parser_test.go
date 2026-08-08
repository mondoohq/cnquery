// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package redisconf_test

import (
	"errors"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers/os/resources/redisconf"
)

// fakeLoader serves inlined fixtures so include resolution can be tested
// without a filesystem.
type fakeLoader struct{ files map[string]string }

func (f fakeLoader) Read(p string) (string, error) {
	if c, ok := f.files[p]; ok {
		return c, nil
	}
	return "", errors.New("not found: " + p)
}

func (f fakeLoader) Glob(pattern string) ([]string, error) {
	var out []string
	for p := range f.files {
		if ok, _ := path.Match(pattern, p); ok {
			out = append(out, p)
		}
	}
	// Deterministic order, since map iteration is not.
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

func load(t *testing.T, files map[string]string, main string) *redisconf.Conf {
	t.Helper()
	c, err := redisconf.Load(main, fakeLoader{files: files})
	require.NoError(t, err)
	return c
}

func TestSplitArgs(t *testing.T) {
	for _, tc := range []struct {
		line     string
		expected []string
	}{
		{`bind 127.0.0.1 -::1`, []string{"bind", "127.0.0.1", "-::1"}},
		{`save 3600 1 300 100`, []string{"save", "3600", "1", "300", "100"}},
		{`logfile ""`, []string{"logfile", ""}},
		{`save ""`, []string{"save", ""}},
		{`requirepass "a b c"`, []string{"requirepass", "a b c"}},
		{`requirepass 'single quoted'`, []string{"requirepass", "single quoted"}},
		{`rename-command CONFIG ""`, []string{"rename-command", "CONFIG", ""}},
		// A # mid-line is an ordinary character, not a comment.
		{`requirepass p#ssw0rd`, []string{"requirepass", "p#ssw0rd"}},
		{`user alice on ~cache:* +@read`, []string{"user", "alice", "on", "~cache:*", "+@read"}},
		// Escapes inside double quotes.
		{`requirepass "a\tb"`, []string{"requirepass", "a\tb"}},
		{`requirepass "\x41\x42"`, []string{"requirepass", "AB"}},
		// An unbalanced quote makes the server reject the line.
		{`requirepass "unterminated`, nil},
		{``, nil},
	} {
		assert.Equal(t, tc.expected, redisconf.SplitArgs(tc.line), "line %q", tc.line)
	}
}

func TestParseDirectivesSkipsCommentsAndBlanks(t *testing.T) {
	ds := redisconf.ParseDirectives("# comment\n\n   \nport 6380\n  # indented comment\nappendonly yes\n")
	require.Len(t, ds, 2)
	assert.Equal(t, "port", ds[0].Name)
	assert.Equal(t, []string{"6380"}, ds[0].Args)
	assert.Equal(t, "appendonly", ds[1].Name)
	assert.Equal(t, 4, ds[0].Line)
}

// An include splices the other file in at the point it appears, so a
// directive after the include wins and one before it loses.
func TestIncludeOrderingDecidesTheWinner(t *testing.T) {
	files := map[string]string{
		"/etc/redis/redis.conf":    "port 6379\ninclude /etc/redis/override.conf\nappendonly yes\n",
		"/etc/redis/override.conf": "port 6380\nappendonly no\n",
	}
	c := load(t, files, "/etc/redis/redis.conf")

	assert.Equal(t, int64(6380), c.Port(), "the include is after port, so it wins")
	assert.True(t, c.AppendOnly(), "appendonly is after the include, so it wins")
	assert.Equal(t, []string{"/etc/redis/redis.conf", "/etc/redis/override.conf"}, c.Files)
}

func TestIncludeGlob(t *testing.T) {
	files := map[string]string{
		"/etc/redis/redis.conf":     "include /etc/redis/conf.d/*.conf\n",
		"/etc/redis/conf.d/10.conf": "protected-mode no\n",
		"/etc/redis/conf.d/20.conf": "requirepass hunter2\n",
	}
	c := load(t, files, "/etc/redis/redis.conf")

	assert.False(t, c.ProtectedMode())
	assert.True(t, c.RequirepassSet())
	assert.Len(t, c.Files, 3)
}

// A missing include is skipped, but a missing main file is an error.
func TestMissingIncludeIsSkippedMissingMainIsNot(t *testing.T) {
	files := map[string]string{
		"/etc/redis/redis.conf": "include /etc/redis/gone.conf\nport 6380\n",
	}
	c := load(t, files, "/etc/redis/redis.conf")
	assert.Equal(t, int64(6380), c.Port())

	_, err := redisconf.Load("/etc/redis/absent.conf", fakeLoader{files: files})
	assert.Error(t, err)
}

func TestIncludeCycleTerminates(t *testing.T) {
	files := map[string]string{
		"/a.conf": "include /b.conf\nport 6380\n",
		"/b.conf": "include /a.conf\nappendonly yes\n",
	}
	c := load(t, files, "/a.conf")
	assert.Equal(t, int64(6380), c.Port())
	assert.True(t, c.AppendOnly())
}

func TestBytes(t *testing.T) {
	for _, tc := range []struct {
		raw      string
		expected int64
	}{
		{"100", 100},
		{"1k", 1000},
		{"1kb", 1024},
		{"1m", 1000 * 1000},
		{"1mb", 1024 * 1024},
		{"2gb", 2 * 1024 * 1024 * 1024},
		{"nonsense", -1},
	} {
		c := load(t, map[string]string{"/c": "maxmemory " + tc.raw + "\n"}, "/c")
		assert.Equal(t, tc.expected, c.Bytes("maxmemory", -1), "raw %q", tc.raw)
	}
}

func TestParseVersion(t *testing.T) {
	for _, tc := range []struct{ output, product, version string }{
		{"Redis server v=7.4.1 sha=00000000:0 malloc=jemalloc-5.3.0 bits=64", "redis", "7.4.1"},
		{"Valkey server v=8.0.1 sha=00000000:0 malloc=jemalloc bits=64", "valkey", "8.0.1"},
		{"", "", ""},
		{"command not found", "", ""},
	} {
		product, version := redisconf.ParseVersion(tc.output)
		assert.Equal(t, tc.product, product, "output %q", tc.output)
		assert.Equal(t, tc.version, version, "output %q", tc.output)
	}
}

func TestIsValkeyKeysOffDirectivesNotPath(t *testing.T) {
	// Named redis.conf, but the directives are Valkey's.
	c := load(t, map[string]string{"/etc/redis/redis.conf": "primaryauth secret\navailability-zone us-east-1a\n"}, "/etc/redis/redis.conf")
	assert.True(t, c.IsValkey())

	plain := load(t, map[string]string{"/c": "port 6379\nmasterauth secret\n"}, "/c")
	assert.False(t, plain.IsValkey())
}

func TestUnknownDirectivesAreIgnoredNotFatal(t *testing.T) {
	c := load(t, map[string]string{"/c": "totally-made-up yes\nport 6380\n"}, "/c")
	assert.Equal(t, int64(6380), c.Port())
}

func TestDirectiveNamesAreCaseInsensitive(t *testing.T) {
	c := load(t, map[string]string{"/c": "PROTECTED-MODE no\nAppendOnly yes\n"}, "/c")
	assert.False(t, c.ProtectedMode())
	assert.True(t, c.AppendOnly())
}

func TestFilesRecordedInLoadOrder(t *testing.T) {
	files := map[string]string{
		"/main.conf": "include /one.conf\ninclude /two.conf\n",
		"/one.conf":  "port 1\n",
		"/two.conf":  "port 2\n",
	}
	c := load(t, files, "/main.conf")
	assert.Equal(t, []string{"/main.conf", "/one.conf", "/two.conf"}, c.Files)
	assert.Equal(t, int64(2), c.Port())
	assert.True(t, strings.HasSuffix(c.Files[len(c.Files)-1], "two.conf"))
}
