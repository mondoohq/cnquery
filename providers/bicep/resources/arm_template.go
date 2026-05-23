// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/bicep/connection"
	"go.mondoo.com/mql/v13/types"
)

type mqlBicepTemplateInternal struct {
	armTemplate *connection.ARMTemplate
	cachePath   string
}

func newMqlBicepTemplate(runtime *plugin.Runtime, filePath string, tmpl *connection.ARMTemplate) (*mqlBicepTemplate, error) {
	res, err := CreateResource(runtime, "bicep.template", map[string]*llx.RawData{
		"__id": llx.StringData("bicep.template:" + filePath),
	})
	if err != nil {
		return nil, err
	}
	mqlT := res.(*mqlBicepTemplate)
	mqlT.armTemplate = tmpl
	mqlT.cachePath = filePath
	return mqlT, nil
}

// id falls back to the connection's path when the Internal struct hasn't
// been populated yet (e.g., the resource came back via StoreData and no
// accessor has run getARMTemplate). If neither source yields a path we
// surface an error rather than returning a "bicep.template:" sentinel —
// a non-unique id would cause every such reconstruction to collide on
// the same cache entry.
func (t *mqlBicepTemplate) id() (string, error) {
	path := t.cachePath
	if path == "" {
		if conn, ok := t.MqlRuntime.Connection.(*connection.BicepConnection); ok {
			path = conn.Path()
		}
	}
	if path == "" {
		return "", errors.New("bicep.template: no path available to derive a stable cache id")
	}
	return "bicep.template:" + path, nil
}

func (t *mqlBicepTemplate) getARMTemplate() *connection.ARMTemplate {
	if t.armTemplate != nil {
		return t.armTemplate
	}
	// Re-fetch from connection when Internal struct wasn't populated
	// (happens when resource is reconstructed via StoreData across gRPC).
	conn := t.MqlRuntime.Connection.(*connection.BicepConnection)
	tmpl := conn.ARMTemplate()
	if tmpl != nil {
		t.armTemplate = tmpl
		t.cachePath = conn.Path()
	}
	return tmpl
}

func (t *mqlBicepTemplate) schema() (string, error) {
	tmpl := t.getARMTemplate()
	if tmpl == nil {
		return "", nil
	}
	return tmpl.Schema, nil
}

func (t *mqlBicepTemplate) contentVersion() (string, error) {
	tmpl := t.getARMTemplate()
	if tmpl == nil {
		return "", nil
	}
	return tmpl.ContentVersion, nil
}

// parameters/variables/outputs return an empty map (not nil) when the
// ARM template is unavailable. Returning nil here used to surface as
// `null` in MQL, which forced every audit to defensively check for
// missing-vs-empty; the connection layer already distinguishes "no ARM
// template at all" by returning a nil bicep.template resource.
func (t *mqlBicepTemplate) parameters() (map[string]any, error) {
	tmpl := t.getARMTemplate()
	if tmpl == nil {
		return map[string]any{}, nil
	}
	return rawMessageMapToDict(tmpl.Parameters)
}

func (t *mqlBicepTemplate) variables() (map[string]any, error) {
	tmpl := t.getARMTemplate()
	if tmpl == nil {
		return map[string]any{}, nil
	}
	return rawMessageMapToDict(tmpl.Variables)
}

func (t *mqlBicepTemplate) outputs() (map[string]any, error) {
	tmpl := t.getARMTemplate()
	if tmpl == nil {
		return map[string]any{}, nil
	}
	return rawMessageMapToDict(tmpl.Outputs)
}

func (t *mqlBicepTemplate) resources() ([]any, error) {
	tmpl := t.getARMTemplate()
	if tmpl == nil {
		return nil, nil
	}
	var mqlResources []any
	for i, raw := range tmpl.Resources {
		var obj map[string]any
		if err := json.Unmarshal(raw, &obj); err != nil {
			log.Warn().Err(err).Int("index", i).Msg("failed to unmarshal ARM template resource")
			continue
		}
		mqlR, err := newMqlBicepTemplateResource(t.MqlRuntime, t.cachePath, i, obj)
		if err != nil {
			log.Warn().Err(err).Int("index", i).Msg("failed to create ARM template resource")
			continue
		}
		mqlResources = append(mqlResources, mqlR)
	}
	return mqlResources, nil
}

func newMqlBicepTemplateResource(runtime *plugin.Runtime, templatePath string, index int, obj map[string]any) (*mqlBicepTemplateResource, error) {
	typ, _ := obj["type"].(string)
	apiVersion, _ := obj["apiVersion"].(string)
	name, _ := obj["name"].(string)
	location, _ := obj["location"].(string)

	var dependsOn []any
	if deps, ok := obj["dependsOn"].([]any); ok {
		for _, d := range deps {
			if s, ok := d.(string); ok {
				dependsOn = append(dependsOn, s)
			}
		}
	}

	// Convert the manifest once, then peel `properties` out of the
	// already-converted dict. JsonToDict walks every nested value, so a
	// separate pass over obj["properties"] was redoing the same work.
	manifest, _ := convert.JsonToDict(obj)
	var properties any
	if manifest != nil {
		properties = manifest["properties"]
	}

	id := "bicep.template.resource:" + templatePath + ":" + typ + ":" + name + ":" + strconv.Itoa(index)
	res, err := CreateResource(runtime, "bicep.template.resource", map[string]*llx.RawData{
		"__id":       llx.StringData(id),
		"type":       llx.StringData(typ),
		"apiVersion": llx.StringData(apiVersion),
		"name":       llx.StringData(name),
		"location":   llx.StringData(location),
		"properties": llx.DictData(properties),
		"dependsOn":  llx.ArrayData(dependsOn, types.String),
		"manifest":   llx.DictData(manifest),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlBicepTemplateResource), nil
}

func rawMessageMapToDict(m map[string]json.RawMessage) (map[string]any, error) {
	result := make(map[string]any, len(m))
	for k, v := range m {
		var val any
		if err := json.Unmarshal(v, &val); err != nil {
			result[k] = string(v)
			continue
		}
		dict, err := convert.JsonToDict(val)
		if err != nil {
			result[k] = val
			continue
		}
		result[k] = dict
	}
	return result, nil
}
