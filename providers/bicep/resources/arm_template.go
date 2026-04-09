// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
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
		"__id":           llx.StringData("bicep.template:" + filePath),
		"schema":         llx.StringData(tmpl.Schema),
		"contentVersion": llx.StringData(tmpl.ContentVersion),
	})
	if err != nil {
		return nil, err
	}
	mqlT := res.(*mqlBicepTemplate)
	mqlT.armTemplate = tmpl
	mqlT.cachePath = filePath
	return mqlT, nil
}

func (t *mqlBicepTemplate) id() (string, error) {
	return "bicep.template:" + t.cachePath, nil
}

func (t *mqlBicepTemplate) parameters() (map[string]any, error) {
	return rawMessageMapToDict(t.armTemplate.Parameters)
}

func (t *mqlBicepTemplate) variables() (map[string]any, error) {
	return rawMessageMapToDict(t.armTemplate.Variables)
}

func (t *mqlBicepTemplate) outputs() (map[string]any, error) {
	return rawMessageMapToDict(t.armTemplate.Outputs)
}

func (t *mqlBicepTemplate) resources() ([]any, error) {
	var mqlResources []any
	for i, raw := range t.armTemplate.Resources {
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

	propsRaw, _ := obj["properties"].(map[string]any)
	properties, _ := convert.JsonToDict(propsRaw)

	var dependsOn []any
	if deps, ok := obj["dependsOn"].([]any); ok {
		for _, d := range deps {
			if s, ok := d.(string); ok {
				dependsOn = append(dependsOn, s)
			}
		}
	}

	manifest, _ := convert.JsonToDict(obj)

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
