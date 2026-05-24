// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"sync"

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
	renderedOnce  sync.Once
	// stampOnce guards the post-construction write of kustomization
	// and kustPath. CreateResource may return a cached instance for
	// concurrent callers with the same __id; stampOnce ensures the
	// stamp happens exactly once across those goroutines.
	stampOnce sync.Once
}

func newMqlKustomization(runtime *plugin.Runtime, entry *connection.KustomizationEntry) (*mqlKustomizeKustomization, error) {
	if entry == nil || entry.Kustomization == nil {
		return nil, errors.New("kustomize: entry has no parsed kustomization")
	}
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
	// CreateResource may return an already-cached instance when two
	// callers ask for the same __id; stampOnce keeps the write
	// race-free under concurrent newMqlKustomization calls and
	// happens-before any subsequent reader on the returned pointer.
	mqlK.stampOnce.Do(func() {
		mqlK.kustomization = k
		mqlK.kustPath = entry.Path
	})
	return mqlK, nil
}

// initKustomizeKustomization handles selector-style lookups
// (`kustomize.kustomization(path: "...")`). It locates the matching entry
// on the connection and routes through newMqlKustomization so Internal
// state stays populated — without this, field accessors would nil-deref
// on a bare resource constructed only from the `path` arg.
//
// Returning `args, nil, nil` for an unknown / empty path matches the
// "bare resource is a valid empty state" convention used elsewhere in
// this repo; callers see a resource that satisfies the type but yields
// errors from the accessors.
func initKustomizeKustomization(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) == 0 {
		return args, nil, nil
	}
	pathArg, ok := args["path"]
	if !ok || pathArg == nil {
		return args, nil, nil
	}
	path, _ := pathArg.Value.(string)
	if path == "" {
		return args, nil, nil
	}

	conn, ok := runtime.Connection.(*connection.KustomizeConnection)
	if !ok {
		return args, nil, nil
	}
	for _, entry := range conn.Kustomizations() {
		if entry.Path == path {
			mqlK, err := newMqlKustomization(runtime, entry)
			if err != nil {
				return nil, nil, err
			}
			return args, mqlK, nil
		}
	}
	return args, nil, nil
}

func (k *mqlKustomizeKustomization) id() (string, error) {
	return "kustomize.kustomization:" + k.Path.Data, nil
}

// resolveEntry returns the parsed kustomization, repopulating Internal
// state from the connection when needed (e.g., when the resource was
// constructed without going through newMqlKustomization). Each accessor
// calls this first instead of dereferencing k.kustomization directly.
func (k *mqlKustomizeKustomization) resolveEntry() (*kustomizeTypes.Kustomization, string, error) {
	if k.kustomization != nil {
		return k.kustomization, k.kustPath, nil
	}
	conn, ok := k.MqlRuntime.Connection.(*connection.KustomizeConnection)
	if !ok {
		return nil, "", errors.New("kustomize: connection is not a KustomizeConnection")
	}
	path := k.Path.Data
	for _, entry := range conn.Kustomizations() {
		if entry.Path == path {
			k.stampOnce.Do(func() {
				k.kustomization = entry.Kustomization
				k.kustPath = entry.Path
			})
			return k.kustomization, k.kustPath, nil
		}
	}
	return nil, "", errors.New("kustomize: no kustomization loaded for path " + path)
}

func (k *mqlKustomizeKustomization) patches() ([]any, error) {
	kust, kustPath, err := k.resolveEntry()
	if err != nil {
		return nil, err
	}
	var mqlPatches []any
	for i, p := range kust.Patches {
		mqlP, err := newMqlKustomizePatch(k.MqlRuntime, kustPath, i, &p)
		if err != nil {
			return nil, err
		}
		mqlPatches = append(mqlPatches, mqlP)
	}
	return mqlPatches, nil
}

func (k *mqlKustomizeKustomization) configMapGenerators() ([]any, error) {
	kust, kustPath, err := k.resolveEntry()
	if err != nil {
		return nil, err
	}
	return newMqlConfigMapGenerators(k.MqlRuntime, kustPath, kust.ConfigMapGenerator)
}

func (k *mqlKustomizeKustomization) secretGenerators() ([]any, error) {
	kust, kustPath, err := k.resolveEntry()
	if err != nil {
		return nil, err
	}
	return newMqlSecretGenerators(k.MqlRuntime, kustPath, kust.SecretGenerator)
}

func (k *mqlKustomizeKustomization) images() ([]any, error) {
	kust, kustPath, err := k.resolveEntry()
	if err != nil {
		return nil, err
	}
	var mqlImages []any
	for _, img := range kust.Images {
		mqlImg, err := newMqlKustomizeImage(k.MqlRuntime, kustPath, img)
		if err != nil {
			return nil, err
		}
		mqlImages = append(mqlImages, mqlImg)
	}
	return mqlImages, nil
}

func (k *mqlKustomizeKustomization) replacements() ([]any, error) {
	kust, kustPath, err := k.resolveEntry()
	if err != nil {
		return nil, err
	}
	var mqlReplacements []any
	for i, r := range kust.Replacements {
		mqlR, err := newMqlKustomizeReplacement(k.MqlRuntime, kustPath, i, &r)
		if err != nil {
			return nil, err
		}
		mqlReplacements = append(mqlReplacements, mqlR)
	}
	return mqlReplacements, nil
}
