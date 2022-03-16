package resources

import (
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubectl/pkg/scheme"
)

func UnstructuredToObject(obj *unstructured.Unstructured) (runtime.Object, error) {
	json, err := runtime.Encode(unstructured.UnstructuredJSONScheme, obj)
	if err != nil {
		return nil, err
	}

	// UniversalDecoder call must specify parameter versions; otherwise it will decode to internal versions.
	decoder := scheme.Codecs.UniversalDecoder(scheme.Scheme.PrioritizedVersionsAllGroups()...)
	return runtime.Decode(decoder, json)
}

var skipResources = []string{"APIService", "CustomResourceDefinition"}

func skip(name string) bool {
	for i := range skipResources {
		if skipResources[i] == name {
			return true
		}
	}
	return false
}

func UnstructuredListToObjectList(list []unstructured.Unstructured) ([]runtime.Object, error) {
	out := []runtime.Object{}
	for i := range list {

		unstructured := list[i]

		if skip(unstructured.GetName()) {
			continue
		}

		d, err := UnstructuredToObject(&unstructured)
		if err != nil {
			log.Debug().Err(err).Str("name", unstructured.GetName()).Msg("error during conversion")
			continue
		}
		out = append(out, d)
	}
	return out, nil
}
