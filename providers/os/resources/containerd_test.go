// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainerdSocketPath(t *testing.T) {
	// Verify the default socket path is set correctly
	assert.Equal(t, "/run/containerd/containerd.sock", defaultContainerdSocket)
}

// Note: Integration tests for containerd.containers require a running containerd daemon.
// The implementation uses the containerd Go SDK directly for better performance and
// reliability compared to shelling out to the `ctr` CLI.
//
// To test manually with containerd running:
//   cnquery shell local -c "containerd.containers { id status namespace }"
