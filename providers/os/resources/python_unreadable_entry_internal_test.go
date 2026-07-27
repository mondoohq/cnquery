// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ghostDirFs lists ghost as a directory but fails to open it, reproducing what a
// container image's layered filesystem does with a dist-info deleted in a later
// layer: the entry survives in the merged listing, opening it does not.
type ghostDirFs struct {
	afero.Fs
	ghost string
}

func (f *ghostDirFs) Open(name string) (afero.File, error) {
	if name == f.ghost {
		return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
	}
	return f.Fs.Open(name)
}

// A single unreadable entry must not abort the whole site-packages directory.
//
// Regression test for a real container scan: site-packages listed a stale
// autocommand-2.2.2.dist-info, reading it failed, collection returned that error
// for the entire directory, and the image reported zero Python packages --
// litellm among them, even though litellm-1.80.15.dist-info sat right there and
// was perfectly readable.
//
// The readable sibling here deliberately carries no METADATA, so nothing has to
// be turned into a resource and the test needs no plugin runtime. Reaching it at
// all is the point: before the fix the ghost aborted the loop and this returned
// an error.
func TestCollectPythonPackages_UnreadableEntryDoesNotAbortDirectory(t *testing.T) {
	const siteDir = "/usr/lib/python3.13/site-packages"

	base := afero.NewMemMapFs()
	// "autocommand" sorts before "litellm", matching the real failure where the
	// bad entry is reached first and kills everything after it
	require.NoError(t, base.MkdirAll(siteDir+"/autocommand-2.2.2.dist-info", 0o755))
	require.NoError(t, base.MkdirAll(siteDir+"/litellm-1.80.15.dist-info", 0o755))

	fs := &ghostDirFs{Fs: base, ghost: siteDir + "/autocommand-2.2.2.dist-info"}

	// precondition: both are listed, and the ghost genuinely fails to open
	entries, err := afero.ReadDir(fs, siteDir)
	require.NoError(t, err)
	require.Len(t, entries, 2, "both entries must appear in the listing")
	_, err = afero.ReadDir(fs, siteDir+"/autocommand-2.2.2.dist-info")
	require.Error(t, err, "the ghost entry must fail to open")

	_, err = collectPythonPackages(nil, fs, siteDir)
	assert.NoError(t, err,
		"one unreadable entry must be skipped, not fail the entire site-packages directory")
}
