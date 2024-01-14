// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/types"
)

func (k *mqlK8s) apiResources() ([]interface{}, error) {
	kt, err := k8sProvider(k.MqlRuntime.Connection)
	if err != nil {
		return nil, err
	}

	resources, err := kt.SupportedResourceTypes()
	if err != nil {
		return nil, err
	}

	// convert to MQL resources
	list := resources.Resources()
	resp := []interface{}{}
	for i := range list {
		entry := list[i]

		mqlK8SResource, err := CreateResource(k.MqlRuntime, "k8s.apiresource", map[string]*llx.RawData{
			"name":         llx.StringData(entry.Resource.Name),
			"singularName": llx.StringData(entry.Resource.SingularName),
			"namespaced":   llx.BoolData(entry.Resource.Namespaced),
			"group":        llx.StringData(entry.GroupVersion.Group),
			"version":      llx.StringData(entry.GroupVersion.Version),
			"kind":         llx.StringData(entry.Resource.Kind),
			"shortNames":   llx.ArrayData(convert.SliceAnyToInterface(entry.Resource.ShortNames), types.String),
			"categories":   llx.ArrayData(convert.SliceAnyToInterface(entry.Resource.Categories), types.String),
		})
		if err != nil {
			return nil, err
		}
		resp = append(resp, mqlK8SResource)
	}

	return resp, nil
}

func (k *mqlK8sApiresource) id() (string, error) {
	return fmt.Sprintf("%s.%s", k.Version.Data, k.Name.Data), nil
}
