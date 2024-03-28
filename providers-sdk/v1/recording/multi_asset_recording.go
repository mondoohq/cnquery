// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package recording

type MultiAsset interface {
	GetAssetRecordings() []*Asset
	SetAssetRecording(uint32, *Asset)
}
