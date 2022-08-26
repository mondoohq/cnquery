package k8s

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (k *mqlK8s) GetCustomresources() ([]interface{}, error) {
	kt, err := k8sProvider(k.MotorRuntime.Motor.Provider)
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

		mqlResources, err := k8sResourceToMql(k.MotorRuntime, crd.GetName(), func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
			ts := obj.GetCreationTimestamp()

			manifest, err := core.JsonToDict(resource)
			if err != nil {
				log.Error().Err(err).Msg("couldn't convert resource to json dict")
				return nil, err
			}

			r, err := k.MotorRuntime.CreateResource("k8s.customresource",
				"id", objIdFromK8sObj(obj, objT),
				"uid", string(obj.GetUID()),
				"resourceVersion", obj.GetResourceVersion(),
				"name", obj.GetName(),
				"namespace", obj.GetNamespace(),
				"kind", objT.GetKind(),
				"created", &ts.Time,
				"manifest", manifest,
			)
			if err != nil {
				log.Error().Err(err).Msg("couldn't create resource")
				return nil, err
			}
			r.MqlResource().Cache.Store("_resource", &resources.CacheEntry{Data: resp})
			return r, nil
		})
		resp = append(resp, mqlResources...)
	}
	return resp, nil
}

func (k *mqlK8sCustomresource) id() (string, error) {
	return k.Id()
}

func (k *mqlK8sCustomresource) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.MqlResource())
}

func (k *mqlK8sCustomresource) GetLabels() (interface{}, error) {
	return k8sLabels(k.MqlResource())
}
