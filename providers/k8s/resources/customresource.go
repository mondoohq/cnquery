// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"sync"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type mqlK8sCustomresourceInternal struct {
	lock sync.Mutex
	obj  metav1.Object
}

func (k *mqlK8s) customresources() ([]interface{}, error) {
	kt, err := k8sProvider(k.MqlRuntime.Connection)
	if err != nil {
		return nil, err
	}

	result, err := kt.Resources("CustomResourceDefinition", "", "")
	if err != nil {
		return nil, err
	}

	resp := []interface{}{}
	for i := range result.Resources {
		resource := result.Resources[i]

		// resource.
		crd, err := meta.Accessor(resource)
		if err != nil {
			log.Error().Err(err).Msg("could not access object attributes")
			return nil, err
		}

		mqlResources, err := k8sResourceToMql(k.MqlRuntime, crd.GetName(), func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
			ts := obj.GetCreationTimestamp()

			r, err := CreateResource(k.MqlRuntime, "k8s.customresource", map[string]*llx.RawData{
				"id":              llx.StringData(objIdFromK8sObj(obj, objT)),
				"uid":             llx.StringData(string(obj.GetUID())),
				"resourceVersion": llx.StringData(obj.GetResourceVersion()),
				"name":            llx.StringData(obj.GetName()),
				"namespace":       llx.StringData(obj.GetNamespace()),
				"kind":            llx.StringData(objT.GetKind()),
				"created":         llx.TimeData(ts.Time),
			})
			if err != nil {
				log.Error().Err(err).Msg("couldn't create resource")
				return nil, err
			}
			r.(*mqlK8sCustomresource).obj = obj
			return r, nil
		})
		resp = append(resp, mqlResources...)
	}
	return resp, nil
}

func (k *mqlK8sCustomresource) manifest() (map[string]interface{}, error) {
	manifest, err := convert.JsonToDict(k.obj)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

func (k *mqlK8sCustomresource) id() (string, error) {
	return k.Id.Data, nil
}

func (k *mqlK8sCustomresource) annotations() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetAnnotations()), nil
}

func (k *mqlK8sCustomresource) labels() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetLabels()), nil
}
