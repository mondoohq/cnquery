// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package explorer

func (r *ResolvedPack) HasFeature(feature ServerFeature) bool {
	for _, f := range r.Features {
		if f == feature {
			return true
		}
	}
	return false
}
