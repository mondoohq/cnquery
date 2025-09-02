// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package explorer

// HasFeature checks if the resolved policy indicates if certain features
// should be supported on the execution and collection of it
func (r *ResolvedPack) HasFeature(feature ServerFeature) bool {
	for _, f := range r.Features {
		if f == feature {
			return true
		}
	}
	return false
}
