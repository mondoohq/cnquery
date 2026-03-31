// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

// Look into provider/providers.go for how asset information is attached.

func (a *mqlAsset) id() (string, error) {
	return "asset", nil
}
