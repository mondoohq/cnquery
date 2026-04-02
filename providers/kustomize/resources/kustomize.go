// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"sync"
	"sync/atomic"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/kustomize/connection"
	"go.mondoo.com/mql/v13/types"
	kustomizeTypes "sigs.k8s.io/kustomize/api/types"
)

func (r *mqlKustomize) id() (string, error) {
	return "kustomize", nil
}

func (r *mqlKustomize) kustomizations() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.KustomizeConnection)
	entries := conn.Kustomizations()

	var mqlKusts []any
	for _, entry := range entries {
		mqlK, err := newMqlKustomization(r.MqlRuntime, entry)
		if err != nil {
			return nil, err
		}
		mqlKusts = append(mqlKusts, mqlK)
	}
	return mqlKusts, nil
}

type mqlKustomizeKustomizationInternal struct {
	kustomization *kustomizeTypes.Kustomization
	kustPath      string
	rendered      []map[string]any
	renderedErr   error
	lock          sync.Mutex
	fetched       atomic.Bool
}

func newMqlKustomization(runtime *plugin.Runtime, entry *connection.KustomizationEntry) (*mqlKustomizeKustomization, error) {
	k := entry.Kustomization

	commonLabels := make(map[string]any, len(k.CommonLabels))
	for key, val := range k.CommonLabels {
		commonLabels[key] = val
	}
	commonAnnotations := make(map[string]any, len(k.CommonAnnotations))
	for key, val := range k.CommonAnnotations {
		commonAnnotations[key] = val
	}

	resourceRefs := convert.SliceAnyToInterface(k.Resources)
	componentRefs := convert.SliceAnyToInterface(k.Components)

	res, err := CreateResource(runtime, "kustomize.kustomization", map[string]*llx.RawData{
		"__id":              llx.StringData("kustomize.kustomization:" + entry.Path),
		"path":              llx.StringData(entry.Path),
		"apiVersion":        llx.StringData(k.APIVersion),
		"kind":              llx.StringData(k.Kind),
		"namespace":         llx.StringData(k.Namespace),
		"namePrefix":        llx.StringData(k.NamePrefix),
		"nameSuffix":        llx.StringData(k.NameSuffix),
		"commonLabels":      llx.MapData(commonLabels, types.String),
		"commonAnnotations": llx.MapData(commonAnnotations, types.String),
		"resourceRefs":      llx.ArrayData(resourceRefs, types.String),
		"componentRefs":     llx.ArrayData(componentRefs, types.String),
	})
	if err != nil {
		return nil, err
	}
	mqlK := res.(*mqlKustomizeKustomization)
	mqlK.kustomization = k
	mqlK.kustPath = entry.Path
	return mqlK, nil
}

func (k *mqlKustomizeKustomization) id() (string, error) {
	return "kustomize.kustomization:" + k.Path.Data, nil
}

func (k *mqlKustomizeKustomization) patches() ([]any, error) {
	var mqlPatches []any
	for i, p := range k.kustomization.Patches {
		mqlP, err := newMqlKustomizePatch(k.MqlRuntime, k.kustPath, i, &p)
		if err != nil {
			return nil, err
		}
		mqlPatches = append(mqlPatches, mqlP)
	}
	return mqlPatches, nil
}

func (k *mqlKustomizeKustomization) configMapGenerators() ([]any, error) {
	return newMqlConfigMapGenerators(k.MqlRuntime, k.kustPath, k.kustomization.ConfigMapGenerator)
}

func (k *mqlKustomizeKustomization) secretGenerators() ([]any, error) {
	return newMqlSecretGenerators(k.MqlRuntime, k.kustPath, k.kustomization.SecretGenerator)
}

func (k *mqlKustomizeKustomization) images() ([]any, error) {
	var mqlImages []any
	for _, img := range k.kustomization.Images {
		mqlImg, err := newMqlKustomizeImage(k.MqlRuntime, k.kustPath, img)
		if err != nil {
			return nil, err
		}
		mqlImages = append(mqlImages, mqlImg)
	}
	return mqlImages, nil
}

func (k *mqlKustomizeKustomization) replacements() ([]any, error) {
	var mqlReplacements []any
	for i, r := range k.kustomization.Replacements {
		mqlR, err := newMqlKustomizeReplacement(k.MqlRuntime, k.kustPath, i, &r)
		if err != nil {
			return nil, err
		}
		mqlReplacements = append(mqlReplacements, mqlR)
	}
	return mqlReplacements, nil
}
