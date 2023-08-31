// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package gcpinstancesnapshot

func SnapshotPlatformMrn(project string, snapshotName string) string {
	return "//platformid.api.mondoo.app/runtime/gcp/compute/v1/projects/" + project + "/snapshots/" + snapshotName
}
