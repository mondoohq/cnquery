// Copyright Mondoo, Inc. 2024, 2026
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

// isCrossZoneClone mirrors the decision cloneDisk makes: a source disk whose
// zone differs from the target zone must be bridged through a snapshot.
func isCrossZoneClone(sourceDisk, targetZone string) bool {
	_, srcZone, _, err := parseDiskUrl(sourceDisk)
	return err == nil && srcZone != "" && srcZone != targetZone
}

func TestCloneDiskCrossZoneDecision(t *testing.T) {
	sourceDisk := "https://www.googleapis.com/compute/v1/projects/my-project-1234/zones/us-central1-a/disks/super-dupa-disk"

	// different zones -> must bridge through a snapshot
	assert.True(t, isCrossZoneClone(sourceDisk, "us-west1-a"),
		"a source disk in us-central1-a with target us-west1-a should be detected as cross-zone")

	// same zone -> direct clone, no snapshot bridge
	assert.False(t, isCrossZoneClone(sourceDisk, "us-central1-a"),
		"a source disk and target in the same zone should not be detected as cross-zone")
}
