// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/types"
	"gopkg.in/yaml.v3"
)

// yamlDocSeparator matches a YAML document separator: `---` alone on a line
// (optionally with trailing whitespace), at column 0.
var yamlDocSeparator = regexp.MustCompile(`(?m)^---[ \t]*$`)

// splitYAMLDocuments splits a multi-document YAML stream on real document
// separators only. The previous strings.Split(content, "---") split on ANY
// occurrence of "---" — including inside a string value, an indented block
// scalar (e.g. a ConfigMap data payload), or a comment — which silently
// corrupted or dropped valid Kubernetes resources.
func splitYAMLDocuments(content string) []string {
	return yamlDocSeparator.Split(content, -1)
}

type mqlHelmResourceInternal struct {
	cacheTemplateKey string
}

// parseK8sResources parses rendered YAML content into Kubernetes resource
// objects. isCRD marks resources that came from the chart's crds/ directory
// (helm installs those ahead of templating) rather than from a template.
func parseK8sResources(runtime *plugin.Runtime, templateKey string, content string, isCRD bool) ([]any, error) {
	var mqlResources []any

	// Split on YAML document separators only (not on `---` inside values)
	docs := splitYAMLDocuments(content)
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

		isHook, hookTypes, hookWeight, hookDeletePolicies := parseHookAnnotations(annotations)

		// Use templateKey (chartName/templateName), namespace, kind, name, and docIndex for uniqueness
		id := "helm.resource:" + templateKey + ":" + kind + ":" + namespace + ":" + name + ":" + strconv.Itoa(docIndex)
		// Store the template ID using ":" separator to match the template __id format (helm.template:chartName:templateName)
		templateID := strings.Replace(templateKey, "/", ":", 1)
		res, err := CreateResource(runtime, "helm.resource", map[string]*llx.RawData{
			"__id":               llx.StringData(id),
			"apiVersion":         llx.StringData(apiVersion),
			"kind":               llx.StringData(kind),
			"name":               llx.StringData(name),
			"namespace":          llx.StringData(namespace),
			"labels":             llx.MapData(labelsStr, types.String),
			"annotations":        llx.MapData(annotationsStr, types.String),
			"manifest":           llx.DictData(manifest),
			"isHook":             llx.BoolData(isHook),
			"hookTypes":          llx.ArrayData(hookTypes, types.String),
			"hookWeight":         llx.IntData(hookWeight),
			"hookDeletePolicies": llx.ArrayData(hookDeletePolicies, types.String),
			"isCRD":              llx.BoolData(isCRD),
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

// parseHookAnnotations reads Helm's hook annotations off a rendered
// resource's metadata.annotations: helm.sh/hook (lifecycle phases),
// helm.sh/hook-weight (ordering), and helm.sh/hook-delete-policy. A
// resource is a hook iff it carries the helm.sh/hook annotation.
func parseHookAnnotations(annotations map[string]any) (isHook bool, hookTypes []any, hookWeight int64, hookDeletePolicies []any) {
	hookTypes = []any{}
	hookDeletePolicies = []any{}

	hookVal, _ := annotations["helm.sh/hook"].(string)
	if strings.TrimSpace(hookVal) == "" {
		return false, hookTypes, 0, hookDeletePolicies
	}
	isHook = true
	hookTypes = splitCSV(hookVal)

	if w, ok := annotations["helm.sh/hook-weight"].(string); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(w)); err == nil {
			hookWeight = int64(n)
		}
	}
	if dp, ok := annotations["helm.sh/hook-delete-policy"].(string); ok {
		hookDeletePolicies = splitCSV(dp)
	}
	return isHook, hookTypes, hookWeight, hookDeletePolicies
}

// splitCSV splits a comma-separated annotation value into trimmed,
// non-empty entries.
func splitCSV(s string) []any {
	parts := strings.Split(s, ",")
	out := make([]any, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
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
