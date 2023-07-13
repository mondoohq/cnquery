//go:build debugtest
// +build debugtest

package gcpinstancesnapshot

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLatestInstanceSnapshot(t *testing.T) {
	sc, err := NewSnapshotCreator()
	require.NoError(t, err)

	projectId := "my-project-1234"
	zone := "us-central1-a"
	instanceName := "super-dupa-instance"

	ii, err := sc.InstanceInfo(projectId, zone, instanceName)
	require.NoError(t, err)
	assert.Equal(t, "super-dupa-instance", ii.InstanceName)

	snap, err := sc.searchLatestSnapshot(projectId, ii.BootDiskSourceURL)
	require.NoError(t, err)
	assert.NotNil(t, snap)
}
