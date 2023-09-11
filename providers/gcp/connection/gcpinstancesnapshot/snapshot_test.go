// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package gcpinstancesnapshot

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDiskUrl(t *testing.T) {
	diskUrl := "https://www.googleapis.com/compute/beta/projects/my-project-1234/zones/us-central1-a/disks/super-dupa-disk"
	projectID, zone, disk, err := parseDiskUrl(diskUrl)
	require.NoError(t, err)
	assert.Equal(t, "my-project-1234", projectID)
	assert.Equal(t, "us-central1-a", zone)
	assert.Equal(t, "super-dupa-disk", disk)
}
