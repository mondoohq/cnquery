// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package device

import (
	"go.mondoo.com/cnquery/v11/providers/os/connection/snapshot"
)

type DeviceManager interface {
	// Name returns the name of the device manager
	Name() string
	// IdentifyMountTargets returns a list of partitions that match the given options and can be mounted
	IdentifyMountTargets(opts map[string]string) ([]*snapshot.Partition, error)
	// Mounts the partition and returns the directories it was mounted to
	Mount(partitions []*snapshot.Partition) ([]*snapshot.MountedPartition, error)
	// UnmountAndClose unmounts the partitions from the specified dirs and closes the device manager
	UnmountAndClose()
}
