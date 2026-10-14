// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The links in /proc/<pid>/fd name the sockets a process holds; without them
// no listening port was attributed to its process over this transport.
func TestReadlinkIfPossible(t *testing.T) {
	fs := &FS{runner: &fakeContainer{}}

	target, err := fs.ReadlinkIfPossible("/proc/71/fd/6")
	require.NoError(t, err)
	assert.Equal(t, "socket:[20845]", target)

	_, err = fs.ReadlinkIfPossible("/proc/71/fd/99")
	assert.Error(t, err, "a path that is not a link")
}
