// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	kustomizeTypes "sigs.k8s.io/kustomize/api/types"
)

func newMqlKustomizePatch(runtime *plugin.Runtime, kustPath string, index int, p *kustomizeTypes.Patch) (*mqlKustomizePatch, error) {
	targetGroup := ""
	targetVersion := ""
	targetKind := ""
	targetName := ""
	targetNamespace := ""
	targetLabelSelector := ""
	targetAnnotationSelector := ""

	if p.Target != nil {
		targetGroup = p.Target.Group
		targetVersion = p.Target.Version
		targetKind = p.Target.Kind
		targetName = p.Target.Name
		targetNamespace = p.Target.Namespace
		targetLabelSelector = p.Target.LabelSelector
		targetAnnotationSelector = p.Target.AnnotationSelector
	}

	id := "kustomize.patch:" + kustPath + ":" + strconv.Itoa(index)

	res, err := CreateResource(runtime, "kustomize.patch", map[string]*llx.RawData{
		"__id":                     llx.StringData(id),
		"content":                  llx.StringData(p.Patch),
		"path":                     llx.StringData(p.Path),
		"targetGroup":              llx.StringData(targetGroup),
		"targetVersion":            llx.StringData(targetVersion),
		"targetKind":               llx.StringData(targetKind),
		"targetName":               llx.StringData(targetName),
		"targetNamespace":          llx.StringData(targetNamespace),
		"targetLabelSelector":      llx.StringData(targetLabelSelector),
		"targetAnnotationSelector": llx.StringData(targetAnnotationSelector),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlKustomizePatch), nil
}

// Satisfy the plugin interface — all fields are static, no computed fields needed.
var _ plugin.Resource = (*mqlKustomizePatch)(nil)
