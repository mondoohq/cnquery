// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/types"
	"gopkg.in/yaml.v3"
)

type mqlHelmResourceInternal struct {
	cacheTemplateKey string
}

// parseK8sResources parses rendered YAML content into Kubernetes resource objects.
func parseK8sResources(runtime *plugin.Runtime, templateKey string, content string) ([]any, error) {
	var mqlResources []any

	// Split on YAML document separator
	docs := strings.Split(content, "---")
	docIndex := 0
	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		var obj map[string]any
		if err := yaml.Unmarshal([]byte(doc), &obj); err != nil {
			continue
		}

		// Skip empty or non-K8s YAML docs
		kind, _ := obj["kind"].(string)
		if kind == "" {
			continue
		}

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
			continue
		}

		// Use templateKey (chartName/templateName), namespace, kind, name, and docIndex for uniqueness
		id := "helm.resource:" + templateKey + ":" + kind + ":" + namespace + ":" + name + ":" + strconv.Itoa(docIndex)
		// Store the template ID using ":" separator to match the template __id format (helm.template:chartName:templateName)
		templateID := strings.Replace(templateKey, "/", ":", 1)
		res, err := CreateResource(runtime, "helm.resource", map[string]*llx.RawData{
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
			continue
		}
		mqlResource := res.(*mqlHelmResource)
		mqlResource.cacheTemplateKey = templateID
		mqlResources = append(mqlResources, mqlResource)
		docIndex++
	}

	return mqlResources, nil
}

func (r *mqlHelmResource) template() (*mqlHelmTemplate, error) {
	// cacheTemplateKey already uses ":" separator matching the template __id format
	res, err := NewResource(r.MqlRuntime, "helm.template", map[string]*llx.RawData{
		"__id": llx.StringData("helm.template:" + r.cacheTemplateKey),
	})
	if err != nil {
		log.Warn().Err(err).Str("templateKey", r.cacheTemplateKey).Msg("failed to resolve helm template for resource")
		r.Template.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	return res.(*mqlHelmTemplate), nil
}
