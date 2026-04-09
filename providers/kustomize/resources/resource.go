// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/types"
	"gopkg.in/yaml.v3"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

func (k *mqlKustomizeKustomization) resources() ([]any, error) {
	rendered, err := k.fetchRendered()
	if err != nil {
		log.Warn().Err(err).Str("path", k.kustPath).Msg("failed to render kustomize overlay, returning empty resources")
		return []any{}, nil
	}

	var allResources []any
	for _, obj := range rendered {
		mqlRes, err := newMqlKustomizeResource(k.MqlRuntime, k.kustPath, obj)
		if err != nil {
			log.Warn().Err(err).Msg("skipping rendered kustomize resource")
			continue
		}
		allResources = append(allResources, mqlRes)
	}
	return allResources, nil
}

func (k *mqlKustomizeKustomization) fetchRendered() ([]map[string]any, error) {
	if k.fetched.Load() {
		return k.rendered, k.renderedErr
	}
	k.lock.Lock()
	defer k.lock.Unlock()
	if k.fetched.Load() {
		return k.rendered, k.renderedErr
	}

	fSys := filesys.MakeFsOnDisk()
	kustomizer := krusty.MakeKustomizer(krusty.MakeDefaultOptions())

	resMap, err := kustomizer.Run(fSys, k.kustPath)
	if err != nil {
		k.renderedErr = err
		k.fetched.Store(true)
		return nil, err
	}

	var resources []map[string]any
	for _, res := range resMap.Resources() {
		yamlBytes, err := res.AsYAML()
		if err != nil {
			log.Warn().Err(err).Msg("skipping kustomize resource: failed to convert to YAML")
			continue
		}
		var obj map[string]any
		if err := yaml.Unmarshal(yamlBytes, &obj); err != nil {
			log.Warn().Err(err).Msg("skipping kustomize resource: failed to unmarshal YAML")
			continue
		}
		resources = append(resources, obj)
	}

	k.rendered = resources
	k.fetched.Store(true)
	return k.rendered, nil
}

func newMqlKustomizeResource(runtime *plugin.Runtime, kustPath string, obj map[string]any) (*mqlKustomizeResource, error) {
	kind, _ := obj["kind"].(string)
	apiVersion, _ := obj["apiVersion"].(string)
	name := ""
	namespace := ""
	labels := map[string]any{}
	annotations := map[string]any{}

	if metadata, ok := obj["metadata"].(map[string]any); ok {
		name, _ = metadata["name"].(string)
		namespace, _ = metadata["namespace"].(string)
		if l, ok := metadata["labels"].(map[string]any); ok {
			labels = l
		}
		if a, ok := metadata["annotations"].(map[string]any); ok {
			annotations = a
		}
	}

	labelsStr := make(map[string]any, len(labels))
	for k, v := range labels {
		if s, ok := v.(string); ok {
			labelsStr[k] = s
		}
	}
	annotationsStr := make(map[string]any, len(annotations))
	for k, v := range annotations {
		if s, ok := v.(string); ok {
			annotationsStr[k] = s
		}
	}

	manifest, err := convert.JsonToDict(obj)
	if err != nil {
		return nil, err
	}

	id := "kustomize.resource:" + kustPath + ":" + apiVersion + ":" + kind + ":" + namespace + "/" + name

	res, err := CreateResource(runtime, "kustomize.resource", map[string]*llx.RawData{
		"__id":        llx.StringData(id),
		"apiVersion":  llx.StringData(apiVersion),
		"kind":        llx.StringData(kind),
		"name":        llx.StringData(name),
		"namespace":   llx.StringData(namespace),
		"labels":      llx.MapData(labelsStr, types.String),
		"annotations": llx.MapData(annotationsStr, types.String),
		"manifest":    llx.DictData(manifest),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlKustomizeResource), nil
}

func initKustomizeResource(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return args, nil, nil
}
