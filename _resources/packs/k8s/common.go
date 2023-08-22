// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package k8s

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/providers"
	k8s_provider "go.mondoo.com/cnquery/motor/providers/k8s"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func k8sProvider(t providers.Instance) (k8s_provider.KubernetesProvider, error) {
	at, ok := t.(k8s_provider.KubernetesProvider)
	if !ok {
		return nil, errors.New("k8s resource is not supported on this provider")
	}
	return at, nil
}

func k8sMetaObject(mqlResource *resources.Resource) (metav1.Object, error) {
	entry, ok := mqlResource.Cache.Load("_resource")
	if !ok {
		return nil, errors.New("cannot get resource from cache")
	}

	obj, ok := entry.Data.(runtime.Object)
	if !ok {
		return nil, errors.New("cannot get resource from cache")
	}

	return meta.Accessor(obj)
}

func k8sAnnotations(mqlResource *resources.Resource) (interface{}, error) {
	objM, err := k8sMetaObject(mqlResource)
	if err != nil {
		return nil, err
	}
	return core.StrMapToInterface(objM.GetAnnotations()), nil
}

func k8sLabels(mqlResource *resources.Resource) (interface{}, error) {
	objM, err := k8sMetaObject(mqlResource)
	if err != nil {
		return nil, err
	}
	return core.StrMapToInterface(objM.GetLabels()), nil
}

type resourceConvertFn func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error)

func k8sResourceToMql(r *resources.Runtime, kind string, fn resourceConvertFn) ([]interface{}, error) {
	kt, err := k8sProvider(r.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ns := ""
	platformId, err := kt.Identifier()
	if err == nil && strings.HasPrefix(platformId, "//platformid.api.mondoo.app/runtime/k8s/namespace/") {
		ns = strings.Split(strings.TrimPrefix(platformId, "//platformid.api.mondoo.app/runtime/k8s/namespace/"), "/")[0]
	}

	// TODO: check if we are running in a namespace scope and retrieve the ns from the provider
	result, err := kt.Resources(kind, "", ns)
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

func getNameAndNamespace(runtime *resources.Runtime) (string, string, error) {
	asset := runtime.Motor.GetAsset()
	if asset == nil || asset.Labels == nil {
		return getPlatformIdentifierElements(runtime.Motor.Provider)
	}
	return asset.Labels["k8s.mondoo.com/name"], asset.Labels["k8s.mondoo.com/namespace"], nil
}

// This function is now (v6.17.0) no longer needed
// Perhaps remove it in v8?
func getPlatformIdentifierElements(transport providers.Instance) (string, string, error) {
	kt, err := k8sProvider(transport)
	if err != nil {
		return "", "", err
	}

	identifier, err := kt.Identifier()
	if err != nil {
		return "", "", err
	}

	var identifierName string
	var identifierNamespace string
	splitIdentifier := strings.Split(identifier, "/")
	arrayLength := len(splitIdentifier)
	if arrayLength >= 1 {
		identifierName = splitIdentifier[arrayLength-1]
	}
	if arrayLength >= 4 {
		identifierNamespace = splitIdentifier[arrayLength-4]
	}

	return identifierName, identifierNamespace, nil
}

type K8sNamespacedObject interface {
	K8sObject
	Namespace() (string, error)
}

type K8sObject interface {
	Id() (string, error)
	Kind() (string, error)
	Name() (string, error)
	Manifest() (interface{}, error)
}

func objId(o K8sNamespacedObject) (string, error) {
	kind, err := o.Kind()
	if err != nil {
		return "", err
	}

	name, err := o.Name()
	if err != nil {
		return "", err
	}

	namespace, err := o.Namespace()
	if err != nil {
		return "", err
	}

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
	args *resources.Args, runtime *resources.Runtime, r func(k8s K8s) ([]interface{}, error),
) (*resources.Args, T, error) {
	// pass-through if all args are already provided
	if len(*args) > 2 {
		return args, *new(T), nil
	}

	// get platform identifier infos
	identifierName, identifierNamespace, err := getNameAndNamespace(runtime)
	if err != nil {
		return args, *new(T), nil
	}

	// search for existing resources if id or name/namespace is provided
	obj, err := runtime.CreateResource("k8s")
	if err != nil {
		return args, *new(T), err
	}
	k8sResource := obj.(K8s)

	nsResources, err := r(k8sResource)
	if err != nil {
		return args, *new(T), err
	}

	var matchFn func(nsR T) bool

	var idRaw string
	if _, ok := (*args)["id"]; ok {
		idRaw = (*args)["id"].(string)
	}

	if idRaw != "" {
		matchFn = func(nsR T) bool {
			id, _ := nsR.Id()
			return id == idRaw
		}
	}

	var nameRaw string
	var namespaceRaw string
	if _, ok := (*args)["name"]; ok {
		nameRaw = (*args)["name"].(string)
	}
	if _, ok := (*args)["namespace"]; ok {
		namespaceRaw = (*args)["namespace"].(string)
	}
	if nameRaw == "" {
		nameRaw = identifierName
		namespaceRaw = identifierNamespace
	}
	if nameRaw != "" {
		matchFn = func(nsR T) bool {
			name, _ := nsR.Name()
			namespace, _ := nsR.Namespace()
			return name == nameRaw && namespace == namespaceRaw
		}
	}

	if matchFn == nil {
		return args, *new(T), fmt.Errorf("cannot use resource without specifying id or name/namespace")
	}

	for i := range nsResources {
		nsR := nsResources[i].(T)
		if matchFn(nsR) {
			return args, nsR, nil
		}
	}

	// the error ResourceNotFound is checked by cnspec
	return args, *new(T), &resources.ResourceNotFound{}
}

func initResource[T K8sObject](
	args *resources.Args, runtime *resources.Runtime, r func(k8s K8s) ([]interface{}, error),
) (*resources.Args, T, error) {
	// pass-through if all args are already provided
	if len(*args) > 1 {
		return args, *new(T), nil
	}

	// get platform identifier infos
	identifierName, _, err := getNameAndNamespace(runtime)
	if err != nil {
		return args, *new(T), nil
	}

	// search for existing resources if id or name is provided
	obj, err := runtime.CreateResource("k8s")
	if err != nil {
		return nil, *new(T), err
	}
	k8sResource := obj.(K8s)

	k8sResources, err := r(k8sResource)
	if err != nil {
		return nil, *new(T), err
	}

	var matchFn func(entry T) bool

	idRaw := (*args)["id"]
	if idRaw != nil {
		matchFn = func(entry T) bool {
			id, _ := entry.Id()
			if id == idRaw.(string) {
				return true
			}
			return false
		}
	}

	var nameRaw string
	if _, ok := (*args)["name"]; ok {
		nameRaw = (*args)["name"].(string)
	}
	if nameRaw == "" {
		nameRaw = identifierName
	}
	if nameRaw != "" {
		matchFn = func(nsR T) bool {
			name, _ := nsR.Name()
			return name == nameRaw
		}
	}

	if matchFn == nil {
		return args, *new(T), fmt.Errorf("cannot use resource without specifying id or name")
	}

	for i := range k8sResources {
		entry := k8sResources[i].(T)
		if matchFn(entry) {
			return nil, entry, nil
		}
	}

	// the error ResourceNotFound is checked by cnspec
	return nil, *new(T), &resources.ResourceNotFound{}
}
