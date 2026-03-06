// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

// TODO: Remove this file once AutoUpdateEngine is set to "default" status
// in features.yaml (i.e. enabled on all platforms including Windows).

//go:build !windows

package mql

func init() {
	DefaultFeatures = append(DefaultFeatures, byte(AutoUpdateEngine))
}
