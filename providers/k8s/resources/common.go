// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/k8s/connection/shared"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func k8sProvider(t plugin.Connection) (shared.Connection, error) {
	at, ok := t.(shared.Connection)
	if !ok {
		return nil, errors.New("k8s resource is not supported on this provider")
	}
	return at, nil
}

type resourceConvertFn func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error)

func k8sResourceToMql(r *plugin.Runtime, kind string, fn resourceConvertFn) ([]interface{}, error) {
	kt, err := k8sProvider(r.Connection)
	if err != nil {
		return nil, err
	}

	// TODO: check if we are running in a namespace scope and retrieve the ns from the provider
	result, err := kt.Resources(kind, "", "")
	if err != nil {
		return nil, err
	}

	resp := []interface{}{}
	for i := range result.Resources {
		resource := result.Resources[i]

		obj, err := meta.Accessor(resource)
		if err != nil {
			log.Error().Err(err).Msg("could not access object attributes")
			return nil, err
		}
		objT, err := meta.TypeAccessor(resource)
		if err != nil {
			log.Error().Err(err).Msg("could not access object attributes")
			return nil, err
		}

		mqlK8sResource, err := fn(kind, resource, obj, objT)
		if err != nil {
			return nil, err
		}

		resp = append(resp, mqlK8sResource)
	}

	return resp, nil
}

func getNameAndNamespace(runtime *plugin.Runtime) (string, string, error) {
	asset := runtime.Connection.(shared.Connection).Asset()
	return asset.Labels["k8s.mondoo.com/name"], asset.Labels["k8s.mondoo.com/namespace"], nil
}

type K8sNamespacedObject interface {
	K8sObject
	GetNamespace() *plugin.TValue[string]
}

type K8sObject interface {
	plugin.Resource
	GetId() *plugin.TValue[string]
	GetKind() *plugin.TValue[string]
	GetName() *plugin.TValue[string]
}

func objId(o runtime.Object, meta metav1.Object) (string, error) {
	kind := o.GetObjectKind().GroupVersionKind().Kind
	name := meta.GetName()
	namespace := meta.GetNamespace()

	return objIdFromFields(kind, namespace, name), nil
}

func objIdFromK8sObj(o metav1.Object, objT metav1.Type) string {
	return objIdFromFields(objT.GetKind(), o.GetNamespace(), o.GetName())
}

func objIdFromFields(kind, namespace, name string) string {
	// Kind is usually capitalized. Make it all lower case for readability
	if namespace == "" {
		return fmt.Sprintf("%s:%s", strings.ToLower(kind), name)
	}
	return fmt.Sprintf("%s:%s:%s", strings.ToLower(kind), namespace, name)
}

func initNamespacedResource[T K8sNamespacedObject](
	runtime *plugin.Runtime, args map[string]*llx.RawData, r func(k8s *mqlK8s) *plugin.TValue[[]interface{}],
) (map[string]*llx.RawData, plugin.Resource, error) {
	// pass-through if all args are already provided
	if len(args) > 2 {
		return args, nil, nil
	}

	// get platform identifier infos
	identifierName, identifierNamespace, err := getNameAndNamespace(runtime)
	if err != nil {
		return args, nil, nil
	}

	// search for existing resources if id or name/namespace is provided
	obj, err := CreateResource(runtime, "k8s", nil)
	if err != nil {
		return args, nil, err
	}
	k8s := obj.(*mqlK8s)

	nsResources := r(k8s)
	if nsResources.Error != nil {
		return args, nil, nsResources.Error
	}

	var matchFn func(nsR T) bool

	var idRaw string
	if _, ok := args["id"]; ok {
		idRaw = args["id"].Value.(string)
	}

	if idRaw != "" {
		matchFn = func(nsR T) bool {
			return nsR.GetId().Data == idRaw
		}
	}

	var nameRaw string
	var namespaceRaw string
	if _, ok := args["name"]; ok {
		nameRaw = args["name"].Value.(string)
	}
	if _, ok := args["namespace"]; ok {
		namespaceRaw = args["namespace"].Value.(string)
	}
	if nameRaw == "" {
		nameRaw = identifierName
		namespaceRaw = identifierNamespace
	}
	if nameRaw != "" {
		matchFn = func(nsR T) bool {
			name := nsR.GetName().Data
			namespace := nsR.GetNamespace().Data
			return name == nameRaw && namespace == namespaceRaw
		}
	}

	if matchFn == nil {
		return args, nil, fmt.Errorf("cannot use resource without specifying id or name/namespace")
	}

	for i := range nsResources.Data {
		nsR := nsResources.Data[i].(T)
		if matchFn(nsR) {
			return args, nsR, nil
		}
	}

	// the error ResourceNotFound is checked by cnspec
	return args, nil, errors.New("not found")
}

func initResource[T K8sObject](
	runtime *plugin.Runtime, args map[string]*llx.RawData, r func(k8s *mqlK8s) *plugin.TValue[[]interface{}],
) (map[string]*llx.RawData, plugin.Resource, error) {
	// pass-through if all args are already provided
	if len(args) > 1 {
		return args, nil, nil
	}

	// get platform identifier infos
	identifierName, _, err := getNameAndNamespace(runtime)
	if err != nil {
		return args, nil, nil
	}

	// search for existing resources if id or name is provided
	obj, err := CreateResource(runtime, "k8s", nil)
	if err != nil {
		return args, nil, err
	}
	k8s := obj.(*mqlK8s)

	k8sResources := r(k8s)
	if k8sResources.Error != nil {
		return nil, nil, k8sResources.Error
	}

	var matchFn func(entry T) bool

	idRaw := args["id"]
	if idRaw != nil {
		matchFn = func(entry T) bool {
			if entry.GetId().Data == idRaw.Value.(string) {
				return true
			}
			return false
		}
	}

	var nameRaw string
	if _, ok := args["name"]; ok {
		nameRaw = args["name"].Value.(string)
	}
	if nameRaw == "" {
		nameRaw = identifierName
	}
	if nameRaw != "" {
		matchFn = func(nsR T) bool {
			return nsR.GetName().Data == nameRaw
		}
	}

	if matchFn == nil {
		return args, *new(T), fmt.Errorf("cannot use resource without specifying id or name")
	}

	for i := range k8sResources.Data {
		entry := k8sResources.Data[i].(T)
		if matchFn(entry) {
			return nil, entry, nil
		}
	}

	// the error ResourceNotFound is checked by cnspec
	return nil, nil, errors.New("not found")
}
