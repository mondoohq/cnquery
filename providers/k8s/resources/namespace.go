// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"sync"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
)

type mqlK8sNamespaceInternal struct {
	lock sync.Mutex
	obj  *corev1.Namespace
}

func initK8sNamespace(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initResource[*mqlK8sNamespace](runtime, args, func(k *mqlK8s) *plugin.TValue[[]interface{}] { return k.GetNamespaces() })
}

func (k *mqlK8s) namespaces() ([]interface{}, error) {
	kp, err := k8sProvider(k.MqlRuntime.Connection)
	if err != nil {
		return nil, err
	}

	nss, err := kp.Namespaces()
	if err != nil {
		return nil, err
	}

	resp := make([]interface{}, 0, len(nss))
	for _, ns := range nss {
		ts := ns.GetCreationTimestamp()

		manifest, err := convert.JsonToDict(ns)
		if err != nil {
			return nil, err
		}

		objT, err := meta.TypeAccessor(&ns)
		if err != nil {
			log.Error().Err(err).Msg("could not access object attributes")
			return nil, err
		}

		r, err := CreateResource(k.MqlRuntime, "k8s.namespace", map[string]*llx.RawData{
			"id":       llx.StringData(objIdFromK8sObj(&ns.ObjectMeta, objT)),
			"uid":      llx.StringData(string(ns.UID)),
			"name":     llx.StringData(ns.Name),
			"created":  llx.TimeData(ts.Time),
			"manifest": llx.DictData(manifest),
			"kind":     llx.StringData(ns.Kind),
		})
		if err != nil {
			return nil, err
		}

		r.(*mqlK8sNamespace).obj = &ns
		resp = append(resp, r)
	}
	return resp, nil
}

func (k *mqlK8sNamespace) id() (string, error) {
	return k.Id.Data, nil
}

func (k *mqlK8sNamespace) annotations() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetAnnotations()), nil
}

func (k *mqlK8sNamespace) labels() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetLabels()), nil
}
