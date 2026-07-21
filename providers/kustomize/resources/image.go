// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	kustomizeTypes "sigs.k8s.io/kustomize/api/types"
)

func newMqlKustomizeImage(runtime *plugin.Runtime, kustPath string, idx int, img kustomizeTypes.Image) (*mqlKustomizeImage, error) {
	// Include the list index in the __id: an images: list can legally contain
	// two entries with the same name (e.g. one overriding the tag, one the
	// digest); without the index the second collides with the first in the
	// resource cache and is lost.
	id := "kustomize.image:" + kustPath + ":" + strconv.Itoa(idx) + ":" + img.Name

	res, err := CreateResource(runtime, "kustomize.image", map[string]*llx.RawData{
		"__id":    llx.StringData(id),
		"name":    llx.StringData(img.Name),
		"newName": llx.StringData(img.NewName),
		"newTag":  llx.StringData(img.NewTag),
		"digest":  llx.StringData(img.Digest),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlKustomizeImage), nil
}

var _ plugin.Resource = (*mqlKustomizeImage)(nil)
