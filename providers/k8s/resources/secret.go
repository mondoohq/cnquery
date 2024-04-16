// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"sync"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type mqlK8sSecretInternal struct {
	lock    sync.Mutex
	obj     *corev1.Secret
	metaObj metav1.Object
}

func (k *mqlK8s) secrets() ([]interface{}, error) {
	return k8sResourceToMql(k.MqlRuntime, "secrets.v1.", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		s, ok := resource.(*corev1.Secret)
		if !ok {
			return nil, errors.New("not a k8s secret")
		}

		r, err := CreateResource(k.MqlRuntime, "k8s.secret", map[string]*llx.RawData{
			"id":              llx.StringData(objIdFromK8sObj(obj, objT)),
			"uid":             llx.StringData(string(obj.GetUID())),
			"resourceVersion": llx.StringData(obj.GetResourceVersion()),
			"name":            llx.StringData(obj.GetName()),
			"namespace":       llx.StringData(obj.GetNamespace()),
			"kind":            llx.StringData(objT.GetKind()),
			"created":         llx.TimeData(ts.Time),
			"type":            llx.StringData(string(s.Type)),
		})
		if err != nil {
			return nil, err
		}
		r.(*mqlK8sSecret).obj = s
		r.(*mqlK8sSecret).metaObj = obj
		return r, nil
	})
}

func (k *mqlK8sSecret) manifest() (map[string]interface{}, error) {
	manifest, err := convert.JsonToDict(k.obj)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

func (k *mqlK8sSecret) id() (string, error) {
	return k.Id.Data, nil
}

func initK8sSecret(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initNamespacedResource[*mqlK8sSecret](runtime, args, func(k *mqlK8s) *plugin.TValue[[]interface{}] { return k.GetSecrets() })
}

func (k *mqlK8sSecret) annotations() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetAnnotations()), nil
}

func (k *mqlK8sSecret) labels() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetLabels()), nil
}

func (k *mqlK8sSecret) certificates() ([]interface{}, error) {
	if k.obj.Type != corev1.SecretTypeTLS {
		// this is not an error, it just does not contain a certificate
		return nil, nil
	}

	certRawData, ok := k.obj.Data["tls.crt"]
	if !ok {
		return nil, errors.New("could not find the 'tls.crt' key")
	}

	c, err := k.MqlRuntime.CreateSharedResource("certificates", map[string]*llx.RawData{
		"pem": llx.StringData(string(certRawData)),
	})
	if err != nil {
		return nil, err
	}

	list, err := k.MqlRuntime.GetSharedData("certificates", c.MqlID(), "list")
	if err != nil {
		return nil, err
	}

	return list.Value.([]interface{}), nil
}
