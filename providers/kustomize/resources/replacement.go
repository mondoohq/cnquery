// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	kustomizeTypes "sigs.k8s.io/kustomize/api/types"
)

type mqlKustomizeReplacementInternal struct {
	replacementTargets []*kustomizeTypes.TargetSelector
	kustPath           string
}

func newMqlKustomizeReplacement(runtime *plugin.Runtime, kustPath string, index int, r *kustomizeTypes.ReplacementField) (*mqlKustomizeReplacement, error) {
	id := "kustomize.replacement:" + kustPath + ":" + strconv.Itoa(index)

	sourcePath := ""
	sourceKind := ""
	sourceName := ""

	if r.Source != nil {
		sourcePath = r.Source.FieldPath
		sourceKind = r.Source.Gvk.Kind
		sourceName = r.Source.Name
	}

	res, err := CreateResource(runtime, "kustomize.replacement", map[string]*llx.RawData{
		"__id":       llx.StringData(id),
		"sourcePath": llx.StringData(sourcePath),
		"sourceKind": llx.StringData(sourceKind),
		"sourceName": llx.StringData(sourceName),
	})
	if err != nil {
		return nil, err
	}

	mqlR := res.(*mqlKustomizeReplacement)
	mqlR.kustPath = kustPath
	mqlR.replacementTargets = r.Targets
	return mqlR, nil
}

func (r *mqlKustomizeReplacement) targets() ([]any, error) {
	var mqlTargets []any
	for i, t := range r.replacementTargets {
		kind := ""
		name := ""
		if t.Select != nil {
			kind = t.Select.Gvk.Kind
			name = t.Select.Name
		}

		for j, fp := range t.FieldPaths {
			id := "kustomize.replacementTarget:" + r.kustPath + ":" + strconv.Itoa(i) + ":" + kind + ":" + name + ":" + strconv.Itoa(j) + ":" + fp

			res, err := CreateResource(r.MqlRuntime, "kustomize.replacementTarget", map[string]*llx.RawData{
				"__id":      llx.StringData(id),
				"fieldPath": llx.StringData(fp),
				"kind":      llx.StringData(kind),
				"name":      llx.StringData(name),
			})
			if err != nil {
				return nil, err
			}
			mqlTargets = append(mqlTargets, res)
		}
	}
	return mqlTargets, nil
}
