// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package updates

import (
	"bytes"
	"compress/gzip"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAptHistory(t *testing.T) {
	f, err := os.Open("./testdata/apt_history.log")
	require.NoError(t, err)
	defer f.Close()

	got, ok := ParseAptHistory(f, time.UTC)
	require.True(t, ok)

	// The 05-09 block only removed a package and the 05-11 block failed, so the
	// unattended upgrade on 05-04 is the newest run that installed anything.
	assert.Equal(t, time.Date(2026, 5, 4, 3, 22, 7, 0, time.UTC), got)
}

func TestParseAptHistoryCases(t *testing.T) {
	tests := []struct {
		name string
		log  string
		want string
	}{
		{
			name: "install only",
			log: "Start-Date: 2026-01-02  03:04:05\n" +
				"Install: curl:amd64 (8.5.0)\n" +
				"End-Date: 2026-01-02  03:04:06\n",
			want: "2026-01-02T03:04:05Z",
		},
		{
			name: "reinstall counts as a change",
			log: "Start-Date: 2026-01-02  03:04:05\n" +
				"Reinstall: apt:arm64 (2.7.14build2)\n" +
				"End-Date: 2026-01-02  03:04:06\n",
			want: "2026-01-02T03:04:05Z",
		},
		{
			name: "removal only is not an update",
			log: "Start-Date: 2026-01-02  03:04:05\n" +
				"Remove: telnet:amd64 (0.17)\n" +
				"End-Date: 2026-01-02  03:04:06\n",
			want: "",
		},
		{
			name: "purge only is not an update",
			log: "Start-Date: 2026-01-02  03:04:05\n" +
				"Purge: telnet:amd64 (0.17)\n" +
				"End-Date: 2026-01-02  03:04:06\n",
			want: "",
		},
		{
			name: "failed run does not count",
			log: "Start-Date: 2026-01-02  03:04:05\n" +
				"Upgrade: openssl:amd64 (3.0.13, 3.0.14)\n" +
				"Error: Sub-process /usr/bin/dpkg returned an error code (1)\n" +
				"End-Date: 2026-01-02  03:04:09\n",
			want: "",
		},
		{
			name: "empty change field does not count",
			log: "Start-Date: 2026-01-02  03:04:05\n" +
				"Upgrade:\n" +
				"End-Date: 2026-01-02  03:04:06\n",
			want: "",
		},
		{
			name: "block truncated by a kill still counts",
			log: "Start-Date: 2026-01-02  03:04:05\n" +
				"Upgrade: openssl:amd64 (3.0.13, 3.0.14)\n" +
				"Start-Date: 2026-01-03  03:04:05\n" +
				"Remove: telnet:amd64 (0.17)\n",
			want: "2026-01-02T03:04:05Z",
		},
		{
			name: "out of order blocks take the newest",
			log: "Start-Date: 2026-03-01  10:00:00\n" +
				"Install: a:amd64 (1)\n" +
				"End-Date: 2026-03-01  10:00:01\n" +
				"\n" +
				"Start-Date: 2026-02-01  10:00:00\n" +
				"Install: b:amd64 (1)\n" +
				"End-Date: 2026-02-01  10:00:01\n",
			want: "2026-03-01T10:00:00Z",
		},
		{
			name: "single space separator is tolerated",
			log: "Start-Date: 2026-01-02 03:04:05\n" +
				"Install: curl:amd64 (8.5.0)\n" +
				"End-Date: 2026-01-02 03:04:06\n",
			want: "2026-01-02T03:04:05Z",
		},
		{
			name: "unparseable date is skipped",
			log: "Start-Date: not-a-date\n" +
				"Install: curl:amd64 (8.5.0)\n" +
				"End-Date: 2026-01-02  03:04:06\n",
			want: "",
		},
		{
			name: "empty log",
			log:  "",
			want: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := ParseAptHistory(strings.NewReader(test.log), time.UTC)
			if test.want == "" {
				assert.False(t, ok)
				assert.True(t, got.IsZero())
				return
			}
			require.True(t, ok)
			assert.Equal(t, test.want, got.Format(time.RFC3339))
		})
	}
}

func TestParseDpkgLog(t *testing.T) {
	f, err := os.Open("./testdata/dpkg.log")
	require.NoError(t, err)
	defer f.Close()

	got, ok := ParseDpkgLog(f, time.UTC)
	require.True(t, ok)

	// The 05-09 lines are a removal, a not-installed status and a trigger, none
	// of which is an install.
	assert.Equal(t, time.Date(2026, 5, 4, 3, 22, 8, 0, time.UTC), got)
}

func TestParseDpkgLogCases(t *testing.T) {
	tests := []struct {
		name string
		log  string
		want string
	}{
		{
			name: "status installed",
			log:  "2026-01-02 03:04:05 status installed curl:amd64 8.5.0\n",
			want: "2026-01-02T03:04:05Z",
		},
		{
			name: "install action",
			log:  "2026-01-02 03:04:05 install curl:amd64 <none> 8.5.0\n",
			want: "2026-01-02T03:04:05Z",
		},
		{
			name: "upgrade action",
			log:  "2026-01-02 03:04:05 upgrade openssl:amd64 3.0.13 3.0.14\n",
			want: "2026-01-02T03:04:05Z",
		},
		{
			name: "removal is not an update",
			log:  "2026-01-02 03:04:05 remove telnet:amd64 0.17 <none>\n",
			want: "",
		},
		{
			name: "status half-configured is not an install",
			log:  "2026-01-02 03:04:05 status half-configured curl:amd64 8.5.0\n",
			want: "",
		},
		{
			name: "status not-installed is not an install",
			log:  "2026-01-02 03:04:05 status not-installed telnet:amd64 <none>\n",
			want: "",
		},
		{
			name: "trigger is not an install",
			log:  "2026-01-02 03:04:05 trigproc man-db:amd64 2.12.0-4build2 <none>\n",
			want: "",
		},
		{
			name: "startup is not an install",
			log:  "2026-01-02 03:04:05 startup archives unpack\n",
			want: "",
		},
		{
			name: "configure alone is not an install",
			log:  "2026-01-02 03:04:05 configure curl:amd64 8.5.0 <none>\n",
			want: "",
		},
		{
			name: "newest of several installs wins",
			log: "2026-03-01 10:00:00 status installed a:amd64 1\n" +
				"2026-02-01 10:00:00 status installed b:amd64 1\n",
			want: "2026-03-01T10:00:00Z",
		},
		{
			name: "short line is skipped",
			log:  "2026-01-02 03:04:05 status\n",
			want: "",
		},
		{
			name: "empty log",
			log:  "",
			want: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := ParseDpkgLog(strings.NewReader(test.log), time.UTC)
			if test.want == "" {
				assert.False(t, ok)
				assert.True(t, got.IsZero())
				return
			}
			require.True(t, ok)
			assert.Equal(t, test.want, got.Format(time.RFC3339))
		})
	}
}

// A dpkg or apt timestamp carries no zone, so the caller's location decides what
// instant it names. Getting this wrong shifts the reported patch age by the
// offset.
func TestParseDpkgTimeUsesLocation(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	require.NoError(t, err)

	got, err := parseDpkgTime("2026-01-02  03:04:05", berlin)
	require.NoError(t, err)
	assert.Equal(t, "2026-01-02T02:04:05Z", got.UTC().Format(time.RFC3339))

	got, err = parseDpkgTime("2026-01-02 03:04:05", nil)
	require.NoError(t, err)
	assert.Equal(t, "2026-01-02T03:04:05Z", got.UTC().Format(time.RFC3339),
		"a nil location must fall back to UTC rather than panic")

	_, err = parseDpkgTime("2026-01-02", time.UTC)
	assert.Error(t, err)
}

func TestRotationPaths(t *testing.T) {
	paths := rotationPaths("/var/log/dpkg.log")

	require.Equal(t, "/var/log/dpkg.log", paths[0], "the live log must be tried first")
	assert.Equal(t, "/var/log/dpkg.log.1", paths[1])
	assert.Equal(t, "/var/log/dpkg.log.1.gz", paths[2])
	assert.Len(t, paths, 1+2*dpkgLogMaxRotations)
}

func TestLastInstalledDebianSourcePriority(t *testing.T) {
	aptLog := "Start-Date: 2026-05-04  03:22:07\n" +
		"Upgrade: libpam-modules:amd64 (1.5.3-5ubuntu5, 1.5.3-5ubuntu5.5)\n" +
		"End-Date: 2026-05-04  03:22:09\n"
	dpkgLog := "2026-05-01 09:14:13 status installed curl:amd64 8.5.0\n"
	statusMtime := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)

	t.Run("apt history wins", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		require.NoError(t, afero.WriteFile(fs, aptHistoryPath, []byte(aptLog), 0o644))
		require.NoError(t, afero.WriteFile(fs, dpkgLogPath, []byte(dpkgLog), 0o644))

		got, err := lastInstalledDebianFS(fs, time.UTC)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, LastUpdateSourceAptHistory, got.Source)
		assert.Equal(t, "2026-05-04T03:22:07Z", got.Time.Format(time.RFC3339))
	})

	t.Run("dpkg log when apt history is absent", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		require.NoError(t, afero.WriteFile(fs, dpkgLogPath, []byte(dpkgLog), 0o644))

		got, err := lastInstalledDebianFS(fs, time.UTC)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, LastUpdateSourceDpkgLog, got.Source)
		assert.Equal(t, "2026-05-01T09:14:13Z", got.Time.Format(time.RFC3339))
	})

	t.Run("dpkg log when apt history holds no install", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		removalOnly := "Start-Date: 2026-05-09  14:17:19\nRemove: telnet:amd64 (0.17)\nEnd-Date: 2026-05-09  14:17:20\n"
		require.NoError(t, afero.WriteFile(fs, aptHistoryPath, []byte(removalOnly), 0o644))
		require.NoError(t, afero.WriteFile(fs, dpkgLogPath, []byte(dpkgLog), 0o644))

		got, err := lastInstalledDebianFS(fs, time.UTC)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, LastUpdateSourceDpkgLog, got.Source)
	})

	t.Run("status mtime when both logs are stripped", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		require.NoError(t, afero.WriteFile(fs, dpkgStatusPath, []byte("Package: curl\n"), 0o644))
		require.NoError(t, fs.Chtimes(dpkgStatusPath, statusMtime, statusMtime))

		got, err := lastInstalledDebianFS(fs, time.UTC)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, LastUpdateSourceDpkgStatus, got.Source)
		assert.Equal(t, "2026-04-01T12:00:00Z", got.Time.Format(time.RFC3339))
	})

	t.Run("nothing to read", func(t *testing.T) {
		got, err := lastInstalledDebianFS(afero.NewMemMapFs(), time.UTC)
		require.NoError(t, err)
		assert.Nil(t, got, "an absent record is null, not an error")
	})
}

// Right after a rotation the live log is empty and the answer is in the gzipped
// copy. Without the walk this reports the status-file mtime, or nothing at all.
func TestLastInstalledDebianReadsRotatedLog(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, aptHistoryPath, []byte(""), 0o644))

	rotated := "Start-Date: 2026-04-02  01:02:03\n" +
		"Install: curl:amd64 (8.5.0)\n" +
		"End-Date: 2026-04-02  01:02:04\n"

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write([]byte(rotated))
	require.NoError(t, err)
	require.NoError(t, gz.Close())
	require.NoError(t, afero.WriteFile(fs, aptHistoryPath+".1.gz", buf.Bytes(), 0o644))

	got, err := lastInstalledDebianFS(fs, time.UTC)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, LastUpdateSourceAptHistory, got.Source)
	assert.Equal(t, "2026-04-02T01:02:03Z", got.Time.Format(time.RFC3339))
}

// A file that ends in .gz but is not gzipped must be skipped rather than
// aborting the walk, so a later rotation can still answer.
func TestLastInstalledDebianSkipsCorruptGzip(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, aptHistoryPath+".1.gz", []byte("not gzip"), 0o644))
	require.NoError(t, afero.WriteFile(fs, dpkgLogPath, []byte("2026-05-01 09:14:13 status installed curl:amd64 8.5.0\n"), 0o644))

	got, err := lastInstalledDebianFS(fs, time.UTC)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, LastUpdateSourceDpkgLog, got.Source)
}
