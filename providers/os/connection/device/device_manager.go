// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package device

import "go.mondoo.com/cnquery/v11/providers/os/connection/device/shared"

type DeviceManager interface {
	Name() string
	IdentifyBlock(opts map[string]string) (shared.MountInfo, error)
	Mount() (string, error)
	UnmountAndClose()
}
