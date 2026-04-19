// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows

package firmware

// CollectNative is a no-op on non-Windows platforms.
func CollectNative() []Device {
	return nil
}
