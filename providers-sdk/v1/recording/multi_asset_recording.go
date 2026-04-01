// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package recording

type MultiAsset interface {
	GetAssetRecordings() []*Asset
	SetAssetRecording(uint32, *Asset)
}
