// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	kustomizeTypes "sigs.k8s.io/kustomize/api/types"
)

func newMqlKustomizeImage(runtime *plugin.Runtime, kustPath string, img kustomizeTypes.Image) (*mqlKustomizeImage, error) {
	id := "kustomize.image:" + kustPath + ":" + img.Name

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
