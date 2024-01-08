// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azureinstancesnapshot

func SnapshotPlatformMrn(snapshotId string) string {
	return "//platformid.api.mondoo.app/runtime/azure" + snapshotId
}

func DiskPlatformMrn(diskId string) string {
	return "//platformid.api.mondoo.app/runtime/azure" + diskId
}
